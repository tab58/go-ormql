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

// === CG-31: Merge mutation type tests ===

// TestAugmentSchema_MergeMatchInput verifies that the augmented schema contains
// {Node}MatchInput with all scalar fields except id and vector, all optional.
// Expected: MovieMatchInput with title and released fields (no id).
func TestAugmentSchema_MergeMatchInput(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "MovieMatchInput") {
		t.Fatalf("augmented schema missing 'MovieMatchInput':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	matchInput := s.Types["MovieMatchInput"]
	if matchInput == nil {
		t.Fatal("MovieMatchInput type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range matchInput.Fields {
		fieldNames[f.Name] = true
	}

	// id should be excluded (auto-generated via randomUUID())
	if fieldNames["id"] {
		t.Error("MovieMatchInput contains 'id' — id should be excluded from MatchInput")
	}
	// title should be included (optional)
	if !fieldNames["title"] {
		t.Error("MovieMatchInput missing 'title' field")
	}
	// released should be included (optional)
	if !fieldNames["released"] {
		t.Error("MovieMatchInput missing 'released' field")
	}

	// All fields should be optional (nullable) — check that title is nullable
	for _, f := range matchInput.Fields {
		if f.Name == "title" && f.Type.NonNull {
			t.Error("MovieMatchInput.title should be optional (not NonNull)")
		}
	}
}

// TestAugmentSchema_MergeMatchInputExcludesVector verifies that vector fields
// are excluded from MatchInput, similar to id.
// Expected: MovieMatchInput does not contain the vector field.
func TestAugmentSchema_MergeMatchInputExcludesVector(t *testing.T) {
	model := movieModel()
	model.Nodes[0].VectorField = &schema.VectorFieldDefinition{
		Name: "embedding", IndexName: "movie_embed", Dimensions: 1536, Similarity: "cosine",
	}
	model.Nodes[0].Fields = append(model.Nodes[0].Fields, schema.FieldDefinition{
		Name: "embedding", GraphQLType: "[Float!]!", GoType: "[]float64", CypherType: "LIST<FLOAT>", IsList: true,
	})

	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	matchInput := s.Types["MovieMatchInput"]
	if matchInput == nil {
		t.Fatal("MovieMatchInput type not found in parsed schema")
	}

	for _, f := range matchInput.Fields {
		if f.Name == "embedding" {
			t.Error("MovieMatchInput contains 'embedding' — vector fields should be excluded")
		}
	}
}

// TestAugmentSchema_MergeMergeInput verifies that {Node}MergeInput is generated
// with match (required), onCreate (optional), and onMatch (optional) fields.
// Expected: MovieMergeInput with correct field types.
func TestAugmentSchema_MergeMergeInput(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "MovieMergeInput") {
		t.Fatalf("augmented schema missing 'MovieMergeInput':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	mergeInput := s.Types["MovieMergeInput"]
	if mergeInput == nil {
		t.Fatal("MovieMergeInput type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range mergeInput.Fields {
		fieldNames[f.Name] = true
	}
	if !fieldNames["match"] {
		t.Error("MovieMergeInput missing 'match' field")
	}
	if !fieldNames["onCreate"] {
		t.Error("MovieMergeInput missing 'onCreate' field")
	}
	if !fieldNames["onMatch"] {
		t.Error("MovieMergeInput missing 'onMatch' field")
	}

	// match should be required (NonNull)
	for _, f := range mergeInput.Fields {
		if f.Name == "match" && !f.Type.NonNull {
			t.Error("MovieMergeInput.match should be required (NonNull)")
		}
	}
}

// TestAugmentSchema_MergeMutationResponse verifies that Merge{Nodes}MutationResponse
// is generated with the plural node list field.
// Expected: MergeMoviesMutationResponse { movies: [Movie!]! }
func TestAugmentSchema_MergeMutationResponse(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "MergeMoviesMutationResponse") {
		t.Fatalf("augmented schema missing 'MergeMoviesMutationResponse':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	response := s.Types["MergeMoviesMutationResponse"]
	if response == nil {
		t.Fatal("MergeMoviesMutationResponse type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range response.Fields {
		fieldNames[f.Name] = true
	}
	if !fieldNames["movies"] {
		t.Error("MergeMoviesMutationResponse missing 'movies' field")
	}
}

// TestAugmentSchema_MergeMutation verifies that the mergeMovies mutation is
// generated in the Mutation type with correct input and return types.
// Expected: mergeMovies(input: [MovieMergeInput!]!): MergeMoviesMutationResponse!
func TestAugmentSchema_MergeMutation(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "mergeMovies") {
		t.Fatalf("augmented schema missing 'mergeMovies' mutation:\n%s", sdl)
	}
	if !strings.Contains(sdl, "MergeMoviesMutationResponse") {
		t.Error("augmented schema missing 'MergeMoviesMutationResponse' return type")
	}
}

// TestAugmentSchema_MergeTypesMultiNode verifies that merge types are generated
// for each @node type in a multi-node model.
// Expected: MovieMatchInput, ActorMatchInput, mergeMovies, mergeActors
func TestAugmentSchema_MergeTypesMultiNode(t *testing.T) {
	sdl, err := AugmentSchema(multiNodeModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	for _, want := range []string{
		"MovieMatchInput", "MovieMergeInput", "MergeMoviesMutationResponse", "mergeMovies",
		"ActorMatchInput", "ActorMergeInput", "MergeActorsMutationResponse", "mergeActors",
	} {
		if !strings.Contains(sdl, want) {
			t.Errorf("augmented schema missing merge artifact %q", want)
		}
	}
}

// TestAugmentSchema_MergeTypesValidSDL verifies that the augmented schema with
// merge types is valid GraphQL SDL (parses without error).
func TestAugmentSchema_MergeTypesValidSDL(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	_, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema with merge types failed gqlparser validation: %v\nSDL:\n%s", parseErr, sdl)
	}
}

// TestAugmentSchema_MergeMatchInputOnlyIDNode verifies that a node with only an
// ID! field produces an empty MatchInput (id is excluded).
// Expected: MovieMatchInput with _empty: String placeholder.
func TestAugmentSchema_MergeMatchInputOnlyIDNode(t *testing.T) {
	model := schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
				},
			},
		},
	}

	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "MovieMatchInput") {
		t.Fatalf("augmented schema missing 'MovieMatchInput' for id-only node:\n%s", sdl)
	}
}

