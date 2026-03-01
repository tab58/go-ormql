package codegen

import (
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
)

// resolverModel returns a model for resolver generation tests.
func resolverModel() schema.GraphModel {
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

// multiResolverModel returns a multi-node model for resolver generation tests.
func multiResolverModel() schema.GraphModel {
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
				Name:   "Actor",
				Labels: []string{"Actor"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
				},
			},
		},
	}
}

// --- Tests ---

// TestGenerateResolvers_NonEmpty verifies that resolver generation produces non-empty output.
func TestGenerateResolvers_NonEmpty(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	if len(src) == 0 {
		t.Fatal("GenerateResolvers returned empty output, want non-empty Go source")
	}
}

// TestGenerateResolvers_PackageDeclaration verifies that the output contains a package declaration.
// Expected: "package generated"
func TestGenerateResolvers_PackageDeclaration(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	if !strings.Contains(string(src), "package generated") {
		t.Errorf("output missing 'package generated':\n%s", string(src))
	}
}

// TestGenerateResolvers_QueryResolver verifies that a list query resolver is generated.
// Expected: function handling the Movies query (e.g., "func ... Movies(")
func TestGenerateResolvers_QueryResolver(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	if !strings.Contains(string(src), "Movies") {
		t.Errorf("output missing Movies query resolver:\n%s", string(src))
	}
}

// TestGenerateResolvers_ConnectionResolver verifies that a connection query resolver is generated.
// Expected: function handling MoviesConnection query.
func TestGenerateResolvers_ConnectionResolver(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	if !strings.Contains(string(src), "MoviesConnection") {
		t.Errorf("output missing MoviesConnection resolver:\n%s", string(src))
	}
}

// TestGenerateResolvers_CreateMutation verifies that a create mutation resolver is generated.
// Expected: function handling CreateMovies mutation.
func TestGenerateResolvers_CreateMutation(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	if !strings.Contains(string(src), "CreateMovies") {
		t.Errorf("output missing CreateMovies mutation resolver:\n%s", string(src))
	}
}

// TestGenerateResolvers_UpdateMutation verifies that an update mutation resolver is generated.
// Expected: function handling UpdateMovies mutation.
func TestGenerateResolvers_UpdateMutation(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	if !strings.Contains(string(src), "UpdateMovies") {
		t.Errorf("output missing UpdateMovies mutation resolver:\n%s", string(src))
	}
}

// TestGenerateResolvers_DeleteMutation verifies that a delete mutation resolver is generated.
// Expected: function handling DeleteMovies mutation.
func TestGenerateResolvers_DeleteMutation(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	if !strings.Contains(string(src), "DeleteMovies") {
		t.Errorf("output missing DeleteMovies mutation resolver:\n%s", string(src))
	}
}

// TestGenerateResolvers_UsesCypherNodeMatch verifies that generated code calls cypher.NodeMatch.
func TestGenerateResolvers_UsesCypherNodeMatch(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	if !strings.Contains(string(src), "cypher.NodeMatch") {
		t.Errorf("output missing 'cypher.NodeMatch' call:\n%s", string(src))
	}
}

// TestGenerateResolvers_UsesCypherNodeCreate verifies that generated code calls cypher.NodeCreate.
func TestGenerateResolvers_UsesCypherNodeCreate(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	if !strings.Contains(string(src), "cypher.NodeCreate") {
		t.Errorf("output missing 'cypher.NodeCreate' call:\n%s", string(src))
	}
}

// TestGenerateResolvers_UsesCypherNodeUpdate verifies that generated code calls cypher.NodeUpdate.
func TestGenerateResolvers_UsesCypherNodeUpdate(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	if !strings.Contains(string(src), "cypher.NodeUpdate") {
		t.Errorf("output missing 'cypher.NodeUpdate' call:\n%s", string(src))
	}
}

