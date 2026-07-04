package cmd

import (
	"context"
	"fmt"
	"io"
)

func Execute(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	_, _, _ = ctx, args, stdin
	fmt.Fprintln(stdout, "magent: llm provider scaffold ready")
	return 0
}
