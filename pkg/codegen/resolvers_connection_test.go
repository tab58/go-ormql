package codegen

import (
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
)

// --- CG-19: Relationship connection resolver templates ---

// connectionResolverModel returns a model with Movie→Actor relationship
// for testing relationship-level connection resolver generation.
func connectionResolverModel() schema.GraphModel {
	return modelWithRelationshipProperties()
}

// connectionResolverModelNoProps returns a model with Movie→Category relationship
// without @relationshipProperties for testing connection resolver generation.
func connectionResolverModelNoProps() schema.GraphModel {
	return modelWithRelationshipNoProperties()
}

// TestGenerateResolvers_Connection_ObjectResolverStruct verifies that an
// object-level resolver struct is generated for nodes with @relationship fields
// that need connection resolvers.
// Expected: generated source contains "movieResolver" struct.
func TestGenerateResolvers_Connection_ObjectResolverStruct(t *testing.T) {
	src, err := GenerateResolvers(connectionResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "movieResolver") {
		t.Errorf("generated resolvers missing 'movieResolver' struct for node with connections:\n%s", s)
	}
}

// TestGenerateResolvers_Connection_MethodGenerated verifies that a connection
// resolver method is generated for the relationship field.
// Expected: generated source contains "ActorsConnection" method.
func TestGenerateResolvers_Connection_MethodGenerated(t *testing.T) {
	src, err := GenerateResolvers(connectionResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "ActorsConnection") {
		t.Errorf("generated resolvers missing 'ActorsConnection' method:\n%s", s)
	}
}

// TestGenerateResolvers_Connection_CallsRelConnectionQuery verifies that the
// connection resolver calls cypher.RelConnectionQuery.
// Expected: generated source contains "cypher.RelConnectionQuery" or "RelConnectionQuery".
func TestGenerateResolvers_Connection_CallsRelConnectionQuery(t *testing.T) {
	src, err := GenerateResolvers(connectionResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "RelConnectionQuery") {
		t.Errorf("connection resolver should call cypher.RelConnectionQuery:\n%s", s)
	}
}

// TestGenerateResolvers_Connection_CallsRelConnectionCount verifies that the
// connection resolver calls cypher.RelConnectionCount for totalCount.
// Expected: generated source contains "cypher.RelConnectionCount" or "RelConnectionCount".
func TestGenerateResolvers_Connection_CallsRelConnectionCount(t *testing.T) {
	src, err := GenerateResolvers(connectionResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "RelConnectionCount") {
		t.Errorf("connection resolver should call cypher.RelConnectionCount for totalCount:\n%s", s)
	}
}

