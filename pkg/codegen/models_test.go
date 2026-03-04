package codegen

import (
	"go/format"
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
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

// --- Custom scalar type alias tests ---

// dateTimeModel returns a GraphModel with a node that uses a DateTime custom scalar.
func dateTimeModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Event",
				Labels: []string{"Event"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
					{Name: "startTime", GraphQLType: "DateTime!", GoType: "time.Time", CypherType: "LOCAL DATETIME"},
					{Name: "endTime", GraphQLType: "DateTime", GoType: "*time.Time", CypherType: "LOCAL DATETIME", Nullable: true},
				},
			},
		},
		CustomScalars: []string{"DateTime"},
	}
}

// Test: GenerateModels uses scalar alias (DateTime) instead of raw Go type (time.Time)
// when the schema declares a custom scalar.
// Expected: output contains "DateTime" for fields, not "time.Time".
func TestGenerateModels_CustomScalarUsesAlias(t *testing.T) {
	out, err := GenerateModels(dateTimeModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)

	// Non-nullable DateTime! field should use "DateTime" type
	if !strings.Contains(src, "DateTime") {
		t.Error("expected DateTime alias in output, got none")
	}
	// Should NOT use time.Time directly (the alias is in scalars_gen.go)
	if strings.Contains(src, "time.Time") {
		t.Errorf("expected scalar alias DateTime instead of time.Time in:\n%s", src)
	}
}

// Test: GenerateModels with custom scalar does not require "time" import.
// Expected: output does NOT contain import "time".
func TestGenerateModels_CustomScalarNoTimeImport(t *testing.T) {
	out, err := GenerateModels(dateTimeModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if strings.Contains(src, `"time"`) {
		t.Errorf("models should not import \"time\" when using scalar aliases:\n%s", src)
	}
}

// Test: GenerateModels with custom scalar still passes gofmt.
func TestGenerateModels_CustomScalar_PassesGofmt(t *testing.T) {
	out, err := GenerateModels(dateTimeModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, fmtErr := format.Source(out)
	if fmtErr != nil {
		t.Errorf("output with custom scalar does not pass gofmt: %v\nOutput:\n%s", fmtErr, string(out))
	}
}

// Test: Nullable DateTime field uses *DateTime pointer type.
func TestGenerateModels_NullableCustomScalar(t *testing.T) {
	out, err := GenerateModels(dateTimeModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "*DateTime") {
		t.Errorf("expected *DateTime for nullable field, got:\n%s", src)
	}
}

// Test: modelsResolveGoType returns scalar alias for custom scalars.
func TestModelsResolveGoType(t *testing.T) {
	customScalars := map[string]bool{"DateTime": true, "Money": true}

	tests := []struct {
		name        string
		graphqlType string
		goType      string
		want        string
	}{
		{"non-nullable scalar", "DateTime!", "time.Time", "DateTime"},
		{"nullable scalar", "DateTime", "*time.Time", "*DateTime"},
		{"list scalar", "[DateTime!]!", "[]time.Time", "[]DateTime"},
		{"unknown scalar", "Money!", "any", "Money"},
		{"non-scalar type", "String!", "string", "string"},
		{"non-scalar nullable", "Int", "*int", "*int"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := modelsResolveGoType(tt.graphqlType, tt.goType, customScalars)
			if got != tt.want {
				t.Errorf("modelsResolveGoType(%q, %q) = %q, want %q", tt.graphqlType, tt.goType, got, tt.want)
			}
		})
	}
}

// Test: modelsResolveGoType with nil customScalars returns original GoType.
func TestModelsResolveGoType_NilScalars(t *testing.T) {
	got := modelsResolveGoType("DateTime!", "time.Time", nil)
	if got != "time.Time" {
		t.Errorf("with nil customScalars, expected original GoType, got %q", got)
	}
}

// === CG-34: Merge/connect Go model struct tests ===

// TestGenerateModels_MatchInput verifies that {Node}MatchInput structs are
// generated with all-pointer fields (all optional), excluding id and vector.
// Expected: MovieMatchInput with *string Title, *int Released — no ID field.
func TestGenerateModels_MatchInput(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type MovieMatchInput struct") {
		t.Errorf("missing MovieMatchInput struct:\n%s", src)
	}
	// All fields should be pointer types (optional)
	if !strings.Contains(src, "*string") {
		t.Errorf("MovieMatchInput should have pointer string fields:\n%s", src)
	}
}

// TestGenerateModels_MergeInput verifies that {Node}MergeInput structs are
// generated with Match, OnCreate, and OnMatch fields.
// Expected: MovieMergeInput { Match *MovieMatchInput, OnCreate *MovieCreateInput, OnMatch *MovieUpdateInput }
func TestGenerateModels_MergeInput(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type MovieMergeInput struct") {
		t.Errorf("missing MovieMergeInput struct:\n%s", src)
	}
	if !strings.Contains(src, "MovieMatchInput") {
		t.Errorf("MovieMergeInput should reference MovieMatchInput:\n%s", src)
	}
}

// TestGenerateModels_MergeMutationResponse verifies that Merge{Nodes}MutationResponse
// structs are generated following the same pattern as CreateMutationResponse.
// Expected: MergeMoviesMutationResponse { Movies []*Movie }
func TestGenerateModels_MergeMutationResponse(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type MergeMoviesMutationResponse struct") {
		t.Errorf("missing MergeMoviesMutationResponse struct:\n%s", src)
	}
}

// TestGenerateModels_ConnectInput verifies that Connect{Source}{Field}Input structs
// are generated with From, To, and optional Edge fields.
// Expected: ConnectMovieActorsInput { From *MovieWhere, To *ActorWhere, Edge *ActedInPropertiesCreateInput }
func TestGenerateModels_ConnectInput(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type ConnectMovieActorsInput struct") {
		t.Errorf("missing ConnectMovieActorsInput struct:\n%s", src)
	}
}

// TestGenerateModels_ConnectInfo verifies that ConnectInfo is generated once
// with RelationshipsCreated int field.
// Expected: type ConnectInfo struct { RelationshipsCreated int }
func TestGenerateModels_ConnectInfo(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "type ConnectInfo struct") {
		t.Errorf("missing ConnectInfo struct:\n%s", src)
	}
	if !strings.Contains(src, "RelationshipsCreated") {
		t.Errorf("ConnectInfo missing RelationshipsCreated field:\n%s", src)
	}
}

// TestGenerateModels_MergeConnectPassesGofmt verifies that generated code with
// merge/connect types passes gofmt formatting.
func TestGenerateModels_MergeConnectPassesGofmt(t *testing.T) {
	out, err := GenerateModels(fullModel(), "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, fmtErr := format.Source(out)
	if fmtErr != nil {
		t.Errorf("generated models with merge/connect types do not pass gofmt: %v\n%s", fmtErr, string(out))
	}
}
