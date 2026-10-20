package main

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Entry is a single guestbook entry.
type Entry struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// Store is the storage abstraction behind the API.
type Store interface {
	Add(ctx context.Context, name, message string) (Entry, error)
	List(ctx context.Context) ([]Entry, error)
	Close()
}

// MemoryStore keeps entries in process memory.
type MemoryStore struct {
	mu      sync.Mutex
	entries []Entry
}

func NewMemoryStore() *MemoryStore { return &MemoryStore{} }

func (s *MemoryStore) Add(_ context.Context, name, message string) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := Entry{ID: int64(len(s.entries) + 1), Name: name, Message: message, CreatedAt: time.Now().UTC()}
	s.entries = append(s.entries, e)
	return e, nil
}

func (s *MemoryStore) List(_ context.Context) ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Entry{}, s.entries...), nil
}

func (s *MemoryStore) Close() {}

// PostgresStore persists entries in Postgres via pgx.
type PostgresStore struct {
	pool *pgxpool.Pool
}

const schema = `CREATE TABLE IF NOT EXISTS entries (
	id         BIGSERIAL PRIMARY KEY,
	name       TEXT NOT NULL,
	message    TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`

func NewPostgresStore(ctx context.Context, url string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgresStore{pool: pool}, nil
}

func (s *PostgresStore) Add(ctx context.Context, name, message string) (Entry, error) {
	e := Entry{Name: name, Message: message}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO entries (name, message) VALUES ($1, $2) RETURNING id, created_at`,
		name, message).Scan(&e.ID, &e.CreatedAt)
	return e, err
}

func (s *PostgresStore) List(ctx context.Context) ([]Entry, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, message, created_at FROM entries ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []Entry{}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.Name, &e.Message, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *PostgresStore) Close() { s.pool.Close() }
