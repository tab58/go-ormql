package codegen

import (
	"go/format"
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
)

// --- CG-27: GenerateIndexes tests ---

// singleVectorModel returns a GraphModel with one node with VectorField.
func singleVectorModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
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

// multiVectorModel returns a GraphModel with two nodes that each have a VectorField.
func multiVectorModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "embedding", GraphQLType: "[Float!]!", GoType: "[]float64", CypherType: "LIST<FLOAT>", IsList: true},
				},
				VectorField: &schema.VectorFieldDefinition{
					Name:       "embedding",
					IndexName:  "movie_embeddings",
					Dimensions: 1536,
					Similarity: "cosine",
				},
			},
			{
				Name:   "Article",
				Labels: []string{"Article"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "vector", GraphQLType: "[Float!]!", GoType: "[]float64", CypherType: "LIST<FLOAT>", IsList: true},
				},
				VectorField: &schema.VectorFieldDefinition{
					Name:       "vector",
					IndexName:  "article_vectors",
					Dimensions: 768,
					Similarity: "euclidean",
				},
			},
		},
	}
}

// Test: GenerateIndexes returns nil,nil when no nodes have VectorField.
// Expected: nil output, nil error.
func TestGenerateIndexes_NoVectorReturnsNil(t *testing.T) {
	model := schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Actor",
				Labels: []string{"Actor"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
				},
				// No VectorField
			},
		},
	}
	out, err := GenerateIndexes(model, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != nil {
		t.Errorf("expected nil output for model without VectorField, got %d bytes", len(out))
	}
}

// Test: GenerateIndexes returns non-nil output when a node has VectorField.
// Expected: non-nil, non-empty byte slice.
func TestGenerateIndexes_SingleVectorReturnsOutput(t *testing.T) {
	out, err := GenerateIndexes(singleVectorModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || len(out) == 0 {
		t.Fatal("GenerateIndexes should return non-nil output for model with VectorField")
	}
}

// Test: GenerateIndexes output contains CreateIndexes function declaration.
// Expected: output contains "func CreateIndexes(ctx context.Context, drv driver.Driver) error".
func TestGenerateIndexes_ContainsCreateIndexesFunc(t *testing.T) {
	out, err := GenerateIndexes(singleVectorModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil — cannot check for CreateIndexes function")
	}
	src := string(out)
	if !strings.Contains(src, "func CreateIndexes") {
		t.Errorf("expected 'func CreateIndexes' in output:\n%s", src)
	}
	if !strings.Contains(src, "context.Context") {
		t.Errorf("expected 'context.Context' parameter in CreateIndexes:\n%s", src)
	}
	if !strings.Contains(src, "driver.Driver") {
		t.Errorf("expected 'driver.Driver' parameter in CreateIndexes:\n%s", src)
	}
}

// Test: GenerateIndexes output contains CREATE VECTOR INDEX DDL with correct index name.
// Expected: output contains "CREATE VECTOR INDEX movie_embeddings IF NOT EXISTS".
func TestGenerateIndexes_ContainsCreateVectorIndexDDL(t *testing.T) {
	out, err := GenerateIndexes(singleVectorModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil — cannot check DDL")
	}
	src := string(out)
	if !strings.Contains(src, "CREATE VECTOR INDEX") {
		t.Errorf("expected 'CREATE VECTOR INDEX' DDL in output:\n%s", src)
	}
	if !strings.Contains(src, "movie_embeddings") {
		t.Errorf("expected index name 'movie_embeddings' in DDL:\n%s", src)
	}
	if !strings.Contains(src, "IF NOT EXISTS") {
		t.Errorf("expected 'IF NOT EXISTS' in DDL:\n%s", src)
	}
}

// Test: DDL references correct label and field name.
// Expected: DDL contains FOR (n:Movie) ON (n.embedding).
func TestGenerateIndexes_DDLLabelAndField(t *testing.T) {
	out, err := GenerateIndexes(singleVectorModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil — cannot check label/field")
	}
	src := string(out)
	if !strings.Contains(src, "Movie") {
		t.Errorf("expected node label 'Movie' in DDL:\n%s", src)
	}
	if !strings.Contains(src, "embedding") {
		t.Errorf("expected field name 'embedding' in DDL:\n%s", src)
	}
}

// Test: DDL contains OPTIONS with vector dimensions and similarity function.
// Expected: DDL contains dimensions 1536 and similarity 'cosine'.
func TestGenerateIndexes_DDLOptions(t *testing.T) {
	out, err := GenerateIndexes(singleVectorModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil — cannot check OPTIONS")
	}
	src := string(out)
	if !strings.Contains(src, "1536") {
		t.Errorf("expected dimensions 1536 in DDL OPTIONS:\n%s", src)
	}
	if !strings.Contains(src, "cosine") {
		t.Errorf("expected similarity 'cosine' in DDL OPTIONS:\n%s", src)
	}
}

// Test: GenerateIndexes uses drv.ExecuteWrite for DDL execution.
// Expected: output contains "ExecuteWrite" call.
func TestGenerateIndexes_UsesExecuteWrite(t *testing.T) {
	out, err := GenerateIndexes(singleVectorModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil — cannot check ExecuteWrite")
	}
	src := string(out)
	if !strings.Contains(src, "ExecuteWrite") {
		t.Errorf("expected 'ExecuteWrite' call in CreateIndexes function:\n%s", src)
	}
}

// Test: Multiple VectorField nodes produce multiple DDL statements.
// Expected: output contains both "movie_embeddings" and "article_vectors" indexes.
func TestGenerateIndexes_MultipleVectors(t *testing.T) {
	out, err := GenerateIndexes(multiVectorModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil for model with 2 VectorFields")
	}
	src := string(out)
	if !strings.Contains(src, "movie_embeddings") {
		t.Errorf("expected 'movie_embeddings' index in output:\n%s", src)
	}
	if !strings.Contains(src, "article_vectors") {
		t.Errorf("expected 'article_vectors' index in output:\n%s", src)
	}
}

// Test: GenerateIndexes output passes gofmt.
// Expected: gofmt.Source returns no error.
func TestGenerateIndexes_PassesGofmt(t *testing.T) {
	out, err := GenerateIndexes(singleVectorModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Skip("output is nil — skipping gofmt check")
	}
	_, fmtErr := format.Source(out)
	if fmtErr != nil {
		t.Errorf("GenerateIndexes output does not pass gofmt: %v\nOutput:\n%s", fmtErr, string(out))
	}
}

// Test: GenerateIndexes output contains correct package declaration.
// Expected: output starts with "package generated".
func TestGenerateIndexes_PackageDeclaration(t *testing.T) {
	out, err := GenerateIndexes(singleVectorModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil — cannot check package declaration")
	}
	src := string(out)
	if !strings.Contains(src, "package generated") {
		t.Errorf("expected 'package generated' in output:\n%s", src)
	}
}
