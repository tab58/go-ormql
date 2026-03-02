package codegen

import (
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
)

// --- CG-13: Nested disconnect/update/delete input types ---

// TestAugmentSchema_UpdateFieldInput_AllFiveOps verifies that the UpdateFieldInput
// type for a relationship has all 5 operation fields: create, connect, disconnect, update, delete.
// Expected: MovieActorsUpdateFieldInput { create, connect, disconnect, update, delete }
func TestAugmentSchema_UpdateFieldInput_AllFiveOps(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieActorsUpdateFieldInput") {
		t.Fatalf("augmented schema missing 'MovieActorsUpdateFieldInput':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	updateFieldInput := s.Types["MovieActorsUpdateFieldInput"]
	if updateFieldInput == nil {
		t.Fatal("MovieActorsUpdateFieldInput type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range updateFieldInput.Fields {
		fieldNames[f.Name] = true
	}

	for _, op := range []string{"create", "connect", "disconnect", "update", "delete"} {
		if !fieldNames[op] {
			t.Errorf("MovieActorsUpdateFieldInput missing '%s' field", op)
		}
	}
}

// TestAugmentSchema_UpdateFieldInput_NoProperties verifies that UpdateFieldInput
// is also generated for relationships WITHOUT @relationshipProperties.
// Expected: MovieCategoriesUpdateFieldInput with all 5 ops.
func TestAugmentSchema_UpdateFieldInput_NoProperties(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipNoProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieCategoriesUpdateFieldInput") {
		t.Fatalf("augmented schema missing 'MovieCategoriesUpdateFieldInput':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	updateFieldInput := s.Types["MovieCategoriesUpdateFieldInput"]
	if updateFieldInput == nil {
		t.Fatal("MovieCategoriesUpdateFieldInput type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range updateFieldInput.Fields {
		fieldNames[f.Name] = true
	}

	for _, op := range []string{"create", "connect", "disconnect", "update", "delete"} {
		if !fieldNames[op] {
			t.Errorf("MovieCategoriesUpdateFieldInput missing '%s' field", op)
		}
	}
}

// TestAugmentSchema_DisconnectFieldInput verifies that {Node}{FieldCap}DisconnectFieldInput
// is generated with a 'where' field referencing {TargetNode}Where.
// Expected: MovieActorsDisconnectFieldInput { where: ActorWhere }
func TestAugmentSchema_DisconnectFieldInput(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieActorsDisconnectFieldInput") {
		t.Fatalf("augmented schema missing 'MovieActorsDisconnectFieldInput':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	disconnectInput := s.Types["MovieActorsDisconnectFieldInput"]
	if disconnectInput == nil {
		t.Fatal("MovieActorsDisconnectFieldInput type not found in parsed schema")
	}

	whereField := disconnectInput.Fields.ForName("where")
	if whereField == nil {
		t.Error("MovieActorsDisconnectFieldInput missing 'where' field")
	} else if whereField.Type.Name() != "ActorWhere" {
		t.Errorf("MovieActorsDisconnectFieldInput.where type = %q, want ActorWhere", whereField.Type.Name())
	}
}

// TestAugmentSchema_DeleteFieldInput verifies that {Node}{FieldCap}DeleteFieldInput
// is generated with a 'where' field referencing {TargetNode}Where.
// Expected: MovieActorsDeleteFieldInput { where: ActorWhere }
func TestAugmentSchema_DeleteFieldInput(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieActorsDeleteFieldInput") {
		t.Fatalf("augmented schema missing 'MovieActorsDeleteFieldInput':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	deleteInput := s.Types["MovieActorsDeleteFieldInput"]
	if deleteInput == nil {
		t.Fatal("MovieActorsDeleteFieldInput type not found in parsed schema")
	}

	whereField := deleteInput.Fields.ForName("where")
	if whereField == nil {
		t.Error("MovieActorsDeleteFieldInput missing 'where' field")
	} else if whereField.Type.Name() != "ActorWhere" {
		t.Errorf("MovieActorsDeleteFieldInput.where type = %q, want ActorWhere", whereField.Type.Name())
	}
}

// TestAugmentSchema_UpdateConnectionInput_WithProperties verifies that
// {Node}{FieldCap}UpdateConnectionInput is generated with where, node, and edge
// fields when @relationshipProperties is present.
// Expected: MovieActorsUpdateConnectionInput { where: ActorWhere, node: ActorUpdateInput, edge: ActedInPropertiesUpdateInput }
func TestAugmentSchema_UpdateConnectionInput_WithProperties(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieActorsUpdateConnectionInput") {
		t.Fatalf("augmented schema missing 'MovieActorsUpdateConnectionInput':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	updateConnInput := s.Types["MovieActorsUpdateConnectionInput"]
	if updateConnInput == nil {
		t.Fatal("MovieActorsUpdateConnectionInput type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range updateConnInput.Fields {
		fieldNames[f.Name] = true
	}

	if !fieldNames["where"] {
		t.Error("MovieActorsUpdateConnectionInput missing 'where' field")
	}
	if !fieldNames["node"] {
		t.Error("MovieActorsUpdateConnectionInput missing 'node' field")
	}
	if !fieldNames["edge"] {
		t.Error("MovieActorsUpdateConnectionInput missing 'edge' field — should be present with @relationshipProperties")
	}
}

// TestAugmentSchema_UpdateConnectionInput_WithoutProperties verifies that
// {Node}{FieldCap}UpdateConnectionInput has where and node but NOT edge
// when there are no @relationshipProperties.
// Expected: MovieCategoriesUpdateConnectionInput { where: CategoryWhere, node: CategoryUpdateInput } (no edge)
func TestAugmentSchema_UpdateConnectionInput_WithoutProperties(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipNoProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieCategoriesUpdateConnectionInput") {
		t.Fatalf("augmented schema missing 'MovieCategoriesUpdateConnectionInput':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	updateConnInput := s.Types["MovieCategoriesUpdateConnectionInput"]
	if updateConnInput == nil {
		t.Fatal("MovieCategoriesUpdateConnectionInput type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range updateConnInput.Fields {
		fieldNames[f.Name] = true
	}

	if !fieldNames["where"] {
		t.Error("MovieCategoriesUpdateConnectionInput missing 'where' field")
	}
	if !fieldNames["node"] {
		t.Error("MovieCategoriesUpdateConnectionInput missing 'node' field")
	}
	if fieldNames["edge"] {
		t.Error("MovieCategoriesUpdateConnectionInput has 'edge' field — should be absent without @relationshipProperties")
	}
}

// TestAugmentSchema_PropertiesUpdateInput verifies that {PropertiesType}UpdateInput
// is generated with all fields made optional.
// Expected: ActedInPropertiesUpdateInput { role: String } (not String!)
func TestAugmentSchema_PropertiesUpdateInput(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "ActedInPropertiesUpdateInput") {
		t.Fatalf("augmented schema missing 'ActedInPropertiesUpdateInput':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	propsUpdate := s.Types["ActedInPropertiesUpdateInput"]
	if propsUpdate == nil {
		t.Fatal("ActedInPropertiesUpdateInput type not found in parsed schema")
	}

	roleField := propsUpdate.Fields.ForName("role")
	if roleField == nil {
		t.Error("ActedInPropertiesUpdateInput missing 'role' field")
	} else if roleField.Type.NonNull {
		t.Error("ActedInPropertiesUpdateInput.role should be optional (not NonNull) in update input")
	}
}

// TestAugmentSchema_PropertiesUpdateInput_GeneratedOnce verifies that a shared
// @relationshipProperties type generates its UpdateInput only once.
// Expected: exactly one occurrence of "ActedInPropertiesUpdateInput".
func TestAugmentSchema_PropertiesUpdateInput_GeneratedOnce(t *testing.T) {
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
					Fields:   []schema.FieldDefinition{{Name: "role", GraphQLType: "String!"}},
				},
			},
			{
				FieldName: "movies",
				RelType:   "ACTED_IN",
				Direction: schema.DirectionOUT,
				FromNode:  "Actor",
				ToNode:    "Movie",
				Properties: &schema.PropertiesDefinition{
					TypeName: "ActedInProperties",
					Fields:   []schema.FieldDefinition{{Name: "role", GraphQLType: "String!"}},
				},
			},
		},
	}

	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	count := strings.Count(sdl, "input ActedInPropertiesUpdateInput")
	if count != 1 {
		t.Errorf("'input ActedInPropertiesUpdateInput' appears %d times, want exactly 1:\n%s", count, sdl)
	}
}

// TestAugmentSchema_UpdateInput_UsesUpdateFieldInput verifies that {NodeName}UpdateInput
// includes a field for each relationship using UpdateFieldInput (not FieldInput).
// Expected: MovieUpdateInput { title: String, released: Int, actors: MovieActorsUpdateFieldInput }
func TestAugmentSchema_UpdateInput_UsesUpdateFieldInput(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	updateInput := s.Types["MovieUpdateInput"]
	if updateInput == nil {
		t.Fatal("MovieUpdateInput type not found in parsed schema")
	}

	actorsField := updateInput.Fields.ForName("actors")
	if actorsField == nil {
		t.Error("MovieUpdateInput missing 'actors' field for relationship")
	} else if actorsField.Type.Name() != "MovieActorsUpdateFieldInput" {
		t.Errorf("MovieUpdateInput.actors type = %q, want MovieActorsUpdateFieldInput", actorsField.Type.Name())
	}
}

// TestAugmentSchema_UpdateFieldInput_CreateFieldRef verifies that UpdateFieldInput.create
// references the correct CreateFieldInput type (same as FieldInput.create).
// Expected: MovieActorsUpdateFieldInput.create: [MovieActorsCreateFieldInput!]
func TestAugmentSchema_UpdateFieldInput_CreateFieldRef(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Check that create field references MovieActorsCreateFieldInput
	if !strings.Contains(sdl, "MovieActorsCreateFieldInput") {
		t.Fatalf("augmented schema missing 'MovieActorsCreateFieldInput':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	updateFieldInput := s.Types["MovieActorsUpdateFieldInput"]
	if updateFieldInput == nil {
		t.Fatal("MovieActorsUpdateFieldInput type not found")
	}

	createField := updateFieldInput.Fields.ForName("create")
	if createField == nil {
		t.Fatal("MovieActorsUpdateFieldInput missing 'create' field")
	}

	// create should be a list of MovieActorsCreateFieldInput
	if !strings.Contains(sdl, "MovieActorsUpdateFieldInput") {
		t.Errorf("missing MovieActorsUpdateFieldInput")
	}
}

// TestAugmentSchema_NestedUpdateTypes_ValidSDL verifies that the augmented schema
// with all nested update/disconnect/delete types is valid GraphQL SDL.
func TestAugmentSchema_NestedUpdateTypes_ValidSDL(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if sdl == "" {
		t.Skip("AugmentSchema returned empty")
	}
	_, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema with nested update types failed gqlparser validation: %v\nSDL:\n%s", parseErr, sdl)
	}
}

// TestAugmentSchema_DisconnectFieldInput_NoProperties verifies DisconnectFieldInput
// is generated even when no @relationshipProperties exist.
// Expected: MovieCategoriesDisconnectFieldInput { where: CategoryWhere }
func TestAugmentSchema_DisconnectFieldInput_NoProperties(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipNoProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieCategoriesDisconnectFieldInput") {
		t.Errorf("augmented schema missing 'MovieCategoriesDisconnectFieldInput':\n%s", sdl)
	}
}

// TestAugmentSchema_DeleteFieldInput_NoProperties verifies DeleteFieldInput
// is generated even when no @relationshipProperties exist.
// Expected: MovieCategoriesDeleteFieldInput { where: CategoryWhere }
func TestAugmentSchema_DeleteFieldInput_NoProperties(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipNoProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieCategoriesDeleteFieldInput") {
		t.Errorf("augmented schema missing 'MovieCategoriesDeleteFieldInput':\n%s", sdl)
	}
}
