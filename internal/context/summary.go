package context

const MaxContextChars = 30000

type Summary struct {
	Root          string
	TotalFiles    int
	GoFiles       int
	Languages     map[string]int
	ImportantDirs []string
	Tree          string
}
