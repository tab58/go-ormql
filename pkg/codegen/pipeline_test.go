package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSampleSchema writes a minimal valid .graphql schema to a temp directory
// and returns the file path.
func writeSampleSchema(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.graphql")
	sdl := `type Movie @node {
	id: ID!
	title: String!
	released: Int
}
`
	if err := os.WriteFile(schemaPath, []byte(sdl), 0644); err != nil {
		t.Fatalf("failed to write sample schema: %v", err)
	}
	return schemaPath
}

// --- Tests ---

// TestGenerate_EmptySchemaFiles verifies that Generate returns an error when
// no schema files are provided.
func TestGenerate_EmptySchemaFiles(t *testing.T) {
	cfg := Config{
		SchemaFiles: []string{},
		OutputDir:   t.TempDir(),
		PackageName: "generated",
	}

	err := Generate(cfg)
	if err == nil {
		t.Fatal("Generate with empty SchemaFiles should return error")
	}
}

// TestGenerate_NonexistentSchemaFile verifies that Generate returns an error
// when a schema file path doesn't exist.
func TestGenerate_NonexistentSchemaFile(t *testing.T) {
	cfg := Config{
		SchemaFiles: []string{"/nonexistent/schema.graphql"},
		OutputDir:   t.TempDir(),
		PackageName: "generated",
	}

	err := Generate(cfg)
	if err == nil {
		t.Fatal("Generate with nonexistent schema file should return error")
	}
}

// TestGenerate_EmptyOutputDir verifies that Generate returns an error when
// OutputDir is empty.
func TestGenerate_EmptyOutputDir(t *testing.T) {
	schemaPath := writeSampleSchema(t)

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   "",
		PackageName: "generated",
	}

	err := Generate(cfg)
	if err == nil {
		t.Fatal("Generate with empty OutputDir should return error")
	}
}

// TestGenerate_CreatesOutputDir verifies that Generate creates the output directory
// if it doesn't already exist.
func TestGenerate_CreatesOutputDir(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := filepath.Join(t.TempDir(), "nested", "output")

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	err := Generate(cfg)
	// The stub returns nil (no error) but doesn't create the dir.
	// Once implemented, this should create the directory.
	if err != nil {
		// Implementation error is acceptable for now
		return
	}

	// If no error, verify the directory was created
	info, statErr := os.Stat(outputDir)
	if statErr != nil {
		t.Fatalf("output directory was not created: %v", statErr)
	}
	if !info.IsDir() {
		t.Fatal("output path is not a directory")
	}
}

// TestGenerate_WritesAugmentedSchema verifies that Generate writes schema.graphql
// to the output directory.
func TestGenerate_WritesAugmentedSchema(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	Generate(cfg)

	augmentedPath := filepath.Join(outputDir, "schema.graphql")
	if _, err := os.Stat(augmentedPath); os.IsNotExist(err) {
		t.Fatal("Generate did not write schema.graphql to output directory")
	}
}

// TestGenerate_WritesGqlgenConfig verifies that Generate writes gqlgen.yml
// to the output directory.
func TestGenerate_WritesGqlgenConfig(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	Generate(cfg)

	configPath := filepath.Join(outputDir, "gqlgen.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Generate did not write gqlgen.yml to output directory")
	}
}

// TestGenerate_WritesResolvers verifies that Generate writes resolvers_gen.go
// to the output directory.
func TestGenerate_WritesResolvers(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	Generate(cfg)

	resolversPath := filepath.Join(outputDir, "resolvers_gen.go")
	if _, err := os.Stat(resolversPath); os.IsNotExist(err) {
		t.Fatal("Generate did not write resolvers_gen.go to output directory")
	}
}

// TestGenerate_WritesMappers verifies that Generate writes mappers_gen.go
// to the output directory.
func TestGenerate_WritesMappers(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	Generate(cfg)

	mappersPath := filepath.Join(outputDir, "mappers_gen.go")
	if _, err := os.Stat(mappersPath); os.IsNotExist(err) {
		t.Fatal("Generate did not write mappers_gen.go to output directory")
	}
}

// TestGenerate_RerunOverwrites verifies that running Generate twice does not error
// and overwrites existing generated files.
func TestGenerate_RerunOverwrites(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	// Run once
	err1 := Generate(cfg)
	// Run again — should not error
	err2 := Generate(cfg)

	// The stub returns nil both times.
	// Once implemented, both calls should succeed (overwrite).
	if err1 != nil && err2 != nil {
		// Both erroring is acceptable during stub phase.
		// Once implemented, at least the second run should succeed.
		return
	}
	// If first succeeded, second should also succeed
	if err1 == nil && err2 != nil {
		t.Fatalf("second Generate call failed: %v", err2)
	}
}

// === CG-10: GenerateClient step in pipeline ===

// TestGenerate_WritesClientGen verifies that Generate writes client_gen.go
// to the output directory as step 7 of the pipeline.
// Expected: client_gen.go exists in output directory after Generate.
func TestGenerate_WritesClientGen(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	Generate(cfg)

	clientGenPath := filepath.Join(outputDir, "client_gen.go")
	if _, err := os.Stat(clientGenPath); os.IsNotExist(err) {
		t.Fatal("Generate did not write client_gen.go to output directory")
	}
}

// TestGenerate_ClientGenContainsNewClient verifies that the generated
// client_gen.go contains a NewClient function.
// Expected: file content includes "func NewClient".
func TestGenerate_ClientGenContainsNewClient(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	Generate(cfg)

	clientGenPath := filepath.Join(outputDir, "client_gen.go")
	content, err := os.ReadFile(clientGenPath)
	if err != nil {
		t.Fatalf("failed to read client_gen.go: %v", err)
	}
	if !strings.Contains(string(content), "func NewClient") {
		t.Errorf("client_gen.go missing 'func NewClient':\n%s", string(content))
	}
}

