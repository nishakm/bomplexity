package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"strings"
	"testing"
)

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestMemoryStore(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	a, _ := s.Add(ctx, "ann", "hi")
	b, _ := s.Add(ctx, "bob", "yo")
	if a.ID != 1 || b.ID != 2 {
		t.Fatalf("unexpected ids %d %d", a.ID, b.ID)
	}
	got, _ := s.List(ctx)
	if len(got) != 2 || got[0].Name != "ann" {
		t.Fatalf("unexpected list %+v", got)
	}
}

func TestHealthz(t *testing.T) {
	rec := do(t, newMux(NewMemoryStore()), "GET", "/healthz", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestPostAndListEntries(t *testing.T) {
	h := newMux(NewMemoryStore())
	rec := do(t, h, "POST", "/entries", `{"name":"ann","message":"hello"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("post: got %d: %s", rec.Code, rec.Body)
	}
	rec = do(t, h, "GET", "/entries", "")
	var got []Entry
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "ann" || got[0].Message != "hello" {
		t.Fatalf("unexpected entries %+v", got)
	}
}

func TestPostValidation(t *testing.T) {
	h := newMux(NewMemoryStore())
	for name, body := range map[string]string{
		"not json":      `nope`,
		"missing field": `{"name":"ann"}`,
		"blank":         `{"name":" ","message":" "}`,
		"unknown field": `{"name":"a","message":"b","x":1}`,
		"too long":      `{"name":"` + strings.Repeat("a", maxNameLen+1) + `","message":"m"}`,
	} {
		if rec := do(t, h, "POST", "/entries", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", name, rec.Code)
		}
	}
}

func TestBuildSBOM(t *testing.T) {
	bi := &debug.BuildInfo{
		GoVersion: "go1.25.1",
		Main:      debug.Module{Path: "example.com/app", Version: "(devel)"},
		Deps: []*debug.Module{
			{Path: "github.com/jackc/pgx/v5", Version: "v5.7.0", Sum: "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="},
			{Path: "golang.org/x/text", Version: "v0.20.0", Sum: "bogus"},
		},
	}
	b := buildSBOM(bi)
	if b.BOMFormat != "CycloneDX" || b.SpecVersion != "1.6" || !strings.HasPrefix(b.SerialNumber, "urn:uuid:") {
		t.Fatalf("bad header %+v", b)
	}
	if b.Metadata.Component.Name != "example.com/app" {
		t.Errorf("main component = %q", b.Metadata.Component.Name)
	}
	if len(b.Components) != 3 {
		t.Fatalf("want 2 deps + toolchain, got %d", len(b.Components))
	}
	pgx := b.Components[0]
	if pgx.PURL != "pkg:golang/github.com/jackc/pgx/v5@v5.7.0" || len(pgx.Hashes) != 1 || len(pgx.Hashes[0].Content) != 64 {
		t.Errorf("bad pgx component %+v", pgx)
	}
	if len(b.Components[1].Hashes) != 0 {
		t.Errorf("bogus sum should yield no hash")
	}
	if tc := b.Components[2]; tc.Name != "go" || tc.Version != "1.25.1" {
		t.Errorf("bad toolchain %+v", tc)
	}
}

func TestSBOMEndpoint(t *testing.T) {
	rec := do(t, newMux(NewMemoryStore()), "GET", "/sbom", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
	var b bom
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil || b.BOMFormat != "CycloneDX" {
		t.Fatalf("invalid sbom: %v", err)
	}
}
