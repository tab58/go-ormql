package codegen

import (
	"go/format"
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
)

// --- CG-25: SimilarResult model + vector field on node struct tests ---

// vectorFullModel returns a GraphModel with a Movie node that has a VectorField
// for model generation testing.
func vectorFullModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
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

// Test: GenerateModels produces MovieSimilarResult struct for node with VectorField.
// Expected: output contains "type MovieSimilarResult struct".
func TestGenerateModels_VectorSimilarResultStruct(t *testing.T) {
	out, err := GenerateModels(vectorFullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type MovieSimilarResult struct") {
		t.Errorf("expected 'type MovieSimilarResult struct' in output:\n%s", src)
	}
}

// Test: MovieSimilarResult has Score float64 field with json:"score" tag.
// Expected: output contains Score field with float64 type and correct JSON tag.
func TestGenerateModels_VectorSimilarResultScoreField(t *testing.T) {
	out, err := GenerateModels(vectorFullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "Score") {
		t.Errorf("MovieSimilarResult missing 'Score' field:\n%s", src)
	}
	if !strings.Contains(src, `json:"score"`) {
		t.Errorf("MovieSimilarResult Score field missing json:\"score\" tag:\n%s", src)
	}
}

// Test: MovieSimilarResult has Node *Movie field with json:"node" tag.
// Expected: SimilarResult struct block contains Node *Movie.
func TestGenerateModels_VectorSimilarResultNodeField(t *testing.T) {
	out, err := GenerateModels(vectorFullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)

	// Find the MovieSimilarResult struct block
	idx := strings.Index(src, "type MovieSimilarResult struct")
	if idx == -1 {
		t.Fatal("MovieSimilarResult struct not found in output")
	}
	block := src[idx:]
	closeIdx := strings.Index(block, "}")
	if closeIdx == -1 {
		t.Fatal("MovieSimilarResult struct block not properly closed")
	}
	block = block[:closeIdx]

	if !strings.Contains(block, "Node") || !strings.Contains(block, "*Movie") {
		t.Errorf("MovieSimilarResult missing 'Node *Movie' field in block:\n%s", block)
	}
}

// Test: Node struct includes vector field (embedding) as []float64.
// Expected: Movie struct contains Embedding []float64 field.
func TestGenerateModels_VectorFieldOnNodeStruct(t *testing.T) {
	out, err := GenerateModels(vectorFullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "Embedding") || !strings.Contains(src, "[]float64") {
		t.Errorf("Movie struct missing 'Embedding []float64' field:\n%s", src)
	}
}

// Test: GenerateModels does NOT produce SimilarResult for node without VectorField.
// Expected: output does NOT contain "ActorSimilarResult".
func TestGenerateModels_NoSimilarResultWithoutVector(t *testing.T) {
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
	out, err := GenerateModels(model, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if strings.Contains(src, "ActorSimilarResult") {
		t.Errorf("should NOT generate ActorSimilarResult for node without VectorField:\n%s", src)
	}
}

// Test: GenerateModels output with vector types passes gofmt.
// Expected: gofmt.Source returns no error.
func TestGenerateModels_VectorPassesGofmt(t *testing.T) {
	out, err := GenerateModels(vectorFullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, fmtErr := format.Source(out)
	if fmtErr != nil {
		t.Errorf("GenerateModels with vector output does not pass gofmt: %v\nOutput:\n%s", fmtErr, string(out))
	}
}
