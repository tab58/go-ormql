package codegen

import (
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

// --- Test helpers ---

// movieModel returns a simple GraphModel with a Movie node for testing.
func movieModel() schema.GraphModel {
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
			},
		},
	}
}

// multiNodeModel returns a GraphModel with Movie and Actor nodes plus a relationship.
func multiNodeModel() schema.GraphModel {
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
			},
			{
				Name:   "Actor",
				Labels: []string{"Actor"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
				},
			},
		},
		Relationships: []schema.RelationshipDefinition{
			{FieldName: "movies", RelType: "ACTED_IN", Direction: schema.DirectionOUT, FromNode: "Actor", ToNode: "Movie"},
		},
	}
}

// parseSDL is a helper that parses a GraphQL SDL string and returns an error if invalid.
func parseSDL(sdl string) (*ast.Schema, error) {
	src := &ast.Source{Name: "augmented.graphql", Input: sdl}
	return gqlparser.LoadSchema(src)
}

// --- Tests ---

// TestAugmentSchema_NonEmpty verifies that augmenting a valid model produces non-empty output.
func TestAugmentSchema_NonEmpty(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if sdl == "" {
		t.Fatal("AugmentSchema returned empty string, want non-empty augmented schema")
	}
}

// TestAugmentSchema_ParsesWithGqlparser verifies that the output is valid GraphQL SDL
// by parsing it with gqlparser.
func TestAugmentSchema_ParsesWithGqlparser(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if sdl == "" {
		t.Skip("AugmentSchema returned empty — skipping parse test")
	}
	_, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed gqlparser validation: %v\nSDL:\n%s", parseErr, sdl)
	}
}

// TestAugmentSchema_OriginalTypePreserved verifies that the original Movie type
// with its scalar fields is present in the augmented schema.
func TestAugmentSchema_OriginalTypePreserved(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "type Movie") {
		t.Errorf("augmented schema missing 'type Movie':\n%s", sdl)
	}
}

// TestAugmentSchema_QueriesGenerated verifies that query fields are generated for the node.
// Expected: movies(where: MovieWhere): [Movie!]! and moviesConnection(...)
func TestAugmentSchema_QueriesGenerated(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "movies") {
		t.Errorf("augmented schema missing 'movies' query field:\n%s", sdl)
	}
	if !strings.Contains(sdl, "moviesConnection") {
		t.Errorf("augmented schema missing 'moviesConnection' query field:\n%s", sdl)
	}
}

// TestAugmentSchema_MutationsGenerated verifies that mutation fields are generated.
// Expected: createMovies, updateMovies, deleteMovies
func TestAugmentSchema_MutationsGenerated(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "createMovies") {
		t.Errorf("augmented schema missing 'createMovies' mutation:\n%s", sdl)
	}
	if !strings.Contains(sdl, "updateMovies") {
		t.Errorf("augmented schema missing 'updateMovies' mutation:\n%s", sdl)
	}
	if !strings.Contains(sdl, "deleteMovies") {
		t.Errorf("augmented schema missing 'deleteMovies' mutation:\n%s", sdl)
	}
}

// TestAugmentSchema_InputTypesGenerated verifies that input types are generated.
// Expected: MovieWhere, MovieCreateInput, MovieUpdateInput
func TestAugmentSchema_InputTypesGenerated(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "MovieWhere") {
		t.Errorf("augmented schema missing 'MovieWhere' input type:\n%s", sdl)
	}
	if !strings.Contains(sdl, "MovieCreateInput") {
		t.Errorf("augmented schema missing 'MovieCreateInput' input type:\n%s", sdl)
	}
	if !strings.Contains(sdl, "MovieUpdateInput") {
		t.Errorf("augmented schema missing 'MovieUpdateInput' input type:\n%s", sdl)
	}
}

// TestAugmentSchema_ResponseTypesGenerated verifies that mutation response types are generated.
// Expected: CreateMoviesMutationResponse, UpdateMoviesMutationResponse
func TestAugmentSchema_ResponseTypesGenerated(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "CreateMoviesMutationResponse") {
		t.Errorf("augmented schema missing 'CreateMoviesMutationResponse':\n%s", sdl)
	}
	if !strings.Contains(sdl, "UpdateMoviesMutationResponse") {
		t.Errorf("augmented schema missing 'UpdateMoviesMutationResponse':\n%s", sdl)
	}
}

