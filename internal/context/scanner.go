package context

import (
	stdcontext "context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ignoredDirs = map[string]bool{
	".agent":       true,
	".agents":      true,
	".codex":       true,
	".git":         true,
	"build":        true,
	"dist":         true,
	"node_modules": true,
	"vendor":       true,
}

var ignoredFiles = map[string]bool{
	".env": true,
}

var languageByExtension = map[string]string{
	".css":  "CSS",
	".go":   "Go",
	".html": "HTML",
	".js":   "JavaScript",
	".json": "JSON",
	".md":   "Markdown",
	".py":   "Python",
	".rs":   "Rust",
	".sh":   "Shell",
	".ts":   "TypeScript",
	".tsx":  "TypeScript",
	".yaml": "YAML",
	".yml":  "YAML",
}

type Scanner struct {
	Root            string
	MaxContextChars int
}

func NewScanner(root string) *Scanner {
	return &Scanner{
		Root:            root,
		MaxContextChars: MaxContextChars,
	}
}

func (s *Scanner) Scan(ctx stdcontext.Context) (*Summary, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	root := s.Root
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}
		root = wd
	}

	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	maxChars := s.MaxContextChars
	if maxChars <= 0 {
		maxChars = MaxContextChars
	}

	summary := &Summary{
		Root:      root,
		Languages: make(map[string]int),
	}

	var tree strings.Builder
	dirSet := make(map[string]bool)

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return fmt.Errorf("walk %s: %w", path, walkErr)
		}

		if path == root {
			return nil
		}

		name := entry.Name()
		if entry.IsDir() {
			if ignoredDirs[name] {
				return filepath.SkipDir
			}
			addImportantDir(root, path, dirSet)
			addTreeLine(&tree, root, path, true, maxChars)
			return nil
		}

		if ignoredFiles[name] {
			return nil
		}

		summary.TotalFiles++
		extension := strings.ToLower(filepath.Ext(name))
		if extension == ".go" {
			summary.GoFiles++
		}
		if language := languageByExtension[extension]; language != "" {
			summary.Languages[language]++
		}

		addImportantDir(root, filepath.Dir(path), dirSet)
		addTreeLine(&tree, root, path, false, maxChars)
		return nil
	})
	if err != nil {
		return nil, err
	}

	summary.ImportantDirs = sortedKeys(dirSet)
	summary.Tree = strings.TrimRight(tree.String(), "\n")
	return summary, nil
}

func addImportantDir(root, path string, dirs map[string]bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return
	}

	first := strings.Split(filepath.ToSlash(rel), "/")[0]
	if first != "" && !ignoredDirs[first] {
		dirs[first] = true
	}
}

func addTreeLine(tree *strings.Builder, root, path string, isDir bool, maxChars int) {
	if tree.Len() >= maxChars {
		return
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return
	}
	rel = filepath.ToSlash(rel)
	if isDir {
		rel += "/"
	}

	line := rel + "\n"
	remaining := maxChars - tree.Len()
	if len(line) > remaining {
		tree.WriteString(line[:remaining])
		return
	}
	tree.WriteString(line)
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
