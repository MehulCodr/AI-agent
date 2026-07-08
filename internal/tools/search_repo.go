package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/MehulCodr/AI-agent/internal/rag"
)

type RepoSearcher interface {
	Search(ctx context.Context, query string, limit int) ([]rag.SearchResult, error)
}

type SearchRepoTool struct {
	Searcher RepoSearcher
}

func (SearchRepoTool) Name() string {
	return "search_repo"
}

func (SearchRepoTool) Description() string {
	return "Searches the Redis-backed repository index for relevant code snippets."
}

func (SearchRepoTool) Parameters() map[string]any {
	return objectSchema([]string{"query"}, map[string]any{
		"query": stringProperty("Search terms or question about the repository."),
		"limit": map[string]any{
			"type":        "number",
			"description": "Maximum number of results. Defaults to 5.",
		},
	})
}

func (t SearchRepoTool) Execute(ctx context.Context, input map[string]any) (string, error) {
	if err := contextError(ctx); err != nil {
		return "", err
	}
	if t.Searcher == nil {
		return "", fmt.Errorf("search_repo requires a Redis repository index")
	}

	query, err := requiredString(input, "query", "search_repo")
	if err != nil {
		return "", err
	}
	limit := 5
	if value, ok := input["limit"]; ok {
		switch typed := value.(type) {
		case float64:
			if typed > 0 {
				limit = int(typed)
			}
		case int:
			if typed > 0 {
				limit = typed
			}
		default:
			return "", fmt.Errorf("search_repo limit must be a number")
		}
	}

	results, err := t.Searcher.Search(ctx, query, limit)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "no repository matches found", nil
	}

	var builder strings.Builder
	for i, result := range results {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(result.Path)
		builder.WriteString("\n")
		builder.WriteString(result.Snippet)
	}
	return builder.String(), nil
}
