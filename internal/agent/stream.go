package agent

import (
	"context"
	"fmt"
	"strings"

	apperrors "github.com/MehulCodr/AI-agent/internal/errors"
	"github.com/MehulCodr/AI-agent/internal/llm"
)

func (a *Agent) Stream(ctx context.Context, input string) (<-chan llm.StreamChunk, error) {
	if a == nil {
		return nil, fmt.Errorf("%w: agent is required", apperrors.ErrInvalidInput)
	}
	if ctx == nil {
		return nil, fmt.Errorf("%w: context is required", apperrors.ErrInvalidInput)
	}
	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("%w: input cannot be empty", apperrors.ErrInvalidInput)
	}
	if a.provider == nil {
		return nil, fmt.Errorf("%w: llm provider is required", apperrors.ErrInvalidInput)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	provider, ok := a.provider.(llm.StreamProvider)
	if !ok {
		return a.fallbackStream(ctx, input), nil
	}

	a.memory.Add(llm.Message{
		Role:    "user",
		Content: input,
	})

	messages, err := a.providerMessages(ctx)
	if err != nil {
		return nil, err
	}

	providerChunks, err := provider.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}
	if providerChunks == nil {
		return nil, fmt.Errorf("%w: stream channel is required", apperrors.ErrInvalidInput)
	}

	chunks := make(chan llm.StreamChunk)
	go a.forwardStream(ctx, providerChunks, chunks)
	return chunks, nil
}

func (a *Agent) fallbackStream(ctx context.Context, input string) <-chan llm.StreamChunk {
	chunks := make(chan llm.StreamChunk)
	go func() {
		defer close(chunks)

		response, err := a.Run(ctx, input)
		if err != nil {
			sendAgentStreamChunk(ctx, chunks, llm.StreamChunk{Error: err, Done: true})
			return
		}
		if response != "" {
			if !sendAgentStreamChunk(ctx, chunks, llm.StreamChunk{Content: response}) {
				return
			}
		}
		sendAgentStreamChunk(ctx, chunks, llm.StreamChunk{Done: true})
	}()
	return chunks
}

func (a *Agent) forwardStream(ctx context.Context, providerChunks <-chan llm.StreamChunk, chunks chan<- llm.StreamChunk) {
	defer close(chunks)

	var content strings.Builder
	for {
		select {
		case <-ctx.Done():
			return
		case chunk, ok := <-providerChunks:
			if !ok {
				a.storeStreamResponse(content.String())
				return
			}

			if chunk.Content != "" {
				content.WriteString(chunk.Content)
			}
			if !sendAgentStreamChunk(ctx, chunks, chunk) {
				return
			}
			if chunk.Error != nil {
				return
			}
			if chunk.Done {
				a.storeStreamResponse(content.String())
				return
			}
		}
	}
}

func (a *Agent) storeStreamResponse(content string) {
	a.memory.Add(llm.Message{Role: "assistant", Content: content})
	a.syncSession()
}

func sendAgentStreamChunk(ctx context.Context, chunks chan<- llm.StreamChunk, chunk llm.StreamChunk) bool {
	select {
	case <-ctx.Done():
		return false
	case chunks <- chunk:
		return true
	}
}
