package schema

import (
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// --- ExtractCypherDirective tests ---

// TestExtractCypherDirective_WithDirective verifies that a field with @cypher(statement: "...")
// returns HasDirective=true and the statement string.
// Expected: HasDirective=true, Statement="MATCH (this)-[:ACTED_IN]->(m) RETURN m"
func TestExtractCypherDirective_WithDirective(t *testing.T) {
	dir := makeDirective("cypher",
		[2]string{"statement", "MATCH (this)-[:ACTED_IN]->(m) RETURN m"},
	)
	field := makeFieldDef("movies", "Movie", dir)

	info := ExtractCypherDirective(field)
	if !info.HasDirective {
		t.Error("ExtractCypherDirective returned HasDirective=false, want true")
	}
	if info.Statement != "MATCH (this)-[:ACTED_IN]->(m) RETURN m" {
		t.Errorf("Statement = %q, want %q", info.Statement, "MATCH (this)-[:ACTED_IN]->(m) RETURN m")
	}
}

// TestExtractCypherDirective_WithoutDirective verifies that a field without @cypher
// returns HasDirective=false.
func TestExtractCypherDirective_WithoutDirective(t *testing.T) {
	field := makeFieldDef("title", "String")

	info := ExtractCypherDirective(field)
	if info.HasDirective {
		t.Error("ExtractCypherDirective returned HasDirective=true for field without @cypher, want false")
	}
	if info.Statement != "" {
		t.Errorf("Statement = %q, want empty", info.Statement)
	}
}

// TestExtractCypherDirective_NilField verifies nil field returns HasDirective=false without panicking.
func TestExtractCypherDirective_NilField(t *testing.T) {
	info := ExtractCypherDirective(nil)
	if info.HasDirective {
		t.Error("ExtractCypherDirective returned HasDirective=true for nil field, want false")
	}
}

// --- BuiltinDirectiveDefs @cypher test ---

// TestBuiltinDirectiveDefs_ContainsCypherDirective verifies that BuiltinDirectiveDefs
// includes the @cypher directive definition.
func TestBuiltinDirectiveDefs_ContainsCypherDirective(t *testing.T) {
	defs := BuiltinDirectiveDefs()
	if !strings.Contains(defs, "directive @cypher") {
		t.Error("BuiltinDirectiveDefs() missing 'directive @cypher'")
	}
	if !strings.Contains(defs, "statement: String!") {
		t.Error("BuiltinDirectiveDefs() missing 'statement: String!' argument for @cypher")
	}
}

// --- ValidateDirectives @cypher tests ---

// TestValidateDirectives_CypherAndRelationshipMutualExclusivity verifies that a field with both
// @cypher and @relationship produces a validation error (mutual exclusivity).
func TestValidateDirectives_CypherAndRelationshipMutualExclusivity(t *testing.T) {
	cypherDir := makeDirective("cypher",
		[2]string{"statement", "MATCH (this)-[:ACTED_IN]->(m) RETURN m"},
	)
	relDir := makeDirective("relationship",
		[2]string{"type", "ACTED_IN"},
		[2]string{"direction", "OUT"},
	)
	nodeType := makeTypeDef("Actor", makeDirective("node"))
	nodeType.Fields = ast.FieldList{
		makeFieldDef("movies", "Movie", cypherDir, relDir),
	}

	doc := &ast.SchemaDocument{
		Definitions: []*ast.Definition{nodeType},
	}

	errs := ValidateDirectives(doc)
	if len(errs) == 0 {
		t.Fatal("ValidateDirectives returned 0 errors for field with both @cypher and @relationship, want >= 1")
	}
	// Verify the error mentions mutual exclusivity
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "cypher") && strings.Contains(e.Error(), "relationship") {
			found = true
		}
	}
	if !found {
		t.Errorf("error should mention both @cypher and @relationship: %v", errs)
	}
}

// TestValidateDirectives_CypherEmptyStatement verifies that @cypher with an empty statement
// produces a validation error.
func TestValidateDirectives_CypherEmptyStatement(t *testing.T) {
	cypherDir := makeDirective("cypher",
		[2]string{"statement", ""},
	)
	nodeType := makeTypeDef("Actor", makeDirective("node"))
	nodeType.Fields = ast.FieldList{
		makeFieldDef("movies", "Movie", cypherDir),
	}

	doc := &ast.SchemaDocument{
		Definitions: []*ast.Definition{nodeType},
	}

	errs := ValidateDirectives(doc)
	if len(errs) == 0 {
		t.Fatal("ValidateDirectives returned 0 errors for @cypher with empty statement, want >= 1")
	}
	// Verify the error mentions empty statement
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "statement") {
			found = true
		}
	}
	if !found {
		t.Errorf("error should mention empty statement: %v", errs)
	}
}

