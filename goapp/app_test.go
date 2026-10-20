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

func TestRenderSBOM(t *testing.T) {
	bi := &debug.BuildInfo{
		GoVersion: "go1.25.1",
		Main:      debug.Module{Path: "example.com/app", Version: "(devel)"},
		Deps: []*debug.Module{
			{Path: "github.com/jackc/pgx/v5", Version: "v5.7.0"},
			{Path: "golang.org/x/text", Version: "v0.20.0"},
		},
	}
	want := "pkg:golang/example.com/app\n" +
		"+-- pkg:golang/github.com/jackc/pgx/v5@v5.7.0\n" +
		"+-- pkg:golang/golang.org/x/text@v0.20.0\n" +
		"`-- pkg:golang/std@1.25.1\n"
	if got := renderSBOM(bi); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderSBOMReplace(t *testing.T) {
	bi := &debug.BuildInfo{
		GoVersion: "go1.25.1",
		Main:      debug.Module{Path: "example.com/app", Version: "v1.2.3"},
		Deps: []*debug.Module{{
			Path: "example.com/lib", Version: "v1.0.0",
			Replace: &debug.Module{Path: "example.com/fork", Version: "v1.0.1"},
		}},
	}
	got := renderSBOM(bi)
	if !strings.HasPrefix(got, "pkg:golang/example.com/app@v1.2.3\n") ||
		!strings.Contains(got, "+-- pkg:golang/example.com/lib@v1.0.1\n") {
		t.Fatalf("unexpected output:\n%s", got)
	}
}

func TestSBOMEndpoint(t *testing.T) {
	rec := do(t, newMux(NewMemoryStore()), "GET", "/sbom", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content type = %q", ct)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, "pkg:golang/") || !strings.Contains(body, "pkg:golang/std@") {
		t.Fatalf("unexpected body:\n%s", body)
	}
}
