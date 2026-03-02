package translate

import (
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// --- TR-3: Map projection tests ---

// Test: buildProjection with scalar fields produces "n { .title, .released }".
func TestBuildProjection_ScalarFields(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	selSet := ast.SelectionSet{
		&ast.Field{Name: "title"},
		&ast.Field{Name: "released"},
	}

	result, _, err := tr.buildProjection(selSet, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty projection, got empty")
	}
	if !strings.Contains(result, ".title") {
		t.Errorf("expected .title in projection, got %q", result)
	}
	if !strings.Contains(result, ".released") {
		t.Errorf("expected .released in projection, got %q", result)
	}
}

// Test: buildProjection includes only selected fields (no over-fetching).
// If only "title" is selected, "released" should NOT appear.
func TestBuildProjection_OnlySelectedFields(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	selSet := ast.SelectionSet{
		&ast.Field{Name: "title"},
	}

	result, _, err := tr.buildProjection(selSet, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, ".title") {
		t.Errorf("expected .title in projection, got %q", result)
	}
	if strings.Contains(result, ".released") {
		t.Errorf("projection should not contain .released when not selected, got %q", result)
	}
}

// Test: buildProjection uses the correct variable name.
func TestBuildProjection_UsesVariable(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Actor")
	fc := fieldContext{node: node, variable: "a", depth: 0}

	selSet := ast.SelectionSet{
		&ast.Field{Name: "name"},
	}

	result, _, err := tr.buildProjection(selSet, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "a {") || !strings.Contains(result, "a{") {
		// Accept either "a {" or "a{" formatting
		if !strings.HasPrefix(strings.TrimSpace(result), "a") {
			t.Errorf("expected projection to start with variable 'a', got %q", result)
		}
	}
}

// Test: buildProjection with empty selection set returns empty map projection.
func TestBuildProjection_EmptySelectionSet(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	result, _, err := tr.buildProjection(ast.SelectionSet{}, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should produce an empty projection like "n {}" or similar
	if !strings.Contains(result, "n") {
		t.Errorf("expected variable in empty projection, got %q", result)
	}
}

// Test: buildProjection with ID field includes .id.
func TestBuildProjection_IncludesIDField(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node, _ := testModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	selSet := ast.SelectionSet{
		&ast.Field{Name: "id"},
		&ast.Field{Name: "title"},
	}

	result, _, err := tr.buildProjection(selSet, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, ".id") {
		t.Errorf("expected .id in projection, got %q", result)
	}
}

// --- H1 regression: Multi-relationship unique aliases ---

// multiRelModel returns a model where Movie has 2 relationships (actors + directors)
// to verify that buildProjection assigns unique aliases to each subquery.
func multiRelModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
				},
			},
			{
				Name:   "Actor",
				Labels: []string{"Actor"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
				},
			},
			{
				Name:   "Director",
				Labels: []string{"Director"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
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
			},
			{
				FieldName: "directors",
				RelType:   "DIRECTED",
				Direction: schema.DirectionIN,
				FromNode:  "Movie",
				ToNode:    "Director",
			},
		},
	}
}

// Test: buildProjection with 2 relationship fields produces unique subquery aliases.
// Before H1 fix, both got __sub0 causing a Cypher alias collision.
func TestBuildProjection_MultiRelUniqueAliases(t *testing.T) {
	tr := New(multiRelModel())
	scope := newParamScope()
	node, _ := multiRelModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	selSet := ast.SelectionSet{
		&ast.Field{Name: "title"},
		&ast.Field{Name: "actors", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "name"},
		}},
		&ast.Field{Name: "directors", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "name"},
		}},
	}

	proj, subqueries, err := tr.buildProjection(selSet, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 subqueries with different aliases
	if len(subqueries) != 2 {
		t.Fatalf("expected 2 subqueries for 2 relationships, got %d", len(subqueries))
	}

	// Aliases should be different (__sub0 and __sub1)
	if strings.Contains(subqueries[0], "__sub0") && strings.Contains(subqueries[1], "__sub0") {
		t.Error("both subqueries use __sub0 alias — alias collision (H1 bug)")
	}

	// Projection should reference both unique aliases
	if !strings.Contains(proj, "actors: __sub0") {
		t.Errorf("expected 'actors: __sub0' in projection, got %q", proj)
	}
	if !strings.Contains(proj, "directors: __sub1") {
		t.Errorf("expected 'directors: __sub1' in projection, got %q", proj)
	}
}

// Test: buildProjection with relationship + connection produces unique aliases.
func TestBuildProjection_RelAndConnectionUniqueAliases(t *testing.T) {
	tr := New(multiRelModel())
	scope := newParamScope()
	node, _ := multiRelModel().NodeByName("Movie")
	fc := fieldContext{node: node, variable: "n", depth: 0}

	selSet := ast.SelectionSet{
		&ast.Field{Name: "actors", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "name"},
		}},
		&ast.Field{Name: "directorsConnection", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "edges", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "node", SelectionSet: ast.SelectionSet{
					&ast.Field{Name: "name"},
				}},
			}},
		}},
	}

	_, subqueries, err := tr.buildProjection(selSet, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(subqueries) != 2 {
		t.Fatalf("expected 2 subqueries (rel + connection), got %d", len(subqueries))
	}

	// First subquery should be __sub0, second should be __conn1
	if !strings.Contains(subqueries[0], "__sub0") {
		t.Errorf("expected first subquery to use __sub0, got %q", subqueries[0])
	}
	if !strings.Contains(subqueries[1], "__conn1") {
		t.Errorf("expected second subquery to use __conn1, got %q", subqueries[1])
	}
}

// Test: E2E Translate with multiple relationships produces valid Cypher.
func TestTranslate_E2E_MultiRelQuery(t *testing.T) {
	tr := New(multiRelModel())

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Name: "title"},
					&ast.Field{Name: "actors", SelectionSet: ast.SelectionSet{
						&ast.Field{Name: "name"},
					}},
					&ast.Field{Name: "directors", SelectionSet: ast.SelectionSet{
						&ast.Field{Name: "name"},
					}},
				}),
			),
		},
	}
	op := doc.Operations[0]

	stmt, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have both subquery aliases referenced
	if !strings.Contains(stmt.Query, "__sub0") {
		t.Errorf("expected __sub0 in query, got %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "__sub1") {
		t.Errorf("expected __sub1 in query, got %q", stmt.Query)
	}

	// Both should be in separate CALL blocks
	callCount := strings.Count(stmt.Query, "CALL")
	if callCount < 3 {
		t.Errorf("expected at least 3 CALL blocks (1 root + 2 subqueries), got %d", callCount)
	}
}

// --- Root list query translation tests ---

// makeQueryOp creates a query operation with root fields.
func makeQueryOp(fields ...*ast.Field) *ast.OperationDefinition {
	selSet := make(ast.SelectionSet, len(fields))
	for i, f := range fields {
		selSet[i] = f
	}
	return &ast.OperationDefinition{
		Operation:    ast.Query,
		SelectionSet: selSet,
	}
}

// makeField creates an ast.Field with name, alias, arguments, and selection set.
func makeField(name string, selSet ast.SelectionSet, args ...*ast.Argument) *ast.Field {
	return &ast.Field{
		Name:         name,
		Alias:        name,
		Arguments:    args,
		SelectionSet: selSet,
	}
}

// makeArg creates an ast.Argument.
func makeArg(name string, value *ast.Value) *ast.Argument {
	return &ast.Argument{Name: name, Value: value}
}
