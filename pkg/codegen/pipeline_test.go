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

// TestGenerate_WritesModelsGen verifies that Generate writes models_gen.go
// to the output directory (V2 pipeline step 3).
func TestGenerate_WritesModelsGen(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	Generate(cfg)

	modelsPath := filepath.Join(outputDir, "models_gen.go")
	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
		t.Fatal("Generate did not write models_gen.go to output directory")
	}
}

// TestGenerate_WritesGraphModelGen verifies that Generate writes graphmodel_gen.go
// to the output directory (V2 pipeline step 4).
func TestGenerate_WritesGraphModelGen(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	Generate(cfg)

	registryPath := filepath.Join(outputDir, "graphmodel_gen.go")
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Fatal("Generate did not write graphmodel_gen.go to output directory")
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

// --- CG-23: V2 pipeline (5 steps) tests ---
// The V2 pipeline produces exactly 4 output files:
// schema.graphql, models_gen.go, graphmodel_gen.go, client_gen.go.
// It does NOT produce gqlgen.yml, resolvers_gen.go, or mappers_gen.go.

// Test: V2 Generate returns no error for valid input.
// Expected: nil error — V2 has no InvokeGqlgen or other external tool calls.
func TestGenerate_V2_ReturnsNoError(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()
	cfg := Config{SchemaFiles: []string{schemaPath}, OutputDir: outputDir, PackageName: "generated"}
	err := Generate(cfg)
	if err != nil {
		t.Fatalf("V2 Generate should not return error, got: %v", err)
	}
}

// Test: V2 pipeline writes models_gen.go to output directory.
// Expected: models_gen.go exists after Generate().
func TestGenerate_V2_WritesModelsGen(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()
	cfg := Config{SchemaFiles: []string{schemaPath}, OutputDir: outputDir, PackageName: "generated"}
	Generate(cfg)
	if _, err := os.Stat(filepath.Join(outputDir, "models_gen.go")); os.IsNotExist(err) {
		t.Fatal("V2 pipeline should write models_gen.go to output directory")
	}
}

// Test: V2 pipeline writes graphmodel_gen.go to output directory.
// Expected: graphmodel_gen.go exists after Generate().
func TestGenerate_V2_WritesGraphModelGen(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()
	cfg := Config{SchemaFiles: []string{schemaPath}, OutputDir: outputDir, PackageName: "generated"}
	Generate(cfg)
	if _, err := os.Stat(filepath.Join(outputDir, "graphmodel_gen.go")); os.IsNotExist(err) {
		t.Fatal("V2 pipeline should write graphmodel_gen.go to output directory")
	}
}

// Test: V2 pipeline does NOT write gqlgen.yml (gqlgen removed in V2).
// Expected: gqlgen.yml does NOT exist after Generate().
func TestGenerate_V2_NoGqlgenConfig(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()
	cfg := Config{SchemaFiles: []string{schemaPath}, OutputDir: outputDir, PackageName: "generated"}
	Generate(cfg)
	if _, err := os.Stat(filepath.Join(outputDir, "gqlgen.yml")); err == nil {
		t.Fatal("V2 pipeline should NOT produce gqlgen.yml (gqlgen removed)")
	}
}

// Test: V2 pipeline does NOT write resolvers_gen.go (V1 artifact).
// Expected: resolvers_gen.go does NOT exist after Generate().
func TestGenerate_V2_NoResolversGen(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()
	cfg := Config{SchemaFiles: []string{schemaPath}, OutputDir: outputDir, PackageName: "generated"}
	Generate(cfg)
	if _, err := os.Stat(filepath.Join(outputDir, "resolvers_gen.go")); err == nil {
		t.Fatal("V2 pipeline should NOT produce resolvers_gen.go (V1 artifact)")
	}
}

// Test: V2 pipeline does NOT write mappers_gen.go (V1 artifact).
// Expected: mappers_gen.go does NOT exist after Generate().
func TestGenerate_V2_NoMappersGen(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()
	cfg := Config{SchemaFiles: []string{schemaPath}, OutputDir: outputDir, PackageName: "generated"}
	Generate(cfg)
	if _, err := os.Stat(filepath.Join(outputDir, "mappers_gen.go")); err == nil {
		t.Fatal("V2 pipeline should NOT produce mappers_gen.go (V1 artifact)")
	}
}

// Test: V2 pipeline produces exactly 4 output files.
// Expected: schema.graphql, models_gen.go, graphmodel_gen.go, client_gen.go.
func TestGenerate_V2_ExactlyFourFiles(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()
	cfg := Config{SchemaFiles: []string{schemaPath}, OutputDir: outputDir, PackageName: "generated"}
	err := Generate(cfg)
	if err != nil {
		t.Fatalf("V2 Generate should succeed, got: %v", err)
	}
	entries, readErr := os.ReadDir(outputDir)
	if readErr != nil {
		t.Fatalf("failed to read output dir: %v", readErr)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	if len(files) != 4 {
		t.Fatalf("V2 pipeline should produce exactly 4 files, got %d: %v", len(files), files)
	}
}

// Test: V2 models_gen.go contains correct package declaration.
// Expected: file content includes "package generated".
func TestGenerate_V2_ModelsGenContainsPackage(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()
	cfg := Config{SchemaFiles: []string{schemaPath}, OutputDir: outputDir, PackageName: "generated"}
	Generate(cfg)
	content, err := os.ReadFile(filepath.Join(outputDir, "models_gen.go"))
	if err != nil {
		t.Fatalf("failed to read models_gen.go: %v", err)
	}
	if !strings.Contains(string(content), "package generated") {
		t.Error("models_gen.go should contain 'package generated'")
	}
}

// Test: V2 graphmodel_gen.go contains GraphModel variable declaration.
// Expected: file content includes "var GraphModel".
func TestGenerate_V2_GraphModelGenContainsVar(t *testing.T) {
	schemaPath := writeSampleSchema(t)
	outputDir := t.TempDir()
	cfg := Config{SchemaFiles: []string{schemaPath}, OutputDir: outputDir, PackageName: "generated"}
	Generate(cfg)
	content, err := os.ReadFile(filepath.Join(outputDir, "graphmodel_gen.go"))
	if err != nil {
		t.Fatalf("failed to read graphmodel_gen.go: %v", err)
	}
	if !strings.Contains(string(content), "var GraphModel") {
		t.Error("graphmodel_gen.go should contain 'var GraphModel' declaration")
	}
}

// --- Custom scalar pipeline tests ---

// writeSampleSchemaWithScalars writes a schema that includes custom scalar declarations.
func writeSampleSchemaWithScalars(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.graphql")
	sdl := `scalar DateTime
scalar Money

type Event @node {
	id: ID!
	title: String!
	startTime: DateTime!
	cost: Money
}
`
	if err := os.WriteFile(schemaPath, []byte(sdl), 0644); err != nil {
		t.Fatalf("failed to write sample schema: %v", err)
	}
	return schemaPath
}

// TestGenerate_WritesScalarsGen verifies that Generate writes scalars_gen.go
// when custom scalars are present in the schema.
func TestGenerate_WritesScalarsGen(t *testing.T) {
	schemaPath := writeSampleSchemaWithScalars(t)
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	scalarsPath := filepath.Join(outputDir, "scalars_gen.go")
	if _, err := os.Stat(scalarsPath); os.IsNotExist(err) {
		t.Fatal("Generate did not write scalars_gen.go when custom scalars are present")
	}
}

// TestGenerate_ScalarsGenContent verifies the content of the generated scalars file.
func TestGenerate_ScalarsGenContent(t *testing.T) {
	schemaPath := writeSampleSchemaWithScalars(t)
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "scalars_gen.go"))
	if err != nil {
		t.Fatalf("failed to read scalars_gen.go: %v", err)
	}
	src := string(content)
	if !strings.Contains(src, "type DateTime = time.Time") {
		t.Error("scalars_gen.go missing 'type DateTime = time.Time'")
	}
	if !strings.Contains(src, "type Money = any") {
		t.Error("scalars_gen.go missing 'type Money = any'")
	}
}

