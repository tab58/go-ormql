package codegen

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// === CLN-1: Verify pkg/server/ removal ===

// projectRoot returns the absolute path to the project root by walking up
// from the current test file's directory until we find go.mod.
func projectRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine test file path")
	}
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// TestCleanup_ServerPackageRemoved verifies that the pkg/server/ directory
// does not contain any .go files. After CLN-1 is implemented, the entire
// directory should be deleted.
// Expected: no .go files exist under pkg/server/.
func TestCleanup_ServerPackageRemoved(t *testing.T) {
	root := projectRoot(t)
	serverDir := filepath.Join(root, "pkg", "server")

	// If the directory doesn't exist at all, the test passes.
	info, err := os.Stat(serverDir)
	if os.IsNotExist(err) {
		return // directory already removed — pass
	}
	if err != nil {
		t.Fatalf("error checking pkg/server/: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("pkg/server exists but is not a directory")
	}

	// Directory exists — check for .go files
	entries, err := os.ReadDir(serverDir)
	if err != nil {
		t.Fatalf("failed to read pkg/server/: %v", err)
	}
	var goFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			goFiles = append(goFiles, e.Name())
		}
	}
	if len(goFiles) > 0 {
		t.Errorf("pkg/server/ still contains Go files (should be removed): %v", goFiles)
	}
}

// TestCleanup_NoServerImports verifies that no Go source files outside
// pkg/server/ import the server package.
// Expected: zero files import "github.com/tab58/gql-orm/pkg/server".
func TestCleanup_NoServerImports(t *testing.T) {
	root := projectRoot(t)

	var offending []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		// Skip hidden directories and the server package itself
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			if path == filepath.Join(root, "pkg", "server") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip this test file to avoid self-detection
		if filepath.Base(path) == "cleanup_test.go" {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		// Search for the import path (split to avoid this test file matching itself)
		importPath := "github.com/tab58/gql-orm/pkg/" + "server"
		if strings.Contains(string(content), `"`+importPath+`"`) {
			rel, _ := filepath.Rel(root, path)
			offending = append(offending, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("error walking project: %v", err)
	}
	if len(offending) > 0 {
		t.Errorf("files still importing pkg/server (should be removed): %v", offending)
	}
}
