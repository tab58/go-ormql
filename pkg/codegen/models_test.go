package codegen

import (
	"go/format"
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
)

// --- CG-20: Model generator (GenerateModels) tests ---

// fullModel returns a GraphModel with Movie + Actor nodes, ACTED_IN relationship
// with properties, @cypher field, and an enum for comprehensive model generation testing.
func fullModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
					{Name: "released", GraphQLType: "Int", GoType: "*int", CypherType: "INTEGER", Nullable: true},
				},
				CypherFields: []schema.CypherFieldDefinition{
					{
						Name:        "averageRating",
						GraphQLType: "Float",
						GoType:      "*float64",
						Statement:   "MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.score)",
						Nullable:    true,
					},
				},
			},
			{
				Name:   "Actor",
				Labels: []string{"Actor"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
				},
			},
		},
		Relationships: []schema.RelationshipDefinition{
			{
				FieldName: "actors",
				RelType:   "ACTED_IN",
				Direction: schema.DirectionIN,
				FromNode:  "Movie",
				ToNode:    "Actor",
				Properties: &schema.PropertiesDefinition{
					TypeName: "ActedInProperties",
					Fields: []schema.FieldDefinition{
						{Name: "role", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
					},
				},
			},
		},
		Enums: []schema.EnumDefinition{
			{Name: "Genre", Values: []string{"ACTION", "COMEDY", "DRAMA"}},
		},
	}
}

// Test: GenerateModels returns non-nil output for a valid model.
// Expected: non-nil, non-empty byte slice.
func TestGenerateModels_ReturnsOutput(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || len(out) == 0 {
		t.Fatal("GenerateModels should return non-nil, non-empty output")
	}
}

// Test: GenerateModels output starts with the correct package declaration.
// Expected: output contains "package generated".
func TestGenerateModels_PackageDeclaration(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), "package generated") {
		t.Errorf("expected 'package generated' in output, got:\n%s", string(out))
	}
}

// Test: GenerateModels output contains node struct type declarations.
// Expected: output contains "type Movie struct" and "type Actor struct".
func TestGenerateModels_NodeStructs(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type Movie struct") {
		t.Error("expected 'type Movie struct' in output")
	}
	if !strings.Contains(src, "type Actor struct") {
		t.Error("expected 'type Actor struct' in output")
	}
}

// Test: Node struct fields have JSON tags matching GraphQL field names.
// Expected: output contains json:"title" and json:"released,omitempty".
func TestGenerateModels_JSONTags(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, `json:"title"`) {
		t.Error("expected json:\"title\" tag in output")
	}
	// Nullable fields should have omitempty
	if !strings.Contains(src, `json:"released,omitempty"`) {
		t.Error("expected json:\"released,omitempty\" tag for nullable field")
	}
}

// Test: Node struct includes relationship fields as pointer/slice types.
// Expected: Movie struct contains Actors field with []*Actor type.
func TestGenerateModels_RelationshipFields(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "Actors") || !strings.Contains(src, "[]*Actor") {
		t.Errorf("expected Actors []*Actor field in Movie struct, got:\n%s", src)
	}
}

// Test: Node struct includes @cypher fields as pointer types.
// Expected: Movie struct contains AverageRating *float64 field.
func TestGenerateModels_CypherFields(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "AverageRating") || !strings.Contains(src, "*float64") {
		t.Errorf("expected AverageRating *float64 field in Movie struct, got:\n%s", src)
	}
}

// Test: GenerateModels output contains CreateInput types.
// Expected: output contains "type MovieCreateInput struct".
func TestGenerateModels_CreateInput(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type MovieCreateInput struct") {
		t.Error("expected 'type MovieCreateInput struct' in output")
	}
}

// Test: GenerateModels output contains UpdateInput types with all fields optional.
// Expected: output contains "type MovieUpdateInput struct" with pointer fields.
func TestGenerateModels_UpdateInput(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type MovieUpdateInput struct") {
		t.Error("expected 'type MovieUpdateInput struct' in output")
	}
	// UpdateInput should have pointer fields (optional)
	if !strings.Contains(src, "*string") {
		t.Error("expected pointer fields in UpdateInput (e.g., *string for Title)")
	}
}

// Test: GenerateModels output contains Where type with operator-suffixed fields.
// Expected: output contains "type MovieWhere struct" with fields like TitleContains.
func TestGenerateModels_WhereInput(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type MovieWhere struct") {
		t.Error("expected 'type MovieWhere struct' in output")
	}
	// Should have operator-suffixed fields
	for _, op := range []string{"TitleContains", "TitleGt", "TitleIn", "TitleNot"} {
		if !strings.Contains(src, op) {
			t.Errorf("expected operator field %q in MovieWhere, got:\n%s", op, src)
		}
	}
	// Should have boolean composition fields
	if !strings.Contains(src, "AND") || !strings.Contains(src, "OR") || !strings.Contains(src, "NOT") {
		t.Error("expected AND/OR/NOT fields in MovieWhere")
	}
}

// Test: GenerateModels output contains Sort type.
// Expected: output contains "type MovieSort struct".
func TestGenerateModels_SortInput(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type MovieSort struct") {
		t.Error("expected 'type MovieSort struct' in output")
	}
}

// Test: GenerateModels output contains SortDirection enum.
// Expected: output contains "type SortDirection string" with ASC and DESC constants.
func TestGenerateModels_SortDirectionEnum(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type SortDirection string") {
		t.Error("expected 'type SortDirection string' in output")
	}
	if !strings.Contains(src, "SortDirectionASC") || !strings.Contains(src, "SortDirectionDESC") {
		t.Error("expected SortDirectionASC and SortDirectionDESC constants")
	}
}

