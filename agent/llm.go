package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// message is one chat message. It is kept as a generic map so provider
// extras (e.g. Gemini's thought signatures on tool calls) survive the
// round trip when the assistant message is sent back.
type message = map[string]any

// llm talks to an OpenAI-compatible chat completions endpoint.
type llm struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

type statusError struct {
	code int
	body string
}

func (e *statusError) Error() string {
	if e.code == http.StatusTooManyRequests {
		return "the model API rate limit or quota was exceeded (HTTP 429); try again in a minute"
	}
	return fmt.Sprintf("model API returned %d: %s", e.code, e.body)
}

// chat sends the conversation and returns the assistant message. Rate limits
// and overload (429/503) are retried a few times with backoff.
func (l *llm) chat(ctx context.Context, msgs []message, tools []any) (message, error) {
	body := map[string]any{"model": l.model, "messages": msgs}
	if len(tools) > 0 {
		body["tools"] = tools
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(time.Duration(1<<(attempt-1)) * time.Second):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		m, err := l.once(ctx, payload)
		if err == nil {
			return m, nil
		}
		lastErr = err
		if se, ok := err.(*statusError); !ok || (se.code != http.StatusTooManyRequests && se.code != http.StatusServiceUnavailable) {
			return nil, err
		}
	}
	return nil, lastErr
}

func (l *llm) once(ctx context.Context, payload []byte) (message, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("model API unreachable: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		if len(raw) > 300 {
			raw = raw[:300]
		}
		return nil, &statusError{code: resp.StatusCode, body: string(raw)}
	}
	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil || len(out.Choices) == 0 {
		return nil, fmt.Errorf("unexpected model response")
	}
	return out.Choices[0].Message, nil
}