// TestValidateDirectives_CypherValidStatement verifies that @cypher with a valid statement
// produces no errors.
func TestValidateDirectives_CypherValidStatement(t *testing.T) {
	cypherDir := makeDirective("cypher",
		[2]string{"statement", "MATCH (this)-[:ACTED_IN]->(m) RETURN m"},
	)
	nodeType := makeTypeDef("Actor", makeDirective("node"))
	nodeType.Fields = ast.FieldList{
		makeFieldDef("name", "String"),
		makeFieldDef("movies", "Movie", cypherDir),
	}

	doc := &ast.SchemaDocument{
		Definitions: []*ast.Definition{nodeType},
	}

	errs := ValidateDirectives(doc)
	if len(errs) != 0 {
		t.Errorf("ValidateDirectives returned %d errors for valid @cypher, want 0: %v", len(errs), errs)
	}
}

// --- ParseSchemaString @cypher integration tests ---

// TestParseSchemaString_CypherField_Basic verifies that a @cypher field is parsed into
// CypherFieldDefinition and stored in NodeDefinition.CypherFields, separate from Fields.
// Expected: CypherFields has 1 entry, Fields does NOT contain the @cypher field.
func TestParseSchemaString_CypherField_Basic(t *testing.T) {
	sdl := `
		type Actor @node {
			name: String!
			movies: [Movie!]! @cypher(statement: "MATCH (this)-[:ACTED_IN]->(m:Movie) RETURN m")
		}
		type Movie @node {
			title: String!
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}

	// Find Actor node
	var actorNode *NodeDefinition
	for i := range model.Nodes {
		if model.Nodes[i].Name == "Actor" {
			actorNode = &model.Nodes[i]
			break
		}
	}
	if actorNode == nil {
		t.Fatal("Actor node not found")
	}

	// @cypher field should NOT be in Fields (scalar fields only)
	for _, f := range actorNode.Fields {
		if f.Name == "movies" {
			t.Error("@cypher field 'movies' should not appear in Fields (only scalar fields)")
		}
	}

	// @cypher field should be in CypherFields
	if len(actorNode.CypherFields) != 1 {
		t.Fatalf("len(CypherFields) = %d, want 1", len(actorNode.CypherFields))
	}
	cf := actorNode.CypherFields[0]
	if cf.Name != "movies" {
		t.Errorf("CypherFields[0].Name = %q, want %q", cf.Name, "movies")
	}
	if cf.Statement != "MATCH (this)-[:ACTED_IN]->(m:Movie) RETURN m" {
		t.Errorf("CypherFields[0].Statement = %q, want %q", cf.Statement, "MATCH (this)-[:ACTED_IN]->(m:Movie) RETURN m")
	}
}

// TestParseSchemaString_CypherField_ListReturnType verifies that a @cypher field returning
// a list type (e.g., [Movie!]!) has IsList=true.
func TestParseSchemaString_CypherField_ListReturnType(t *testing.T) {
	sdl := `
		type Actor @node {
			name: String!
			movies: [Movie!]! @cypher(statement: "MATCH (this)-[:ACTED_IN]->(m:Movie) RETURN m")
		}
		type Movie @node {
			title: String!
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}

	var actorNode *NodeDefinition
	for i := range model.Nodes {
		if model.Nodes[i].Name == "Actor" {
			actorNode = &model.Nodes[i]
			break
		}
	}
	if actorNode == nil {
		t.Fatal("Actor node not found")
	}
	if len(actorNode.CypherFields) == 0 {
		t.Fatal("CypherFields is empty")
	}
	if !actorNode.CypherFields[0].IsList {
		t.Error("CypherFields[0].IsList = false, want true for [Movie!]! return type")
	}
}

