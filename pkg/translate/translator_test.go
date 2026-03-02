package translate

import (
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// --- Test Fixtures ---

// testModel returns a minimal GraphModel for testing with Movie and Actor nodes
// connected by an ACTED_IN relationship with properties.
func testModel() schema.GraphModel {
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
					{Name: "born", GraphQLType: "Int", GoType: "*int", CypherType: "INTEGER", Nullable: true},
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
	}
}

// --- TR-1: Translator types + paramScope + New() ---

// Test: New() stores the model and returns a non-nil Translator.
func TestNew_ReturnsNonNilTranslator(t *testing.T) {
	model := testModel()
	tr := New(model)
	if tr == nil {
		t.Fatal("New() returned nil")
	}
}

// Test: New() stores the model so it can be used during translation.
// The translator should retain the model's nodes for lookup.
func TestNew_StoresModel(t *testing.T) {
	model := testModel()
	tr := New(model)
	// Verify the model is stored by checking the translator has access to it.
	if len(tr.model.Nodes) != 2 {
		t.Fatalf("expected model to have 2 nodes, got %d", len(tr.model.Nodes))
	}
	if tr.model.Nodes[0].Name != "Movie" {
		t.Errorf("expected first node to be Movie, got %s", tr.model.Nodes[0].Name)
	}
}

// Test: Translate() returns error for subscription operations (not supported).
func TestTranslate_SubscriptionReturnsError(t *testing.T) {
	model := testModel()
	tr := New(model)
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			{Operation: ast.Subscription, Name: "OnMovieCreated"},
		},
	}
	op := doc.Operations[0]

	_, err := tr.Translate(doc, op, nil)
	if err == nil {
		t.Fatal("expected error for subscription operation, got nil")
	}
}

// Test: Translate() does not return error for query operations.
func TestTranslate_QueryDoesNotError(t *testing.T) {
	model := testModel()
	tr := New(model)
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			{Operation: ast.Query, Name: "GetMovies"},
		},
	}
	op := doc.Operations[0]

	_, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error for query operation: %v", err)
	}
}

// Test: Translate() does not return error for mutation operations.
func TestTranslate_MutationDoesNotError(t *testing.T) {
	model := testModel()
	tr := New(model)
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			{Operation: ast.Mutation, Name: "CreateMovies"},
		},
	}
	op := doc.Operations[0]

	_, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error for mutation operation: %v", err)
	}
}

// --- paramScope tests ---

// Test: newParamScope creates a root scope with empty prefix and zero counter.
func TestNewParamScope_RootScope(t *testing.T) {
	s := newParamScope()
	if s == nil {
		t.Fatal("newParamScope() returned nil")
	}
	if s.prefix != "" {
		t.Errorf("expected empty prefix, got %q", s.prefix)
	}
	if s.next != 0 {
		t.Errorf("expected next=0, got %d", s.next)
	}
	if s.params == nil {
		t.Fatal("expected non-nil params map")
	}
}

// Test: add() registers a parameter and returns namespaced placeholder "$p0", "$p1", etc.
func TestParamScope_Add_ReturnsNamespacedPlaceholder(t *testing.T) {
	s := newParamScope()

	// First parameter should be $p0
	p0 := s.add("Matrix")
	if p0 != "$p0" {
		t.Errorf("expected $p0, got %s", p0)
	}

	// Second parameter should be $p1
	p1 := s.add(1999)
	if p1 != "$p1" {
		t.Errorf("expected $p1, got %s", p1)
	}
}

// Test: add() stores the value in the params map keyed by the parameter name (without $).
func TestParamScope_Add_StoresValue(t *testing.T) {
	s := newParamScope()
	s.add("Matrix")

	val, ok := s.params["p0"]
	if !ok {
		t.Fatal("expected p0 key in params")
	}
	if val != "Matrix" {
		t.Errorf("expected params[p0]='Matrix', got %v", val)
	}
}