// Test: GenerateModels output contains user-defined enum types.
// Expected: output contains "type Genre string" with constants.
func TestGenerateModels_UserEnum(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type Genre string") {
		t.Error("expected 'type Genre string' in output")
	}
	for _, v := range []string{"GenreACTION", "GenreCOMEDY", "GenreDRAMA"} {
		if !strings.Contains(src, v) {
			t.Errorf("expected enum constant %q in output", v)
		}
	}
}

// Test: GenerateModels output contains connection types.
// Expected: output contains MoviesConnection, MovieEdge, PageInfo types.
func TestGenerateModels_ConnectionTypes(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type MoviesConnection struct") {
		t.Error("expected 'type MoviesConnection struct' in output")
	}
	if !strings.Contains(src, "type MovieEdge struct") {
		t.Error("expected 'type MovieEdge struct' in output")
	}
	if !strings.Contains(src, "type PageInfo struct") {
		t.Error("expected 'type PageInfo struct' in output")
	}
}

// Test: GenerateModels output contains nested connection type for relationships.
// Expected: output contains MovieActorsConnection with edge type including Properties.
func TestGenerateModels_RelationshipConnectionTypes(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type MovieActorsConnection struct") {
		t.Error("expected 'type MovieActorsConnection struct' in output")
	}
	if !strings.Contains(src, "type MovieActorsEdge struct") {
		t.Error("expected 'type MovieActorsEdge struct' in output")
	}
	// Edge should include Properties field when @relationshipProperties defined
	if !strings.Contains(src, "ActedInProperties") {
		t.Error("expected ActedInProperties reference in MovieActorsEdge")
	}
}

// Test: GenerateModels output contains mutation response types.
// Expected: output contains CreateMoviesMutationResponse and DeleteInfo.
func TestGenerateModels_ResponseTypes(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type CreateMoviesMutationResponse struct") {
		t.Error("expected 'type CreateMoviesMutationResponse struct' in output")
	}
	if !strings.Contains(src, "type UpdateMoviesMutationResponse struct") {
		t.Error("expected 'type UpdateMoviesMutationResponse struct' in output")
	}
	if !strings.Contains(src, "type DeleteInfo struct") {
		t.Error("expected 'type DeleteInfo struct' in output")
	}
}

// Test: GenerateModels output contains relationship properties type.
// Expected: output contains "type ActedInProperties struct" with Role field.
func TestGenerateModels_RelationshipProperties(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type ActedInProperties struct") {
		t.Error("expected 'type ActedInProperties struct' in output")
	}
	if !strings.Contains(src, "Role") {
		t.Error("expected Role field in ActedInProperties")
	}
}

// Test: GenerateModels output contains nested mutation input types.
// Expected: output contains FieldInput, CreateFieldInput, ConnectFieldInput, etc.
func TestGenerateModels_NestedMutationInputTypes(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	expectedTypes := []string{
		"type MovieActorsFieldInput struct",
		"type MovieActorsUpdateFieldInput struct",
		"type MovieActorsCreateFieldInput struct",
		"type MovieActorsConnectFieldInput struct",
		"type MovieActorsDisconnectFieldInput struct",
		"type MovieActorsDeleteFieldInput struct",
	}
	for _, expected := range expectedTypes {
		if !strings.Contains(src, expected) {
			t.Errorf("expected %q in output", expected)
		}
	}
}

// Test: GenerateModels output contains properties input types for create/update.
// Expected: output contains ActedInPropertiesCreateInput and ActedInPropertiesUpdateInput.
func TestGenerateModels_PropertiesInputTypes(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type ActedInPropertiesCreateInput struct") {
		t.Error("expected 'type ActedInPropertiesCreateInput struct' in output")
	}
	if !strings.Contains(src, "type ActedInPropertiesUpdateInput struct") {
		t.Error("expected 'type ActedInPropertiesUpdateInput struct' in output")
	}
}

// Test: GenerateModels output passes gofmt (well-formatted Go source).
// Expected: gofmt.Source returns no error.
func TestGenerateModels_PassesGofmt(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil, cannot check gofmt")
	}
	_, fmtErr := format.Source(out)
	if fmtErr != nil {
		t.Errorf("GenerateModels output does not pass gofmt: %v\nOutput:\n%s", fmtErr, string(out))
	}
}

// Test: GenerateModels for a node with no relationships produces model with
// only scalar fields (no connection/field input types for that node).
// Expected: output contains node struct without relationship fields.
func TestGenerateModels_NodeWithNoRelationships(t *testing.T) {
	model := schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Genre",
				Labels: []string{"Genre"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string"},
				},
			},
		},
	}
	out, err := GenerateModels(model, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type Genre struct") {
		t.Error("expected 'type Genre struct' in output")
	}
	// Should NOT have relationship-specific types for Genre
	if strings.Contains(src, "GenreFieldInput") {
		t.Error("should not generate FieldInput for node with no relationships")
	}
}

// Test: GenerateModels with empty packageName returns error.
// Expected: non-nil error.
func TestGenerateModels_EmptyPackageName_ReturnsError(t *testing.T) {
	_, err := GenerateModels(fullModel(), "")
	if err == nil {
		t.Error("GenerateModels with empty packageName should return error")
	}
}
