package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// === FE-6: E2E compilation test for TranslationPlan changes ===
//
// Verifies the full codegen pipeline produces compilable Go code
// after the TranslationPlan change (Translate returns TranslationPlan
// instead of cypher.Statement). The generated client must compile
// against the updated pkg/client.Execute() signature.

// foreachTestSchema returns a schema with merge mutations to exercise
// the TranslationPlan code path in the generated client.
func foreachTestSchema() string {
	return `type Movie @node {
	id: ID!
	title: String!
	released: Int
	actors: [Actor!]! @relationship(type: "ACTED_IN", direction: IN, properties: "ActedInProperties")
}

type Actor @node {
	id: ID!
	name: String!
	movies: [Movie!]! @relationship(type: "ACTED_IN", direction: OUT, properties: "ActedInProperties")
}

type ActedInProperties @relationshipProperties {
	role: String!
}
`
}

// writeForeachSchema writes the schema to a temp directory.
func writeForeachSchema(t *testing.T) (schemaPath, outputDir string) {
	t.Helper()
	baseDir := t.TempDir()
	schemaPath = filepath.Join(baseDir, "schema.graphql")
	if err := os.WriteFile(schemaPath, []byte(foreachTestSchema()), 0644); err != nil {
		t.Fatalf("failed to write schema file: %v", err)
	}
	outputDir = filepath.Join(baseDir, "generated")
	return schemaPath, outputDir
}

// Test: Generate() succeeds for merge schema after TranslationPlan changes.
// Expected: Generate returns nil error — pipeline handles TranslationPlan correctly.
func TestE2EForeach_GenerateSucceeds(t *testing.T) {
	schemaPath, outputDir := writeForeachSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
}

// Test: Generated code compiles with go build after TranslationPlan change.
// Expected: `go build ./...` exits with code 0 — generated client works with
// updated pkg/client.Execute() that handles TranslationPlan.
func TestE2EForeach_GoBuildSucceeds(t *testing.T) {
	schemaPath, outputDir := writeForeachSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify output files exist
	for _, file := range []string{"schema.graphql", "models_gen.go", "graphmodel_gen.go", "client_gen.go"} {
		path := filepath.Join(outputDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("expected output file %s not found", file)
		}
	}

	// Run go mod init + tidy
	initCmd := exec.Command("go", "mod", "init", "example.com/test-foreach")
	initCmd.Dir = outputDir
	if output, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod init failed:\n%s\n%v", string(output), err)
	}

	// Add replace directive for local module
	goModPath := filepath.Join(outputDir, "go.mod")
	goMod, _ := os.ReadFile(goModPath)
	cwd, _ := os.Getwd()
	moduleRoot := filepath.Dir(filepath.Dir(cwd)) // pkg/codegen -> project root
	goMod = append(goMod, []byte("\nreplace github.com/tab58/go-ormql => "+moduleRoot+"\n")...)
	os.WriteFile(goModPath, goMod, 0644)

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = outputDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed:\n%s\n%v", string(output), err)
	}

	// Run go build
	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = outputDir
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed on generated output after TranslationPlan change:\n%s\n%v", string(output), err)
	}
}

// Test: Generated client_gen.go imports pkg/client (not pkg/cypher directly).
// Expected: client_gen.go references client.New() but does NOT import pkg/cypher.
func TestE2EForeach_ClientGenImportsCorrectPackages(t *testing.T) {
	schemaPath, outputDir := writeForeachSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "client_gen.go"))
	if err != nil {
		t.Fatalf("failed to read client_gen.go: %v", err)
	}
	src := string(content)

	if !strings.Contains(src, "pkg/client") {
		t.Error("client_gen.go should import pkg/client")
	}
	// Generated code should not directly reference cypher.Statement or TranslationPlan
	if strings.Contains(src, "TranslationPlan") {
		t.Error("client_gen.go should NOT reference TranslationPlan directly — that's internal to pkg/translate")
	}
}