// Test: sub() creates a child scope with namespaced prefix.
// e.g., scope.sub("sub0") creates prefix "sub0_" for nested params.
func TestParamScope_Sub_CreatesNamespacedChild(t *testing.T) {
	s := newParamScope()
	child := s.sub("sub0")
	if child == nil {
		t.Fatal("sub() returned nil")
	}
	if child.prefix != "sub0_" {
		t.Errorf("expected prefix 'sub0_', got %q", child.prefix)
	}
	if child.next != 0 {
		t.Errorf("expected next=0 in child, got %d", child.next)
	}
}

// Test: add() in a child scope returns namespaced placeholders like "$sub0_p0".
func TestParamScope_Sub_Add_ReturnsNamespacedPlaceholder(t *testing.T) {
	s := newParamScope()
	child := s.sub("sub0")

	p := child.add("value")
	if p != "$sub0_p0" {
		t.Errorf("expected $sub0_p0, got %s", p)
	}
}

// Test: addNamed() registers a parameter with a specific name.
func TestParamScope_AddNamed_ReturnsNamedPlaceholder(t *testing.T) {
	s := newParamScope()

	p := s.addNamed("set_title", "New Title")
	if p != "$set_title" {
		t.Errorf("expected $set_title, got %s", p)
	}

	val, ok := s.params["set_title"]
	if !ok {
		t.Fatal("expected set_title key in params")
	}
	if val != "New Title" {
		t.Errorf("expected params[set_title]='New Title', got %v", val)
	}
}

// Test: addNamed() in a child scope prefixes the name.
func TestParamScope_Sub_AddNamed_PrefixesName(t *testing.T) {
	s := newParamScope()
	child := s.sub("sub0")

	p := child.addNamed("offset", 5)
	if p != "$sub0_offset" {
		t.Errorf("expected $sub0_offset, got %s", p)
	}

	val, ok := child.params["sub0_offset"]
	if !ok {
		t.Fatal("expected sub0_offset key in params")
	}
	if val != 5 {
		t.Errorf("expected params[sub0_offset]=5, got %v", val)
	}
}

// Test: collect() returns all parameters from this scope.
func TestParamScope_Collect_ReturnsAllParams(t *testing.T) {
	s := newParamScope()
	s.add("v1")
	s.add("v2")
	s.addNamed("offset", 10)

	params := s.collect()
	if len(params) != 3 {
		t.Fatalf("expected 3 params, got %d", len(params))
	}
	if params["p0"] != "v1" {
		t.Errorf("expected p0='v1', got %v", params["p0"])
	}
	if params["p1"] != "v2" {
		t.Errorf("expected p1='v2', got %v", params["p1"])
	}
	if params["offset"] != 10 {
		t.Errorf("expected offset=10, got %v", params["offset"])
	}
}

// Test: collect() merges params from child scopes into the parent scope's params.
func TestParamScope_Collect_MergesChildParams(t *testing.T) {
	s := newParamScope()
	s.add("root_val")

	child := s.sub("sub0")
	child.add("child_val")

	params := s.collect()
	if len(params) < 2 {
		t.Fatalf("expected at least 2 params after merge, got %d", len(params))
	}
	if params["p0"] != "root_val" {
		t.Errorf("expected p0='root_val', got %v", params["p0"])
	}
	if params["sub0_p0"] != "child_val" {
		t.Errorf("expected sub0_p0='child_val', got %v", params["sub0_p0"])
	}
}

// Test: Deeply nested scopes produce unique parameter names.
func TestParamScope_DeepNesting_UniqueParams(t *testing.T) {
	s := newParamScope()
	child := s.sub("sub0")
	grandchild := child.sub("sub1")

	p := grandchild.add("deep_val")
	if p != "$sub0_sub1_p0" {
		t.Errorf("expected $sub0_sub1_p0, got %s", p)
	}
}

// Test: fieldContext carries node, variable, and depth.
func TestFieldContext_HoldsFields(t *testing.T) {
	node := schema.NodeDefinition{Name: "Movie", Labels: []string{"Movie"}}
	fc := fieldContext{node: node, variable: "n", depth: 0}
	if fc.node.Name != "Movie" {
		t.Errorf("expected node name 'Movie', got %s", fc.node.Name)
	}
	if fc.variable != "n" {
		t.Errorf("expected variable 'n', got %s", fc.variable)
	}
	if fc.depth != 0 {
		t.Errorf("expected depth 0, got %d", fc.depth)
	}
}
