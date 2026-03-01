package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// --- GenerateGqlgenConfig tests ---

// TestGenerateGqlgenConfig_NonEmpty verifies that config generation produces non-empty output.
func TestGenerateGqlgenConfig_NonEmpty(t *testing.T) {
	cfg := GqlgenConfig{
		SchemaPath:  "/output/schema.graphql",
		OutputDir:   "/output",
		PackageName: "generated",
	}

	content, err := GenerateGqlgenConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateGqlgenConfig returned error: %v", err)
	}
	if content == "" {
		t.Fatal("GenerateGqlgenConfig returned empty string, want non-empty YAML")
	}
}

// TestGenerateGqlgenConfig_ValidYAML verifies that the output parses as valid YAML.
func TestGenerateGqlgenConfig_ValidYAML(t *testing.T) {
	cfg := GqlgenConfig{
		SchemaPath:  "/output/schema.graphql",
		OutputDir:   "/output",
		PackageName: "generated",
	}

	content, err := GenerateGqlgenConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateGqlgenConfig returned error: %v", err)
	}
	if content == "" {
		t.Skip("GenerateGqlgenConfig returned empty — skipping YAML parse test")
	}

	var parsed map[string]any
	if yamlErr := yaml.Unmarshal([]byte(content), &parsed); yamlErr != nil {
		t.Fatalf("output is not valid YAML: %v\nContent:\n%s", yamlErr, content)
	}
}

// TestGenerateGqlgenConfig_ContainsSchemaPath verifies that the config references
// the augmented schema file path.
func TestGenerateGqlgenConfig_ContainsSchemaPath(t *testing.T) {
	cfg := GqlgenConfig{
		SchemaPath:  "/output/schema.graphql",
		OutputDir:   "/output",
		PackageName: "generated",
	}

	content, err := GenerateGqlgenConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateGqlgenConfig returned error: %v", err)
	}
	if !strings.Contains(content, "schema.graphql") {
		t.Errorf("config missing schema path 'schema.graphql':\n%s", content)
	}
}

// TestGenerateGqlgenConfig_ContainsOutputDir verifies that the config points to
// the output directory for generated model and resolver files.
func TestGenerateGqlgenConfig_ContainsOutputDir(t *testing.T) {
	cfg := GqlgenConfig{
		SchemaPath:  "/output/schema.graphql",
		OutputDir:   "/output",
		PackageName: "generated",
	}

	content, err := GenerateGqlgenConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateGqlgenConfig returned error: %v", err)
	}
	if !strings.Contains(content, "exec_gen.go") {
		t.Errorf("config missing exec filename 'exec_gen.go':\n%s", content)
	}
}

// TestGenerateGqlgenConfig_ContainsPackageName verifies that the config uses
// the configured Go package name.
func TestGenerateGqlgenConfig_ContainsPackageName(t *testing.T) {
	cfg := GqlgenConfig{
		SchemaPath:  "/output/schema.graphql",
		OutputDir:   "/output",
		PackageName: "generated",
	}

	content, err := GenerateGqlgenConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateGqlgenConfig returned error: %v", err)
	}
	if !strings.Contains(content, "generated") {
		t.Errorf("config missing package name 'generated':\n%s", content)
	}
}

// === H1-1: Resolver section in gqlgen config tests ===

// TestGenerateGqlgenConfig_ContainsResolverSection verifies that the generated
// gqlgen.yml contains a top-level "resolver:" YAML key to control gqlgen's
// resolver scaffold output.
// Expected: YAML output contains "resolver:" key.
func TestGenerateGqlgenConfig_ContainsResolverSection(t *testing.T) {
	cfg := GqlgenConfig{
		SchemaPath:  "/output/schema.graphql",
		OutputDir:   "/output",
		PackageName: "generated",
	}

	content, err := GenerateGqlgenConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateGqlgenConfig returned error: %v", err)
	}
	if !strings.Contains(content, "resolver:") {
		t.Errorf("config missing 'resolver:' section:\n%s", content)
	}
}

