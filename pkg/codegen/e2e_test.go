package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// === H1-3: E2E compilation test ===

// movieActorSchema returns a GraphQL schema string with Movie and Actor nodes,
// an ACTED_IN relationship with ActedInProperties containing a "role" field.
// This is the reference schema for the E2E compilation test.
func movieActorSchema() string {
	return `type Movie @node {
	id: ID!
	title: String!
	released: Int
	actors: [Actor!]! @relationship(type: "ACTED_IN", direction: IN, properties: "ActedInProperties")
}

type Actor @node {
	id: ID!
	name: String!
}

type ActedInProperties @relationshipProperties {
	role: String!
}
`
}

// writeE2ESchema writes the Movie/Actor/ActedInProperties schema to a temp directory
// and returns the schema file path and the output directory for generated code.
func writeE2ESchema(t *testing.T) (schemaPath, outputDir string) {
	t.Helper()
	baseDir := t.TempDir()
	schemaPath = filepath.Join(baseDir, "schema.graphql")
	if err := os.WriteFile(schemaPath, []byte(movieActorSchema()), 0644); err != nil {
		t.Fatalf("failed to write schema file: %v", err)
	}
	outputDir = filepath.Join(baseDir, "generated")
	return schemaPath, outputDir
}

// TestE2ECompilation_GenerateSucceeds verifies that the full Generate() pipeline
// completes without error on the Movie/Actor/ActedInProperties schema.
// Expected: Generate returns nil error.
func TestE2ECompilation_GenerateSucceeds(t *testing.T) {
	schemaPath, outputDir := writeE2ESchema(t)

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

// TestE2ECompilation_AllExpectedFilesExist verifies that Generate() produces all
// expected output files: schema.graphql, gqlgen.yml, exec_gen.go, models_gen.go,
// resolvers_gen.go, mappers_gen.go, client_gen.go.
// Expected: all 7 files exist in the output directory.
func TestE2ECompilation_AllExpectedFilesExist(t *testing.T) {
	schemaPath, outputDir := writeE2ESchema(t)

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expectedFiles := []string{
		"schema.graphql",
		"gqlgen.yml",
		"exec_gen.go",
		"models_gen.go",
		"resolvers_gen.go",
		"mappers_gen.go",
		"client_gen.go",
	}
	for _, name := range expectedFiles {
		path := filepath.Join(outputDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s not found in output directory", name)
		}
	}
}

// TestE2ECompilation_NoScaffoldFilesRemain verifies that after Generate(),
// no gqlgen resolver scaffold files (resolver.go, *.resolvers.go) remain
// in the output directory.
// Expected: resolver.go and *.resolvers.go are absent.
func TestE2ECompilation_NoScaffoldFilesRemain(t *testing.T) {
	schemaPath, outputDir := writeE2ESchema(t)

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check for resolver.go
	if _, err := os.Stat(filepath.Join(outputDir, "resolver.go")); err == nil {
		t.Error("resolver.go should have been removed by deleteResolverScaffold")
	}

	// Check for *.resolvers.go
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("failed to read output directory: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".resolvers.go") {
			t.Errorf("scaffold file %s should have been removed by deleteResolverScaffold", e.Name())
		}
	}
}

// TestE2ECompilation_GqlgenConfigHasResolverSection verifies that the
// generated gqlgen.yml contains a resolver section. Without it, gqlgen
// silently skips scaffold generation — the pipeline appears to work but
// the resolver config isn't controlling gqlgen's output as required by spec.
// Expected: parsed gqlgen.yml has a non-empty resolver.layout field.
func TestE2ECompilation_GqlgenConfigHasResolverSection(t *testing.T) {
	schemaPath, outputDir := writeE2ESchema(t)

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	configData, err := os.ReadFile(filepath.Join(outputDir, "gqlgen.yml"))
	if err != nil {
		t.Fatalf("failed to read gqlgen.yml: %v", err)
	}

	var parsed struct {
		Resolver struct {
			Layout  string `yaml:"layout"`
			Package string `yaml:"package"`
		} `yaml:"resolver"`
	}
	if yamlErr := yaml.Unmarshal(configData, &parsed); yamlErr != nil {
		t.Fatalf("failed to parse gqlgen.yml: %v", yamlErr)
	}
	if parsed.Resolver.Layout == "" {
		t.Error("gqlgen.yml missing resolver.layout — gqlgen resolver scaffold is not being controlled by config")
	}
	if parsed.Resolver.Package == "" {
		t.Error("gqlgen.yml missing resolver.package — gqlgen resolver scaffold will use wrong package")
	}
}

// TestE2ECompilation_PipelineDeletesScaffoldAfterGqlgen verifies that the
// pipeline actively removes gqlgen scaffold files, not just passively avoids
// generating them. This test manually creates scaffold files in the output
// directory before calling Generate(), simulating what gqlgen produces when
// the resolver config section is present.
// Expected: resolver.go and schema.resolvers.go are absent after Generate().
func TestE2ECompilation_PipelineDeletesScaffoldAfterGqlgen(t *testing.T) {
	schemaPath, outputDir := writeE2ESchema(t)

	// Pre-create the output dir and plant scaffold files to simulate
	// what gqlgen would produce with a resolver config section.
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}
	scaffoldFiles := []string{"resolver.go", "schema.resolvers.go"}
	for _, name := range scaffoldFiles {
		if err := os.WriteFile(filepath.Join(outputDir, name), []byte("package generated\n"), 0644); err != nil {
			t.Fatalf("failed to plant scaffold file %s: %v", name, err)
		}
	}

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify scaffold files were deleted by the pipeline
	for _, name := range scaffoldFiles {
		path := filepath.Join(outputDir, name)
		if _, err := os.Stat(path); err == nil {
			t.Errorf("scaffold file %s was not removed by pipeline — deleteResolverScaffold not wired", name)
		}
	}
}

// TestE2ECompilation_GoBuildSucceeds verifies that the generated output from
// the full pipeline (Movie/Actor/ActedInProperties schema) compiles successfully.
// This is the ultimate acceptance gate for H1: the generated Go code must be
// valid and compilable.
// Expected: `go build ./...` on the output directory exits with code 0.
func TestE2ECompilation_GoBuildSucceeds(t *testing.T) {
	schemaPath, outputDir := writeE2ESchema(t)

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// The generated package needs a go.mod to compile. Generate one that
	// references the main module so import paths resolve correctly.
	goModContent := "module generated\n\ngo 1.25\n\nrequire (\n" +
		"\tgithub.com/99designs/gqlgen v0.17.87\n" +
		"\tgithub.com/tab58/gql-orm v0.0.0\n" +
		")\n\n" +
		"replace github.com/tab58/gql-orm => " + projectRoot(t) + "\n"
	goModPath := filepath.Join(outputDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go mod tidy to resolve transitive dependencies
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = outputDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %s\n%v", string(output), err)
	}

	// Run go build to verify the generated code compiles
	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = outputDir
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed on generated output:\n%s\n%v", string(output), err)
	}
}