// TestAugmentSchema_RelayTypesGenerated verifies that Relay connection types are generated.
// Expected: MoviesConnection, MovieEdge
func TestAugmentSchema_RelayTypesGenerated(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "MoviesConnection") {
		t.Errorf("augmented schema missing 'MoviesConnection':\n%s", sdl)
	}
	if !strings.Contains(sdl, "MovieEdge") {
		t.Errorf("augmented schema missing 'MovieEdge':\n%s", sdl)
	}
}

// TestAugmentSchema_SharedTypesGenerated verifies that shared types are generated exactly once.
// Expected: DeleteInfo and PageInfo types present in the output.
func TestAugmentSchema_SharedTypesGenerated(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "DeleteInfo") {
		t.Errorf("augmented schema missing 'DeleteInfo' shared type:\n%s", sdl)
	}
	if !strings.Contains(sdl, "PageInfo") {
		t.Errorf("augmented schema missing 'PageInfo' shared type:\n%s", sdl)
	}
}

// TestAugmentSchema_SharedTypesNotDuplicated verifies that shared types appear only once
// even with multiple nodes.
func TestAugmentSchema_SharedTypesNotDuplicated(t *testing.T) {
	sdl, err := AugmentSchema(multiNodeModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if sdl == "" {
		t.Skip("AugmentSchema returned empty — skipping duplication check")
	}
	// Count occurrences of "type DeleteInfo"
	deleteCount := strings.Count(sdl, "type DeleteInfo")
	if deleteCount != 1 {
		t.Errorf("'type DeleteInfo' appears %d times, want exactly 1", deleteCount)
	}
	pageCount := strings.Count(sdl, "type PageInfo")
	if pageCount != 1 {
		t.Errorf("'type PageInfo' appears %d times, want exactly 1", pageCount)
	}
}

// TestAugmentSchema_MultiNode verifies that types are generated for each @node in a multi-node model.
// Expected: Movie and Actor both get queries, mutations, input types, connection types.
func TestAugmentSchema_MultiNode(t *testing.T) {
	sdl, err := AugmentSchema(multiNodeModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Movie types
	for _, want := range []string{"type Movie", "movies", "createMovies", "MovieWhere", "MoviesConnection"} {
		if !strings.Contains(sdl, want) {
			t.Errorf("augmented schema missing Movie artifact %q:\n%s", want, sdl)
		}
	}

	// Actor types
	for _, want := range []string{"type Actor", "actors", "createActors", "ActorWhere", "ActorsConnection"} {
		if !strings.Contains(sdl, want) {
			t.Errorf("augmented schema missing Actor artifact %q:\n%s", want, sdl)
		}
	}
}

// TestAugmentSchema_WhereInputOnlyScalarFields verifies that the Where input type
// contains only scalar fields from the node, not relationship fields.
// For Movie with fields (id, title, released), MovieWhere should have those fields.
func TestAugmentSchema_WhereInputOnlyScalarFields(t *testing.T) {
	sdl, err := AugmentSchema(multiNodeModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if sdl == "" {
		t.Skip("AugmentSchema returned empty — skipping field check")
	}

	// Parse to inspect fields
	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Skipf("augmented schema failed parse: %v", parseErr)
	}

	movieWhere := s.Types["MovieWhere"]
	if movieWhere == nil {
		t.Fatal("MovieWhere type not found in parsed schema")
	}

	// Should have scalar fields but not relationship field "movies"
	fieldNames := map[string]bool{}
	for _, f := range movieWhere.Fields {
		fieldNames[f.Name] = true
	}
	if fieldNames["movies"] {
		t.Error("MovieWhere contains 'movies' field — relationship fields should be excluded")
	}
}

// TestAugmentSchema_EmptyModel verifies that augmenting an empty model does not error.
func TestAugmentSchema_EmptyModel(t *testing.T) {
	sdl, err := AugmentSchema(schema.GraphModel{})
	if err != nil {
		t.Fatalf("AugmentSchema returned error for empty model: %v", err)
	}
	// Empty model should return empty or minimal schema — no specific content required
	_ = sdl
}

// === CG-6: Nested mutation input types tests ===

// modelWithRelationshipProperties returns a GraphModel with Movie and Actor nodes,
// a relationship with @relationshipProperties (ActedInProperties with "role" field),
// and the relationship field is "actors" on Movie (direction IN from Actor).
func modelWithRelationshipProperties() schema.GraphModel {
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
			},
			{
				Name:   "Actor",
				Labels: []string{"Actor"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
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
						{Name: "role", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
					},
				},
			},
		},
	}
}