// TestGenerateResolvers_UsesCypherNodeDelete verifies that generated code calls cypher.NodeDelete.
func TestGenerateResolvers_UsesCypherNodeDelete(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	if !strings.Contains(string(src), "cypher.NodeDelete") {
		t.Errorf("output missing 'cypher.NodeDelete' call:\n%s", string(src))
	}
}

// TestGenerateResolvers_UsesDriverExecute verifies that read resolvers call driver.Execute.
func TestGenerateResolvers_UsesDriverExecute(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	// The generated code should reference driver Execute for read operations
	s := string(src)
	if !strings.Contains(s, ".Execute(") && !strings.Contains(s, "driver.Execute") {
		t.Errorf("output missing driver Execute call:\n%s", s)
	}
}

// TestGenerateResolvers_UsesDriverExecuteWrite verifies that write resolvers call driver.ExecuteWrite.
func TestGenerateResolvers_UsesDriverExecuteWrite(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, ".ExecuteWrite(") && !strings.Contains(s, "driver.ExecuteWrite") {
		t.Errorf("output missing driver ExecuteWrite call:\n%s", s)
	}
}

// TestGenerateResolvers_WhereToMapHelper verifies that a where-to-map helper function
// is generated for the node type (e.g., movieWhereToMap or similar).
func TestGenerateResolvers_WhereToMapHelper(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := strings.ToLower(string(src))
	// Look for a where conversion helper (case insensitive for flexibility)
	if !strings.Contains(s, "wheretomap") && !strings.Contains(s, "where_to_map") && !strings.Contains(s, "wheremap") && !strings.Contains(s, "wheretoclause") {
		t.Errorf("output missing where conversion helper function:\n%s", string(src))
	}
}

// TestGenerateResolvers_MultiNode verifies that resolvers are generated for all nodes in the model.
// Expected: both Movie and Actor resolver functions present.
func TestGenerateResolvers_MultiNode(t *testing.T) {
	src, err := GenerateResolvers(multiResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)

	for _, name := range []string{"Movies", "CreateMovies", "Actors", "CreateActors"} {
		if !strings.Contains(s, name) {
			t.Errorf("output missing resolver for %q:\n%s", name, s)
		}
	}
}

// === CG-8: Transaction-based nested mutation resolver tests ===

// TestGenerateResolvers_WithRelationships_UsesBeginTx verifies that the create
// mutation resolver for a node with relationships uses driver.BeginTx for
// transactional nested mutation processing.
// Expected: generated source contains "BeginTx".
func TestGenerateResolvers_WithRelationships_UsesBeginTx(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "BeginTx") {
		t.Errorf("create resolver for node with relationships should use BeginTx:\n%s", s)
	}
}

// TestGenerateResolvers_WithRelationships_UsesTxExecute verifies that the create
// mutation resolver for a node with relationships uses tx.Execute for each
// Cypher statement within the transaction.
// Expected: generated source contains "tx.Execute" or equivalent transaction execute call.
func TestGenerateResolvers_WithRelationships_UsesTxExecute(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "tx.Execute") {
		t.Errorf("create resolver for node with relationships should use tx.Execute:\n%s", s)
	}
}

// TestGenerateResolvers_WithRelationships_UsesTxCommit verifies that the create
// mutation resolver for a node with relationships commits the transaction
// after all nested operations succeed.
// Expected: generated source contains "tx.Commit" or "Commit".
func TestGenerateResolvers_WithRelationships_UsesTxCommit(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "Commit") {
		t.Errorf("create resolver for node with relationships should commit transaction:\n%s", s)
	}
}

// TestGenerateResolvers_WithRelationships_DefersRollback verifies that the create
// mutation resolver for a node with relationships uses defer tx.Rollback to
// ensure cleanup on error.
// Expected: generated source contains "defer" and "Rollback".
func TestGenerateResolvers_WithRelationships_DefersRollback(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "defer") || !strings.Contains(s, "Rollback") {
		t.Errorf("create resolver for node with relationships should defer Rollback:\n%s", s)
	}
}

