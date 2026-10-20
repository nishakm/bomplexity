package main

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
)

const (
	maxBodyBytes = 16 << 10
	maxNameLen   = 100
	maxMsgLen    = 2000
)

// newMux wires the HTTP routes using the standard library router.
func newMux(store Store) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /entries", func(w http.ResponseWriter, r *http.Request) {
		entries, err := store.List(r.Context())
		if err != nil {
			log.Printf("list entries: %v", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, entries)
	})
	mux.HandleFunc("POST /entries", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		var in struct {
			Name    string `json:"name"`
			Message string `json:"message"`
		}
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		in.Name, in.Message = strings.TrimSpace(in.Name), strings.TrimSpace(in.Message)
		if in.Name == "" || in.Message == "" {
			writeError(w, http.StatusBadRequest, "name and message are required")
			return
		}
		if len(in.Name) > maxNameLen || len(in.Message) > maxMsgLen {
			writeError(w, http.StatusBadRequest, "name or message too long")
			return
		}
		e, err := store.Add(r.Context(), in.Name, in.Message)
		if err != nil {
			log.Printf("add entry: %v", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusCreated, e)
	})
	mux.HandleFunc("GET /sbom", func(w http.ResponseWriter, _ *http.Request) {
		bi, ok := debug.ReadBuildInfo()
		if !ok {
			writeError(w, http.StatusInternalServerError, "build info unavailable (binary not built with module support)")
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(renderSBOM(bi)))
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
