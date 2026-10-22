// Command agent is the "ask the guestbook" service: POST /ask runs a model
// with tools served by the guestbook MCP server.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	cfg := loadConfig()
	if cfg.APIKey == "" {
		log.Print("warning: LLM_API_KEY is not set; /ask will answer 503")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("POST /ask", askHandler(cfg))

	addr := ":" + envOr("PORT", "8082")
	log.Printf("guestbook-agent listening on %s (model %s, MCP %s)", addr, cfg.Model, cfg.MCPURL)
	log.Fatal((&http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}).ListenAndServe())
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
