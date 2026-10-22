// Command mcp is the guestbook MCP server. It exposes two read-only tools
// that call the guestbook's own GET /entries API, so it never touches the
// database and cannot write.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultLimit = 20

type entry struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type guestbook struct {
	url    string
	client *http.Client
}

// entries fetches all entries from the guestbook, newest first.
func (g *guestbook) entries(ctx context.Context) ([]entry, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("guestbook unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("guestbook returned %s", resp.Status)
	}
	var out []entry
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode entries: %w", err)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].ID > out[j].ID // entries made in one transaction share a timestamp
	})
	return out, nil
}

type listInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"maximum number of entries to return, newest first (default 20)"`
}

type listOutput struct {
	Entries []entry `json:"entries"`
}

type statsOutput struct {
	Count  int    `json:"count"`
	Newest *entry `json:"newest,omitempty"`
	Oldest *entry `json:"oldest,omitempty"`
}

func (g *guestbook) listEntries(ctx context.Context, _ *mcp.CallToolRequest, in listInput) (*mcp.CallToolResult, listOutput, error) {
	all, err := g.entries(ctx)
	if err != nil {
		return nil, listOutput{}, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if len(all) > limit {
		all = all[:limit]
	}
	return nil, listOutput{Entries: all}, nil
}

func (g *guestbook) entryStats(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, statsOutput, error) {
	all, err := g.entries(ctx)
	if err != nil {
		return nil, statsOutput{}, err
	}
	out := statsOutput{Count: len(all)}
	if len(all) > 0 {
		out.Newest, out.Oldest = &all[0], &all[len(all)-1]
	}
	return nil, out, nil
}

func newServer(g *guestbook) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "guestbook-mcp", Version: "v1"}, nil)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_entries",
		Description: "List guestbook entries, newest first. Optional limit (default 20).",
	}, g.listEntries)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "entry_stats",
		Description: "Guestbook statistics: total entry count plus the newest and oldest entry.",
	}, g.entryStats)
	return s
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	g := &guestbook{
		url:    envOr("ENTRIES_URL", "http://guestbook/entries"),
		client: &http.Client{Timeout: 10 * time.Second},
	}
	server := newServer(g)

	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})

	addr := ":" + envOr("PORT", "8081")
	log.Printf("guestbook-mcp listening on %s, entries from %s", addr, g.url)
	log.Fatal((&http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}).ListenAndServe())
}