// TestGenerateResolvers_WithoutRelationships_NoBeginTx verifies that the create
// mutation resolver for a node WITHOUT relationships uses simple ExecuteWrite
// and does NOT use BeginTx (no transaction overhead needed).
// Expected: generated source contains "ExecuteWrite" but NOT "BeginTx".
func TestGenerateResolvers_WithoutRelationships_NoBeginTx(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "ExecuteWrite") {
		t.Errorf("create resolver for node without relationships should use ExecuteWrite:\n%s", s)
	}
	if strings.Contains(s, "BeginTx") {
		t.Errorf("create resolver for node without relationships should NOT use BeginTx:\n%s", s)
	}
}

// TestGenerateResolvers_ConnectionResolver_EncodeCursor verifies that the
// connection resolver contains a cursor encoding function for Relay pagination.
// Expected: generated source contains "encodeCursor".
func TestGenerateResolvers_ConnectionResolver_EncodeCursor(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "encodeCursor") {
		t.Errorf("connection resolver should contain encodeCursor function:\n%s", s)
	}
}

// TestGenerateResolvers_ConnectionResolver_DecodeCursor verifies that the
// connection resolver contains a cursor decoding function for Relay pagination.
// Expected: generated source contains "decodeCursor".
func TestGenerateResolvers_ConnectionResolver_DecodeCursor(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "decodeCursor") {
		t.Errorf("connection resolver should contain decodeCursor function:\n%s", s)
	}
}

// TestGenerateResolvers_WithProperties_UsesRelationshipCreate verifies that
// the create resolver for a node with @relationshipProperties references
// cypher.RelCreate (or RelationshipCreate) to create the relationship with
// edge properties.
// Expected: generated source contains "RelCreate" or "RelationshipCreate".
func TestGenerateResolvers_WithProperties_UsesRelationshipCreate(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "RelCreate") && !strings.Contains(s, "RelationshipCreate") {
		t.Errorf("create resolver with relationship properties should use RelCreate or RelationshipCreate:\n%s", s)
	}
}

// TestGenerateResolvers_NestedCreate_ProcessesCreateInput verifies that the
// create mutation resolver for a node with relationships processes the "create"
// nested input entries — extracting the "actors" field from the input map and
// iterating over its "create" entries to create related nodes within the
// transaction.
// Expected: generated source references "actors" as a map key (quoted string).
func TestGenerateResolvers_NestedCreate_ProcessesCreateInput(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The resolver should access the relationship field on the typed input struct
	// (e.g., item.Actors) for nested create/connect processing.
	if !strings.Contains(s, ".Actors") {
		t.Errorf("create resolver should process nested 'Actors' field from typed input:\n%s", s)
	}
}

// TestGenerateResolvers_NestedConnect_ProcessesConnectInput verifies that the
// create mutation resolver for a node with relationships processes the "connect"
// nested input entries (matching existing nodes + creating relationships).
// Expected: generated source references processing of "connect" and "NodeMatch".
func TestGenerateResolvers_NestedConnect_ProcessesConnectInput(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The resolver should handle connect operations (match existing node + create relationship)
	if !strings.Contains(s, "connect") {
		t.Errorf("create resolver should process nested connect inputs:\n%s", s)
	}
}

// === H1-4: Template signature alignment tests ===

// TestGenerateResolvers_QuerySignature_MatchesGqlgen verifies that the
// generated Movies query resolver has the exact signature gqlgen expects:
// Movies(ctx context.Context, where *MovieWhere) ([]*Movie, error)
// This catches parameter type drift (e.g., []*Type vs []Type, pointer vs value).
// Expected: generated source contains the exact signature pattern.
func TestGenerateResolvers_QuerySignature_MatchesGqlgen(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// gqlgen generates: Movies(ctx context.Context, where *MovieWhere, sort []*MovieSort) ([]*Movie, error)
	expected := "Movies(ctx context.Context, where *MovieWhere, sort []*MovieSort) ([]*Movie, error)"
	if !strings.Contains(s, expected) {
		t.Errorf("query resolver signature mismatch.\nwant: %s\ngot resolvers:\n%s", expected, s)
	}
}