// TestGenerateGqlgenConfig_ResolverLayout verifies that the resolver section
// contains layout: follow-schema, which tells gqlgen to generate per-schema
// resolver files.
// Expected: YAML output contains "layout: follow-schema".
func TestGenerateGqlgenConfig_ResolverLayout(t *testing.T) {
	cfg := GqlgenConfig{
		SchemaPath:  "/output/schema.graphql",
		OutputDir:   "/output",
		PackageName: "generated",
	}

	content, err := GenerateGqlgenConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateGqlgenConfig returned error: %v", err)
	}
	if !strings.Contains(content, "follow-schema") {
		t.Errorf("config resolver section missing 'layout: follow-schema':\n%s", content)
	}
}

// TestGenerateGqlgenConfig_ResolverPackage verifies that the resolver section
// uses the same package name as the exec/model sections.
// Expected: parsed resolver.package matches the configured PackageName.
func TestGenerateGqlgenConfig_ResolverPackage(t *testing.T) {
	cfg := GqlgenConfig{
		SchemaPath:  "/output/schema.graphql",
		OutputDir:   "/output",
		PackageName: "mypkg",
	}

	content, err := GenerateGqlgenConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateGqlgenConfig returned error: %v", err)
	}

	// Parse and verify the resolver section has the correct package
	var parsed struct {
		Resolver struct {
			Package string `yaml:"package"`
		} `yaml:"resolver"`
	}
	if yamlErr := yaml.Unmarshal([]byte(content), &parsed); yamlErr != nil {
		t.Fatalf("failed to parse YAML: %v", yamlErr)
	}
	if parsed.Resolver.Package != "mypkg" {
		t.Errorf("resolver.package = %q, want %q", parsed.Resolver.Package, "mypkg")
	}
}

// TestGenerateGqlgenConfig_ResolverFilenameTemplate verifies that the resolver
// section contains a filename_template field for controlling scaffold file names.
// Expected: YAML output contains "filename_template" in the resolver section.
func TestGenerateGqlgenConfig_ResolverFilenameTemplate(t *testing.T) {
	cfg := GqlgenConfig{
		SchemaPath:  "/output/schema.graphql",
		OutputDir:   "/output",
		PackageName: "generated",
	}

	content, err := GenerateGqlgenConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateGqlgenConfig returned error: %v", err)
	}
	if !strings.Contains(content, "filename_template") {
		t.Errorf("config resolver section missing 'filename_template':\n%s", content)
	}
}

// TestGenerateGqlgenConfig_ResolverDir verifies that the resolver section
// contains a dir field set to "." (same directory as other generated files).
// Expected: parsed resolver.dir is ".".
func TestGenerateGqlgenConfig_ResolverDir(t *testing.T) {
	cfg := GqlgenConfig{
		SchemaPath:  "/output/schema.graphql",
		OutputDir:   "/output",
		PackageName: "generated",
	}

	content, err := GenerateGqlgenConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateGqlgenConfig returned error: %v", err)
	}

	var parsed struct {
		Resolver struct {
			Dir string `yaml:"dir"`
		} `yaml:"resolver"`
	}
	if yamlErr := yaml.Unmarshal([]byte(content), &parsed); yamlErr != nil {
		t.Fatalf("failed to parse YAML: %v", yamlErr)
	}
	if parsed.Resolver.Dir != "." {
		t.Errorf("resolver.dir = %q, want %q", parsed.Resolver.Dir, ".")
	}
}

// --- InvokeGqlgen tests ---

// TestInvokeGqlgen_MissingConfig verifies that invoking gqlgen with a nonexistent
// config file returns an error.
func TestInvokeGqlgen_MissingConfig(t *testing.T) {
	err := InvokeGqlgen("/nonexistent/path/gqlgen.yml")
	if err == nil {
		t.Fatal("InvokeGqlgen with nonexistent config should return error")
	}
}