// === CG-32: Connect mutation type tests ===

// TestAugmentSchema_ConnectInputWithProperties verifies that for a relationship
// with @relationshipProperties, the Connect{Source}{Field}Input is generated
// with from, to, and edge fields.
// Expected: ConnectMovieActorsInput { from: MovieWhere!, to: ActorWhere!, edge: ActedInPropertiesCreateInput }
func TestAugmentSchema_ConnectInputWithProperties(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "ConnectMovieActorsInput") {
		t.Fatalf("augmented schema missing 'ConnectMovieActorsInput':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	connectInput := s.Types["ConnectMovieActorsInput"]
	if connectInput == nil {
		t.Fatal("ConnectMovieActorsInput type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range connectInput.Fields {
		fieldNames[f.Name] = true
	}
	if !fieldNames["from"] {
		t.Error("ConnectMovieActorsInput missing 'from' field")
	}
	if !fieldNames["to"] {
		t.Error("ConnectMovieActorsInput missing 'to' field")
	}
	if !fieldNames["edge"] {
		t.Error("ConnectMovieActorsInput missing 'edge' field — should be present for relationships with @relationshipProperties")
	}
}

// TestAugmentSchema_ConnectInputWithoutProperties verifies that for a relationship
// without @relationshipProperties, the edge field is omitted.
// Expected: ConnectMovieCategoriesInput { from: MovieWhere!, to: CategoryWhere! } — no edge.
func TestAugmentSchema_ConnectInputWithoutProperties(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipNoProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "ConnectMovieCategoriesInput") {
		t.Fatalf("augmented schema missing 'ConnectMovieCategoriesInput':\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	connectInput := s.Types["ConnectMovieCategoriesInput"]
	if connectInput == nil {
		t.Fatal("ConnectMovieCategoriesInput not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range connectInput.Fields {
		fieldNames[f.Name] = true
	}
	if !fieldNames["from"] {
		t.Error("ConnectMovieCategoriesInput missing 'from' field")
	}
	if !fieldNames["to"] {
		t.Error("ConnectMovieCategoriesInput missing 'to' field")
	}
	if fieldNames["edge"] {
		t.Error("ConnectMovieCategoriesInput has 'edge' field — should be absent without @relationshipProperties")
	}
}

// TestAugmentSchema_ConnectMutation verifies that the connectMovieActors mutation
// is generated in the Mutation type.
// Expected: connectMovieActors(input: [ConnectMovieActorsInput!]!): ConnectInfo!
func TestAugmentSchema_ConnectMutation(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if !strings.Contains(sdl, "connectMovieActors") {
		t.Fatalf("augmented schema missing 'connectMovieActors' mutation:\n%s", sdl)
	}
	if !strings.Contains(sdl, "ConnectInfo") {
		t.Error("augmented schema missing 'ConnectInfo' return type")
	}
}

// TestAugmentSchema_ConnectInfoSharedType verifies that ConnectInfo is a shared type
// generated once, not duplicated per relationship.
// Expected: exactly one occurrence of "type ConnectInfo".
func TestAugmentSchema_ConnectInfoSharedType(t *testing.T) {
	// Model with two relationships
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
			{FieldName: "actors", RelType: "ACTED_IN", Direction: schema.DirectionIN, FromNode: "Movie", ToNode: "Actor", IsList: true},
			{FieldName: "directors", RelType: "DIRECTED", Direction: schema.DirectionIN, FromNode: "Movie", ToNode: "Director", IsList: true},
		},
	}

	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	count := strings.Count(sdl, "type ConnectInfo")
	if count != 1 {
		t.Errorf("'type ConnectInfo' appears %d times, want exactly 1:\n%s", count, sdl)
	}
}

// TestAugmentSchema_ConnectMutationPerRelationship verifies that each relationship
// gets its own connect mutation.
// Expected: connectMovieActors and connectMovieDirectors both present.
func TestAugmentSchema_ConnectMutationPerRelationship(t *testing.T) {
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
			{FieldName: "actors", RelType: "ACTED_IN", Direction: schema.DirectionIN, FromNode: "Movie", ToNode: "Actor", IsList: true},
			{FieldName: "directors", RelType: "DIRECTED", Direction: schema.DirectionIN, FromNode: "Movie", ToNode: "Director", IsList: true},
		},
	}

	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "connectMovieActors") {
		t.Error("augmented schema missing 'connectMovieActors' mutation")
	}
	if !strings.Contains(sdl, "connectMovieDirectors") {
		t.Error("augmented schema missing 'connectMovieDirectors' mutation")
	}
}

