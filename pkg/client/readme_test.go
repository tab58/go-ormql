package client

import (
	"os"
	"strings"
	"testing"
)

// --- CH-5: README documents auto-chunking ---

// Test: README.md contains documentation about auto-chunking/batching.
// Expected: README mentions WithBatchSize, batch size, and chunking behavior.
func TestREADME_DocumentsBatching(t *testing.T) {
	content, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}
	readme := string(content)
	lower := strings.ToLower(readme)

	checks := []struct {
		name    string
		keyword string
	}{
		{"WithBatchSize option", "withbatchsize"},
		{"default batch size", "50"},
		{"bulk mutations", "bulk"},
		{"chunking/batching behavior", "batch"},
	}

	for _, check := range checks {
		if !strings.Contains(lower, check.keyword) {
			t.Errorf("README should document %s (keyword: %q)", check.name, check.keyword)
		}
	}
}
