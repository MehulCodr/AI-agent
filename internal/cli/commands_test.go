package cli

import "testing"

func TestParseOptionsSupportsStreamBoolean(t *testing.T) {
	options, rest, err := parseOptions([]string{"--stream=false", "hello"})
	if err != nil {
		t.Fatalf("parseOptions returned error: %v", err)
	}
	if options.Stream == nil || *options.Stream {
		t.Fatalf("Stream = %v, want false", options.Stream)
	}
	if len(rest) != 1 || rest[0] != "hello" {
		t.Fatalf("rest = %#v, want hello", rest)
	}
}
