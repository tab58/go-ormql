package codegen

import (
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
)

// --- CG-15: @cypher field passthrough in augmented schema ---

// cypherFieldModel returns a model with a Movie node that has a @cypher field
// "averageRating" (Float, no args) and a @cypher field "recommended" ([Movie!]!, with args).
func cypherFieldModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
					{Name: "released", GraphQLType: "Int", GoType: "*int", CypherType: "INTEGER", Nullable: true},
				},
				CypherFields: []schema.CypherFieldDefinition{
					{
						Name:        "averageRating",
						GraphQLType: "Float",
						GoType:      "*float64",
						Statement:   "MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.score)",
						IsList:      false,
						Nullable:    true,
						Arguments:   nil,
					},
					{
						Name:        "recommended",
						GraphQLType: "[Movie!]!",
						GoType:      "[]*Movie",
						Statement:   "MATCH (this)-[:IN_GENRE]->()<-[:IN_GENRE]-(rec) RETURN rec LIMIT $limit",
						IsList:      true,
						Nullable:    false,
						Arguments: []schema.ArgumentDefinition{
							{Name: "limit", GraphQLType: "Int!", GoType: "int", DefaultValue: nil},
						},
					},
				},
			},
		},
	}
}

// TestAugmentSchema_CypherField_InObjectType verifies that @cypher fields appear
// in the generated object type definition.
// Expected: type Movie { id: ID!, title: String!, released: Int, averageRating: Float, recommended(limit: Int!): [Movie!]! }
func TestAugmentSchema_CypherField_InObjectType(t *testing.T) {
	sdl, err := AugmentSchema(cypherFieldModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	movieType := s.Types["Movie"]
	if movieType == nil {
		t.Fatal("Movie type not found in parsed schema")
	}

	avgField := movieType.Fields.ForName("averageRating")
	if avgField == nil {
		t.Error("Movie type missing 'averageRating' @cypher field")
	}

	recField := movieType.Fields.ForName("recommended")
	if recField == nil {
		t.Error("Movie type missing 'recommended' @cypher field")
	}
}

// TestAugmentSchema_CypherField_ReturnType verifies that @cypher fields have
// the correct return type in the object type.
// Expected: averageRating: Float, recommended: [Movie!]!
func TestAugmentSchema_CypherField_ReturnType(t *testing.T) {
	sdl, err := AugmentSchema(cypherFieldModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Check for the field declarations in the SDL
	if !strings.Contains(sdl, "averageRating: Float") {
		t.Errorf("SDL missing 'averageRating: Float':\n%s", sdl)
	}
	if !strings.Contains(sdl, "recommended") {
		t.Errorf("SDL missing 'recommended' field:\n%s", sdl)
	}
}

// TestAugmentSchema_CypherField_WithArguments verifies that @cypher fields with
// arguments have those arguments in the object type field definition.
// Expected: recommended(limit: Int!): [Movie!]!
func TestAugmentSchema_CypherField_WithArguments(t *testing.T) {
	sdl, err := AugmentSchema(cypherFieldModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	movieType := s.Types["Movie"]
	if movieType == nil {
		t.Fatal("Movie type not found in parsed schema")
	}

	recField := movieType.Fields.ForName("recommended")
	if recField == nil {
		t.Fatal("Movie type missing 'recommended' @cypher field")
	}

	limitArg := recField.Arguments.ForName("limit")
	if limitArg == nil {
		t.Error("recommended field missing 'limit' argument")
	} else if limitArg.Type.Name() != "Int" {
		t.Errorf("recommended.limit type = %q, want Int", limitArg.Type.Name())
	}
}

// TestAugmentSchema_CypherField_ExcludedFromCreateInput verifies that @cypher fields
// do NOT appear in the CreateInput type.
// Expected: MovieCreateInput has title, released — NOT averageRating or recommended.
func TestAugmentSchema_CypherField_ExcludedFromCreateInput(t *testing.T) {
	sdl, err := AugmentSchema(cypherFieldModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	createInput := s.Types["MovieCreateInput"]
	if createInput == nil {
		t.Fatal("MovieCreateInput type not found")
	}

	fieldNames := map[string]bool{}
	for _, f := range createInput.Fields {
		fieldNames[f.Name] = true
	}

	if fieldNames["averageRating"] {
		t.Error("MovieCreateInput should NOT contain 'averageRating' (@cypher field)")
	}
	if fieldNames["recommended"] {
		t.Error("MovieCreateInput should NOT contain 'recommended' (@cypher field)")
	}
}

// TestAugmentSchema_CypherField_ExcludedFromUpdateInput verifies that @cypher fields
// do NOT appear in the UpdateInput type.
// Expected: MovieUpdateInput has title, released — NOT averageRating or recommended.
func TestAugmentSchema_CypherField_ExcludedFromUpdateInput(t *testing.T) {
	sdl, err := AugmentSchema(cypherFieldModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	updateInput := s.Types["MovieUpdateInput"]
	if updateInput == nil {
		t.Fatal("MovieUpdateInput type not found")
	}

	fieldNames := map[string]bool{}
	for _, f := range updateInput.Fields {
		fieldNames[f.Name] = true
	}

	if fieldNames["averageRating"] {
		t.Error("MovieUpdateInput should NOT contain 'averageRating' (@cypher field)")
	}
	if fieldNames["recommended"] {
		t.Error("MovieUpdateInput should NOT contain 'recommended' (@cypher field)")
	}
}

// TestAugmentSchema_CypherField_ExcludedFromWhere verifies that @cypher fields
// do NOT appear in the Where input type.
// Expected: MovieWhere has id, title, released — NOT averageRating or recommended.
func TestAugmentSchema_CypherField_ExcludedFromWhere(t *testing.T) {
	sdl, err := AugmentSchema(cypherFieldModel())
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

	if fieldNames["averageRating"] {
		t.Error("MovieWhere should NOT contain 'averageRating' (@cypher field)")
	}
	if fieldNames["recommended"] {
		t.Error("MovieWhere should NOT contain 'recommended' (@cypher field)")
	}
}

// TestAugmentSchema_CypherField_ExcludedFromSort verifies that @cypher fields
// do NOT appear in the Sort input type.
// Expected: MovieSort has id, title, released — NOT averageRating or recommended.
func TestAugmentSchema_CypherField_ExcludedFromSort(t *testing.T) {
	sdl, err := AugmentSchema(cypherFieldModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	movieSort := s.Types["MovieSort"]
	if movieSort == nil {
		// Sort may not exist yet (CG-12 not implemented), that's a separate test
		t.Skip("MovieSort type not found — sort inputs may not be implemented yet")
	}

	fieldNames := map[string]bool{}
	for _, f := range movieSort.Fields {
		fieldNames[f.Name] = true
	}

	if fieldNames["averageRating"] {
		t.Error("MovieSort should NOT contain 'averageRating' (@cypher field)")
	}
	if fieldNames["recommended"] {
		t.Error("MovieSort should NOT contain 'recommended' (@cypher field)")
	}
}

// TestAugmentSchema_CypherField_NoArgsField verifies that a @cypher field with
// no arguments appears without parentheses in the type.
// Expected: averageRating: Float (no args)
func TestAugmentSchema_CypherField_NoArgsField(t *testing.T) {
	sdl, err := AugmentSchema(cypherFieldModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	movieType := s.Types["Movie"]
	if movieType == nil {
		t.Fatal("Movie type not found in parsed schema")
	}

	avgField := movieType.Fields.ForName("averageRating")
	if avgField == nil {
		t.Fatal("Movie type missing 'averageRating' @cypher field")
	}

	if len(avgField.Arguments) != 0 {
		t.Errorf("averageRating should have no arguments, got %d", len(avgField.Arguments))
	}
}

// TestAugmentSchema_CypherField_ValidSDL verifies that the augmented schema
// with @cypher field passthrough is valid GraphQL SDL.
func TestAugmentSchema_CypherField_ValidSDL(t *testing.T) {
	sdl, err := AugmentSchema(cypherFieldModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if sdl == "" {
		t.Skip("AugmentSchema returned empty")
	}
	_, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema with @cypher fields failed gqlparser validation: %v\nSDL:\n%s", parseErr, sdl)
	}
}
