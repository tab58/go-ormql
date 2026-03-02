package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- CG-28: Pipeline step 6 (GenerateIndexes) tests ---

// writeSampleSchemaWithVector writes a .graphql schema that includes a @vector
// directive on a [Float!]! field to a temp directory and returns the file path.
func writeSampleSchemaWithVector(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.graphql")
	sdl := `type Movie @node {
	id: ID!
	title: String!
	embedding: [Float!]! @vector(indexName: "movie_embeddings", dimensions: 1536, similarity: "cosine")
}
`
	if err := os.WriteFile(schemaPath, []byte(sdl), 0644); err != nil {
		t.Fatalf("failed to write sample schema with @vector: %v", err)
	}
	return schemaPath
}

// Test: Generate writes indexes_gen.go when schema has @vector directive.
// Expected: indexes_gen.go exists in output directory after Generate().
func TestGenerate_WritesIndexesGenWhenVectorPresent(t *testing.T) {
	schemaPath := writeSampleSchemaWithVector(t)
	outputDir := t.TempDir()
	cfg := Config{SchemaFiles: []string{schemaPath}, OutputDir: outputDir, PackageName: "generated"}

	Generate(cfg) // may error if @vector parsing not yet implemented

	indexesPath := filepath.Join(outputDir, "indexes_gen.go")
	if _, statErr := os.Stat(indexesPath); os.IsNotExist(statErr) {
		t.Fatal("Generate should write indexes_gen.go when schema has @vector directive")
	}
}

// Test: Generate does NOT write indexes_gen.go when schema has no @vector directive.
// Expected: indexes_gen.go does NOT exist in output directory.
func TestGenerate_SkipsIndexesGenWhenNoVector(t *testing.T) {
	schemaPath := writeSampleSchema(t) // no @vector
	outputDir := t.TempDir()
	cfg := Config{SchemaFiles: []string{schemaPath}, OutputDir: outputDir, PackageName: "generated"}

	Generate(cfg)

	indexesPath := filepath.Join(outputDir, "indexes_gen.go")
	if _, statErr := os.Stat(indexesPath); !os.IsNotExist(statErr) {
		t.Fatal("Generate should NOT write indexes_gen.go when schema has no @vector directive")
	}
}

// Test: Generated indexes_gen.go contains CreateIndexes function declaration.
// Expected: indexes_gen.go content includes "func CreateIndexes".
func TestGenerate_IndexesGenContainsCreateIndexes(t *testing.T) {
	schemaPath := writeSampleSchemaWithVector(t)
	outputDir := t.TempDir()
	cfg := Config{SchemaFiles: []string{schemaPath}, OutputDir: outputDir, PackageName: "generated"}

	Generate(cfg) // may error if @vector parsing not yet implemented

	content, readErr := os.ReadFile(filepath.Join(outputDir, "indexes_gen.go"))
	if readErr != nil {
		t.Fatalf("indexes_gen.go not found or unreadable: %v", readErr)
	}
	if !strings.Contains(string(content), "func CreateIndexes") {
		t.Errorf("indexes_gen.go missing 'func CreateIndexes':\n%s", string(content))
	}
}

// Test: cleanGeneratedGoFiles removes stale indexes_gen.go on re-runs.
// Expected: pre-existing indexes_gen.go is removed by cleanGeneratedGoFiles.
func TestCleanGeneratedGoFiles_CleansIndexesGen(t *testing.T) {
	dir := t.TempDir()
	indexesPath := filepath.Join(dir, "indexes_gen.go")
	if err := os.WriteFile(indexesPath, []byte("package generated\n"), 0644); err != nil {
		t.Fatalf("failed to create indexes_gen.go: %v", err)
	}

	cleanGeneratedGoFiles(dir)

	if _, err := os.Stat(indexesPath); !os.IsNotExist(err) {
		t.Fatal("cleanGeneratedGoFiles should remove stale indexes_gen.go")
	}
}