// modelWithRelationshipNoProperties returns a GraphModel with Movie and Category nodes,
// a relationship WITHOUT @relationshipProperties.
// The relationship field is "categories" on Movie (direction OUT to Category).
func modelWithRelationshipNoProperties() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
				},
			},
			{
				Name:   "Category",
				Labels: []string{"Category"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
					{Name: "categoryName", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
				},
			},
		},
		Relationships: []schema.RelationshipDefinition{
			{
				FieldName:  "categories",
				RelType:    "IN_CATEGORY",
				Direction:  schema.DirectionOUT,
				FromNode:   "Movie",
				ToNode:     "Category",
				Properties: nil, // no @relationshipProperties
			},
		},
	}
}

// TestAugmentSchema_NestedFieldInputType verifies that for a node with a
// @relationship field, the augmented schema contains the FieldInput type.
// Per spec: MovieActorsFieldInput { create: [...], connect: [...] }
// Expected: "MovieActorsFieldInput" present in output.
func TestAugmentSchema_NestedFieldInputType(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "MovieActorsFieldInput") {
		t.Errorf("augmented schema missing 'MovieActorsFieldInput':\n%s", sdl)
	}
}

// TestAugmentSchema_NestedCreateFieldInputType verifies that the
// CreateFieldInput type is generated for each relationship.
// Per spec: MovieActorsCreateFieldInput { node: ActorCreateInput!, edge: ActedInPropertiesCreateInput }
// Expected: "MovieActorsCreateFieldInput" present in output.
func TestAugmentSchema_NestedCreateFieldInputType(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "MovieActorsCreateFieldInput") {
		t.Errorf("augmented schema missing 'MovieActorsCreateFieldInput':\n%s", sdl)
	}
}

// TestAugmentSchema_NestedConnectFieldInputType verifies that the
// ConnectFieldInput type is generated for each relationship.
// Per spec: MovieActorsConnectFieldInput { where: ActorWhere, edge: ActedInPropertiesCreateInput }
// Expected: "MovieActorsConnectFieldInput" present in output.
func TestAugmentSchema_NestedConnectFieldInputType(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "MovieActorsConnectFieldInput") {
		t.Errorf("augmented schema missing 'MovieActorsConnectFieldInput':\n%s", sdl)
	}
}

// TestAugmentSchema_PropertiesCreateInputType verifies that for a relationship
// with @relationshipProperties, the PropertiesCreateInput type is generated.
// Per spec: ActedInPropertiesCreateInput { role: String! }
// Expected: "ActedInPropertiesCreateInput" present in output.
func TestAugmentSchema_PropertiesCreateInputType(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "ActedInPropertiesCreateInput") {
		t.Errorf("augmented schema missing 'ActedInPropertiesCreateInput':\n%s", sdl)
	}
}

