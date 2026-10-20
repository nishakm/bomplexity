// Command guestbook is a tiny guestbook HTTP API used as an SBOM demo.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// ctrl+c stops the app
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := openStore(ctx)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}
	defer store.Close()

	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           newMux(store),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// openStore uses Postgres when DATABASE_URL is set, otherwise memory.
func openStore(ctx context.Context) (Store, error) {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		log.Print("using Postgres store")
		return NewPostgresStore(ctx, url)
	}
	log.Print("DATABASE_URL not set: using in-memory store")
	return NewMemoryStore(), nil
}

// envOr get environment variable or string
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
