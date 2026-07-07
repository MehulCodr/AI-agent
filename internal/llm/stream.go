package llm

import "context"

type StreamChunk struct {
	Content string
	Done    bool
	Error   error
}

type StreamProvider interface {
	Stream(ctx context.Context, messages []Message) (<-chan StreamChunk, error)
}
