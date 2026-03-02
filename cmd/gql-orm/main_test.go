package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTempSchema writes a minimal .graphql file to a temp directory
// and returns the file path.
func writeTempSchema(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.graphql")
	sdl := `type Movie @node {
	id: ID!
	title: String!
}
`
	if err := os.WriteFile(path, []byte(sdl), 0644); err != nil {
		t.Fatalf("failed to write temp schema: %v", err)
	}
	return path
}

// --- Tests ---

// TestRun_NoArgs verifies that run with no arguments returns an error
// indicating a subcommand is required.
// Expected: non-nil error mentioning "subcommand" or usage.
func TestRun_NoArgs(t *testing.T) {
	err := run([]string{})
	if err == nil {
		t.Fatal("run() with no args should return error")
	}
}

// TestRun_UnknownSubcommand verifies that run with an unknown subcommand
// returns an error.
// Expected: non-nil error mentioning the unknown command.
func TestRun_UnknownSubcommand(t *testing.T) {
	err := run([]string{"foobar"})
	if err == nil {
		t.Fatal("run('foobar') should return error for unknown subcommand")
	}
}

// TestRun_GenerateRoutes verifies that "generate" is recognized as a valid
// subcommand. Without required flags, it should still error (missing --schema).
// Expected: error about missing required flag, not about unknown subcommand.
func TestRun_GenerateRoutes(t *testing.T) {
	err := run([]string{"generate"})
	if err == nil {
		t.Fatal("run('generate') with no flags should return error (missing required flags)")
	}
	// Should not be an "unknown subcommand" error
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "unknown") {
		t.Errorf("error should be about missing flags, not unknown subcommand: %v", err)
	}
}