// TestParseSchemaString_CypherField_NullableScalarReturn verifies that a @cypher field
// returning a nullable scalar (e.g., Float) has Nullable=true.
func TestParseSchemaString_CypherField_NullableScalarReturn(t *testing.T) {
	sdl := `
		type Movie @node {
			title: String!
			averageRating: Float @cypher(statement: "MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.rating)")
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}

	if len(model.Nodes) == 0 {
		t.Fatal("no nodes found")
	}
	node := model.Nodes[0]
	if len(node.CypherFields) == 0 {
		t.Fatal("CypherFields is empty")
	}
	cf := node.CypherFields[0]
	if cf.Name != "averageRating" {
		t.Errorf("CypherFields[0].Name = %q, want %q", cf.Name, "averageRating")
	}
	if !cf.Nullable {
		t.Error("CypherFields[0].Nullable = false, want true for nullable Float return type")
	}
	if cf.IsList {
		t.Error("CypherFields[0].IsList = true, want false for scalar Float return type")
	}
}

// TestParseSchemaString_CypherField_WithArguments verifies that a @cypher field with
// GraphQL arguments extracts them as ArgumentDefinitions.
func TestParseSchemaString_CypherField_WithArguments(t *testing.T) {
	sdl := `
		type Movie @node {
			title: String!
			similarMovies(limit: Int! = 10): [Movie!]! @cypher(statement: "MATCH (this)-[:SIMILAR]->(m:Movie) RETURN m LIMIT $limit")
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}

	if len(model.Nodes) == 0 {
		t.Fatal("no nodes found")
	}
	node := model.Nodes[0]
	if len(node.CypherFields) == 0 {
		t.Fatal("CypherFields is empty")
	}
	cf := node.CypherFields[0]
	if cf.Name != "similarMovies" {
		t.Errorf("CypherFields[0].Name = %q, want %q", cf.Name, "similarMovies")
	}
	if len(cf.Arguments) != 1 {
		t.Fatalf("len(Arguments) = %d, want 1", len(cf.Arguments))
	}
	arg := cf.Arguments[0]
	if arg.Name != "limit" {
		t.Errorf("Arguments[0].Name = %q, want %q", arg.Name, "limit")
	}
	if arg.GraphQLType != "Int!" {
		t.Errorf("Arguments[0].GraphQLType = %q, want %q", arg.GraphQLType, "Int!")
	}
}

// TestParseSchemaString_CypherField_NoArguments verifies that a @cypher field with no
// arguments has an empty Arguments slice.
func TestParseSchemaString_CypherField_NoArguments(t *testing.T) {
	sdl := `
		type Movie @node {
			title: String!
			actorCount: Int! @cypher(statement: "MATCH (this)<-[:ACTED_IN]-(a) RETURN count(a)")
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}

	if len(model.Nodes) == 0 {
		t.Fatal("no nodes found")
	}
	node := model.Nodes[0]
	if len(node.CypherFields) == 0 {
		t.Fatal("CypherFields is empty")
	}
	cf := node.CypherFields[0]
	if len(cf.Arguments) != 0 {
		t.Errorf("len(Arguments) = %d, want 0 for @cypher with no args", len(cf.Arguments))
	}
}

// TestParseSchemaString_CypherField_ArgumentWithDefault verifies that a @cypher field
// argument with a default value has DefaultValue populated.
func TestParseSchemaString_CypherField_ArgumentWithDefault(t *testing.T) {
	sdl := `
		type Movie @node {
			title: String!
			similarMovies(limit: Int! = 10): [Movie!]! @cypher(statement: "MATCH (this)-[:SIMILAR]->(m:Movie) RETURN m LIMIT $limit")
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}

	if len(model.Nodes) == 0 {
		t.Fatal("no nodes found")
	}
	node := model.Nodes[0]
	if len(node.CypherFields) == 0 {
		t.Fatal("CypherFields is empty")
	}
	cf := node.CypherFields[0]
	if len(cf.Arguments) == 0 {
		t.Fatal("Arguments is empty")
	}
	arg := cf.Arguments[0]
	if arg.DefaultValue == nil {
		t.Error("Arguments[0].DefaultValue is nil, want non-nil for argument with default = 10")
	}
}

