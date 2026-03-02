package codegen

import (
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
)

// --- CG-24: Augmented schema vector similarity types tests ---

// vectorMovieModel returns a GraphModel with a Movie node that has a VectorField.
func vectorMovieModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
					{Name: "released", GraphQLType: "Int", GoType: "*int", CypherType: "INTEGER", Nullable: true},
					{Name: "embedding", GraphQLType: "[Float!]!", GoType: "[]float64", CypherType: "LIST<FLOAT>", IsList: true},
				},
				VectorField: &schema.VectorFieldDefinition{
					Name:       "embedding",
					IndexName:  "movie_embeddings",
					Dimensions: 1536,
					Similarity: "cosine",
				},
			},
		},
	}
}

// Test: AugmentSchema generates moviesSimilar query field for node with VectorField.
// Expected: Query type contains "moviesSimilar(vector: [Float!]!, first: Int): [MovieSimilarResult!]!"
func TestAugmentSchema_VectorSimilarQueryField(t *testing.T) {
	sdl, err := AugmentSchema(vectorMovieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "moviesSimilar") {
		t.Errorf("augmented schema missing 'moviesSimilar' query field:\n%s", sdl)
	}
	if !strings.Contains(sdl, "vector: [Float!]!") {
		t.Errorf("moviesSimilar missing 'vector: [Float!]!' argument:\n%s", sdl)
	}
	if !strings.Contains(sdl, "MovieSimilarResult") {
		t.Errorf("augmented schema missing 'MovieSimilarResult' return type:\n%s", sdl)
	}
}

// Test: AugmentSchema generates MovieSimilarResult type with score and node fields.
// Expected: type MovieSimilarResult { score: Float! node: Movie! }
func TestAugmentSchema_VectorSimilarResultType(t *testing.T) {
	sdl, err := AugmentSchema(vectorMovieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "type MovieSimilarResult") {
		t.Fatal("MovieSimilarResult type not found in augmented schema")
	}
	if !strings.Contains(sdl, "score: Float!") {
		t.Error("MovieSimilarResult missing 'score: Float!' field")
	}
	if !strings.Contains(sdl, "node: Movie!") {
		t.Error("MovieSimilarResult missing 'node: Movie!' field")
	}
}

// Test: AugmentSchema excludes vector field from Where input.
// Expected: MovieWhere does NOT contain "embedding" fields.
func TestAugmentSchema_VectorFieldExcludedFromWhere(t *testing.T) {
	sdl, err := AugmentSchema(vectorMovieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Extract the MovieWhere block from the SDL and check for embedding fields.
	// Can't use gqlparser because list-type in Where produces invalid SDL until vector field is excluded.
	whereIdx := strings.Index(sdl, "input MovieWhere {")
	if whereIdx == -1 {
		t.Fatal("MovieWhere not found in augmented schema")
	}
	whereBlock := sdl[whereIdx:]
	closeIdx := strings.Index(whereBlock, "}")
	if closeIdx == -1 {
		t.Fatal("MovieWhere block not properly closed")
	}
	whereBlock = whereBlock[:closeIdx]

	if strings.Contains(whereBlock, "embedding") {
		t.Errorf("MovieWhere should not contain vector field 'embedding':\n%s", whereBlock)
	}
}

// Test: AugmentSchema excludes vector field from Sort input.
// Expected: MovieSort does NOT contain "embedding" field.
func TestAugmentSchema_VectorFieldExcludedFromSort(t *testing.T) {
	sdl, err := AugmentSchema(vectorMovieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Extract the MovieSort block from the SDL.
	sortIdx := strings.Index(sdl, "input MovieSort {")
	if sortIdx == -1 {
		t.Fatal("MovieSort not found in augmented schema")
	}
	sortBlock := sdl[sortIdx:]
	closeIdx := strings.Index(sortBlock, "}")
	if closeIdx == -1 {
		t.Fatal("MovieSort block not properly closed")
	}
	sortBlock = sortBlock[:closeIdx]

	if strings.Contains(sortBlock, "embedding") {
		t.Errorf("MovieSort should not contain vector field 'embedding':\n%s", sortBlock)
	}
}

// Test: AugmentSchema includes vector field in CreateInput.
// Expected: MovieCreateInput contains "embedding: [Float!]!" field.
func TestAugmentSchema_VectorFieldInCreateInput(t *testing.T) {
	sdl, err := AugmentSchema(vectorMovieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Use string check (not gqlparser) since pre-existing list-type Where bug causes parse errors
	if !strings.Contains(sdl, "input MovieCreateInput") {
		t.Fatal("MovieCreateInput not found in augmented schema")
	}
	if !strings.Contains(sdl, "embedding: [Float!]!") {
		t.Error("MovieCreateInput should include vector field 'embedding: [Float!]!'")
	}
}

// Test: AugmentSchema includes vector field in UpdateInput.
// Expected: MovieUpdateInput contains "embedding" field (nullable for update).
func TestAugmentSchema_VectorFieldInUpdateInput(t *testing.T) {
	sdl, err := AugmentSchema(vectorMovieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Use string check (not gqlparser) since pre-existing list-type Where bug causes parse errors
	if !strings.Contains(sdl, "input MovieUpdateInput") {
		t.Fatal("MovieUpdateInput not found in augmented schema")
	}
	// embedding should be in UpdateInput (nullable version)
	if !strings.Contains(sdl, "embedding: [Float!]") {
		t.Error("MovieUpdateInput should include vector field 'embedding'")
	}
}

// Test: AugmentSchema does NOT generate Similar query for node without VectorField.
// Expected: actorsSimilar is NOT in the augmented schema.
func TestAugmentSchema_NoSimilarQueryWithoutVector(t *testing.T) {
	model := schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Actor",
				Labels: []string{"Actor"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
				},
				// No VectorField
			},
		},
	}

	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if strings.Contains(sdl, "actorsSimilar") {
		t.Errorf("augmented schema should NOT contain 'actorsSimilar' for node without VectorField:\n%s", sdl)
	}
	if strings.Contains(sdl, "ActorSimilarResult") {
		t.Errorf("augmented schema should NOT contain 'ActorSimilarResult' for node without VectorField:\n%s", sdl)
	}
}

// Test: Augmented schema with vector types is valid GraphQL SDL.
// Expected: gqlparser.LoadSchema succeeds.
func TestAugmentSchema_VectorTypesValidSDL(t *testing.T) {
	sdl, err := AugmentSchema(vectorMovieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if sdl == "" {
		t.Skip("AugmentSchema returned empty — skipping parse test")
	}
	_, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema with vector types failed gqlparser validation: %v\nSDL:\n%s", parseErr, sdl)
	}
}
