package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/autokeren/kancil/internal/agent"
	"github.com/autokeren/kancil/internal/channels/messenger"
	"github.com/autokeren/kancil/internal/config"
	"github.com/autokeren/kancil/internal/sitefs"
)

type Server struct {
	Cfg    *config.Config
	Port   int
	Sites  *sitefs.Store
	msg    *messenger.Sender
	agents map[string]*agent.Agent
	mu     sync.Mutex
	http   *http.Server
}

func New(cfg *config.Config, sites *sitefs.Store) *Server {
	return &Server{
		Cfg:    cfg,
		Port:   cfg.Port,
		Sites:  sites,
		agents: map[string]*agent.Agent{},
	}
}

// PublicBaseURL returns the address sites are reachable at.
func (s *Server) PublicBaseURL() string {
	return fmt.Sprintf("http://localhost:%d", s.Port)
}

func (s *Server) Start(ctx context.Context, mkAgent func(userID string) *agent.Agent) error {
	mux := http.NewServeMux()

	// sites
	mux.Handle("/sites/", http.StripPrefix("/sites/", s.sitesHandler()))

	// messenger webhook
	mux.HandleFunc("/webhook/messenger", func(w http.ResponseWriter, r *http.Request) {
		s.handleMessengerWebhook(w, r, mkAgent)
	})

	// whatsapp webhook
	mux.HandleFunc("/webhook/whatsapp", func(w http.ResponseWriter, r *http.Request) {
		s.handleWhatsAppWebhook(w, r, mkAgent)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "🦌 kancil is alive")
	})

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return err
	}
	s.http = &http.Server{Handler: mux}
	go func() { <-ctx.Done(); _ = s.http.Close() }()
	fmt.Printf("  🦌 kancil serving on :%d (webhooks: /webhook/messenger, /webhook/whatsapp; sites: /sites/)\n", s.Port)
	return s.http.Serve(ln)
}

func (s *Server) sitesHandler() http.Handler {
	root := filepath.Join(s.Cfg.Workspace, "sites")
	fs := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// never list parent dirs
		p := r.URL.Path
		if strings.Contains(p, "..") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		fs.ServeHTTP(w, r)
	})
}

func (s *Server) agentFor(userID string, mk func(string) *agent.Agent) *agent.Agent {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[userID]
	if !ok {
		a = mk(userID)
		s.agents[userID] = a
	}
	return a
}

func (s *Server) handleMessengerWebhook(w http.ResponseWriter, r *http.Request, mk func(string) *agent.Agent) {
	// verification handshake
	if r.Method == http.MethodGet {
		hub := r.URL.Query()
		if hub.Get("hub.mode") == "subscribe" && hub.Get("hub.verify_token") == s.Cfg.Messenger.VerifyToken {
			fmt.Fprint(w, hub.Get("hub.challenge"))
			return
		}
		http.Error(w, "verify token mismatch", http.StatusForbidden)
		return
	}

	events, err := messenger.ParseWebhook(r.Body)
	if err != nil {
		http.Error(w, "bad payload", http.StatusBadRequest)
		return
	}
	for _, ev := range events {
		if ev.Text == "" {
			continue
		}
		a := s.agentFor(ev.SenderID, mk)
		go func(ev messenger.Event) {
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			defer cancel()
			reply, err := a.Handle(ctx, ev.Text)
			if err != nil {
				reply = "ampun, error: " + err.Error()
			}
			s.sendMessenger(ev.SenderID, reply)
		}(ev)
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) sendMessenger(recipient, text string) {
	if s.msg == nil {
		s.msg = messenger.NewSender(s.Cfg.Messenger.AccessToken)
	}
	if s.msg == nil || s.Cfg.Messenger.AccessToken == "" {
		return
	}
	_ = s.msg.Send(recipient, text)
}

func (s *Server) handleWhatsAppWebhook(w http.ResponseWriter, r *http.Request, mk func(string) *agent.Agent) {
	if r.Method == http.MethodGet {
		q := r.URL.Query()
		if q.Get("hub.verify_token") == s.Cfg.WhatsApp.VerifyToken {
			fmt.Fprint(w, q.Get("hub.challenge"))
			return
		}
		http.Error(w, "verify", http.StatusForbidden)
		return
	}
	// WhatsApp Cloud API events handled in D3
	w.WriteHeader(http.StatusOK)
}