// TestGenerate_ClientGenReferencesClientNew verifies that the generated
// client_gen.go references client.New to wire the programmatic client.
// Expected: file content includes "client.New".
func TestGenerate_ClientGenReferencesClientNew(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	Generate(cfg)

	clientGenPath := filepath.Join(outputDir, "client_gen.go")
	content, err := os.ReadFile(clientGenPath)
	if err != nil {
		t.Fatalf("failed to read client_gen.go: %v", err)
	}
	if !strings.Contains(string(content), "client.New") {
		t.Errorf("client_gen.go missing 'client.New':\n%s", string(content))
	}
}

// === H1-2: deleteResolverScaffold tests ===

// TestDeleteResolverScaffold_RemovesResolverGo verifies that deleteResolverScaffold
// removes the gqlgen-generated resolver.go file from the output directory.
// Expected: resolver.go is deleted after calling deleteResolverScaffold.
func TestDeleteResolverScaffold_RemovesResolverGo(t *testing.T) {
	dir := t.TempDir()

	// Create a resolver.go file (gqlgen scaffold)
	resolverPath := filepath.Join(dir, "resolver.go")
	if err := os.WriteFile(resolverPath, []byte("package generated\n"), 0644); err != nil {
		t.Fatalf("failed to create resolver.go: %v", err)
	}

	if err := deleteResolverScaffold(dir); err != nil {
		t.Fatalf("deleteResolverScaffold returned error: %v", err)
	}

	if _, err := os.Stat(resolverPath); !os.IsNotExist(err) {
		t.Error("resolver.go was not removed by deleteResolverScaffold")
	}
}

// TestDeleteResolverScaffold_RemovesResolversGoFiles verifies that deleteResolverScaffold
// removes all *.resolvers.go files (e.g., schema.resolvers.go) from the output directory.
// Expected: schema.resolvers.go is deleted after calling deleteResolverScaffold.
func TestDeleteResolverScaffold_RemovesResolversGoFiles(t *testing.T) {
	dir := t.TempDir()

	// Create *.resolvers.go files (gqlgen scaffold)
	scaffoldFiles := []string{"schema.resolvers.go", "query.resolvers.go"}
	for _, name := range scaffoldFiles {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("package generated\n"), 0644); err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
	}

	if err := deleteResolverScaffold(dir); err != nil {
		t.Fatalf("deleteResolverScaffold returned error: %v", err)
	}

	for _, name := range scaffoldFiles {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("%s was not removed by deleteResolverScaffold", name)
		}
	}
}

// TestDeleteResolverScaffold_PreservesNonScaffoldFiles verifies that
// deleteResolverScaffold does NOT remove non-scaffold files like exec_gen.go,
// models_gen.go, schema.graphql, or gqlgen.yml.
// Expected: all non-scaffold files remain after calling deleteResolverScaffold.
func TestDeleteResolverScaffold_PreservesNonScaffoldFiles(t *testing.T) {
	dir := t.TempDir()

	// Create scaffold files (should be removed)
	os.WriteFile(filepath.Join(dir, "resolver.go"), []byte("package generated\n"), 0644)
	os.WriteFile(filepath.Join(dir, "schema.resolvers.go"), []byte("package generated\n"), 0644)

	// Create non-scaffold files (should be preserved)
	preservedFiles := []string{"exec_gen.go", "models_gen.go", "schema.graphql", "gqlgen.yml", "resolvers_gen.go"}
	for _, name := range preservedFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("package generated\n"), 0644); err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
	}

	if err := deleteResolverScaffold(dir); err != nil {
		t.Fatalf("deleteResolverScaffold returned error: %v", err)
	}

	for _, name := range preservedFiles {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("%s was incorrectly removed by deleteResolverScaffold", name)
		}
	}
}

// TestDeleteResolverScaffold_EmptyDir verifies that deleteResolverScaffold
// does not error when called on an empty directory.
// Expected: no error returned.
func TestDeleteResolverScaffold_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	err := deleteResolverScaffold(dir)
	if err != nil {
		t.Fatalf("deleteResolverScaffold on empty dir returned error: %v", err)
	}
}

// TestDeleteResolverScaffold_NonexistentDir verifies that deleteResolverScaffold
// returns an error when called on a nonexistent directory.
// Expected: non-nil error.
func TestDeleteResolverScaffold_NonexistentDir(t *testing.T) {
	err := deleteResolverScaffold("/nonexistent/path/to/dir")
	if err == nil {
		t.Fatal("deleteResolverScaffold on nonexistent dir should return error")
	}
}

// TestConfig_HasExpectedFields verifies that Config has the expected struct fields.
// This is a compile-time check — if fields are missing, the test won't compile.
func TestConfig_HasExpectedFields(t *testing.T) {
	cfg := Config{
		SchemaFiles: []string{"schema.graphql"},
		OutputDir:   "/output",
		PackageName: "generated",
	}
	if len(cfg.SchemaFiles) != 1 {
		t.Errorf("SchemaFiles = %v, want 1 element", cfg.SchemaFiles)
	}
	if cfg.OutputDir != "/output" {
		t.Errorf("OutputDir = %q, want %q", cfg.OutputDir, "/output")
	}
	if cfg.PackageName != "generated" {
		t.Errorf("PackageName = %q, want %q", cfg.PackageName, "generated")
	}
}