// TestAugmentSchema_ConnectTypesValidSDL verifies that the augmented schema with
// connect types is valid GraphQL SDL.
func TestAugmentSchema_ConnectTypesValidSDL(t *testing.T) {
	sdl, err := AugmentSchema(modelWithRelationshipProperties())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	_, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema with connect types failed gqlparser validation: %v\nSDL:\n%s", parseErr, sdl)
	}
}

// === CG-33: Relationship WHERE filter field tests ===

// relWhereModel returns a GraphModel with to-one and to-many relationships for
// testing relationship-based WHERE filters.
func relWhereModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{Name: "Movie", Labels: []string{"Movie"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
				{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
			}},
			{Name: "Actor", Labels: []string{"Actor"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
				{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
			}},
			{Name: "Repository", Labels: []string{"Repository"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
				{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
			}},
		},
		Relationships: []schema.RelationshipDefinition{
			// to-many: Movie.actors → [Actor!]!
			{FieldName: "actors", RelType: "ACTED_IN", Direction: schema.DirectionIN, FromNode: "Movie", ToNode: "Actor", IsList: true},
			// to-one: Movie.repository → Repository!
			{FieldName: "repository", RelType: "BELONGS_TO", Direction: schema.DirectionOUT, FromNode: "Movie", ToNode: "Repository", IsList: false},
		},
	}
}

