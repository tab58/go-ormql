package codegen

import (
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
)

// --- CG-14: Relationship connection types in augmented schema ---

// TestAugmentSchema_RelConnection_TypeGenerated verifies that each @relationship field
// generates a {Node}{FieldCap}Connection type.
// Expected: MovieActorsConnection { edges, pageInfo, totalCount }
func TestAugmentSchema_RelConnection_TypeGenerated(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieActorsConnection") {
		t.Fatalf("augmented schema missing 'MovieActorsConnection':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	connType := s.Types["MovieActorsConnection"]
	if connType == nil {
		t.Fatal("MovieActorsConnection type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range connType.Fields {
		fieldNames[f.Name] = true
	}

	for _, want := range []string{"edges", "pageInfo", "totalCount"} {
		if !fieldNames[want] {
			t.Errorf("MovieActorsConnection missing '%s' field", want)
		}
	}
}

// TestAugmentSchema_RelConnection_EdgeType verifies that {Node}{FieldCap}Edge type
// is generated with node and cursor fields.
// Expected: MovieActorsEdge { node: Actor!, cursor: String! }
func TestAugmentSchema_RelConnection_EdgeType(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieActorsEdge") {
		t.Fatalf("augmented schema missing 'MovieActorsEdge':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	edgeType := s.Types["MovieActorsEdge"]
	if edgeType == nil {
		t.Fatal("MovieActorsEdge type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range edgeType.Fields {
		fieldNames[f.Name] = true
	}

	if !fieldNames["node"] {
		t.Error("MovieActorsEdge missing 'node' field")
	}
	if !fieldNames["cursor"] {
		t.Error("MovieActorsEdge missing 'cursor' field")
	}
}

// TestAugmentSchema_RelConnection_EdgeNodeType verifies that the edge's node field
// has the correct target type.
// Expected: MovieActorsEdge.node: Actor!
func TestAugmentSchema_RelConnection_EdgeNodeType(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	edgeType := s.Types["MovieActorsEdge"]
	if edgeType == nil {
		t.Fatal("MovieActorsEdge type not found in parsed schema")
	}

	nodeField := edgeType.Fields.ForName("node")
	if nodeField == nil {
		t.Fatal("MovieActorsEdge missing 'node' field")
	}
	if nodeField.Type.Name() != "Actor" {
		t.Errorf("MovieActorsEdge.node type = %q, want Actor", nodeField.Type.Name())
	}
}

// TestAugmentSchema_RelConnection_EdgeProperties_Present verifies that when
// @relationshipProperties exists, the edge type has a 'properties' field.
// Expected: MovieActorsEdge { node, cursor, properties: ActedInProperties }
func TestAugmentSchema_RelConnection_EdgeProperties_Present(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	edgeType := s.Types["MovieActorsEdge"]
	if edgeType == nil {
		t.Fatal("MovieActorsEdge type not found in parsed schema")
	}

	propsField := edgeType.Fields.ForName("properties")
	if propsField == nil {
		t.Error("MovieActorsEdge missing 'properties' field — should be present with @relationshipProperties")
	} else if propsField.Type.Name() != "ActedInProperties" {
		t.Errorf("MovieActorsEdge.properties type = %q, want ActedInProperties", propsField.Type.Name())
	}
}

// TestAugmentSchema_RelConnection_EdgeProperties_Absent verifies that when
// no @relationshipProperties exist, the edge type does NOT have a 'properties' field.
// Expected: MovieCategoriesEdge { node, cursor } (no properties)
func TestAugmentSchema_RelConnection_EdgeProperties_Absent(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipNoProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	edgeType := s.Types["MovieCategoriesEdge"]
	if edgeType == nil {
		t.Fatal("MovieCategoriesEdge type not found in parsed schema")
	}

	propsField := edgeType.Fields.ForName("properties")
	if propsField != nil {
		t.Error("MovieCategoriesEdge has 'properties' field — should be absent without @relationshipProperties")
	}
}

// TestAugmentSchema_RelConnection_FieldOnParentType verifies that the parent node type
// gets a {fieldName}Connection field.
// Expected: type Movie { ..., actorsConnection(...): MovieActorsConnection! }
func TestAugmentSchema_RelConnection_FieldOnParentType(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
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

	connField := movieType.Fields.ForName("actorsConnection")
	if connField == nil {
		t.Error("Movie type missing 'actorsConnection' field")
	} else if connField.Type.Name() != "MovieActorsConnection" {
		t.Errorf("Movie.actorsConnection type = %q, want MovieActorsConnection", connField.Type.Name())
	}
}

// TestAugmentSchema_RelConnection_FieldParams verifies that the connection field
// on the parent type accepts first, after, where, and sort parameters.
// Expected: actorsConnection(first: Int, after: String, where: ActorWhere, sort: [ActorSort!]): MovieActorsConnection!
func TestAugmentSchema_RelConnection_FieldParams(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
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

	connField := movieType.Fields.ForName("actorsConnection")
	if connField == nil {
		t.Fatal("Movie type missing 'actorsConnection' field")
	}

	for _, argName := range []string{"first", "after", "where", "sort"} {
		arg := connField.Arguments.ForName(argName)
		if arg == nil {
			t.Errorf("actorsConnection missing '%s' parameter", argName)
		}
	}
}

// TestAugmentSchema_RelConnection_WhereParamType verifies that the where parameter
// on the connection field references {TargetNode}Where.
// Expected: actorsConnection(where: ActorWhere)
func TestAugmentSchema_RelConnection_WhereParamType(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
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

	connField := movieType.Fields.ForName("actorsConnection")
	if connField == nil {
		t.Fatal("Movie type missing 'actorsConnection' field")
	}

	whereArg := connField.Arguments.ForName("where")
	if whereArg == nil {
		t.Fatal("actorsConnection missing 'where' parameter")
	}
	if whereArg.Type.Name() != "ActorWhere" {
		t.Errorf("actorsConnection where param type = %q, want ActorWhere", whereArg.Type.Name())
	}
}

// TestAugmentSchema_RelConnection_NoProperties verifies that connection types
// are generated even for relationships without @relationshipProperties.
// Expected: MovieCategoriesConnection and MovieCategoriesEdge present.
func TestAugmentSchema_RelConnection_NoProperties(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipNoProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieCategoriesConnection") {
		t.Errorf("augmented schema missing 'MovieCategoriesConnection':\n%s", sdl)
	}
	if !strings.Contains(sdl, "MovieCategoriesEdge") {
		t.Errorf("augmented schema missing 'MovieCategoriesEdge':\n%s", sdl)
	}
}

// TestAugmentSchema_RelConnection_PropertiesTypeEmitted verifies that when
// @relationshipProperties exists, the properties type definition is emitted
// as a GraphQL type (not input) for reading edge properties.
// Expected: type ActedInProperties { role: String! }
func TestAugmentSchema_RelConnection_PropertiesTypeEmitted(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "type ActedInProperties") {
		t.Errorf("augmented schema missing 'type ActedInProperties':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	propsType := s.Types["ActedInProperties"]
	if propsType == nil {
		t.Fatal("ActedInProperties type not found in parsed schema")
	}

	roleField := propsType.Fields.ForName("role")
	if roleField == nil {
		t.Error("ActedInProperties missing 'role' field")
	}
}

// TestAugmentSchema_RelConnection_MultiRelationship verifies that a model with
// multiple relationships on the same node generates separate connection types for each.
func TestAugmentSchema_RelConnection_MultiRelationship(t *testing.T) {
	model := schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{Name: "Movie", Labels: []string{"Movie"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", IsID: true},
				{Name: "title", GraphQLType: "String!"},
			}},
			{Name: "Actor", Labels: []string{"Actor"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", IsID: true},
				{Name: "name", GraphQLType: "String!"},
			}},
			{Name: "Director", Labels: []string{"Director"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", IsID: true},
				{Name: "name", GraphQLType: "String!"},
			}},
		},
		Relationships: []schema.RelationshipDefinition{
			{
				FieldName:  "actors",
				RelType:    "ACTED_IN",
				Direction:  schema.DirectionIN,
				FromNode:   "Movie",
				ToNode:     "Actor",
				Properties: nil,
			},
			{
				FieldName:  "directors",
				RelType:    "DIRECTED",
				Direction:  schema.DirectionIN,
				FromNode:   "Movie",
				ToNode:     "Director",
				Properties: nil,
			},
		},
	}

	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieActorsConnection") {
		t.Errorf("augmented schema missing 'MovieActorsConnection':\n%s", sdl)
	}
	if !strings.Contains(sdl, "MovieDirectorsConnection") {
		t.Errorf("augmented schema missing 'MovieDirectorsConnection':\n%s", sdl)
	}
	if !strings.Contains(sdl, "MovieActorsEdge") {
		t.Errorf("augmented schema missing 'MovieActorsEdge':\n%s", sdl)
	}
	if !strings.Contains(sdl, "MovieDirectorsEdge") {
		t.Errorf("augmented schema missing 'MovieDirectorsEdge':\n%s", sdl)
	}
}

// TestAugmentSchema_RelConnection_ValidSDL verifies that the augmented schema
// with relationship connection types is valid GraphQL SDL.
func TestAugmentSchema_RelConnection_ValidSDL(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if sdl == "" {
		t.Skip("AugmentSchema returned empty")
	}
	_, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema with relationship connections failed gqlparser validation: %v\nSDL:\n%s", parseErr, sdl)
	}
}
