package rag

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/redis/go-redis/v9"
)

const maxIndexedFileBytes = 512 * 1024
const maxSnippetRunes = 700

type RedisConfig struct {
	Addr      string
	Password  string
	DB        int
	Namespace string
	Root      string
}

type SearchResult struct {
	Path    string
	Snippet string
	Score   int
}

type RedisIndex struct {
	client    *redis.Client
	namespace string
	root      string
}

func NewRedisIndex(config RedisConfig) (*RedisIndex, error) {
	if strings.TrimSpace(config.Addr) == "" {
		return nil, fmt.Errorf("redis addr is required")
	}
	root := config.Root
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("resolve project root: %w", err)
		}
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}

	namespace := strings.TrimSpace(config.Namespace)
	if namespace == "" {
		namespace = "ai-agent"
	}

	return &RedisIndex{
		client: redis.NewClient(&redis.Options{
			Addr:     config.Addr,
			Password: config.Password,
			DB:       config.DB,
		}),
		namespace: namespace,
		root:      root,
	}, nil
}

func (r *RedisIndex) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

func (r *RedisIndex) Ping(ctx context.Context) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("redis index is required")
	}
	return r.client.Ping(ctx).Err()
}

func (r *RedisIndex) Index(ctx context.Context) (int, error) {
	if r == nil || r.client == nil {
		return 0, fmt.Errorf("redis index is required")
	}
	if err := r.Ping(ctx); err != nil {
		return 0, fmt.Errorf("connect to redis: %w", err)
	}

	key := r.docsKey()
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return 0, err
	}

	count := 0
	err := filepath.WalkDir(r.root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == r.root {
			return nil
		}
		rel, err := filepath.Rel(r.root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if shouldSkipDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkipFile(rel) {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > maxIndexedFileBytes {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !utf8.Valid(data) || looksBinary(data) {
			return nil
		}
		if strings.TrimSpace(string(data)) == "" {
			return nil
		}

		if err := r.client.HSet(ctx, key, rel, string(data)).Err(); err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		return count, err
	}

	if err := r.client.Set(ctx, r.metaKey("root"), r.root, 0).Err(); err != nil {
		return count, err
	}
	return count, nil
}

func (r *RedisIndex) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("redis index is required")
	}
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}
	if limit <= 0 {
		limit = 5
	}
	if err := r.Ping(ctx); err != nil {
		return nil, fmt.Errorf("connect to redis: %w", err)
	}

	docs, err := r.client.HGetAll(ctx, r.docsKey()).Result()
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("repository index is empty; run agent index first")
	}

	terms := queryTerms(query)
	results := make([]SearchResult, 0, len(docs))
	for path, content := range docs {
		score := scoreDocument(path, content, terms)
		if score <= 0 {
			continue
		}
		results = append(results, SearchResult{
			Path:    path,
			Snippet: snippet(content, terms),
			Score:   score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Path < results[j].Path
		}
		return results[i].Score > results[j].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (r *RedisIndex) docsKey() string {
	sum := sha1.Sum([]byte(strings.ToLower(r.root)))
	return r.namespace + ":rag:" + hex.EncodeToString(sum[:8]) + ":docs"
}

func (r *RedisIndex) metaKey(name string) string {
	sum := sha1.Sum([]byte(strings.ToLower(r.root)))
	return r.namespace + ":rag:" + hex.EncodeToString(sum[:8]) + ":" + name
}

func shouldSkipDir(path string) bool {
	name := filepath.Base(path)
	switch name {
	case ".git", ".agent", "node_modules", "vendor", "dist", "build", ".next", "tmp", "bin":
		return true
	default:
		return false
	}
}

func shouldSkipFile(path string) bool {
	name := filepath.Base(path)
	if strings.HasPrefix(name, ".") && name != ".gitignore" {
		return true
	}
	switch name {
	case "go.sum":
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".exe", ".dll", ".so", ".dylib", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico", ".pdf", ".zip", ".tar", ".gz":
		return true
	default:
		return false
	}
}

func looksBinary(data []byte) bool {
	limit := len(data)
	if limit > 8000 {
		limit = 8000
	}
	for _, b := range data[:limit] {
		if b == 0 {
			return true
		}
	}
	return false
}

func queryTerms(query string) []string {
	fields := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})
	seen := map[string]bool{}
	terms := make([]string, 0, len(fields))
	for _, field := range fields {
		if len(field) < 2 || seen[field] {
			continue
		}
		seen[field] = true
		terms = append(terms, field)
	}
	return terms
}

func scoreDocument(path, content string, terms []string) int {
	lowerPath := strings.ToLower(path)
	lowerContent := strings.ToLower(content)
	score := 0
	for _, term := range terms {
		score += 5 * strings.Count(lowerPath, term)
		score += strings.Count(lowerContent, term)
	}
	return score
}

func snippet(content string, terms []string) string {
	lower := strings.ToLower(content)
	index := 0
	for _, term := range terms {
		if i := strings.Index(lower, term); i >= 0 {
			index = i
			break
		}
	}

	start := index - maxSnippetRunes/3
	if start < 0 {
		start = 0
	}
	if start > len(content) {
		start = 0
	}
	end := start + maxSnippetRunes
	if end > len(content) {
		end = len(content)
	}
	return strings.TrimSpace(content[start:end])
}