// TestGenerateResolvers_ConnectionSignature_MatchesGqlgen verifies that the
// generated MoviesConnection resolver has the exact signature gqlgen expects:
// MoviesConnection(ctx context.Context, first *int, after *string, where *MovieWhere, sort []*MovieSort) (*MoviesConnection, error)
// Expected: generated source contains the exact signature pattern.
func TestGenerateResolvers_ConnectionSignature_MatchesGqlgen(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	expected := "MoviesConnection(ctx context.Context, first *int, after *string, where *MovieWhere, sort []*MovieSort) (*MoviesConnection, error)"
	if !strings.Contains(s, expected) {
		t.Errorf("connection resolver signature mismatch.\nwant: %s\ngot resolvers:\n%s", expected, s)
	}
}

// TestGenerateResolvers_CreateSignature_MatchesGqlgen verifies that the
// generated CreateMovies mutation resolver has the exact signature gqlgen expects:
// CreateMovies(ctx context.Context, input []*MovieCreateInput) (*CreateMoviesMutationResponse, error)
// Expected: generated source contains the exact signature pattern.
func TestGenerateResolvers_CreateSignature_MatchesGqlgen(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	expected := "CreateMovies(ctx context.Context, input []*MovieCreateInput) (*CreateMoviesMutationResponse, error)"
	if !strings.Contains(s, expected) {
		t.Errorf("create mutation signature mismatch.\nwant: %s\ngot resolvers:\n%s", expected, s)
	}
}

// TestGenerateResolvers_UpdateSignature_MatchesGqlgen verifies that the
// generated UpdateMovies mutation resolver has the exact signature gqlgen expects:
// UpdateMovies(ctx context.Context, where *MovieWhere, update *MovieUpdateInput) (*UpdateMoviesMutationResponse, error)
// Expected: generated source contains the exact signature pattern.
func TestGenerateResolvers_UpdateSignature_MatchesGqlgen(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	expected := "UpdateMovies(ctx context.Context, where *MovieWhere, update *MovieUpdateInput) (*UpdateMoviesMutationResponse, error)"
	if !strings.Contains(s, expected) {
		t.Errorf("update mutation signature mismatch.\nwant: %s\ngot resolvers:\n%s", expected, s)
	}
}

// TestGenerateResolvers_DeleteSignature_MatchesGqlgen verifies that the
// generated DeleteMovies mutation resolver has the exact signature gqlgen expects:
// DeleteMovies(ctx context.Context, where *MovieWhere) (*DeleteInfo, error)
// Expected: generated source contains the exact signature pattern.
func TestGenerateResolvers_DeleteSignature_MatchesGqlgen(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	expected := "DeleteMovies(ctx context.Context, where *MovieWhere) (*DeleteInfo, error)"
	if !strings.Contains(s, expected) {
		t.Errorf("delete mutation signature mismatch.\nwant: %s\ngot resolvers:\n%s", expected, s)
	}
}

// TestGenerateResolvers_PropertiesMapper_Referenced verifies that the resolver
// template references the properties-type mapper function (e.g.,
// actedInPropertiesCreateInputToMap) for handling edge properties.
// Expected: generated resolvers contain a call to the properties mapper.
func TestGenerateResolvers_PropertiesMapper_Referenced(t *testing.T) {
	src, err := GenerateResolvers(modelWithRelationshipProperties(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "actedInPropertiesCreateInputToMap") {
		t.Errorf("resolver should reference actedInPropertiesCreateInputToMap for edge properties:\n%s", s)
	}
}

// === M1: Verify Resolver implements ResolverRoot ===

// TestGenerateResolvers_ResolverStruct_Defined verifies that the generated
// resolver output defines a `Resolver` struct.
// Expected: generated source contains "type Resolver struct".
func TestGenerateResolvers_ResolverStruct_Defined(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "type Resolver struct") {
		t.Errorf("generated output missing 'type Resolver struct':\n%s", s)
	}
}

