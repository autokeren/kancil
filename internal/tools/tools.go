package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/autokeren/kancil/internal/meta"
	"github.com/autokeren/kancil/internal/model"
)

// Publisher publishes a folder as a live static site and returns its public URL.
type Publisher interface {
	Publish(name string) (string, error)
}

type Registry struct {
	Workspace string
	Client    *meta.Client
	Publisher Publisher
}

func New(workspace string, client *meta.Client, pub Publisher) *Registry {
	return &Registry{Workspace: workspace, Client: client, Publisher: pub}
}

func (r *Registry) Definitions() []model.ToolDefinition {
	return []model.ToolDefinition{
		{Type: "function", Function: model.ToolFunction{
			Name: "write_file", Description: "Write a file into the workspace (websites, code, text). Use forward slashes. Directories are created automatically.",
			Parameters: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"relative path, e.g. sites/kopi/index.html"},"content":{"type":"string","description":"full file content"}},"required":["path","content"]}`),
		}},
		{Type: "function", Function: model.ToolFunction{
			Name: "read_file", Description: "Read a file from the workspace.",
			Parameters: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
		}},
		{Type: "function", Function: model.ToolFunction{
			Name: "list_files", Description: "List files in the workspace.",
			Parameters: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string","description":"optional subfolder"}}}`),
		}},
		{Type: "function", Function: model.ToolFunction{
			Name: "gen_image", Description: "Generate an image with Muse Image (banners, logos, illustrations). Saved into workspace, returns its public URL.",
			Parameters: json.RawMessage(`{"type":"object","properties":{"prompt":{"type":"string","description":"what to draw, be specific"},"filename":{"type":"string","description":"output filename like banners/promo.png"}},"required":["prompt","filename"]}`),
		}},
		{Type: "function", Function: model.ToolFunction{
			Name: "publish_site", Description: "Publish a site folder as a LIVE public website and get its URL. Write the index.html first with write_file, then call this.",
			Parameters: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","description":"site name (folder under sites/)"}},"required":["name"]}`),
		}},
	}
}

func (r *Registry) Execute(ctx context.Context, call model.ToolCall) string {
	var args map[string]any
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return fmt.Sprintf("error: bad arguments: %v", err)
	}
	switch call.Function.Name {
	case "write_file":
		return r.writeFile(str(args["path"]), str(args["content"]))
	case "read_file":
		return r.readFile(str(args["path"]))
	case "list_files":
		return r.listFiles(str(args["path"]))
	case "gen_image":
		return r.genImage(ctx, str(args["prompt"]), str(args["filename"]))
	case "publish_site":
		return r.publishSite(str(args["name"]))
	default:
		return fmt.Sprintf("error: unknown tool %s", call.Function.Name)
	}
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

var safePath = regexp.MustCompile(`^[a-zA-Z0-9._\-/]+$`)

func (r *Registry) resolve(rel string) (string, error) {
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || !safePath.MatchString(rel) || strings.Contains(rel, "..") {
		return "", fmt.Errorf("invalid path: %q", rel)
	}
	return filepath.Join(r.Workspace, rel), nil
}

func (r *Registry) writeFile(rel, content string) string {
	full, err := r.resolve(rel)
	if err != nil {
		return "error: " + err.Error()
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "error: " + err.Error()
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return "error: " + err.Error()
	}
	return fmt.Sprintf("written %s (%d bytes)", rel, len(content))
}

func (r *Registry) readFile(rel string) string {
	full, err := r.resolve(rel)
	if err != nil {
		return "error: " + err.Error()
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "error: " + err.Error()
	}
	return string(data)
}

func (r *Registry) listFiles(rel string) string {
	base, err := r.resolve(rel)
	if err != nil {
		return "error: " + err.Error()
	}
	if rel == "" {
		base = r.Workspace
	}
	var files []string
	_ = filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel2, _ := filepath.Rel(r.Workspace, p)
			files = append(files, rel2)
		}
		return nil
	})
	if len(files) == 0 {
		return "(empty)"
	}
	return strings.Join(files, "\n")
}

func (r *Registry) genImage(ctx context.Context, prompt, filename string) string {
	if filename == "" {
		filename = "images/generated.png"
	}
	full, err := r.resolve(filename)
	if err != nil {
		return "error: " + err.Error()
	}
	if !strings.HasSuffix(full, ".png") && !strings.HasSuffix(full, ".jpg") {
		full += ".png"
	}
	res, err := r.Client.GenerateImage(ctx, prompt, "1024x1024")
	if err != nil {
		return "error: " + err.Error()
	}
	var data []byte
	if res.B64JSON != "" {
		if b, err := base64.StdEncoding.DecodeString(res.B64JSON); err == nil {
			data = b
		}
	}
	if len(data) == 0 && res.URL != "" {
		return fmt.Sprintf("image generated at %s — tell the user the link (and save via download if supported)", res.URL)
	}
	if len(data) == 0 {
		return "error: no image data in response"
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "error: " + err.Error()
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "error: " + err.Error()
	}
	return fmt.Sprintf("image saved to %s", filename)
}

func (r *Registry) publishSite(name string) string {
	if r.Publisher == nil {
		return "error: publisher not configured"
	}
	if name == "" {
		return "error: site name required"
	}
	url, err := r.Publisher.Publish(name)
	if err != nil {
		return "error: " + err.Error()
	}
	return fmt.Sprintf("site is LIVE at %s", url)
}

