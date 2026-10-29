package agent

import (
	"context"
	"encoding/json"
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
	Client    *meta.Client
	Tools     *tools.Registry
	History   []model.Message
}

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
}

var _ = json.Marshal

func shortArgs(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 60 {
		return s[:60] + "…"
	}
	return s
}
