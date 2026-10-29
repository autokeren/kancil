package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/autokeren/kancil/internal/agent"
	"github.com/autokeren/kancil/internal/config"
	"github.com/autokeren/kancil/internal/meta"
	"github.com/autokeren/kancil/internal/server"
	"github.com/autokeren/kancil/internal/sitefs"
	"github.com/autokeren/kancil/internal/tools"
	"github.com/autokeren/kancil/internal/wizard"
)

const version = "0.1.0"

func main() {
	args := os.Args[1:]
	switch {
	case len(args) == 0:
		runDefault()
	case args[0] == "chat":
		runChat()
	case args[0] == "serve":
		runServe()
	case args[0] == "wizard":
		runWizard()
	case args[0] == "--version":
		fmt.Println("kancil", version)
	default:
		fmt.Println("kancil — the clever agent in your Messenger 🦌")
		fmt.Println("  kancil        setup wizard + serve")
		fmt.Println("  kancil chat   chat locally in terminal")
		fmt.Println("  kancil serve  run webhook server only")
	}
}

func ensureConfig() (*config.Config, bool) {
	cfg, _ := config.Load()
	if cfg != nil && cfg.Validate() == nil {
		return cfg, true
	}
	fmt.Println("  no config found — starting wizard\n")
	newCfg, err := wizard.Run(bufio.NewReader(os.Stdin))
	if err != nil {
		fmt.Println("  ✗", err)
		os.Exit(1)
	}
	return newCfg, true
}

func runDefault() {
	cfg, _ := ensureConfig()
	_ = cfg
	runServe()
}

func runWizard() {
	if _, err := wizard.Run(bufio.NewReader(os.Stdin)); err != nil {
		fmt.Println("  ✗", err)
		os.Exit(1)
	}
}

func buildAgent(cfg *config.Config, _ string, onImage func(string)) *agent.Agent {
	client := meta.New(cfg.MetaAPIKey, cfg.BaseURL)
	sites := sitefs.New(cfg.Workspace, cfg.PublicURL)
	reg := tools.New(cfg.Workspace, cfg.PublicURL, client, sites)
	reg.OnImageSaved = onImage
	return agent.New(client, reg)
}

func runServe() {
	cfg, _ := ensureConfig()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv := server.New(cfg, sitefs.New(cfg.Workspace, fmt.Sprintf("http://localhost:%d", cfg.Port)))
	if err := srv.Start(ctx, func(userID string, onImage func(string)) *agent.Agent {
		return buildAgent(cfg, userID, onImage)
	}); err != nil {
		fmt.Println("  ✗ server:", err)
		os.Exit(1)
	}
}

func runChat() {
	cfg, _ := ensureConfig()
	a := buildAgent(cfg, "terminal", func(url string) {
		fmt.Printf("  🖼 %s\n", url)
	})
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("  🦌 kancil chat — muse-spark-1.3 (Ctrl+C to exit)\n")
	for {
		fmt.Print("  you ▸ ")
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println()
			return
		}
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		reply, err := a.Handle(ctx, text)
		cancel()
		if err != nil {
			fmt.Println("  ✗", err)
			continue
		}
		fmt.Printf("  kancil ▸ %s\n\n", reply)
	}
}