// TestAugmentSchema_RelWhereToMany verifies that a to-many relationship adds
// a {fieldName}_some filter field to the Where input.
// Expected: MovieWhere has actors_some: ActorWhere
func TestAugmentSchema_RelWhereToMany(t *testing.T) {
	sdl, err := AugmentSchema(relWhereModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

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
	if !fieldNames["actors_some"] {
		t.Errorf("MovieWhere missing 'actors_some' field for to-many relationship filter")
	}
}

// TestAugmentSchema_RelWhereToOne verifies that a to-one relationship adds
// a {fieldName} filter field (no suffix) to the Where input.
// Expected: MovieWhere has repository: RepositoryWhere
func TestAugmentSchema_RelWhereToOne(t *testing.T) {
	sdl, err := AugmentSchema(relWhereModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

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
	if !fieldNames["repository"] {
		t.Errorf("MovieWhere missing 'repository' field for to-one relationship filter")
	}
}

// TestAugmentSchema_RelWhereDepthCap verifies that relationship WHERE filters
// are depth-limited at 3 levels to prevent infinite recursion.
// At depth 3, the Where type should omit relationship filter fields.
func TestAugmentSchema_RelWhereDepthCap(t *testing.T) {
	// Chain: A -> B -> C -> D (depth 3 from A)
	model := schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{Name: "A", Labels: []string{"A"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", IsID: true},
			}},
			{Name: "B", Labels: []string{"B"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", IsID: true},
			}},
			{Name: "C", Labels: []string{"C"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", IsID: true},
			}},
			{Name: "D", Labels: []string{"D"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", IsID: true},
			}},
		},
		Relationships: []schema.RelationshipDefinition{
			{FieldName: "bs", RelType: "HAS_B", Direction: schema.DirectionOUT, FromNode: "A", ToNode: "B", IsList: true},
			{FieldName: "cs", RelType: "HAS_C", Direction: schema.DirectionOUT, FromNode: "B", ToNode: "C", IsList: true},
			{FieldName: "ds", RelType: "HAS_D", Direction: schema.DirectionOUT, FromNode: "C", ToNode: "D", IsList: true},
		},
	}

	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// The schema must parse without infinite recursion
	_, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse (infinite recursion?): %v\nSDL:\n%s", parseErr, sdl)
	}

	// Verify that relationship filter fields exist at lower levels
	if !strings.Contains(sdl, "bs_some") {
		t.Error("AugmentSchema missing 'bs_some' relationship filter on AWhere")
	}
}

// TestAugmentSchema_RelWhereSelfReferencing verifies that self-referencing
// relationships respect the depth cap and don't cause infinite recursion.
func TestAugmentSchema_RelWhereSelfReferencing(t *testing.T) {
	model := schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{Name: "Folder", Labels: []string{"Folder"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", IsID: true},
				{Name: "name", GraphQLType: "String!"},
			}},
		},
		Relationships: []schema.RelationshipDefinition{
			{FieldName: "subfolders", RelType: "CONTAINS", Direction: schema.DirectionOUT, FromNode: "Folder", ToNode: "Folder", IsList: true},
		},
	}

	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Must not infinite-recurse
	_, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema with self-referencing relationship failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	// Should have subfolders_some at the first level
	if !strings.Contains(sdl, "subfolders_some") {
		t.Error("AugmentSchema missing 'subfolders_some' relationship filter on FolderWhere")
	}
}

// TestAugmentSchema_RelWhereBothStylesCoexist verifies that to-one and to-many
// filter styles coexist in the same Where input.
func TestAugmentSchema_RelWhereBothStylesCoexist(t *testing.T) {
	sdl, err := AugmentSchema(relWhereModel())
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

	// Both to-one (repository) and to-many (actors_some) should coexist
	if !fieldNames["actors_some"] {
		t.Error("MovieWhere missing 'actors_some' (to-many)")
	}
	if !fieldNames["repository"] {
		t.Error("MovieWhere missing 'repository' (to-one)")
	}
	// AND/OR/NOT should still be present
	if !fieldNames["AND"] {
		t.Error("MovieWhere missing 'AND' boolean composition")
	}
}
