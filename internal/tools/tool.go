package tools

import "context"

type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input map[string]any) (string, error)
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
