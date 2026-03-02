package codegen

import (
	"go/format"
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
)

// --- CG-26: VectorField serialization in registry tests ---

// vectorRegistryModel returns a GraphModel with a Movie node that has a VectorField
// for registry serialization testing.
func vectorRegistryModel() schema.GraphModel {
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

// Test: GenerateGraphModelRegistry serializes VectorField when present on a node.
// Expected: output contains "VectorField:" with a non-nil *schema.VectorFieldDefinition.
func TestGenerateGraphModelRegistry_SerializesVectorField(t *testing.T) {
	out, err := GenerateGraphModelRegistry(vectorRegistryModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "VectorField") {
		t.Errorf("expected 'VectorField' in serialized GraphModel:\n%s", src)
	}
	if !strings.Contains(src, "schema.VectorFieldDefinition") {
		t.Errorf("expected 'schema.VectorFieldDefinition' type reference:\n%s", src)
	}
}

// Test: Serialized VectorField contains correct IndexName.
// Expected: output contains IndexName: "movie_embeddings".
func TestGenerateGraphModelRegistry_VectorFieldIndexName(t *testing.T) {
	out, err := GenerateGraphModelRegistry(vectorRegistryModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, `"movie_embeddings"`) {
		t.Errorf("expected IndexName 'movie_embeddings' in serialized VectorField:\n%s", src)
	}
}

// Test: Serialized VectorField contains correct Dimensions.
// Expected: output contains Dimensions: 1536.
func TestGenerateGraphModelRegistry_VectorFieldDimensions(t *testing.T) {
	out, err := GenerateGraphModelRegistry(vectorRegistryModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "1536") {
		t.Errorf("expected Dimensions 1536 in serialized VectorField:\n%s", src)
	}
}

// Test: Serialized VectorField contains correct Similarity.
// Expected: output contains Similarity: "cosine".
func TestGenerateGraphModelRegistry_VectorFieldSimilarity(t *testing.T) {
	out, err := GenerateGraphModelRegistry(vectorRegistryModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, `"cosine"`) {
		t.Errorf("expected Similarity 'cosine' in serialized VectorField:\n%s", src)
	}
}

// Test: Serialized VectorField contains correct Name.
// Expected: output contains Name: "embedding" within the VectorField block.
func TestGenerateGraphModelRegistry_VectorFieldName(t *testing.T) {
	out, err := GenerateGraphModelRegistry(vectorRegistryModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	// The VectorField block should contain the field name "embedding"
	// (distinct from the FieldDefinition for embedding which also exists)
	if !strings.Contains(src, "VectorField") {
		t.Fatal("VectorField not serialized — cannot verify Name")
	}
	// Find VectorField block and check it contains "embedding"
	idx := strings.Index(src, "VectorField")
	if idx == -1 {
		t.Fatal("VectorField not found in output")
	}
	block := src[idx:]
	if !strings.Contains(block, `"embedding"`) {
		t.Errorf("VectorField block missing Name 'embedding':\n%s", block[:min(len(block), 200)])
	}
}

// Test: Node without VectorField does NOT get VectorField serialized.
// Expected: output for Actor node does NOT contain "VectorField".
func TestGenerateGraphModelRegistry_NoVectorFieldWhenAbsent(t *testing.T) {
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
	out, err := GenerateGraphModelRegistry(model, testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if strings.Contains(src, "VectorField") {
		t.Errorf("should NOT serialize VectorField for node without vector:\n%s", src)
	}
}

// Test: Output with VectorField passes gofmt.
// Expected: gofmt.Source returns no error.
func TestGenerateGraphModelRegistry_VectorPassesGofmt(t *testing.T) {
	out, err := GenerateGraphModelRegistry(vectorRegistryModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, fmtErr := format.Source(out)
	if fmtErr != nil {
		t.Errorf("output with VectorField does not pass gofmt: %v\nOutput:\n%s", fmtErr, string(out))
	}
}