// TestAugmentSchema_CreateInputIncludesRelationshipField verifies that
// MovieCreateInput includes a field for the relationship.
// Per spec: MovieCreateInput { title: String!, released: Int, actors: MovieActorsFieldInput }
// Expected: "actors" field referencing "MovieActorsFieldInput" in MovieCreateInput.
func TestAugmentSchema_CreateInputIncludesRelationshipField(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Parse to inspect MovieCreateInput fields
	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed gqlparser parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	createInput := s.Types["MovieCreateInput"]
	if createInput == nil {
		t.Fatal("MovieCreateInput type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range createInput.Fields {
		fieldNames[f.Name] = true
	}
	if !fieldNames["actors"] {
		t.Errorf("MovieCreateInput missing 'actors' field:\n%s", sdl)
	}
}

// TestAugmentSchema_FieldInputHasCreateAndConnect verifies that the FieldInput type
// has both 'create' and 'connect' array fields.
// Per spec: MovieActorsFieldInput { create: [MovieActorsCreateFieldInput!], connect: [MovieActorsConnectFieldInput!] }
// Expected: both 'create' and 'connect' fields present.
func TestAugmentSchema_FieldInputHasCreateAndConnect(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	fieldInput := s.Types["MovieActorsFieldInput"]
	if fieldInput == nil {
		t.Fatal("MovieActorsFieldInput type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range fieldInput.Fields {
		fieldNames[f.Name] = true
	}
	if !fieldNames["create"] {
		t.Error("MovieActorsFieldInput missing 'create' field")
	}
	if !fieldNames["connect"] {
		t.Error("MovieActorsFieldInput missing 'connect' field")
	}
}

// TestAugmentSchema_EdgeFieldAbsentWithoutProperties verifies that when a
// relationship has NO @relationshipProperties, the 'edge' field is absent
// from the CreateFieldInput and ConnectFieldInput types.
// Expected: MovieCategoriesCreateFieldInput has 'node' but NOT 'edge'.
func TestAugmentSchema_EdgeFieldAbsentWithoutProperties(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipNoProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// First check that the nested types exist at all
	if !strings.Contains(sdl, "MovieCategoriesCreateFieldInput") {
		t.Fatalf("augmented schema missing 'MovieCategoriesCreateFieldInput':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	createFieldInput := s.Types["MovieCategoriesCreateFieldInput"]
	if createFieldInput == nil {
		t.Fatal("MovieCategoriesCreateFieldInput not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range createFieldInput.Fields {
		fieldNames[f.Name] = true
	}
	if !fieldNames["node"] {
		t.Error("MovieCategoriesCreateFieldInput missing 'node' field")
	}
	if fieldNames["edge"] {
		t.Error("MovieCategoriesCreateFieldInput has 'edge' field — should be absent for relationships without @relationshipProperties")
	}
}

// TestAugmentSchema_EdgeFieldPresentWithProperties verifies that when a
// relationship HAS @relationshipProperties, the 'edge' field is present
// on both CreateFieldInput and ConnectFieldInput.
// Expected: MovieActorsCreateFieldInput has both 'node' and 'edge'.
func TestAugmentSchema_EdgeFieldPresentWithProperties(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	createFieldInput := s.Types["MovieActorsCreateFieldInput"]
	if createFieldInput == nil {
		t.Fatal("MovieActorsCreateFieldInput not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range createFieldInput.Fields {
		fieldNames[f.Name] = true
	}
	if !fieldNames["node"] {
		t.Error("MovieActorsCreateFieldInput missing 'node' field")
	}
	if !fieldNames["edge"] {
		t.Error("MovieActorsCreateFieldInput missing 'edge' field — should be present for relationships with @relationshipProperties")
	}
}

// TestAugmentSchema_PropertiesCreateInputGeneratedOnce verifies that a shared
// @relationshipProperties type generates its CreateInput only once,
// even if multiple relationships reference it.
// Expected: exactly one occurrence of "ActedInPropertiesCreateInput".
func TestAugmentSchema_PropertiesCreateInputGeneratedOnce(t *testing.T) {
	// Model with two relationships sharing the same properties type
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

	count := strings.Count(sdl, "input ActedInPropertiesCreateInput")
	if count != 1 {
		t.Errorf("'input ActedInPropertiesCreateInput' appears %d times, want exactly 1:\n%s", count, sdl)
	}
}

// TestAugmentSchema_NestedInputsValidSDL verifies that the augmented schema
// with nested input types is valid GraphQL SDL (parses without error).
// Expected: gqlparser.LoadSchema succeeds.
func TestAugmentSchema_NestedInputsValidSDL(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if sdl == "" {
		t.Skip("AugmentSchema returned empty — skipping parse test")
	}
	_, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema with nested inputs failed gqlparser validation: %v\nSDL:\n%s", parseErr, sdl)
	}
}

// TestAugmentSchema_NodeWithNoRelationships_NoNestedInputs verifies that
// nodes without any @relationship fields do NOT get nested input types.
// The MovieCreateInput should contain only scalar fields (no relationship fields).
// Expected: no FieldInput types for a standalone node.
func TestAugmentSchema_NodeWithNoRelationships_NoNestedInputs(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	// movieModel() has no relationships, so no nested types should appear
	if strings.Contains(sdl, "FieldInput") {
		t.Errorf("augmented schema should not contain 'FieldInput' for a model with no relationships:\n%s", sdl)
	}
}
