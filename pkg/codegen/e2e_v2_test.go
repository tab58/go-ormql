package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// --- E2E-4: V2 pipeline E2E compilation test ---
// The V2 pipeline generates code that compiles without gqlgen.
// Output: schema.graphql, models_gen.go, graphmodel_gen.go, client_gen.go.

// v2FullSchema returns a GraphQL schema exercising ALL V2 features:
// - Multiple node types (Movie, Actor, Genre)
// - @relationship with properties (ACTED_IN + ActedInProperties)
// - @relationship without properties (IN_GENRE)
// - @cypher field with arguments (recommended)
// - @cypher field without arguments (averageRating)
// - Multiple scalar types (String, Int, Float, Boolean, ID)
// - Enum type (Genre enum)
func v2FullSchema() string {
	return `type Movie @node {
	id: ID!
	title: String!
	released: Int
	rating: Float
	active: Boolean
	actors: [Actor!]! @relationship(type: "ACTED_IN", direction: IN, properties: "ActedInProperties")
	genres: [Genre!]! @relationship(type: "IN_GENRE", direction: OUT)
	averageRating: Float @cypher(statement: "MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.rating)")
	recommended(limit: Int!): [Movie!]! @cypher(statement: "MATCH (this)-[:ACTED_IN]->()<-[:ACTED_IN]-(rec) RETURN rec LIMIT $limit")
}

type Actor @node {
	id: ID!
	name: String!
	born: Int
	movies: [Movie!]! @relationship(type: "ACTED_IN", direction: OUT, properties: "ActedInProperties")
}

type Genre @node {
	id: ID!
	name: String!
}

type ActedInProperties @relationshipProperties {
	role: String!
	screenTime: Int
}

enum MovieGenre {
	ACTION
	COMEDY
	DRAMA
	HORROR
	SCIFI
}
`
}

// v2ProjectRoot returns the project root by walking up from the test file
// directory until go.mod is found.
func v2ProjectRoot(t *testing.T) string {
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

// writeV2Schema writes the full-featured V2 schema to a temp directory
// and returns the schema file path and output directory.
func writeV2Schema(t *testing.T) (schemaPath, outputDir string) {
	t.Helper()
	baseDir := t.TempDir()
	schemaPath = filepath.Join(baseDir, "schema.graphql")
	if err := os.WriteFile(schemaPath, []byte(v2FullSchema()), 0644); err != nil {
		t.Fatalf("failed to write schema file: %v", err)
	}
	outputDir = filepath.Join(baseDir, "generated")
	return schemaPath, outputDir
}

// Test: V2 Generate() completes without error on full-featured schema.
// Expected: Generate returns nil error.
func TestE2EV2_GenerateSucceeds(t *testing.T) {
	schemaPath, outputDir := writeV2Schema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	err := Generate(cfg)
	if err != nil {
		t.Fatalf("V2 Generate failed: %v", err)
	}
}

// Test: V2 pipeline produces exactly 4 output files.
// Expected: schema.graphql, models_gen.go, graphmodel_gen.go, client_gen.go.
func TestE2EV2_ExactlyFourFiles(t *testing.T) {
	schemaPath, outputDir := writeV2Schema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("V2 Generate failed: %v", err)
	}

	expected := map[string]bool{
		"schema.graphql":    false,
		"models_gen.go":     false,
		"graphmodel_gen.go": false,
		"client_gen.go":     false,
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("failed to read output dir: %v", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
			if _, ok := expected[e.Name()]; ok {
				expected[e.Name()] = true
			}
		}
	}
	if len(files) != 4 {
		t.Fatalf("V2 pipeline should produce exactly 4 files, got %d: %v", len(files), files)
	}
	for name, found := range expected {
		if !found {
			t.Errorf("expected file %s not found in output", name)
		}
	}
}

// Test: V2 pipeline does NOT produce any gqlgen artifacts.
// Expected: no gqlgen.yml, exec_gen.go, resolvers_gen.go, mappers_gen.go.
func TestE2EV2_NoGqlgenArtifacts(t *testing.T) {
	schemaPath, outputDir := writeV2Schema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	Generate(cfg)

	gqlgenArtifacts := []string{
		"gqlgen.yml",
		"exec_gen.go",
		"resolvers_gen.go",
		"mappers_gen.go",
		"resolver.go",
	}
	for _, name := range gqlgenArtifacts {
		t.Run(name, func(t *testing.T) {
			if _, err := os.Stat(filepath.Join(outputDir, name)); err == nil {
				t.Errorf("V2 pipeline should NOT produce %s (gqlgen removed)", name)
			}
		})
	}
}

// Test: V2 generated code compiles with go build (no gqlgen dependency).
// Expected: `go build ./...` exits with code 0. go.mod does NOT include gqlgen.
func TestE2EV2_GoBuildSucceeds(t *testing.T) {
	schemaPath, outputDir := writeV2Schema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("V2 Generate failed: %v", err)
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

	// Run go mod tidy to resolve transitive dependencies.
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = outputDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed:\n%s\n%v", string(output), err)
	}

	// Run go build to verify the V2 generated code compiles.
	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = outputDir
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed on V2 generated output:\n%s\n%v", string(output), err)
	}
}

// Test: V2 generated models_gen.go contains node structs for all schema types.
// Expected: Movie, Actor, Genre struct types present.
func TestE2EV2_ModelsContainNodeStructs(t *testing.T) {
	schemaPath, outputDir := writeV2Schema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("V2 Generate failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "models_gen.go"))
	if err != nil {
		t.Fatalf("failed to read models_gen.go: %v", err)
	}
	src := string(content)
	for _, typeName := range []string{"Movie", "Actor", "Genre"} {
		if !contains(src, "type "+typeName+" struct") {
			t.Errorf("models_gen.go missing 'type %s struct'", typeName)
		}
	}
}

// Test: V2 generated graphmodel_gen.go contains GraphModel and AugmentedSchemaSDL.
// Expected: var GraphModel and var AugmentedSchemaSDL declarations present.
func TestE2EV2_RegistryContainsVars(t *testing.T) {
	schemaPath, outputDir := writeV2Schema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("V2 Generate failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "graphmodel_gen.go"))
	if err != nil {
		t.Fatalf("failed to read graphmodel_gen.go: %v", err)
	}
	src := string(content)
	if !contains(src, "var GraphModel") {
		t.Error("graphmodel_gen.go missing 'var GraphModel' declaration")
	}
	if !contains(src, "var AugmentedSchemaSDL") {
		t.Error("graphmodel_gen.go missing 'var AugmentedSchemaSDL' declaration")
	}
}

// contains is a helper to check substring presence.
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && containsSubstr(s, substr)
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
