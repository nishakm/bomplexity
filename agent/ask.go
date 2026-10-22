package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	maxIterations = 5
	maxBodyBytes  = 16 << 10
	askTimeout    = 90 * time.Second
)

// toolSource is the MCP side of the loop: it lists tools and runs them.
type toolSource interface {
	Tools(ctx context.Context) ([]any, error)
	Call(ctx context.Context, name string, args map[string]any) (string, error)
}

// askConfig is read from the environment once at startup.
type askConfig struct {
	APIKey     string
	BaseURL    string
	Model      string
	PromptFile string
	MCPURL     string
}

func loadConfig() askConfig {
	return askConfig{
		APIKey:     os.Getenv("LLM_API_KEY"),
		BaseURL:    envOr("LLM_BASE_URL", "https://generativelanguage.googleapis.com/v1beta/openai"),
		Model:      envOr("LLM_MODEL", "gemini-3.5-flash-lite"),
		PromptFile: os.Getenv("SYSTEM_PROMPT_FILE"),
		MCPURL:     envOr("MCP_SERVER_URL", "http://guestbook-mcp:8081/mcp"),
	}
}

// askHandler serves POST /ask {"question": "..."} -> {"answer": "..."}.
func askHandler(cfg askConfig) http.HandlerFunc {
	model := &llm{baseURL: cfg.BaseURL, apiKey: cfg.APIKey, model: cfg.Model, client: &http.Client{Timeout: 60 * time.Second}}
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.APIKey == "" {
			writeError(w, http.StatusServiceUnavailable, "the agent is not configured: LLM_API_KEY is not set")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			Question string `json:"question"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Question) == "" {
			writeError(w, http.StatusBadRequest, `body must be {"question": "..."}`)
			return
		}
		system, err := readPrompt(cfg.PromptFile)
		if err != nil {
			log.Printf("read system prompt: %v", err)
			writeError(w, http.StatusInternalServerError, "cannot read the system prompt")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), askTimeout)
		defer cancel()
		tools, err := connectMCP(ctx, cfg.MCPURL)
		if err != nil {
			log.Printf("mcp: %v", err)
			writeError(w, http.StatusBadGateway, "cannot reach the MCP server")
			return
		}
		defer tools.Close()

		answer, err := runLoop(ctx, model, tools, system, in.Question)
		if err != nil {
			log.Printf("ask: %v", err)
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"answer": answer})
	}
}

func readPrompt(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	return strings.TrimSpace(string(b)), err
}

// runLoop lets the model call MCP tools until it gives a final answer.
func runLoop(ctx context.Context, model *llm, src toolSource, system, question string) (string, error) {
	tools, err := src.Tools(ctx)
	if err != nil {
		return "", fmt.Errorf("list tools: %w", err)
	}
	msgs := []message{}
	if system != "" {
		msgs = append(msgs, message{"role": "system", "content": system})
	}
	msgs = append(msgs, message{"role": "user", "content": question})

	for i := 0; i < maxIterations; i++ {
		reply, err := model.chat(ctx, msgs, tools)
		if err != nil {
			return "", err
		}
		calls, _ := reply["tool_calls"].([]any)
		if len(calls) == 0 {
			text, _ := reply["content"].(string)
			return strings.TrimSpace(text), nil
		}
		msgs = append(msgs, reply) // sent back unchanged, extras included
		for _, c := range calls {
			msgs = append(msgs, runToolCall(ctx, src, c))
		}
	}
	return "", errors.New("no final answer after the maximum number of tool iterations")
}

func runToolCall(ctx context.Context, src toolSource, call any) message {
	c, _ := call.(map[string]any)
	id, _ := c["id"].(string)
	fn, _ := c["function"].(map[string]any)
	name, _ := fn["name"].(string)
	var args map[string]any
	if s, _ := fn["arguments"].(string); s != "" {
		_ = json.Unmarshal([]byte(s), &args)
	}
	out, err := src.Call(ctx, name, args)
	if err != nil {
		out = "error: " + err.Error()
	}
	return message{"role": "tool", "tool_call_id": id, "content": out}
}

// mcpTools implements toolSource over an MCP streamable HTTP session.
type mcpTools struct{ session *mcp.ClientSession }

func connectMCP(ctx context.Context, url string) (*mcpTools, error) {
	client := mcp.NewClient(&mcp.Implementation{Name: "guestbook-agent", Version: "v1"}, nil)
	s, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: url}, nil)
	if err != nil {
		return nil, err
	}
	return &mcpTools{session: s}, nil
}

func (m *mcpTools) Close() { _ = m.session.Close() }

func (m *mcpTools) Tools(ctx context.Context) ([]any, error) {
	res, err := m.session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	var out []any
	for _, t := range res.Tools {
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			},
		})
	}
	return out, nil
}

func (m *mcpTools) Call(ctx context.Context, name string, args map[string]any) (string, error) {
	res, err := m.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, c := range res.Content {
		if t, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(t.Text)
		}
	}
	if res.IsError {
		return "", errors.New(sb.String())
	}
	return sb.String(), nil
}
