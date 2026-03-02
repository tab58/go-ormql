package schema

import (
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// --- SM-6: @vector directive parsing + VectorFieldDefinition tests ---

// Test: ExtractVectorDirective returns HasDirective=false when field has no @vector directive.
func TestExtractVectorDirective_NoDirective(t *testing.T) {
	field := &ast.FieldDefinition{
		Name: "title",
		Directives: ast.DirectiveList{},
	}
	info := ExtractVectorDirective(field)
	if info.HasDirective {
		t.Error("expected HasDirective=false for field without @vector")
	}
}

// Test: ExtractVectorDirective returns HasDirective=false when field is nil.
func TestExtractVectorDirective_NilField(t *testing.T) {
	info := ExtractVectorDirective(nil)
	if info.HasDirective {
		t.Error("expected HasDirective=false for nil field")
	}
}

// Test: ExtractVectorDirective correctly extracts @vector(indexName, dimensions, similarity).
// Expected: HasDirective=true, IndexName="movie_embeddings", Dimensions=1536, Similarity="cosine"
func TestExtractVectorDirective_ValidDirective(t *testing.T) {
	field := &ast.FieldDefinition{
		Name: "embedding",
		Directives: ast.DirectiveList{
			{
				Name: "vector",
				Arguments: ast.ArgumentList{
					{Name: "indexName", Value: &ast.Value{Raw: "movie_embeddings", Kind: ast.StringValue}},
					{Name: "dimensions", Value: &ast.Value{Raw: "1536", Kind: ast.IntValue}},
					{Name: "similarity", Value: &ast.Value{Raw: "cosine", Kind: ast.StringValue}},
				},
			},
		},
	}
	info := ExtractVectorDirective(field)
	if !info.HasDirective {
		t.Fatal("expected HasDirective=true")
	}
	if info.IndexName != "movie_embeddings" {
		t.Errorf("IndexName = %q, want %q", info.IndexName, "movie_embeddings")
	}
	if info.Dimensions != 1536 {
		t.Errorf("Dimensions = %d, want %d", info.Dimensions, 1536)
	}
	if info.Similarity != "cosine" {
		t.Errorf("Similarity = %q, want %q", info.Similarity, "cosine")
	}
}

// Test: BuiltinDirectiveDefs includes the @vector directive definition.
// Expected: SDL string contains "directive @vector"
func TestBuiltinDirectiveDefs_ContainsVectorDirective(t *testing.T) {
	defs := BuiltinDirectiveDefs()
	if !strings.Contains(defs, "directive @vector") {
		t.Error("BuiltinDirectiveDefs() does not contain @vector directive definition")
	}
	// Should include all 3 required args
	if !strings.Contains(defs, "indexName: String!") {
		t.Error("@vector directive missing indexName argument")
	}
	if !strings.Contains(defs, "dimensions: Int!") {
		t.Error("@vector directive missing dimensions argument")
	}
	if !strings.Contains(defs, "similarity: String!") {
		t.Error("@vector directive missing similarity argument")
	}
}

// Test: ValidateDirectives reports error when @vector field type is not [Float!]!.
// Expected: validation error mentioning the field name and expected type.
func TestValidateDirectives_VectorWrongType(t *testing.T) {
	tests := []struct {
		name    string
		gqlType string
	}{
		{"String type", "String!"},
		{"nullable list", "[Float]"},
		{"non-list Float", "Float!"},
		{"nullable inner", "[Float!]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &ast.SchemaDocument{
				Definitions: ast.DefinitionList{
					{
						Kind: ast.Object,
						Name: "Movie",
						Directives: ast.DirectiveList{{Name: "node"}},
						Fields: ast.FieldList{
							{
								Name: "embedding",
								Type: parseASTType(tt.gqlType),
								Directives: ast.DirectiveList{
									{
										Name: "vector",
										Arguments: ast.ArgumentList{
											{Name: "indexName", Value: &ast.Value{Raw: "idx", Kind: ast.StringValue}},
											{Name: "dimensions", Value: &ast.Value{Raw: "1536", Kind: ast.IntValue}},
											{Name: "similarity", Value: &ast.Value{Raw: "cosine", Kind: ast.StringValue}},
										},
									},
								},
							},
						},
					},
				},
			}
			errs := ValidateDirectives(doc)
			if len(errs) == 0 {
				t.Errorf("expected validation error for @vector on %s field, got none", tt.gqlType)
			}
		})
	}
}

