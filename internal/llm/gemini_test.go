package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGeminiProviderChatParsesMockedHTTPResponse(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1beta/models/test-model:generateContent" {
			t.Fatalf("path = %q, want /v1beta/models/test-model:generateContent", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "test-key" {
			t.Fatalf("key = %q, want test-key", r.URL.Query().Get("key"))
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}

		var req geminiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Contents) != 1 || req.Contents[0].Parts[0].Text != "hello" {
			t.Fatalf("contents = %#v", req.Contents)
		}

		return jsonResponse(http.StatusOK, `{"candidates":[{"content":{"parts":[{"text":"hi there"}]}}]}`), nil
	})}

	provider := NewGeminiProvider(GeminiConfig{
		APIKey:     "test-key",
		BaseURL:    "https://example.test/v1beta",
		Model:      "test-model",
		HTTPClient: client,
	})

	got, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if got.Role != "assistant" || got.Content != "hi there" {
		t.Fatalf("message = %#v", got)
	}
}

func TestGeminiProviderChatParsesToolCalls(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"echo","args":{"text":"hello"}}}]}}]}`), nil
	})}

	provider := NewGeminiProvider(GeminiConfig{
		APIKey:     "test-key",
		BaseURL:    "https://example.test/v1beta",
		Model:      "test-model",
		HTTPClient: client,
	})

	got, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if len(got.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(got.ToolCalls))
	}
	if got.ToolCalls[0].Function.Name != "echo" {
		t.Fatalf("tool call name = %q, want echo", got.ToolCalls[0].Function.Name)
	}
	if !strings.Contains(got.ToolCalls[0].Function.Arguments, "hello") {
		t.Fatalf("tool call arguments = %q, want hello", got.ToolCalls[0].Function.Arguments)
	}
}

func TestGeminiProviderRequiresAPIKey(t *testing.T) {
	provider := NewGeminiProvider(GeminiConfig{Model: "test-model"})

	_, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err == nil || !strings.Contains(err.Error(), "api key") {
		t.Fatalf("error = %v, want api key error", err)
	}
}

func TestGeminiProviderReturnsAPIError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, "bad request"), nil
	})}
	provider := NewGeminiProvider(GeminiConfig{
		APIKey:     "test-key",
		BaseURL:    "https://example.test/v1beta",
		Model:      "test-model",
		HTTPClient: client,
	})

	_, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err == nil || !strings.Contains(err.Error(), "400 Bad Request") {
		t.Fatalf("error = %v, want status error", err)
	}
}

func TestGeminiProviderRequiresCandidate(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"candidates":[]}`), nil
	})}
	provider := NewGeminiProvider(GeminiConfig{
		APIKey:     "test-key",
		BaseURL:    "https://example.test/v1beta",
		Model:      "test-model",
		HTTPClient: client,
	})

	_, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err == nil || !strings.Contains(err.Error(), "no candidates") {
		t.Fatalf("error = %v, want no candidates error", err)
	}
}

func TestGeminiContentsMapsAssistantToModel(t *testing.T) {
	got := geminiContents([]Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	})

	if got[0].Role != "user" || got[1].Role != "model" {
		t.Fatalf("contents = %#v, want user then model roles", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}
