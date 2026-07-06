package tools

import (
	"fmt"
	"strings"
)

func editDiff(path, oldText, newText string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("--- %s\n", path))
	builder.WriteString(fmt.Sprintf("+++ %s\n", path))
	builder.WriteString("@@\n")

	for _, line := range diffLines(oldText) {
		builder.WriteString("-")
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	for _, line := range diffLines(newText) {
		builder.WriteString("+")
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	return builder.String()
}

func diffLines(text string) []string {
	if text == "" {
		return []string{""}
	}

	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}
