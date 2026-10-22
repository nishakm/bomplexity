package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeTools struct{ called []string }

func (f *fakeTools) Tools(context.Context) ([]any, error) {
	return []any{map[string]any{"type": "function", "function": map[string]any{"name": "entry_stats"}}}, nil
}

func (f *fakeTools) Call(_ context.Context, name string, _ map[string]any) (string, error) {
	f.called = append(f.called, name)
	return `{"count":4}`, nil
}

// fakeLLM answers each request with the next scripted assistant message and
// records the conversation it was sent.
func fakeLLM(t *testing.T, replies []message, seen *[][]message) *llm {
	i := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []message `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		*seen = append(*seen, body.Messages)
		reply := replies[min(i, len(replies)-1)]
		i++
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": reply}}})
	}))
	t.Cleanup(srv.Close)
	return &llm{baseURL: srv.URL, apiKey: "k", model: "m", client: srv.Client()}
}

func toolCall() message {
	return message{"role": "assistant", "tool_calls": []any{map[string]any{
		"id": "c1", "type": "function",
		"extra_content": map[string]any{"google": map[string]any{"thought_signature": "sig"}},
		"function":      map[string]any{"name": "entry_stats", "arguments": "{}"},
	}}}
}

func TestLoopToolThenAnswer(t *testing.T) {
	var seen [][]message
	tools := &fakeTools{}
	model := fakeLLM(t, []message{toolCall(), {"role": "assistant", "content": "four entries"}}, &seen)
	got, err := runLoop(context.Background(), model, tools, "be brief", "how many?")
	if err != nil || got != "four entries" {
		t.Fatalf("got %q, %v", got, err)
	}
	if len(tools.called) != 1 || tools.called[0] != "entry_stats" {
		t.Fatalf("tool calls = %v", tools.called)
	}
	second, _ := json.Marshal(seen[1])
	if !strings.Contains(string(second), "thought_signature") || !strings.Contains(string(second), `"role":"tool"`) {
		t.Fatalf("second request lost the signature or tool result: %s", second)
	}
}

func TestLoopStopsAfterMaxIterations(t *testing.T) {
	var seen [][]message
	model := fakeLLM(t, []message{toolCall()}, &seen)
	if _, err := runLoop(context.Background(), model, &fakeTools{}, "", "q"); err == nil {
		t.Fatal("expected an error")
	}
	if len(seen) != maxIterations {
		t.Fatalf("model called %d times, want %d", len(seen), maxIterations)
	}
}

func TestAskWithoutKeyIs503(t *testing.T) {
	rec := httptest.NewRecorder()
	askHandler(askConfig{})(rec, httptest.NewRequest("POST", "/ask", strings.NewReader(`{"question":"hi"}`)))
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "LLM_API_KEY") {
		t.Fatalf("got %d %s", rec.Code, rec.Body)
	}
}