// TestRunGenerate_MissingSchemaFlag verifies that runGenerate returns an error
// when --schema is not provided.
// Expected: non-nil error mentioning "schema".
func TestRunGenerate_MissingSchemaFlag(t *testing.T) {
	err := runGenerate([]string{"--output", t.TempDir()})
	if err == nil {
		t.Fatal("runGenerate without --schema should return error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "schema") {
		t.Errorf("error should mention 'schema': %v", err)
	}
}

// TestRunGenerate_MissingOutputFlag verifies that runGenerate returns an error
// when --output is not provided.
// Expected: non-nil error mentioning "output".
func TestRunGenerate_MissingOutputFlag(t *testing.T) {
	schema := writeTempSchema(t)
	err := runGenerate([]string{"--schema", schema})
	if err == nil {
		t.Fatal("runGenerate without --output should return error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "output") {
		t.Errorf("error should mention 'output': %v", err)
	}
}

// TestRunGenerate_ValidFlags verifies that runGenerate accepts valid --schema
// and --output flags without a flag-parsing error.
// Expected: runGenerate either succeeds (nil error) or returns an implementation
// error — not a flag-parsing error.
func TestRunGenerate_ValidFlags(t *testing.T) {
	schema := writeTempSchema(t)
	output := t.TempDir()

	err := runGenerate([]string{"--schema", schema, "--output", output})
	// Stub returns nil, but once implemented this should call codegen.Generate.
	// The key check: no flag-parsing error.
	if err != nil {
		s := strings.ToLower(err.Error())
		if strings.Contains(s, "flag") || strings.Contains(s, "usage") {
			t.Errorf("valid flags should not produce flag-parsing error: %v", err)
		}
	}
}

// TestRunGenerate_DefaultPackageName verifies that when --package is not provided,
// the default "generated" is used. Since the stub doesn't run the pipeline,
// we verify it doesn't error on missing --package.
// Expected: no flag-parsing error when --package is omitted.
func TestRunGenerate_DefaultPackageName(t *testing.T) {
	schema := writeTempSchema(t)
	output := t.TempDir()

	err := runGenerate([]string{"--schema", schema, "--output", output})
	if err != nil {
		s := strings.ToLower(err.Error())
		if strings.Contains(s, "package") {
			t.Errorf("missing --package should not error (has default): %v", err)
		}
	}
}

// TestRunGenerate_CustomPackageName verifies that --package flag is accepted.
// Expected: no flag-parsing error when --package is provided.
func TestRunGenerate_CustomPackageName(t *testing.T) {
	schema := writeTempSchema(t)
	output := t.TempDir()

	err := runGenerate([]string{
		"--schema", schema,
		"--output", output,
		"--package", "myapp",
	})
	if err != nil {
		s := strings.ToLower(err.Error())
		if strings.Contains(s, "flag") || strings.Contains(s, "package") {
			t.Errorf("valid --package flag should not error: %v", err)
		}
	}
}

// TestRunGenerate_MultipleSchemaFiles verifies that --schema accepts
// comma-separated file paths for multiple schema files.
// Expected: no flag-parsing error with comma-separated paths.
func TestRunGenerate_MultipleSchemaFiles(t *testing.T) {
	dir := t.TempDir()
	schema1 := filepath.Join(dir, "types.graphql")
	schema2 := filepath.Join(dir, "queries.graphql")

	os.WriteFile(schema1, []byte("type Movie @node { id: ID! }"), 0644)
	os.WriteFile(schema2, []byte("type Actor @node { id: ID! }"), 0644)

	output := t.TempDir()
	paths := schema1 + "," + schema2

	err := runGenerate([]string{"--schema", paths, "--output", output})
	if err != nil {
		s := strings.ToLower(err.Error())
		if strings.Contains(s, "flag") || strings.Contains(s, "schema") {
			t.Errorf("comma-separated --schema should be accepted: %v", err)
		}
	}
}


// === CLN-2: Remove serve CLI subcommand tests ===

// TestCLN2_ServeIsUnknownSubcommand verifies that after removal of the serve
// subcommand, passing "serve" to run() returns an "unknown subcommand" error.
// Expected: non-nil error containing "unknown".
func TestCLN2_ServeIsUnknownSubcommand(t *testing.T) {
	err := run([]string{"serve"})
	if err == nil {
		t.Fatal("run('serve') should return error after serve subcommand is removed")
	}
	s := strings.ToLower(err.Error())
	if !strings.Contains(s, "unknown") {
		t.Errorf("error for 'serve' should say 'unknown subcommand', got: %v", err)
	}
}

// TestCLN2_UsageMessageExcludesServe verifies that the usage/error message
// when no subcommand is given does NOT mention "serve" as a valid option.
// Only "generate" should be listed.
// Expected: error message does not contain "serve".
func TestCLN2_UsageMessageExcludesServe(t *testing.T) {
	err := run([]string{})
	if err == nil {
		t.Fatal("run() with no args should return error")
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "serve") {
		t.Errorf("usage message should not mention 'serve' after removal: %v", err)
	}
}

// TestCLN2_GenerateStillWorks verifies that the generate subcommand still
// works correctly after serve removal.
// Expected: generate is recognized (no "unknown subcommand" error).
func TestCLN2_GenerateStillWorks(t *testing.T) {
	schema := writeTempSchema(t)
	output := t.TempDir()

	err := run([]string{"generate", "--schema", schema, "--output", output})
	if err != nil {
		s := strings.ToLower(err.Error())
		if strings.Contains(s, "unknown") {
			t.Errorf("generate should still be recognized after serve removal: %v", err)
		}
	}
}

// --- CG-28: CLI @vector warning tests ---

// writeTempSchemaWithVector writes a .graphql schema with @vector directive
// to a temp directory and returns the file path.
func writeTempSchemaWithVector(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.graphql")
	sdl := `type Movie @node {
	id: ID!
	title: String!
	embedding: [Float!]! @vector(indexName: "movie_embeddings", dimensions: 1536, similarity: "cosine")
}
`
	if err := os.WriteFile(path, []byte(sdl), 0644); err != nil {
		t.Fatalf("failed to write temp schema with @vector: %v", err)
	}
	return path
}

// Test: CLI prints @vector warning to stderr when schema has @vector directive.
// Expected: stderr contains "Warning: @vector directive requires Neo4j 5.11+".
func TestRunGenerate_VectorWarningToStderr(t *testing.T) {
	schemaPath := writeTempSchemaWithVector(t)
	outputDir := t.TempDir()

	// Capture stderr by replacing os.Stderr with a pipe.
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	runGenerate([]string{"--schema", schemaPath, "--output", outputDir})

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stderr = origStderr

	stderr := buf.String()
	if !strings.Contains(stderr, "Warning: @vector directive requires Neo4j 5.11+") {
		t.Errorf("expected @vector warning in stderr, got: %q", stderr)
	}
}

// Test: CLI does NOT print @vector warning when schema has no @vector directive.
// Expected: stderr does NOT contain "@vector" or "Neo4j 5.11".
func TestRunGenerate_NoVectorNoWarning(t *testing.T) {
	schemaPath := writeTempSchema(t) // no @vector
	outputDir := t.TempDir()

	// Capture stderr by replacing os.Stderr with a pipe.
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	runGenerate([]string{"--schema", schemaPath, "--output", outputDir})

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stderr = origStderr

	stderr := buf.String()
	if strings.Contains(stderr, "@vector") {
		t.Errorf("stderr should NOT contain @vector warning without @vector directive: %q", stderr)
	}
}
