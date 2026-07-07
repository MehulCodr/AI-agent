package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"

	apperrors "github.com/MehulCodr/AI-agent/internal/errors"
	"github.com/MehulCodr/AI-agent/internal/llm"
)

type StreamingChatRunner interface {
	Stream(ctx context.Context, input string) (<-chan llm.StreamChunk, error)
}

func printStream(ctx context.Context, output io.Writer, chunks <-chan llm.StreamChunk) error {
	if chunks == nil {
		return fmt.Errorf("%w: stream channel is required", apperrors.ErrInvalidInput)
	}
	defer fmt.Fprintln(output)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case chunk, ok := <-chunks:
			if !ok {
				return nil
			}
			if chunk.Content != "" {
				if _, err := io.WriteString(output, chunk.Content); err != nil {
					return err
				}
			}
			if chunk.Error != nil {
				return chunk.Error
			}
			if chunk.Done {
				return nil
			}
		}
	}
}

func runRunnerStream(ctx context.Context, runner ChatRunner, input string) (<-chan llm.StreamChunk, error) {
	if streaming, ok := runner.(StreamingChatRunner); ok {
		return streaming.Stream(ctx, input)
	}

	chunks := make(chan llm.StreamChunk)
	go func() {
		defer close(chunks)

		response, err := runner.Run(ctx, input)
		if err != nil {
			sendCLIStreamChunk(ctx, chunks, llm.StreamChunk{Error: err, Done: true})
			return
		}
		if response != "" {
			if !sendCLIStreamChunk(ctx, chunks, llm.StreamChunk{Content: response}) {
				return
			}
		}
		sendCLIStreamChunk(ctx, chunks, llm.StreamChunk{Done: true})
	}()
	return chunks, nil
}

func sendCLIStreamChunk(ctx context.Context, chunks chan<- llm.StreamChunk, chunk llm.StreamChunk) bool {
	select {
	case <-ctx.Done():
		return false
	case chunks <- chunk:
		return true
	}
}

func interruptContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}