// TestParseSchemaString_CypherField_GoTypeMapping verifies that the GraphQL return type
// of a @cypher field is correctly mapped to a Go type.
func TestParseSchemaString_CypherField_GoTypeMapping(t *testing.T) {
	sdl := `
		type Movie @node {
			title: String!
			actorCount: Int! @cypher(statement: "MATCH (this)<-[:ACTED_IN]-(a) RETURN count(a)")
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}

	if len(model.Nodes) == 0 {
		t.Fatal("no nodes found")
	}
	node := model.Nodes[0]
	if len(node.CypherFields) == 0 {
		t.Fatal("CypherFields is empty")
	}
	cf := node.CypherFields[0]
	if cf.GoType != "int" {
		t.Errorf("CypherFields[0].GoType = %q, want %q for Int! return type", cf.GoType, "int")
	}
	if cf.GraphQLType != "Int!" {
		t.Errorf("CypherFields[0].GraphQLType = %q, want %q", cf.GraphQLType, "Int!")
	}
}

// TestParseSchemaString_CypherField_ExcludedFromScalarFields verifies that @cypher fields
// do not appear in NodeDefinition.Fields (they are computed, not stored).
func TestParseSchemaString_CypherField_ExcludedFromScalarFields(t *testing.T) {
	sdl := `
		type Movie @node {
			title: String!
			released: Int
			actorCount: Int! @cypher(statement: "MATCH (this)<-[:ACTED_IN]-(a) RETURN count(a)")
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}

	if len(model.Nodes) == 0 {
		t.Fatal("no nodes found")
	}
	node := model.Nodes[0]

	// Fields should only contain title and released (scalar/stored fields)
	if len(node.Fields) != 2 {
		t.Fatalf("len(Fields) = %d, want 2 (title, released — @cypher excluded)", len(node.Fields))
	}
	for _, f := range node.Fields {
		if f.Name == "actorCount" {
			t.Error("@cypher field 'actorCount' should not appear in Fields")
		}
	}
}

// TestParseSchemaString_CypherField_MultipleCypherFields verifies that a node with
// multiple @cypher fields captures all of them.
func TestParseSchemaString_CypherField_MultipleCypherFields(t *testing.T) {
	sdl := `
		type Movie @node {
			title: String!
			actorCount: Int! @cypher(statement: "MATCH (this)<-[:ACTED_IN]-(a) RETURN count(a)")
			averageRating: Float @cypher(statement: "MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.rating)")
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}

	if len(model.Nodes) == 0 {
		t.Fatal("no nodes found")
	}
	node := model.Nodes[0]
	if len(node.CypherFields) != 2 {
		t.Fatalf("len(CypherFields) = %d, want 2", len(node.CypherFields))
	}

	names := map[string]bool{}
	for _, cf := range node.CypherFields {
		names[cf.Name] = true
	}
	if !names["actorCount"] {
		t.Error("CypherFields missing 'actorCount'")
	}
	if !names["averageRating"] {
		t.Error("CypherFields missing 'averageRating'")
	}
}

// TestParseSchemaString_CypherField_MutualExclusivityError verifies that a field with
// both @cypher and @relationship produces a parse error.
func TestParseSchemaString_CypherField_MutualExclusivityError(t *testing.T) {
	sdl := `
		type Actor @node {
			name: String!
			movies: [Movie!]! @cypher(statement: "MATCH (this)-[:ACTED_IN]->(m) RETURN m") @relationship(type: "ACTED_IN", direction: OUT)
		}
		type Movie @node {
			title: String!
		}
	`

	_, err := ParseSchemaString(sdl)
	if err == nil {
		t.Fatal("ParseSchemaString returned nil error for field with both @cypher and @relationship, want error")
	}
}

// TestParseSchemaString_CypherField_EmptyStatementError verifies that @cypher with
// an empty statement produces a parse error.
func TestParseSchemaString_CypherField_EmptyStatementError(t *testing.T) {
	sdl := `
		type Movie @node {
			title: String!
			bad: Int @cypher(statement: "")
		}
	`

	_, err := ParseSchemaString(sdl)
	if err == nil {
		t.Fatal("ParseSchemaString returned nil error for @cypher with empty statement, want error")
	}
}

// TestParseSchemaString_NoCypherFields verifies that a node without @cypher fields
// has an empty CypherFields slice (not nil).
func TestParseSchemaString_NoCypherFields(t *testing.T) {
	sdl := `
		type Movie @node {
			title: String!
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}

	if len(model.Nodes) == 0 {
		t.Fatal("no nodes found")
	}
	node := model.Nodes[0]
	if len(node.CypherFields) != 0 {
		t.Errorf("len(CypherFields) = %d, want 0 for node without @cypher fields", len(node.CypherFields))
	}
}
