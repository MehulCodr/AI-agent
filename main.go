package main

import (
	"context"
	"os"

	"AI-agent/cmd"
)

func main() {
	os.Exit(cmd.Execute(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
