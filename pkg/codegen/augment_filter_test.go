package codegen

import (
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
)

// --- CG-11: Filter operator fields in augmented schema ---

// filterTestModel returns a model with a Movie node containing diverse scalar types
// for testing filter operator applicability rules.
func filterTestModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
					{Name: "released", GraphQLType: "Int", GoType: "*int", CypherType: "INTEGER", Nullable: true},
					{Name: "rating", GraphQLType: "Float", GoType: "*float64", CypherType: "FLOAT", Nullable: true},
					{Name: "active", GraphQLType: "Boolean!", GoType: "bool", CypherType: "BOOLEAN", Nullable: false},
				},
			},
		},
	}
}

// TestAugmentSchema_WhereInput_ComparisonOperators verifies that Int, Float, String, ID fields
// get _gt, _gte, _lt, _lte comparison operator fields. Boolean does NOT get these.
func TestAugmentSchema_WhereInput_ComparisonOperators(t *testing.T) {
	sdl, err := AugmentSchema(filterTestModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Int field "released" should have comparison operators
	for _, suffix := range []string{"released_gt", "released_gte", "released_lt", "released_lte"} {
		if !strings.Contains(sdl, suffix) {
			t.Errorf("MovieWhere missing comparison operator field %q:\n%s", suffix, sdl)
		}
	}

	// String field "title" should have comparison operators
	for _, suffix := range []string{"title_gt", "title_gte", "title_lt", "title_lte"} {
		if !strings.Contains(sdl, suffix) {
			t.Errorf("MovieWhere missing comparison operator field %q:\n%s", suffix, sdl)
		}
	}

	// Float field "rating" should have comparison operators
	for _, suffix := range []string{"rating_gt", "rating_gte", "rating_lt", "rating_lte"} {
		if !strings.Contains(sdl, suffix) {
			t.Errorf("MovieWhere missing comparison operator field %q:\n%s", suffix, sdl)
		}
	}

	// Boolean field "active" should NOT have comparison operators
	for _, suffix := range []string{"active_gt", "active_gte", "active_lt", "active_lte"} {
		if strings.Contains(sdl, suffix) {
			t.Errorf("MovieWhere should NOT have comparison operator %q for Boolean field:\n%s", suffix, sdl)
		}
	}
}

// TestAugmentSchema_WhereInput_StringOperators verifies that String and ID fields
// get _contains, _startsWith, _endsWith, _regex. Int, Float, Boolean do NOT.
func TestAugmentSchema_WhereInput_StringOperators(t *testing.T) {
	sdl, err := AugmentSchema(filterTestModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// String field "title" should have string operators
	for _, suffix := range []string{"title_contains", "title_startsWith", "title_endsWith", "title_regex"} {
		if !strings.Contains(sdl, suffix) {
			t.Errorf("MovieWhere missing string operator field %q:\n%s", suffix, sdl)
		}
	}

	// ID field "id" should have string operators
	for _, suffix := range []string{"id_contains", "id_startsWith", "id_endsWith", "id_regex"} {
		if !strings.Contains(sdl, suffix) {
			t.Errorf("MovieWhere missing string operator field %q for ID type:\n%s", suffix, sdl)
		}
	}

	// Int field "released" should NOT have string operators
	for _, suffix := range []string{"released_contains", "released_startsWith", "released_endsWith", "released_regex"} {
		if strings.Contains(sdl, suffix) {
			t.Errorf("MovieWhere should NOT have string operator %q for Int field:\n%s", suffix, sdl)
		}
	}

	// Boolean field "active" should NOT have string operators
	for _, suffix := range []string{"active_contains", "active_startsWith"} {
		if strings.Contains(sdl, suffix) {
			t.Errorf("MovieWhere should NOT have string operator %q for Boolean field:\n%s", suffix, sdl)
		}
	}
}

// TestAugmentSchema_WhereInput_ListOperators verifies that all scalar fields
// get _in and _nin (list membership) operator fields.
func TestAugmentSchema_WhereInput_ListOperators(t *testing.T) {
	sdl, err := AugmentSchema(filterTestModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// All scalar types should have _in and _nin
	for _, field := range []string{"id", "title", "released", "rating", "active"} {
		for _, suffix := range []string{"_in", "_nin"} {
			if !strings.Contains(sdl, field+suffix) {
				t.Errorf("MovieWhere missing list operator field %q:\n%s", field+suffix, sdl)
			}
		}
	}
}

// TestAugmentSchema_WhereInput_NegationOperator verifies that all scalar fields
// get _not (negation) operator field.
func TestAugmentSchema_WhereInput_NegationOperator(t *testing.T) {
	sdl, err := AugmentSchema(filterTestModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	for _, field := range []string{"id", "title", "released", "rating", "active"} {
		if !strings.Contains(sdl, field+"_not") {
			t.Errorf("MovieWhere missing negation operator field %q:\n%s", field+"_not", sdl)
		}
	}
}

// TestAugmentSchema_WhereInput_IsNullOperator verifies that all scalar fields
// get _isNull: Boolean field.
func TestAugmentSchema_WhereInput_IsNullOperator(t *testing.T) {
	sdl, err := AugmentSchema(filterTestModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	for _, field := range []string{"id", "title", "released", "rating", "active"} {
		if !strings.Contains(sdl, field+"_isNull") {
			t.Errorf("MovieWhere missing _isNull field %q:\n%s", field+"_isNull", sdl)
		}
	}
}

// TestAugmentSchema_WhereInput_BooleanComposition verifies that MovieWhere has
// AND, OR, NOT boolean composition fields.
// Expected: AND: [MovieWhere!], OR: [MovieWhere!], NOT: MovieWhere
func TestAugmentSchema_WhereInput_BooleanComposition(t *testing.T) {
	sdl, err := AugmentSchema(filterTestModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Parse to check field types
	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	movieWhere := s.Types["MovieWhere"]
	if movieWhere == nil {
		t.Fatal("MovieWhere type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range movieWhere.Fields {
		fieldNames[f.Name] = true
	}

	if !fieldNames["AND"] {
		t.Error("MovieWhere missing 'AND' boolean composition field")
	}
	if !fieldNames["OR"] {
		t.Error("MovieWhere missing 'OR' boolean composition field")
	}
	if !fieldNames["NOT"] {
		t.Error("MovieWhere missing 'NOT' boolean composition field")
	}
}

// TestAugmentSchema_WhereInput_InFieldIsListType verifies that _in fields are list types.
// Expected: title_in: [String!] (list of the scalar type)
func TestAugmentSchema_WhereInput_InFieldIsListType(t *testing.T) {
	sdl, err := AugmentSchema(filterTestModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// The _in field for "title" (String!) should be a list: [String!]
	// Check that the SDL contains something like "title_in: [String"
	if !strings.Contains(sdl, "title_in: [") {
		t.Errorf("title_in should be a list type (e.g., [String!]):\n%s", sdl)
	}
}

// TestAugmentSchema_WhereInput_IsNullFieldIsBoolean verifies that _isNull fields
// are Boolean type (not the field's scalar type).
func TestAugmentSchema_WhereInput_IsNullFieldIsBoolean(t *testing.T) {
	sdl, err := AugmentSchema(filterTestModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Parse to check field types
	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	movieWhere := s.Types["MovieWhere"]
	if movieWhere == nil {
		t.Fatal("MovieWhere type not found in parsed schema")
	}

	for _, f := range movieWhere.Fields {
		if f.Name == "title_isNull" {
			if f.Type.Name() != "Boolean" {
				t.Errorf("title_isNull type = %q, want Boolean", f.Type.Name())
			}
			return
		}
	}
	t.Error("title_isNull field not found in MovieWhere")
}

// TestAugmentSchema_WhereInput_OperatorsValidSDL verifies that the augmented schema
// with all filter operator fields is still valid GraphQL SDL.
func TestAugmentSchema_WhereInput_OperatorsValidSDL(t *testing.T) {
	sdl, err := AugmentSchema(filterTestModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if sdl == "" {
		t.Skip("AugmentSchema returned empty")
	}
	_, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema with filter operators failed gqlparser validation: %v\nSDL:\n%s", parseErr, sdl)
	}
}

// TestAugmentSchema_WhereInput_MultiNode_EachHasOperators verifies that each node
// gets its own set of operator fields in its Where input.
func TestAugmentSchema_WhereInput_MultiNode_EachHasOperators(t *testing.T) {
	model := schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", IsID: true},
					{Name: "title", GraphQLType: "String!"},
				},
			},
			{
				Name:   "Actor",
				Labels: []string{"Actor"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", IsID: true},
					{Name: "name", GraphQLType: "String!"},
				},
			},
		},
	}

	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Both nodes should have boolean composition in their Where inputs
	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	for _, typeName := range []string{"MovieWhere", "ActorWhere"} {
		whereType := s.Types[typeName]
		if whereType == nil {
			t.Errorf("%s type not found in parsed schema", typeName)
			continue
		}
		fieldNames := map[string]bool{}
		for _, f := range whereType.Fields {
			fieldNames[f.Name] = true
		}
		if !fieldNames["AND"] {
			t.Errorf("%s missing 'AND' boolean composition field", typeName)
		}
		if !fieldNames["OR"] {
			t.Errorf("%s missing 'OR' boolean composition field", typeName)
		}
		if !fieldNames["NOT"] {
			t.Errorf("%s missing 'NOT' boolean composition field", typeName)
		}
	}
}

// TestAugmentSchema_WhereInput_EqualityFieldsStillPresent verifies that the original
// equality fields are still present alongside the new operator fields.
func TestAugmentSchema_WhereInput_EqualityFieldsStillPresent(t *testing.T) {
	sdl, err := AugmentSchema(filterTestModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	movieWhere := s.Types["MovieWhere"]
	if movieWhere == nil {
		t.Fatal("MovieWhere type not found")
	}

	fieldNames := map[string]bool{}
	for _, f := range movieWhere.Fields {
		fieldNames[f.Name] = true
	}

	// Original equality fields should still be present
	for _, name := range []string{"id", "title", "released", "rating", "active"} {
		if !fieldNames[name] {
			t.Errorf("MovieWhere missing original equality field %q", name)
		}
	}
}
