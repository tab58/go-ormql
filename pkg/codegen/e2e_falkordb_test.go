package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- E2E-6: FalkorDB E2E compilation test ---
// The V2 pipeline with TargetFalkorDB generates code that compiles and uses
// FalkorDB-specific DDL format in indexes_gen.go, plus a VectorIndexes var.

// writeV2FalkorDBSchema writes the full-featured V2 + @vector schema to a temp
// directory and returns the schema file path and output directory.
func writeV2FalkorDBSchema(t *testing.T) (schemaPath, outputDir string) {
	t.Helper()
	baseDir := t.TempDir()
	schemaPath = filepath.Join(baseDir, "schema.graphql")
	// Reuse the same schema as the vector test — includes @vector on Movie.embedding
	if err := os.WriteFile(schemaPath, []byte(v2VectorSchema()), 0644); err != nil {
		t.Fatalf("failed to write schema file: %v", err)
	}
	outputDir = filepath.Join(baseDir, "generated")
	return schemaPath, outputDir
}

// Test: Generate() with TargetFalkorDB completes without error.
// Expected: Generate returns nil error.
func TestE2EFalkorDB_GenerateSucceeds(t *testing.T) {
	schemaPath, outputDir := writeV2FalkorDBSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
		Target:      TargetFalkorDB,
	}
	err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate with TargetFalkorDB failed: %v", err)
	}
}

// Test: FalkorDB target produces indexes_gen.go with FalkorDB DDL format.
// Expected: indexes_gen.go exists and contains FalkorDB-specific DDL
//   (NOT Neo4j's indexConfig/vector.dimensions).
func TestE2EFalkorDB_IndexesGenFalkorDBDDL(t *testing.T) {
	schemaPath, outputDir := writeV2FalkorDBSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
		Target:      TargetFalkorDB,
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate with TargetFalkorDB failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "indexes_gen.go"))
	if err != nil {
		t.Fatalf("indexes_gen.go not found: %v", err)
	}
	src := string(content)

	// FalkorDB DDL should contain FalkorDB-specific syntax without IF NOT EXISTS
	if !strings.Contains(src, "CREATE VECTOR INDEX FOR") {
		t.Error("FalkorDB indexes_gen.go missing 'CREATE VECTOR INDEX FOR'")
	}
	if strings.Contains(src, "IF NOT EXISTS") {
		t.Error("FalkorDB indexes_gen.go should NOT contain 'IF NOT EXISTS'")
	}
	if !strings.Contains(src, "dimension") {
		t.Error("FalkorDB indexes_gen.go missing 'dimension' option key")
	}
	if !strings.Contains(src, "similarityFunction") {
		t.Error("FalkorDB indexes_gen.go missing 'similarityFunction' option key")
	}

	// Should NOT contain Neo4j-specific syntax
	if strings.Contains(src, "indexConfig") {
		t.Error("FalkorDB indexes_gen.go should NOT contain 'indexConfig' (Neo4j-specific)")
	}
	if strings.Contains(src, "vector.dimensions") {
		t.Error("FalkorDB indexes_gen.go should NOT contain 'vector.dimensions' (Neo4j-specific)")
	}
}

// Test: FalkorDB target produces VectorIndexes var in indexes_gen.go.
// Expected: indexes_gen.go contains "var VectorIndexes" and "driver.VectorIndex".
func TestE2EFalkorDB_VectorIndexesVar(t *testing.T) {
	schemaPath, outputDir := writeV2FalkorDBSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
		Target:      TargetFalkorDB,
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate with TargetFalkorDB failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "indexes_gen.go"))
	if err != nil {
		t.Fatalf("indexes_gen.go not found: %v", err)
	}
	src := string(content)

	if !strings.Contains(src, "VectorIndexes") {
		t.Error("FalkorDB indexes_gen.go missing 'VectorIndexes' var")
	}
	if !strings.Contains(src, "driver.VectorIndex") {
		t.Error("FalkorDB indexes_gen.go missing 'driver.VectorIndex' type reference")
	}
}

// Test: FalkorDB generated code compiles with go build.
// Expected: `go build ./...` exits with code 0.
func TestE2EFalkorDB_GoBuildSucceeds(t *testing.T) {
	schemaPath, outputDir := writeV2FalkorDBSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
		Target:      TargetFalkorDB,
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate with TargetFalkorDB failed: %v", err)
	}

	// V2 go.mod: only gormql dependency, NO gqlgen.
	root := v2ProjectRoot(t)
	goModContent := "module generated\n\ngo 1.25\n\nrequire (\n" +
		"\tgithub.com/tab58/go-ormql v0.0.0\n" +
		")\n\n" +
		"replace github.com/tab58/go-ormql => " + root + "\n"
	if err := os.WriteFile(filepath.Join(outputDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go mod tidy
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = outputDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed:\n%s\n%v", string(output), err)
	}

	// Run go build to verify FalkorDB-target generated code compiles
	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = outputDir
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed on FalkorDB-target generated output:\n%s\n%v", string(output), err)
	}
}

// Test: FalkorDB @vector warning mentions FalkorDB version (not Neo4j).
// Expected: stderr contains "FalkorDB 4.2+" when target is FalkorDB.
func TestE2EFalkorDB_VectorWarningTarget(t *testing.T) {
	schemaPath, outputDir := writeV2FalkorDBSchema(t)
	var stderr strings.Builder
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
		Target:      TargetFalkorDB,
		Stderr:      &stderr,
	}
	Generate(cfg) // may error if not fully implemented

	warning := stderr.String()
	if !strings.Contains(warning, "FalkorDB") {
		t.Errorf("FalkorDB target @vector warning should mention 'FalkorDB', got: %q", warning)
	}
	if strings.Contains(warning, "Neo4j 5.11") {
		t.Errorf("FalkorDB target @vector warning should NOT mention 'Neo4j 5.11', got: %q", warning)
	}
}
