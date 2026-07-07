package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MehulCodr/AI-agent/internal/llm"
)

func TestAgentStreamEmitsChunksInOrder(t *testing.T) {
	provider := &fakeStreamProvider{
		chunks: []llm.StreamChunk{
			{Content: "The"},
			{Content: " CLI"},
			{Content: " streams"},
			{Done: true},
		},
	}
	agent := New(provider)

	chunks, err := agent.Stream(context.Background(), "explain")
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	got, err := collectAgentStream(t, chunks)
	if err != nil {
		t.Fatalf("collect stream: %v", err)
	}
	if got != "The CLI streams" {
		t.Fatalf("stream content = %q, want ordered chunks", got)
	}
}

func TestAgentStreamFallsBackToChatProvider(t *testing.T) {
	agent := New(&fakeProvider{
		response: llm.Message{Role: "assistant", Content: "fallback response"},
	})

	chunks, err := agent.Stream(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	got, err := collectAgentStream(t, chunks)
	if err != nil {
		t.Fatalf("collect stream: %v", err)
	}
	if got != "fallback response" {
		t.Fatalf("stream content = %q, want fallback response", got)
	}
}

func TestAgentStreamCancellationStopsStreaming(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	providerChunks := make(chan llm.StreamChunk)
	provider := &blockingStreamProvider{chunks: providerChunks}
	agent := New(provider)

	chunks, err := agent.Stream(ctx, "hello")
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	providerChunks <- llm.StreamChunk{Content: "partial"}
	first := readAgentChunk(t, chunks)
	if first.Content != "partial" {
		t.Fatalf("first chunk = %#v, want partial content", first)
	}

	cancel()
	assertAgentStreamClosed(t, chunks)
	if messages := agent.Messages(); len(messages) != 1 {
		t.Fatalf("len(Messages()) = %d, want only user message after cancellation", len(messages))
	}
}

func TestAgentStreamStoresFinalResponseInHistory(t *testing.T) {
	provider := &fakeStreamProvider{
		chunks: []llm.StreamChunk{
			{Content: "hi"},
			{Content: " there"},
			{Done: true},
		},
	}
	agent := New(provider)

	chunks, err := agent.Stream(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	if _, err := collectAgentStream(t, chunks); err != nil {
		t.Fatalf("collect stream: %v", err)
	}

	messages := agent.Messages()
	if len(messages) != 2 {
		t.Fatalf("len(Messages()) = %d, want user and assistant", len(messages))
	}
	if messages[1].Role != "assistant" || messages[1].Content != "hi there" {
		t.Fatalf("assistant message = %#v", messages[1])
	}
}

func TestAgentStreamHandlesEmptyStream(t *testing.T) {
	provider := &fakeStreamProvider{}
	agent := New(provider)

	chunks, err := agent.Stream(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}
	got, err := collectAgentStream(t, chunks)
	if err != nil {
		t.Fatalf("collect stream: %v", err)
	}
	if got != "" {
		t.Fatalf("stream content = %q, want empty", got)
	}

	messages := agent.Messages()
	if len(messages) != 2 {
		t.Fatalf("len(Messages()) = %d, want user and empty assistant", len(messages))
	}
	if messages[1].Role != "assistant" || messages[1].Content != "" {
		t.Fatalf("assistant message = %#v", messages[1])
	}
}

type fakeStreamProvider struct {
	chunks []llm.StreamChunk
	seen   []llm.Message
}

func (p *fakeStreamProvider) Chat(ctx context.Context, messages []llm.Message) (llm.Message, error) {
	if err := ctx.Err(); err != nil {
		return llm.Message{}, err
	}
	return llm.Message{Role: "assistant", Content: "chat response"}, nil
}

func (p *fakeStreamProvider) Stream(ctx context.Context, messages []llm.Message) (<-chan llm.StreamChunk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p.seen = append([]llm.Message(nil), messages...)

	chunks := make(chan llm.StreamChunk)
	go func() {
		defer close(chunks)
		for _, chunk := range p.chunks {
			select {
			case <-ctx.Done():
				return
			case chunks <- chunk:
			}
		}
	}()
	return chunks, nil
}

type blockingStreamProvider struct {
	chunks <-chan llm.StreamChunk
}

func (p *blockingStreamProvider) Chat(ctx context.Context, messages []llm.Message) (llm.Message, error) {
	return llm.Message{Role: "assistant", Content: "chat response"}, ctx.Err()
}

func (p *blockingStreamProvider) Stream(ctx context.Context, messages []llm.Message) (<-chan llm.StreamChunk, error) {
	return p.chunks, ctx.Err()
}

func collectAgentStream(t *testing.T, chunks <-chan llm.StreamChunk) (string, error) {
	t.Helper()

	var content string
	for {
		chunk := readAgentChunk(t, chunks)
		content += chunk.Content
		if chunk.Error != nil {
			return content, chunk.Error
		}
		if chunk.Done {
			return content, nil
		}
	}
}

func readAgentChunk(t *testing.T, chunks <-chan llm.StreamChunk) llm.StreamChunk {
	t.Helper()

	select {
	case chunk, ok := <-chunks:
		if !ok {
			return llm.StreamChunk{Done: true}
		}
		return chunk
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stream chunk")
		return llm.StreamChunk{}
	}
}

func assertAgentStreamClosed(t *testing.T, chunks <-chan llm.StreamChunk) {
	t.Helper()

	select {
	case chunk, ok := <-chunks:
		if ok && !errors.Is(chunk.Error, context.Canceled) {
			t.Fatalf("stream remained open with chunk %#v", chunk)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stream to close")
	}
}