// Test: ValidateDirectives reports error when two @vector fields on same @node.
// Expected: validation error about at most one @vector per node.
func TestValidateDirectives_MultipleVectorPerNode(t *testing.T) {
	doc := &ast.SchemaDocument{
		Definitions: ast.DefinitionList{
			{
				Kind: ast.Object,
				Name: "Movie",
				Directives: ast.DirectiveList{{Name: "node"}},
				Fields: ast.FieldList{
					makeVectorField("embedding1", "idx1", "1536", "cosine"),
					makeVectorField("embedding2", "idx2", "768", "euclidean"),
				},
			},
		},
	}
	errs := ValidateDirectives(doc)
	if len(errs) == 0 {
		t.Error("expected validation error for multiple @vector on same node, got none")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "vector") {
			found = true
		}
	}
	if !found {
		t.Error("expected error message to mention @vector")
	}
}

// Test: ValidateDirectives reports error when @vector and @cypher on same field.
// Expected: mutual exclusivity error.
func TestValidateDirectives_VectorCypherMutualExclusivity(t *testing.T) {
	doc := &ast.SchemaDocument{
		Definitions: ast.DefinitionList{
			{
				Kind: ast.Object,
				Name: "Movie",
				Directives: ast.DirectiveList{{Name: "node"}},
				Fields: ast.FieldList{
					{
						Name: "embedding",
						Type: parseASTType("[Float!]!"),
						Directives: ast.DirectiveList{
							{
								Name: "vector",
								Arguments: ast.ArgumentList{
									{Name: "indexName", Value: &ast.Value{Raw: "idx", Kind: ast.StringValue}},
									{Name: "dimensions", Value: &ast.Value{Raw: "1536", Kind: ast.IntValue}},
									{Name: "similarity", Value: &ast.Value{Raw: "cosine", Kind: ast.StringValue}},
								},
							},
							{
								Name: "cypher",
								Arguments: ast.ArgumentList{
									{Name: "statement", Value: &ast.Value{Raw: "RETURN []", Kind: ast.StringValue}},
								},
							},
						},
					},
				},
			},
		},
	}
	errs := ValidateDirectives(doc)
	if len(errs) == 0 {
		t.Error("expected mutual exclusivity error for @vector + @cypher, got none")
	}
}

// Test: ValidateDirectives reports error when @vector and @relationship on same field.
// Expected: mutual exclusivity error.
func TestValidateDirectives_VectorRelationshipMutualExclusivity(t *testing.T) {
	doc := &ast.SchemaDocument{
		Definitions: ast.DefinitionList{
			{
				Kind: ast.Object,
				Name: "Movie",
				Directives: ast.DirectiveList{{Name: "node"}},
				Fields: ast.FieldList{
					{
						Name: "embedding",
						Type: parseASTType("[Float!]!"),
						Directives: ast.DirectiveList{
							{
								Name: "vector",
								Arguments: ast.ArgumentList{
									{Name: "indexName", Value: &ast.Value{Raw: "idx", Kind: ast.StringValue}},
									{Name: "dimensions", Value: &ast.Value{Raw: "1536", Kind: ast.IntValue}},
									{Name: "similarity", Value: &ast.Value{Raw: "cosine", Kind: ast.StringValue}},
								},
							},
							{
								Name: "relationship",
								Arguments: ast.ArgumentList{
									{Name: "type", Value: &ast.Value{Raw: "SIMILAR_TO", Kind: ast.StringValue}},
									{Name: "direction", Value: &ast.Value{Raw: "OUT", Kind: ast.EnumValue}},
								},
							},
						},
					},
				},
			},
		},
	}
	errs := ValidateDirectives(doc)
	if len(errs) == 0 {
		t.Error("expected mutual exclusivity error for @vector + @relationship, got none")
	}
}

// Test: ValidateDirectives passes for valid @vector on [Float!]! field.
// Expected: no errors.
func TestValidateDirectives_VectorValidField(t *testing.T) {
	doc := &ast.SchemaDocument{
		Definitions: ast.DefinitionList{
			{
				Kind: ast.Object,
				Name: "Movie",
				Directives: ast.DirectiveList{{Name: "node"}},
				Fields: ast.FieldList{
					makeVectorField("embedding", "movie_embeddings", "1536", "cosine"),
				},
			},
		},
	}
	errs := ValidateDirectives(doc)
	for _, e := range errs {
		if strings.Contains(e.Error(), "vector") {
			t.Errorf("unexpected validation error for valid @vector field: %v", e)
		}
	}
}

