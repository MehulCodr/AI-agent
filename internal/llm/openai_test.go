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

func TestOpenAIProviderChat(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
		}

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Fatalf("model = %q, want test-model", req.Model)
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "hello" {
			t.Fatalf("messages = %#v", req.Messages)
		}

		return jsonResponse(http.StatusOK, `{"choices":[{"message":{"role":"assistant","content":"hi there"}}]}`), nil
	})}

	provider := NewOpenAIProvider(OpenAIConfig{
		APIKey:  "test-key",
		BaseURL: "https://example.test/v1",
		Model:   "test-model",
	})
	provider.client = client

	got, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if got.Role != "assistant" || got.Content != "hi there" {
		t.Fatalf("message = %#v", got)
	}
}

func TestOpenAIProviderChatRequiresAPIKey(t *testing.T) {
	provider := NewOpenAIProvider(OpenAIConfig{Model: "test-model"})

	_, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err == nil || !strings.Contains(err.Error(), "api key") {
		t.Fatalf("error = %v, want api key error", err)
	}
}

func TestOpenAIProviderChatRequiresModel(t *testing.T) {
	provider := NewOpenAIProvider(OpenAIConfig{APIKey: "test-key"})

	_, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err == nil || !strings.Contains(err.Error(), "model") {
		t.Fatalf("error = %v, want model error", err)
	}
}

func TestOpenAIProviderChatReturnsAPIError(t *testing.T) {
	provider := NewOpenAIProvider(OpenAIConfig{
		APIKey:  "test-key",
		BaseURL: "https://example.test/v1",
		Model:   "test-model",
	})
	provider.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadRequest, "bad request"), nil
	})}

	_, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err == nil || !strings.Contains(err.Error(), "400 Bad Request") {
		t.Fatalf("error = %v, want status error", err)
	}
}

func TestOpenAIProviderChatRequiresChoice(t *testing.T) {
	provider := NewOpenAIProvider(OpenAIConfig{
		APIKey:  "test-key",
		BaseURL: "https://example.test/v1",
		Model:   "test-model",
	})
	provider.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"choices":[]}`), nil
	})}

	_, err := provider.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err == nil || !strings.Contains(err.Error(), "no choices") {
		t.Fatalf("error = %v, want no choices error", err)
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