// TestInvokeGqlgen_Callable verifies that InvokeGqlgen can be called without panicking.
// The stub returns nil — once implemented, this will verify actual gqlgen invocation.
func TestInvokeGqlgen_Callable(t *testing.T) {
	// With stub, this returns nil for any path (including nonexistent ones).
	// This test just verifies the function is callable.
	_ = InvokeGqlgen("/some/path/gqlgen.yml")
}

// === CG-7: Wire actual gqlgen api.Generate tests ===

// writeValidGqlgenSetup writes a minimal valid augmented schema and gqlgen.yml
// to a temp directory and returns the config file path and output directory.
func writeValidGqlgenSetup(t *testing.T) (configPath, outputDir string) {
	t.Helper()
	dir := t.TempDir()
	outputDir = dir

	// Write a minimal valid augmented GraphQL schema
	schemaSDL := `type Movie {
  id: ID!
  title: String!
  released: Int
}

input MovieWhere {
  id: ID
  title: String
  released: Int
}

input MovieCreateInput {
  title: String!
  released: Int
}

input MovieUpdateInput {
  title: String
  released: Int
}

type CreateMoviesMutationResponse {
  movies: [Movie!]!
}

type UpdateMoviesMutationResponse {
  movies: [Movie!]!
}

type DeleteInfo {
  nodesDeleted: Int!
  relationshipsDeleted: Int!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

type MoviesConnection {
  edges: [MovieEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type MovieEdge {
  node: Movie!
  cursor: String!
}

type Query {
  movies(where: MovieWhere): [Movie!]!
  moviesConnection(first: Int, after: String, where: MovieWhere): MoviesConnection!
}

type Mutation {
  createMovies(input: [MovieCreateInput!]!): CreateMoviesMutationResponse!
  updateMovies(where: MovieWhere, update: MovieUpdateInput): UpdateMoviesMutationResponse!
  deleteMovies(where: MovieWhere): DeleteInfo!
}
`
	schemaPath := filepath.Join(dir, "schema.graphql")
	if err := os.WriteFile(schemaPath, []byte(schemaSDL), 0644); err != nil {
		t.Fatalf("failed to write schema: %v", err)
	}

	// Generate gqlgen.yml pointing to the schema
	cfg := GqlgenConfig{
		SchemaPath:  schemaPath,
		OutputDir:   dir,
		PackageName: "generated",
	}
	configContent, err := GenerateGqlgenConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateGqlgenConfig failed: %v", err)
	}

	configPath = filepath.Join(dir, "gqlgen.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write gqlgen.yml: %v", err)
	}

	return configPath, outputDir
}

// TestInvokeGqlgen_ProducesModelsFile verifies that InvokeGqlgen with a valid
// augmented schema + gqlgen.yml produces a Go models file in the output directory.
// Expected: models_gen.go exists in output directory after invocation.
func TestInvokeGqlgen_ProducesModelsFile(t *testing.T) {
	configPath, outputDir := writeValidGqlgenSetup(t)

	err := InvokeGqlgen(configPath)
	if err != nil {
		t.Fatalf("InvokeGqlgen returned error: %v", err)
	}

	modelsPath := filepath.Join(outputDir, "models_gen.go")
	if _, statErr := os.Stat(modelsPath); os.IsNotExist(statErr) {
		t.Fatal("InvokeGqlgen did not produce models_gen.go — gqlgen api.Generate not wired")
	}
}

// TestInvokeGqlgen_ProducesExecFile verifies that InvokeGqlgen with a valid
// config produces a gqlgen executor file in the output directory.
// Expected: exec_gen.go exists in output directory after invocation.
func TestInvokeGqlgen_ProducesExecFile(t *testing.T) {
	configPath, outputDir := writeValidGqlgenSetup(t)

	err := InvokeGqlgen(configPath)
	if err != nil {
		t.Fatalf("InvokeGqlgen returned error: %v", err)
	}

	execPath := filepath.Join(outputDir, "exec_gen.go")
	if _, statErr := os.Stat(execPath); os.IsNotExist(statErr) {
		t.Fatal("InvokeGqlgen did not produce exec_gen.go — gqlgen api.Generate not wired")
	}
}

