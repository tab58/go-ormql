package codegen

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- CLN-3: V1 codegen file removal verification ---
// After CLN-3 is implemented, all V1-specific production and test files
// should be deleted from pkg/codegen/. These tests verify non-existence.

// Test: V1 production files should not exist after cleanup.
// Expected: gqlgen.go, resolvers.go, resolvers_helpers.go, mappers.go all removed.
func TestCLN3_V1ProductionFilesRemoved(t *testing.T) {
	v1Files := []string{
		"gqlgen.go",
		"resolvers.go",
		"resolvers_helpers.go",
		"mappers.go",
	}
	for _, name := range v1Files {
		t.Run(name, func(t *testing.T) {
			if _, err := os.Stat(name); err == nil {
				t.Errorf("%s should be removed (V1 artifact)", name)
			}
		})
	}
}

// Test: V1 test files should not exist after cleanup.
// Expected: all V1 resolver/mapper/gqlgen test files removed.
func TestCLN3_V1TestFilesRemoved(t *testing.T) {
	v1TestFiles := []string{
		"gqlgen_test.go",
		"resolvers_test.go",
		"resolvers_connection_test.go",
		"resolvers_cypher_test.go",
		"resolvers_nested_test.go",
		"resolvers_where_sort_test.go",
		"mappers_test.go",
		"cleanup_test.go",
	}
	for _, name := range v1TestFiles {
		t.Run(name, func(t *testing.T) {
			if _, err := os.Stat(name); err == nil {
				t.Errorf("%s should be removed (V1 test artifact)", name)
			}
		})
	}
}

// Test: V1 E2E test files should not exist after cleanup.
// Expected: e2e_test.go and e2e_tier1_test.go removed.
func TestCLN3_V1E2ETestFilesRemoved(t *testing.T) {
	v1E2EFiles := []string{
		"e2e_test.go",
		"e2e_tier1_test.go",
	}
	for _, name := range v1E2EFiles {
		t.Run(name, func(t *testing.T) {
			if _, err := os.Stat(name); err == nil {
				t.Errorf("%s should be removed (V1 E2E test)", name)
			}
		})
	}
}

// --- CLN-4: Remove gqlgen dependency + update example ---
// After CLN-4 is implemented, gqlgen should no longer appear in go.mod
// and V1 example generated files should be removed.

// cln4ProjectRoot returns the project root by walking up from the test file
// directory until go.mod is found.
func cln4ProjectRoot(t *testing.T) string {
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

// Test: gqlgen should not be in go.mod after cleanup.
// Expected: go.mod does not contain "gqlgen" or "99designs/gqlgen".
func TestCLN4_GqlgenNotInGoMod(t *testing.T) {
	root := cln4ProjectRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}
	if strings.Contains(string(content), "gqlgen") {
		t.Error("go.mod should not contain 'gqlgen' dependency after cleanup")
	}
}

// Test: V1 example generated files should not exist after cleanup.
// Expected: exec_gen.go, gqlgen.yml, resolvers_gen.go, mappers_gen.go removed
// from cmd/example/generated/.
func TestCLN4_V1ExampleFilesRemoved(t *testing.T) {
	root := cln4ProjectRoot(t)
	exampleDir := filepath.Join(root, "cmd", "example", "generated")

	v1Files := []string{
		"exec_gen.go",
		"gqlgen.yml",
		"resolvers_gen.go",
		"mappers_gen.go",
	}
	for _, name := range v1Files {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(exampleDir, name)
			if _, err := os.Stat(path); err == nil {
				t.Errorf("cmd/example/generated/%s should be removed (V1 artifact)", name)
			}
		})
	}
}

// Test: No Go source files in the project import gqlgen packages.
// Expected: zero files import "github.com/99designs/gqlgen".
func TestCLN4_NoGqlgenImports(t *testing.T) {
	root := cln4ProjectRoot(t)

	var offending []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip this test file to avoid self-detection from comments.
		if filepath.Base(path) == "v1_removal_test.go" {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		// Split the import path to avoid this test file matching itself.
		importPrefix := "github.com/99designs/"
		importPkg := "gqlgen"
		if strings.Contains(string(content), `"`+importPrefix+importPkg) {
			rel, _ := filepath.Rel(root, path)
			offending = append(offending, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("error walking project: %v", err)
	}
	if len(offending) > 0 {
		t.Errorf("files still importing gqlgen (should be removed): %v", offending)
	}
}