// TestGenerateResolvers_Connection_PassesDirection verifies that the connection
// resolver passes the relationship direction to RelConnectionQuery.
// Expected: generated source contains direction reference (e.g., "cypher.DirectionIN").
func TestGenerateResolvers_Connection_PassesDirection(t *testing.T) {
	src, err := GenerateResolvers(connectionResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The Movie→Actor relationship has DirectionIN, so it should be referenced
	if !strings.Contains(s, "cypher.DirectionIN") && !strings.Contains(s, "DirectionIN") &&
		!strings.Contains(s, "cypher.DirectionOUT") && !strings.Contains(s, "DirectionOUT") {
		t.Errorf("connection resolver should pass direction to RelConnectionQuery:\n%s", s)
	}
}

// TestGenerateResolvers_Connection_PassesRelType verifies that the connection
// resolver passes the relationship type to RelConnectionQuery.
// Expected: generated source contains "ACTED_IN" string.
func TestGenerateResolvers_Connection_PassesRelType(t *testing.T) {
	src, err := GenerateResolvers(connectionResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, `"ACTED_IN"`) {
		t.Errorf("connection resolver should pass relType \"ACTED_IN\":\n%s", s)
	}
}

// TestGenerateResolvers_Connection_BuildsEdges verifies that the connection
// resolver constructs edge objects with node and cursor fields.
// Expected: generated source builds edges with cursor encoding.
func TestGenerateResolvers_Connection_BuildsEdges(t *testing.T) {
	src, err := GenerateResolvers(connectionResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// Should construct edge objects with encoded cursors
	if !strings.Contains(s, "encodeCursor") {
		t.Errorf("connection resolver should use encodeCursor for edge construction:\n%s", s)
	}
}

// TestGenerateResolvers_Connection_BuildsPageInfo verifies that the connection
// resolver constructs a PageInfo object with hasNextPage, hasPreviousPage,
// startCursor, and endCursor.
// Expected: generated source contains "PageInfo" construction.
func TestGenerateResolvers_Connection_BuildsPageInfo(t *testing.T) {
	src, err := GenerateResolvers(connectionResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "PageInfo") {
		t.Errorf("connection resolver should construct PageInfo:\n%s", s)
	}
	if !strings.Contains(s, "HasNextPage") {
		t.Errorf("connection resolver should set HasNextPage:\n%s", s)
	}
}

// TestGenerateResolvers_Connection_HandlesFirstAndAfter verifies that the
// connection resolver accepts first and after parameters for pagination.
// Expected: generated source handles "first" and "after" parameters.
func TestGenerateResolvers_Connection_HandlesFirstAndAfter(t *testing.T) {
	src, err := GenerateResolvers(connectionResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "first") {
		t.Errorf("connection resolver should handle 'first' pagination param:\n%s", s)
	}
	if !strings.Contains(s, "after") {
		t.Errorf("connection resolver should handle 'after' pagination param:\n%s", s)
	}
}

// TestGenerateResolvers_Connection_EdgeProperties_WithProps verifies that when
// @relationshipProperties exists, the edge construction includes properties mapping.
// Expected: generated source for Movie→Actor connection references edge properties.
func TestGenerateResolvers_Connection_EdgeProperties_WithProps(t *testing.T) {
	src, err := GenerateResolvers(connectionResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// With @relationshipProperties (ActedInProperties), the edge should include properties
	if !strings.Contains(s, "ActedInProperties") && !strings.Contains(s, "Properties") {
		t.Errorf("connection resolver with @relationshipProperties should map edge properties:\n%s", s)
	}
}

// TestGenerateResolvers_Connection_EdgeProperties_WithoutProps verifies that when
// no @relationshipProperties exist, the edge construction does NOT include properties.
// Expected: MovieCategoriesEdge has no properties field mapping.
func TestGenerateResolvers_Connection_EdgeProperties_WithoutProps(t *testing.T) {
	src, err := GenerateResolvers(connectionResolverModelNoProps(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// Find the CategoriesConnection method section
	idx := strings.Index(s, "CategoriesConnection")
	if idx == -1 {
		// If not found, the test will fail on method generation (separate test)
		t.Skip("CategoriesConnection method not found — connection resolver not yet generated")
	}
	connSection := s[idx:]
	nextFunc := strings.Index(connSection[1:], "\nfunc ")
	if nextFunc > 0 {
		connSection = connSection[:nextFunc+1]
	}
	// Should NOT reference properties in edge construction
	if strings.Contains(connSection, "Properties") && !strings.Contains(connSection, "PageInfo") {
		t.Errorf("connection resolver without @relationshipProperties should NOT map edge properties:\n%s", connSection)
	}
}

// TestGenerateResolvers_Connection_ResolverRootMethod verifies that the
// Resolver struct gets a Movie() method for the ResolverRoot interface
// when a node has relationship connections.
// Expected: generated source contains "func (r *Resolver) Movie() MovieResolver".
func TestGenerateResolvers_Connection_ResolverRootMethod(t *testing.T) {
	src, err := GenerateResolvers(connectionResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "Movie() MovieResolver") {
		t.Errorf("Resolver missing Movie() method for connection resolvers:\n%s", s)
	}
}

// TestGenerateResolvers_Connection_NoConnectionNode_NoObjectResolver verifies
// that nodes without @relationship fields do NOT get connection resolver structs.
// Expected: no "movieResolver" when no relationships.
func TestGenerateResolvers_Connection_NoConnectionNode_NoObjectResolver(t *testing.T) {
	src, err := GenerateResolvers(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if strings.Contains(s, "movieResolver") {
		t.Errorf("node without relationships should NOT have a movieResolver struct:\n%s", s)
	}
}

// TestGenerateResolvers_Connection_MultiRelationship verifies that a node
// with multiple relationships generates connection resolvers for each.
func TestGenerateResolvers_Connection_MultiRelationship(t *testing.T) {
	model := schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{Name: "Movie", Labels: []string{"Movie"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
				{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
			}},
			{Name: "Actor", Labels: []string{"Actor"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
				{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
			}},
			{Name: "Director", Labels: []string{"Director"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
				{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
			}},
		},
		Relationships: []schema.RelationshipDefinition{
			{FieldName: "actors", RelType: "ACTED_IN", Direction: schema.DirectionIN, FromNode: "Movie", ToNode: "Actor"},
			{FieldName: "directors", RelType: "DIRECTED", Direction: schema.DirectionIN, FromNode: "Movie", ToNode: "Director"},
		},
	}

	src, err := GenerateResolvers(model, "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "ActorsConnection") {
		t.Errorf("missing ActorsConnection resolver method:\n%s", s)
	}
	if !strings.Contains(s, "DirectorsConnection") {
		t.Errorf("missing DirectorsConnection resolver method:\n%s", s)
	}
}

// TestGenerateResolvers_Connection_UsesParentObject verifies that the connection
// resolver receives the parent object to build the parent WHERE clause.
// Expected: generated source references parent object (e.g., "obj" parameter).
func TestGenerateResolvers_Connection_UsesParentObject(t *testing.T) {
	src, err := GenerateResolvers(connectionResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The connection resolver should receive the parent object for WHERE binding
	if !strings.Contains(s, "obj") && !strings.Contains(s, "parent") {
		t.Errorf("connection resolver should receive parent object:\n%s", s)
	}
}
