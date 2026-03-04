package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- E2E-7: E2E compilation test with merge, connect, and relationship WHERE ---
// Exercises all 3 features: merge mutations, connect mutations (with and without
// edge properties), and relationship-based WHERE filters (to-one, to-many).

// mergeConnectSchema returns a GraphQL schema exercising:
// - Multiple node types (Movie, Actor, Director, Studio)
// - @relationship to-many with properties (ACTED_IN + ActedInProperties)
// - @relationship to-many without properties (DIRECTED)
// - @relationship to-one without properties (BELONGS_TO)
// - All 3 features rely on IsList for relationship cardinality
func mergeConnectSchema() string {
	return `type Movie @node {
	id: ID!
	title: String!
	released: Int
	actors: [Actor!]! @relationship(type: "ACTED_IN", direction: IN, properties: "ActedInProperties")
	directors: [Director!]! @relationship(type: "DIRECTED", direction: IN)
	studio: Studio! @relationship(type: "BELONGS_TO", direction: OUT)
}

type Actor @node {
	id: ID!
	name: String!
	movies: [Movie!]! @relationship(type: "ACTED_IN", direction: OUT, properties: "ActedInProperties")
}

type Director @node {
	id: ID!
	name: String!
	movies: [Movie!]! @relationship(type: "DIRECTED", direction: OUT)
}

type Studio @node {
	id: ID!
	name: String!
	movies: [Movie!]! @relationship(type: "BELONGS_TO", direction: IN)
}

type ActedInProperties @relationshipProperties {
	role: String!
	screenTime: Int
}
`
}

// writeMergeConnectSchema writes the merge/connect schema to a temp directory
// and returns the schema file path and output directory.
func writeMergeConnectSchema(t *testing.T) (schemaPath, outputDir string) {
	t.Helper()
	baseDir := t.TempDir()
	schemaPath = filepath.Join(baseDir, "schema.graphql")
	if err := os.WriteFile(schemaPath, []byte(mergeConnectSchema()), 0644); err != nil {
		t.Fatalf("failed to write schema file: %v", err)
	}
	outputDir = filepath.Join(baseDir, "generated")
	return schemaPath, outputDir
}

// Test: E2E Generate() succeeds for merge/connect/relWhere schema.
// Expected: Generate returns nil error.
func TestE2EMergeConnect_GenerateSucceeds(t *testing.T) {
	schemaPath, outputDir := writeMergeConnectSchema(t)
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

// Test: Augmented schema contains merge mutation types for each node.
// Expected: mergeMovies, mergeActors fields in Mutation type; MovieMatchInput,
// MovieMergeInput, MergeMoviesMutationResponse types present.
func TestE2EMergeConnect_AugmentedSchemaContainsMergeTypes(t *testing.T) {
	schemaPath, outputDir := writeMergeConnectSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Read the augmented schema from the output directory
	content, err := os.ReadFile(filepath.Join(outputDir, "schema.graphql"))
	if err != nil {
		t.Fatalf("failed to read schema.graphql: %v", err)
	}
	sdl := string(content)

	// Check merge mutation fields
	if !strings.Contains(sdl, "mergeMovies") {
		t.Error("augmented schema missing 'mergeMovies' mutation field")
	}
	// Check merge input types
	if !strings.Contains(sdl, "MovieMatchInput") {
		t.Error("augmented schema missing 'MovieMatchInput' type")
	}
	if !strings.Contains(sdl, "MovieMergeInput") {
		t.Error("augmented schema missing 'MovieMergeInput' type")
	}
	// Check merge response type
	if !strings.Contains(sdl, "MergeMoviesMutationResponse") {
		t.Error("augmented schema missing 'MergeMoviesMutationResponse' type")
	}
}

// Test: Augmented schema contains connect mutation types.
// Expected: connectMovieActors (with properties), connectMovieDirectors (without
// properties), ConnectInfo type present.
func TestE2EMergeConnect_AugmentedSchemaContainsConnectTypes(t *testing.T) {
	schemaPath, outputDir := writeMergeConnectSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "schema.graphql"))
	if err != nil {
		t.Fatalf("failed to read schema.graphql: %v", err)
	}
	sdl := string(content)

	// Connect with properties (ACTED_IN has ActedInProperties)
	if !strings.Contains(sdl, "connectMovieActors") {
		t.Error("augmented schema missing 'connectMovieActors' mutation field")
	}
	// Connect without properties (DIRECTED has no properties)
	if !strings.Contains(sdl, "connectMovieDirectors") {
		t.Error("augmented schema missing 'connectMovieDirectors' mutation field")
	}
	// ConnectInfo shared response type
	if !strings.Contains(sdl, "ConnectInfo") {
		t.Error("augmented schema missing 'ConnectInfo' type")
	}
}

// Test: Augmented schema contains relationship WHERE filter fields.
// Expected: MovieWhere has actors_some (to-many) and studio (to-one) filter fields.
func TestE2EMergeConnect_AugmentedSchemaContainsRelWhereFields(t *testing.T) {
	schemaPath, outputDir := writeMergeConnectSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "schema.graphql"))
	if err != nil {
		t.Fatalf("failed to read schema.graphql: %v", err)
	}
	sdl := string(content)

	// To-many uses _some suffix
	if !strings.Contains(sdl, "actors_some") {
		t.Error("augmented schema missing 'actors_some' relationship WHERE field in MovieWhere")
	}
	// To-one uses direct field name
	if !strings.Contains(sdl, "studio: StudioWhere") {
		t.Error("augmented schema missing 'studio: StudioWhere' relationship WHERE field")
	}
}

// Test: Generated models contain merge/connect Go structs.
// Expected: models_gen.go contains MovieMatchInput, MovieMergeInput,
// MergeMoviesMutationResponse, ConnectMovieActorsInput, ConnectInfo.
func TestE2EMergeConnect_ModelsContainMergeConnectStructs(t *testing.T) {
	schemaPath, outputDir := writeMergeConnectSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "models_gen.go"))
	if err != nil {
		t.Fatalf("failed to read models_gen.go: %v", err)
	}
	src := string(content)

	expectedStructs := []string{
		"MovieMatchInput",
		"MovieMergeInput",
		"MergeMoviesMutationResponse",
		"ConnectMovieActorsInput",
		"ConnectInfo",
	}
	for _, name := range expectedStructs {
		if !strings.Contains(src, "type "+name+" struct") {
			t.Errorf("models_gen.go missing 'type %s struct'", name)
		}
	}
}

// Test: Generated code compiles with go build.
// Expected: `go build ./...` exits with code 0.
func TestE2EMergeConnect_GoBuildSucceeds(t *testing.T) {
	schemaPath, outputDir := writeMergeConnectSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	root := v2ProjectRoot(t)
	goModContent := "module generated\n\ngo 1.25\n\nrequire (\n" +
		"\tgithub.com/tab58/go-ormql v0.0.0\n" +
		")\n\n" +
		"replace github.com/tab58/go-ormql => " + root + "\n"
	if err := os.WriteFile(filepath.Join(outputDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = outputDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed:\n%s\n%v", string(output), err)
	}

	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = outputDir
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed on merge/connect generated output:\n%s\n%v", string(output), err)
	}
}
