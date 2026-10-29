package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"fmt"
	"strings"

	"github.com/autokeren/kancil/internal/meta"
	"github.com/autokeren/kancil/internal/model"
	"github.com/autokeren/kancil/internal/tools"
)

const systemPrompt = `You are Kancil 🦌 — a helpful agent that lives in the user's Messenger and WhatsApp.
You build things for people: websites, images, reports, files. You speak the language the
user speaks (Bahasa Indonesia, English, or both).

RULES:
1. When the user asks you to build/create/make something, USE YOUR TOOLS. Do not just describe — build it.
2. To build a website: use write_file to write an index.html (make it beautiful, modern, mobile-friendly,
   dark terminal aesthetic when in doubt), then use publish_site to make it live, then give the user the link.
3. To make an image: use gen_image.
4. Keep replies short and friendly — this is a chat, not a document. One or two sentences + links.
5. When you finish building, always share the public link so the user can see the result.
6. You may be given voice notes — they arrive as text transcripts; treat them like normal messages.
7. If a task needs many steps, just do them all. You never say "I can't do multi-step".`

const MaxSteps = 8

type Agent struct {
	Client          *meta.Client
	Tools           *tools.Registry
	History         []model.Message
	OnFirstToolCall func()      // fired once when Muse decides to build
	firstToolFired   bool
	MemoryDir       string     // where history persists
	ContextLimit     int        // model context window (tokens)
	CompactAt        float64    // fraction of ContextLimit that triggers compaction
}

const DefaultContextLimit = 1_000_000 // muse-spark-1.3: 1M

func New(client *meta.Client, reg *tools.Registry) *Agent {
	return &Agent{Client: client, Tools: reg}
}

// Handle processes one user message and returns the final reply text.
// Multi-step: executes tool calls until the model answers with plain content.
func (a *Agent) Handle(ctx context.Context, userText string) (string, error) {
	a.History = append(a.History, model.Message{Role: "user", Content: userText})
	if len(a.History) == 1 {
		a.History = append([]model.Message{{Role: "system", Content: systemPrompt}}, a.History...)
	}

	toolDefs := a.Tools.Definitions()

	for step := 0; step < MaxSteps; step++ {
		resp, err := a.Client.Complete(ctx, meta.CompleteOptions{
			Messages:  a.History,
			Tools:     toolDefs,
			MaxTokens: 16000,
		})
		if err != nil {
			return "", fmt.Errorf("muse-spark: %w", err)
		}

		// opencode-style memory: persist every turn + compact near the limit
		a.persist()
		limit := a.ContextLimit
		if limit <= 0 {
			limit = DefaultContextLimit
		}
		if a.CompactAt <= 0 {
			a.CompactAt = 0.85
		}
		if resp.Usage.PromptTokens > int(float64(limit)*a.CompactAt) {
			fmt.Printf("    🗜 compaction: %d tokens > %.0f%% of context — summarizing…\n", resp.Usage.PromptTokens, a.CompactAt*100)
			a.compact(ctx)
		}

		msg := model.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		a.History = append(a.History, msg)

		if len(resp.ToolCalls) == 0 {
			// plain answer — done
			if strings.TrimSpace(resp.Content) == "" {
				return "(muse berpikir lama tapi belum menjawab — coba lagi)", nil
			}
			return resp.Content, nil
		}

		// model-triggered ack: Muse mutusin buat ngebuild → kabarin user
		if len(resp.ToolCalls) > 0 && !a.firstToolFired {
			a.firstToolFired = true
			if a.OnFirstToolCall != nil {
				a.OnFirstToolCall()
			}
		}

		// execute each tool call, append results
		for _, call := range resp.ToolCalls {
			fmt.Printf("    ⚙ %s(%s)\n", call.Function.Name, shortArgs(call.Function.Arguments))
			result := a.Tools.Execute(ctx, call)
			a.History = append(a.History, model.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: call.ID,
			})
		}
	}
	return "Maaf, langkahnya kepanjangan 😅 — coba task yang lebih spesifik?", nil
}

// Reset clears conversation memory (new session per user is default).
func (a *Agent) Reset() {
	a.History = nil
	a.firstToolFired = false
}

// memoryPath returns the JSON file this conversation persists to.
func (a *Agent) memoryPath(userID string) string {
	if a.MemoryDir == "" {
		return ""
	}
	return filepath.Join(a.MemoryDir, userID+".json")
}

// PersistToFile saves history (used with the server's per-user wiring).
func (a *Agent) PersistToFile(path string) {
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	raw, _ := json.Marshal(a.History)
	_ = os.WriteFile(path, raw, 0o600)
}

func (a *Agent) persist() {
	if a.memoryPath("session") != "" {
		a.PersistToFile(filepath.Join(a.MemoryDir, "_last.json"))
	}
}

// LoadHistory replaces in-RAM history with the persisted one.
func (a *Agent) LoadHistory(path string) {
	if path == "" {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var h []model.Message
	if json.Unmarshal(raw, &h) == nil && len(h) > 0 {
		a.History = h
	}
}

// compact summarizes older turns and keeps the recent tail verbatim,
// mirroring opencode's context compaction at the threshold.
func (a *Agent) compact(ctx context.Context) {
	const keep = 6 // keep the last 6 messages verbatim
	if len(a.History) <= keep+2 {
		return
	}
	old := a.History[1 : len(a.History)-keep] // everything after system, before tail
	tail := a.History[len(a.History)-keep:]

	var sb strings.Builder
	for _, m := range old {
		role := m.Role
		text := m.Content
		if len(text) > 500 {
			text = text[:500] + "…"
		}
		if len(m.ToolCalls) > 0 {
			role = "assistant(tool call)"
			text = m.ToolCalls[0].Function.Name
		}
		sb.WriteString(role + ": " + text + "\n")
	}

	summaryReq := []model.Message{
		{Role: "system", Content: "Summarize this conversation history for an agent named Kancil. Keep: what the user asked for, what was built (site names, image names, URLs), and any preferences. Max 200 words. Reply with ONLY the summary."},
		{Role: "user", Content: sb.String()},
	}
	resp, err := a.Client.Complete(ctx, meta.CompleteOptions{Messages: summaryReq, MaxTokens: 3000})
	if err != nil {
		// fallback: drop old messages hard
		a.History = append([]model.Message{a.History[0]}, tail...)
		return
	}
	summary := resp.Content
	a.History = append(
		[]model.Message{a.History[0], {Role: "user", Content: "[memory] Sebelumnya: " + summary}},
		tail...,
	)
}

var _ = json.Marshal

func shortArgs(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 60 {
		return s[:60] + "…"
	}
	return s
}