// TestInvokeGqlgen_InvalidConfig_ReturnsError verifies that InvokeGqlgen
// returns an error when given a config with an invalid schema path.
// Expected: non-nil error (gqlgen fails to load schema).
func TestInvokeGqlgen_InvalidConfig_ReturnsError(t *testing.T) {
	dir := t.TempDir()

	// Write gqlgen.yml pointing to a nonexistent schema
	cfg := GqlgenConfig{
		SchemaPath:  filepath.Join(dir, "nonexistent.graphql"),
		OutputDir:   dir,
		PackageName: "generated",
	}
	configContent, err := GenerateGqlgenConfig(cfg)
	if err != nil {
		t.Fatalf("GenerateGqlgenConfig failed: %v", err)
	}

	configPath := filepath.Join(dir, "gqlgen.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write gqlgen.yml: %v", err)
	}

	err = InvokeGqlgen(configPath)
	if err == nil {
		t.Fatal("InvokeGqlgen with invalid schema path should return error")
	}
}

// === L-chdir: InvokeGqlgen mutex serialization tests ===

// TestGqlgenMu_CanBeLocked verifies that the package-level gqlgenMu variable
// exists and behaves as a sync.Mutex (can be locked and unlocked).
// Expected: Lock/Unlock succeeds without panic. Guardrail test.
func TestGqlgenMu_CanBeLocked(t *testing.T) {
	gqlgenMu.Lock()
	gqlgenMu.Unlock()
}

// TestInvokeGqlgen_BlocksWhenGqlgenMuHeld verifies that InvokeGqlgen acquires
// gqlgenMu before the os.Chdir block. When the mutex is held externally,
// InvokeGqlgen should block until it is released.
// Currently FAILS because InvokeGqlgen does not lock gqlgenMu — it completes
// regardless of whether the mutex is held.
func TestInvokeGqlgen_BlocksWhenGqlgenMuHeld(t *testing.T) {
	configPath, outputDir := writeValidGqlgenSetup(t)

	// Pre-create go.mod and stub_gen.go so InvokeGqlgen skips the slow
	// go-mod-tidy step and reaches the chdir block quickly.
	os.WriteFile(filepath.Join(outputDir, "go.mod"), []byte("module generated\ngo 1.25\n"), 0644)
	os.WriteFile(filepath.Join(outputDir, "stub_gen.go"), []byte("package generated\n"), 0644)

	gqlgenMu.Lock()

	done := make(chan error, 1)
	go func() {
		done <- InvokeGqlgen(configPath)
	}()

	// Without the mutex wired, InvokeGqlgen completes quickly (<2s):
	// stat → parse → skip go.mod → chdir → LoadConfig error → return.
	// With the mutex wired, InvokeGqlgen blocks on gqlgenMu.Lock().
	select {
	case err := <-done:
		gqlgenMu.Unlock()
		t.Fatalf("InvokeGqlgen completed (err=%v) while gqlgenMu held — mutex not wired into InvokeGqlgen", err)
	case <-time.After(2 * time.Second):
		// Expected: InvokeGqlgen is blocked on the mutex
		gqlgenMu.Unlock()
		<-done // let it finish and clean up
	}
}

// TestInvokeGqlgen_ReleasesGqlgenMuAfterReturn verifies that gqlgenMu is
// released after InvokeGqlgen returns (no leaked lock).
// Expected: TryLock succeeds after InvokeGqlgen returns. Guardrail test.
func TestInvokeGqlgen_ReleasesGqlgenMuAfterReturn(t *testing.T) {
	_ = InvokeGqlgen("/nonexistent/path/gqlgen.yml")

	if !gqlgenMu.TryLock() {
		t.Fatal("gqlgenMu still held after InvokeGqlgen returned — lock not released")
	}
	gqlgenMu.Unlock()
}