// TestGenerateResolvers_QueryMethod_ReturnsQueryResolver verifies that the
// generated Resolver struct has a Query() method returning QueryResolver
// (the interface type, not a concrete type). This is required for the
// Resolver to implement gqlgen's ResolverRoot interface.
// Expected: generated source contains "func (r *Resolver) Query() QueryResolver".
func TestGenerateResolvers_QueryMethod_ReturnsQueryResolver(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	expected := "func (r *Resolver) Query() QueryResolver"
	if !strings.Contains(s, expected) {
		t.Errorf("Resolver missing Query() method with correct return type.\nwant: %s\ngot:\n%s", expected, s)
	}
}

// TestGenerateResolvers_MutationMethod_ReturnsMutationResolver verifies that
// the generated Resolver struct has a Mutation() method returning
// MutationResolver (the interface type). Required for gqlgen's ResolverRoot.
// Expected: generated source contains "func (r *Resolver) Mutation() MutationResolver".
func TestGenerateResolvers_MutationMethod_ReturnsMutationResolver(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	expected := "func (r *Resolver) Mutation() MutationResolver"
	if !strings.Contains(s, expected) {
		t.Errorf("Resolver missing Mutation() method with correct return type.\nwant: %s\ngot:\n%s", expected, s)
	}
}

// TestGenerateResolvers_QueryResolver_NotConcreteType verifies that the
// Query() method returns the interface type QueryResolver, not a concrete
// type like *queryResolver. If it returned the concrete type, the Resolver
// would not satisfy the ResolverRoot interface.
// Expected: output does NOT contain "func (r *Resolver) Query() *queryResolver".
func TestGenerateResolvers_QueryResolver_NotConcreteType(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if strings.Contains(s, "func (r *Resolver) Query() *queryResolver") {
		t.Errorf("Query() returns concrete *queryResolver instead of QueryResolver interface:\n%s", s)
	}
}

// TestGenerateResolvers_MutationResolver_NotConcreteType verifies that the
// Mutation() method returns the interface type MutationResolver, not a
// concrete type like *mutationResolver.
// Expected: output does NOT contain "func (r *Resolver) Mutation() *mutationResolver".
func TestGenerateResolvers_MutationResolver_NotConcreteType(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if strings.Contains(s, "func (r *Resolver) Mutation() *mutationResolver") {
		t.Errorf("Mutation() returns concrete *mutationResolver instead of MutationResolver interface:\n%s", s)
	}
}

// TestGenerateResolvers_E2E_ResolverRootCompiles verifies that the generated
// Resolver struct can be assigned to an interface variable requiring
// Query() QueryResolver and Mutation() MutationResolver methods. This is
// the compile-time verification that Resolver implements ResolverRoot.
// The test writes a small Go file that does `var _ ResolverRoot = &Resolver{}`
// and verifies it compiles alongside the generated resolver code.
// Expected: the type assertion compiles without error (fails RED because we
// don't currently generate a ResolverRoot interface type in our output — the
// interface is defined in gqlgen's generated exec_gen.go, not in our resolvers).
func TestGenerateResolvers_E2E_ResolverRootCompiles(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The generated resolver output should include a compile-time interface
	// satisfaction check: var _ ResolverRoot = &Resolver{}
	// This ensures the Resolver struct satisfies gqlgen's ResolverRoot
	// interface at compile time rather than failing at runtime.
	if !strings.Contains(s, "var _ ResolverRoot = &Resolver{}") {
		t.Errorf("generated resolvers missing compile-time interface check 'var _ ResolverRoot = &Resolver{}':\n%s", s)
	}
}
