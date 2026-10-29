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
	"github.com/autokeren/kancil/internal/meta"
	"github.com/autokeren/kancil/internal/model"
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
		Cfg:             cfg,
		Port:            cfg.Port,
		Sites:           sites,
		agents: map[string]*agent.Agent{},
	}
}

// PublicBaseURL returns the address sites are reachable at.
func (s *Server) PublicBaseURL() string {
	return fmt.Sprintf("http://localhost:%d", s.Port)
}

func (s *Server) Start(ctx context.Context, mkAgent func(userID string, onImage func(string)) *agent.Agent) error {
	mux := http.NewServeMux()

	// sites
	mux.Handle("/sites/", http.StripPrefix("/sites/", s.sitesHandler()))
	mux.Handle("/images/", http.StripPrefix("/images/", s.imagesHandler()))

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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func firstLine(s string, n int) string {
	for i, r := range s {
		if r == '\n' {
			return truncateStr(s[:i], n)
		}
	}
	return truncateStr(s, n)
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (s *Server) imagesHandler() http.Handler {
	root := filepath.Join(s.Cfg.Workspace, "images")
	return http.FileServer(http.Dir(root))
}

func looksLikeBuildTask(text string) bool {
	t := strings.ToLower(text)
	words := []string{"bikin", "buat", "buatkan", "membuat", "gambar", "desain", "build", "create", "make", "draw", "design", "generate", "tulis", "render", "bikinin", "buatin"}
	for _, w := range words {
		if strings.Contains(t, w) {
			return true
		}
	}
	return false
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

func (s *Server) agentFor(userID string, mk func(string, func(string)) *agent.Agent) *agent.Agent {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.agents[userID]
	if !ok {
		a = mk(userID, func(url string) { s.sendMessengerImage(userID, url) })
		if s.Cfg.MemoryDir != "" {
			a.LoadHistory(filepath.Join(s.Cfg.MemoryDir, userID+".json"))
		}
		s.agents[userID] = a
	}
	return a
}

func (s *Server) handleMessengerWebhook(w http.ResponseWriter, r *http.Request, mk func(string, func(string)) *agent.Agent) {
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
		fmt.Printf("→ [%s] %s\n", ev.SenderID[:min(8, len(ev.SenderID))], ev.Text)
		a := s.agentFor(ev.SenderID, mk)
		go func(ev messenger.Event) {
			// typing indicator ON + auto-refresh (muse spark reasoning bisa lama)
			s.sendAction(ev.SenderID, "typing_on")
			ticker := time.NewTicker(12 * time.Second)
			done := make(chan struct{})
			defer close(done)
			go func() {
				for {
					select {
					case <-ticker.C:
						s.sendAction(ev.SenderID, "typing_on")
					case <-done:
						ticker.Stop()
						return
					}
				}
			}()

			// state per-message: kalau reply utama udah kekirim, drop ack
			var muAck sync.Mutex
			replySent := false

			// ── MUSE ACK (parallel): model yang mutusin + nulis ──
			go func() {
				ackCtx, ackCancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer ackCancel()
				ackClient := meta.New(s.Cfg.MetaAPIKey, s.Cfg.BaseURL)
				ackReq := []model.Message{
					{Role: "system", Content: "You are Kancil's intent router in a chat app. If the user's message asks to BUILD/CREATE/MAKE something (website, image, file, report), write ONE short friendly sentence (max 12 words) confirming you're working on it — same language as the user, one emoji. If it's just a question/chat/greeting, reply with exactly: SKIP"},
					{Role: "user", Content: ev.Text},
				}
				resp, err := ackClient.Complete(ackCtx, meta.CompleteOptions{Messages: ackReq, MaxTokens: 3000})
				if err != nil {
					return
				}
				text := strings.TrimSpace(resp.Content)
				if text == "" || strings.Contains(strings.ToUpper(text), "SKIP") {
					return
				}
				muAck.Lock()
				if replySent {
					muAck.Unlock()
					return
				}
				muAck.Unlock()
				fmt.Printf("← ack (muse): %s\n", firstLine(text, 60))
				s.sendMessenger(ev.SenderID, text)
			}()

			// ── MAIN AGENT TASK ──
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
			defer cancel()
			reply, err := a.Handle(ctx, ev.Text)
			if err != nil {
				reply = "ampun, error: " + err.Error()
			}
			muAck.Lock()
			replySent = true
			muAck.Unlock()
			fmt.Printf("← reply: %s\n", firstLine(reply, 90))
			s.sendMessenger(ev.SenderID, reply)
			if s.Cfg.MemoryDir != "" {
				a.PersistToFile(filepath.Join(s.Cfg.MemoryDir, ev.SenderID+".json"))
			}
		}(ev)
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) sendAction(recipient, action string) {
	if s.msg == nil {
		s.msg = messenger.NewSender(s.Cfg.Messenger.AccessToken)
	}
	_ = s.msg.Action(recipient, action)
}

func (s *Server) sendMessengerImage(recipient, imageURL string) {
	if s.msg == nil {
		s.msg = messenger.NewSender(s.Cfg.Messenger.AccessToken)
	}
	if s.Cfg.Messenger.AccessToken == "" {
		return
	}
	if err := s.msg.SendImage(recipient, imageURL); err != nil {
		fmt.Printf("⚠ image send: %v\n", err)
	} else {
		fmt.Printf("🖼 image sent to chat: %s\n", imageURL)
	}
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

func (s *Server) handleWhatsAppWebhook(w http.ResponseWriter, r *http.Request, mk func(string, func(string)) *agent.Agent) {
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