// Test: ParseSchemaString populates NodeDefinition.VectorField when @vector is found.
// Expected: VectorField is non-nil with correct IndexName, Dimensions, Similarity.
func TestParseSchemaString_VectorField(t *testing.T) {
	sdl := `
type Movie @node {
  id: ID!
  title: String!
  embedding: [Float!]! @vector(indexName: "movie_embeddings", dimensions: 1536, similarity: "cosine")
}
`
	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString() error: %v", err)
	}
	node, ok := model.NodeByName("Movie")
	if !ok {
		t.Fatal("Movie node not found")
	}
	if node.VectorField == nil {
		t.Fatal("expected VectorField to be non-nil on Movie node")
	}
	if node.VectorField.Name != "embedding" {
		t.Errorf("VectorField.Name = %q, want %q", node.VectorField.Name, "embedding")
	}
	if node.VectorField.IndexName != "movie_embeddings" {
		t.Errorf("VectorField.IndexName = %q, want %q", node.VectorField.IndexName, "movie_embeddings")
	}
	if node.VectorField.Dimensions != 1536 {
		t.Errorf("VectorField.Dimensions = %d, want %d", node.VectorField.Dimensions, 1536)
	}
	if node.VectorField.Similarity != "cosine" {
		t.Errorf("VectorField.Similarity = %q, want %q", node.VectorField.Similarity, "cosine")
	}
}

// Test: ParseSchemaString keeps vector field in Fields as a regular FieldDefinition.
// Expected: Fields includes "embedding" with GraphQLType "[Float!]!" alongside VectorField.
func TestParseSchemaString_VectorFieldAlsoInFields(t *testing.T) {
	sdl := `
type Movie @node {
  id: ID!
  title: String!
  embedding: [Float!]! @vector(indexName: "movie_embeddings", dimensions: 1536, similarity: "cosine")
}
`
	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString() error: %v", err)
	}
	node, ok := model.NodeByName("Movie")
	if !ok {
		t.Fatal("Movie node not found")
	}
	found := false
	for _, f := range node.Fields {
		if f.Name == "embedding" {
			found = true
			if f.GraphQLType != "[Float!]!" {
				t.Errorf("embedding field GraphQLType = %q, want %q", f.GraphQLType, "[Float!]!")
			}
		}
	}
	if !found {
		t.Error("expected 'embedding' to be present in Fields as a regular FieldDefinition")
	}
}

// Test: ParseSchemaString with no @vector field returns nil VectorField.
// Expected: VectorField is nil.
func TestParseSchemaString_NoVectorField(t *testing.T) {
	sdl := `
type Movie @node {
  id: ID!
  title: String!
}
`
	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString() error: %v", err)
	}
	node, ok := model.NodeByName("Movie")
	if !ok {
		t.Fatal("Movie node not found")
	}
	if node.VectorField != nil {
		t.Error("expected VectorField to be nil for node without @vector")
	}
}

// --- Test Helpers ---

// makeVectorField creates a field definition with @vector directive for testing.
func makeVectorField(name, indexName, dimensions, similarity string) *ast.FieldDefinition {
	return &ast.FieldDefinition{
		Name: name,
		Type: parseASTType("[Float!]!"),
		Directives: ast.DirectiveList{
			{
				Name: "vector",
				Arguments: ast.ArgumentList{
					{Name: "indexName", Value: &ast.Value{Raw: indexName, Kind: ast.StringValue}},
					{Name: "dimensions", Value: &ast.Value{Raw: dimensions, Kind: ast.IntValue}},
					{Name: "similarity", Value: &ast.Value{Raw: similarity, Kind: ast.StringValue}},
				},
			},
		},
	}
}

// parseASTType builds a minimal ast.Type from a GraphQL type string.
// Handles: "Float!", "[Float!]!", "[Float!]", "[Float]", "String!", etc.
func parseASTType(gqlType string) *ast.Type {
	nonNull := strings.HasSuffix(gqlType, "!")
	s := strings.TrimSuffix(gqlType, "!")

	if strings.HasPrefix(s, "[") {
		inner := strings.TrimPrefix(s, "[")
		inner = strings.TrimSuffix(inner, "]")
		elemType := parseASTType(inner)
		return &ast.Type{
			Elem:    elemType,
			NonNull: nonNull,
		}
	}

	return &ast.Type{
		NamedType: s,
		NonNull:   nonNull,
	}
}
