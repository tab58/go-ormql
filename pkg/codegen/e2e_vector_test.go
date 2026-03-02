package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- E2E-5: E2E compilation test with @vector schema ---
// The V2 pipeline with @vector generates code that compiles without errors.
// Output: schema.graphql, models_gen.go, graphmodel_gen.go, client_gen.go, indexes_gen.go.

// v2VectorSchema returns a GraphQL schema exercising ALL V2 features PLUS @vector:
// - Multiple node types (Movie, Actor, Genre)
// - @relationship with properties (ACTED_IN + ActedInProperties)
// - @relationship without properties (IN_GENRE)
// - @cypher field with arguments (recommended)
// - @cypher field without arguments (averageRating)
// - @vector directive on [Float!]! field (embedding)
// - Multiple scalar types (String, Int, Float, Boolean, ID)
// - Enum type (MovieGenre)
func v2VectorSchema() string {
	return `type Movie @node {
	id: ID!
	title: String!
	released: Int
	rating: Float
	active: Boolean
	embedding: [Float!]! @vector(indexName: "movie_embeddings", dimensions: 1536, similarity: "cosine")
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

// writeV2VectorSchema writes the full-featured V2 + @vector schema to a temp
// directory and returns the schema file path and output directory.
func writeV2VectorSchema(t *testing.T) (schemaPath, outputDir string) {
	t.Helper()
	baseDir := t.TempDir()
	schemaPath = filepath.Join(baseDir, "schema.graphql")
	if err := os.WriteFile(schemaPath, []byte(v2VectorSchema()), 0644); err != nil {
		t.Fatalf("failed to write schema file: %v", err)
	}
	outputDir = filepath.Join(baseDir, "generated")
	return schemaPath, outputDir
}

// Test: V2 Generate() with @vector schema completes without error.
// Expected: Generate returns nil error.
func TestE2EVector_GenerateSucceeds(t *testing.T) {
	schemaPath, outputDir := writeV2VectorSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	err := Generate(cfg)
	if err != nil {
		t.Fatalf("V2 Generate with @vector schema failed: %v", err)
	}
}

// Test: V2 pipeline with @vector produces indexes_gen.go in output directory.
// Expected: indexes_gen.go exists after Generate().
func TestE2EVector_IndexesGenExists(t *testing.T) {
	schemaPath, outputDir := writeV2VectorSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	Generate(cfg) // may error if @vector parsing not yet implemented

	indexesPath := filepath.Join(outputDir, "indexes_gen.go")
	if _, err := os.Stat(indexesPath); os.IsNotExist(err) {
		t.Fatal("V2 pipeline with @vector should produce indexes_gen.go")
	}
}

// Test: Generated indexes_gen.go contains CreateIndexes function with correct DDL.
// Expected: indexes_gen.go content includes "func CreateIndexes" and "movie_embeddings".
func TestE2EVector_IndexesGenContainsCreateIndexes(t *testing.T) {
	schemaPath, outputDir := writeV2VectorSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	Generate(cfg) // may error if @vector parsing not yet implemented

	content, err := os.ReadFile(filepath.Join(outputDir, "indexes_gen.go"))
	if err != nil {
		t.Fatalf("indexes_gen.go not found or unreadable: %v", err)
	}
	src := string(content)
	if !strings.Contains(src, "func CreateIndexes") {
		t.Error("indexes_gen.go missing 'func CreateIndexes'")
	}
	if !strings.Contains(src, "movie_embeddings") {
		t.Error("indexes_gen.go missing index name 'movie_embeddings'")
	}
}

// Test: V2 generated models_gen.go contains MovieSimilarResult struct.
// Expected: models_gen.go includes "type MovieSimilarResult struct".
func TestE2EVector_ModelsContainSimilarResult(t *testing.T) {
	schemaPath, outputDir := writeV2VectorSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("V2 Generate with @vector schema failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "models_gen.go"))
	if err != nil {
		t.Fatalf("failed to read models_gen.go: %v", err)
	}
	src := string(content)
	if !strings.Contains(src, "type MovieSimilarResult struct") {
		t.Error("models_gen.go missing 'type MovieSimilarResult struct'")
	}
}

// Test: V2 augmented schema contains moviesSimilar query.
// Expected: schema.graphql includes "moviesSimilar" and "MovieSimilarResult".
func TestE2EVector_AugmentedSchemaContainsSimilarQuery(t *testing.T) {
	schemaPath, outputDir := writeV2VectorSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("V2 Generate with @vector schema failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outputDir, "schema.graphql"))
	if err != nil {
		t.Fatalf("failed to read schema.graphql: %v", err)
	}
	sdl := string(content)
	if !strings.Contains(sdl, "moviesSimilar") {
		t.Error("augmented schema missing 'moviesSimilar' query field")
	}
	if !strings.Contains(sdl, "MovieSimilarResult") {
		t.Error("augmented schema missing 'MovieSimilarResult' type")
	}
}

// Test: V2 generated code with @vector compiles with go build.
// Expected: `go build ./...` exits with code 0.
func TestE2EVector_GoBuildSucceeds(t *testing.T) {
	schemaPath, outputDir := writeV2VectorSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("V2 Generate with @vector schema failed: %v", err)
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

	// Run go build to verify the V2 + @vector generated code compiles.
	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = outputDir
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed on V2 + @vector generated output:\n%s\n%v", string(output), err)
	}
}

// Test: V2 pipeline with @vector produces exactly 5 output files.
// Expected: schema.graphql, models_gen.go, graphmodel_gen.go, client_gen.go, indexes_gen.go.
func TestE2EVector_ExactlyFiveFiles(t *testing.T) {
	schemaPath, outputDir := writeV2VectorSchema(t)
	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}
	if err := Generate(cfg); err != nil {
		t.Fatalf("V2 Generate with @vector schema failed: %v", err)
	}

	expected := map[string]bool{
		"schema.graphql":    false,
		"models_gen.go":     false,
		"graphmodel_gen.go": false,
		"client_gen.go":     false,
		"indexes_gen.go":    false,
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
	if len(files) != 5 {
		t.Fatalf("V2 + @vector pipeline should produce exactly 5 files, got %d: %v", len(files), files)
	}
	for name, found := range expected {
		if !found {
			t.Errorf("expected file %s not found in output", name)
		}
	}
}