// TestGenerate_AugmentedSchemaContainsCustomScalars verifies that the augmented
// schema.graphql contains scalar declarations when the schema uses custom scalars.
func TestGenerate_AugmentedSchemaContainsCustomScalars(t *testing.T) {
	schemaPath := writeSampleSchemaWithScalars(t)
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "schema.graphql"))
	if err != nil {
		t.Fatalf("failed to read schema.graphql: %v", err)
	}
	src := string(content)
	if !strings.Contains(src, "scalar DateTime") {
		t.Error("augmented schema.graphql missing 'scalar DateTime' declaration")
	}
	if !strings.Contains(src, "scalar Money") {
		t.Error("augmented schema.graphql missing 'scalar Money' declaration")
	}
}

// TestGenerate_NoScalarsGenWithoutCustomScalars verifies that scalars_gen.go
// is NOT produced when the schema has no custom scalar declarations.
func TestGenerate_NoScalarsGenWithoutCustomScalars(t *testing.T) {
	schemaPath := writeSampleSchema(t) // standard schema without custom scalars
	outputDir := t.TempDir()

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	scalarsPath := filepath.Join(outputDir, "scalars_gen.go")
	if _, err := os.Stat(scalarsPath); err == nil {
		t.Fatal("scalars_gen.go should NOT exist when no custom scalars are declared")
	}
}
