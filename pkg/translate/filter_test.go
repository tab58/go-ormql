package translate

import (
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// movieNode returns a Movie NodeDefinition for filter/sort tests.
func movieNode() schema.NodeDefinition {
	return schema.NodeDefinition{
		Name:   "Movie",
		Labels: []string{"Movie"},
		Fields: []schema.FieldDefinition{
			{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
			{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
			{Name: "released", GraphQLType: "Int", GoType: "*int", CypherType: "INTEGER", Nullable: true},
			{Name: "rating", GraphQLType: "Float", GoType: "*float64", CypherType: "FLOAT", Nullable: true},
			{Name: "active", GraphQLType: "Boolean", GoType: "*bool", CypherType: "BOOLEAN", Nullable: true},
		},
	}
}

// makeWhereValue creates an ast.Value object field list from a map of field name → raw value.
// This is a test helper to construct filter arguments.
func makeWhereValue(fields map[string]*ast.Value) *ast.Value {
	children := make(ast.ChildValueList, 0, len(fields))
	for name, val := range fields {
		children = append(children, &ast.ChildValue{
			Name:  name,
			Value: val,
		})
	}
	return &ast.Value{
		Kind:     ast.ObjectValue,
		Children: children,
	}
}

// strVal creates a string ast.Value.
func strVal(v string) *ast.Value {
	return &ast.Value{Kind: ast.StringValue, Raw: v}
}

// intVal creates an int ast.Value.
func intVal(v string) *ast.Value {
	return &ast.Value{Kind: ast.IntValue, Raw: v}
}

// floatVal creates a float ast.Value.
func floatVal(v string) *ast.Value {
	return &ast.Value{Kind: ast.FloatValue, Raw: v}
}

// boolVal creates a boolean ast.Value.
func boolVal(v bool) *ast.Value {
	raw := "false"
	if v {
		raw = "true"
	}
	return &ast.Value{Kind: ast.BooleanValue, Raw: raw}
}

// listVal creates a list ast.Value from child values.
func listVal(vals ...*ast.Value) *ast.Value {
	children := make(ast.ChildValueList, len(vals))
	for i, v := range vals {
		children[i] = &ast.ChildValue{Value: v}
	}
	return &ast.Value{Kind: ast.ListValue, Children: children}
}

// --- Equality operator (field: value → n.field = $param) ---

// Test: Equality filter on a string field produces "n.title = $p0".
func TestBuildWhereClause_Equality(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"title": strVal("Matrix"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if result == "" {
		t.Fatal("expected non-empty WHERE clause, got empty")
	}
	if !strings.Contains(result, "n.title") {
		t.Errorf("expected n.title in WHERE clause, got %q", result)
	}
	if !strings.Contains(result, "= $") {
		t.Errorf("expected '= $' in WHERE clause, got %q", result)
	}
}

// --- Comparison operators ---

// Test: _gt operator produces "n.released > $param".
func TestBuildWhereClause_GT(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"released_gt": intVal("2000"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "> $") {
		t.Errorf("expected '> $' in WHERE clause for _gt, got %q", result)
	}
}

// Test: _gte operator produces "n.released >= $param".
func TestBuildWhereClause_GTE(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"released_gte": intVal("2000"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, ">= $") {
		t.Errorf("expected '>= $' in WHERE clause for _gte, got %q", result)
	}
}

// Test: _lt operator produces "n.released < $param".
func TestBuildWhereClause_LT(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"released_lt": intVal("2000"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "< $") {
		t.Errorf("expected '< $' in WHERE clause for _lt, got %q", result)
	}
}

// Test: _lte operator produces "n.released <= $param".
func TestBuildWhereClause_LTE(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"released_lte": intVal("2000"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "<= $") {
		t.Errorf("expected '<= $' in WHERE clause for _lte, got %q", result)
	}
}

// --- Not-equal operator ---

// Test: _not operator produces "n.title <> $param".
func TestBuildWhereClause_NotEqual(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"title_not": strVal("Matrix"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "<> $") {
		t.Errorf("expected '<> $' in WHERE clause for _not, got %q", result)
	}
}

// --- String operators ---

// Test: _contains operator produces "n.title CONTAINS $param".
func TestBuildWhereClause_Contains(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"title_contains": strVal("Matrix"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "CONTAINS $") {
		t.Errorf("expected 'CONTAINS $' in WHERE clause, got %q", result)
	}
}

// Test: _startsWith operator produces "n.title STARTS WITH $param".
func TestBuildWhereClause_StartsWith(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"title_startsWith": strVal("The"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "STARTS WITH $") {
		t.Errorf("expected 'STARTS WITH $' in WHERE clause, got %q", result)
	}
}

// Test: _endsWith operator produces "n.title ENDS WITH $param".
func TestBuildWhereClause_EndsWith(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"title_endsWith": strVal("ion"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "ENDS WITH $") {
		t.Errorf("expected 'ENDS WITH $' in WHERE clause, got %q", result)
	}
}

// Test: _regex operator produces "n.title =~ $param".
func TestBuildWhereClause_Regex(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"title_regex": strVal(".*Matrix.*"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "=~ $") {
		t.Errorf("expected '=~ $' in WHERE clause for _regex, got %q", result)
	}
}

// --- List operators ---

// Test: _in operator produces "n.title IN $param".
func TestBuildWhereClause_In(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"title_in": listVal(strVal("Matrix"), strVal("Inception")),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "IN $") {
		t.Errorf("expected 'IN $' in WHERE clause for _in, got %q", result)
	}
}

// Test: _nin operator produces "NOT n.title IN $param".
func TestBuildWhereClause_NotIn(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"title_nin": listVal(strVal("Matrix")),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "NOT") && !strings.Contains(result, "IN $") {
		t.Errorf("expected 'NOT ... IN $' in WHERE clause for _nin, got %q", result)
	}
}

// --- Null check operators ---

// Test: _isNull: true produces "n.rating IS NULL".
func TestBuildWhereClause_IsNull_True(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"rating_isNull": boolVal(true),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "IS NULL") {
		t.Errorf("expected 'IS NULL' in WHERE clause, got %q", result)
	}
}

// Test: _isNull: false produces "n.rating IS NOT NULL".
func TestBuildWhereClause_IsNull_False(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"rating_isNull": boolVal(false),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "IS NOT NULL") {
		t.Errorf("expected 'IS NOT NULL' in WHERE clause, got %q", result)
	}
}

// --- Boolean composition ---

// Test: AND composition joins predicates with AND and correct parenthesization.
func TestBuildWhereClause_AND(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()

	andList := listVal(
		makeWhereValue(map[string]*ast.Value{"title": strVal("Matrix")}),
		makeWhereValue(map[string]*ast.Value{"released_gte": intVal("2000")}),
	)
	where := makeWhereValue(map[string]*ast.Value{
		"AND": andList,
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "AND") {
		t.Errorf("expected 'AND' in WHERE clause, got %q", result)
	}
}

// Test: OR composition joins predicates with OR and correct parenthesization.
func TestBuildWhereClause_OR(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()

	orList := listVal(
		makeWhereValue(map[string]*ast.Value{"title": strVal("Matrix")}),
		makeWhereValue(map[string]*ast.Value{"title": strVal("Inception")}),
	)
	where := makeWhereValue(map[string]*ast.Value{
		"OR": orList,
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "OR") {
		t.Errorf("expected 'OR' in WHERE clause, got %q", result)
	}
}

// Test: NOT composition wraps with NOT and parenthesization.
func TestBuildWhereClause_NOT(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()

	notChild := makeWhereValue(map[string]*ast.Value{"title": strVal("Matrix")})
	where := makeWhereValue(map[string]*ast.Value{
		"NOT": notChild,
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "NOT") {
		t.Errorf("expected 'NOT' in WHERE clause, got %q", result)
	}
}

// Test: Multiple fields at top level are implicitly ANDed.
func TestBuildWhereClause_MultipleFieldsImplicitAND(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"title":        strVal("Matrix"),
		"released_gte": intVal("1999"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if result == "" {
		t.Fatal("expected non-empty WHERE clause for multiple fields")
	}
	// Both fields should appear in the clause
	if !strings.Contains(result, "n.title") {
		t.Errorf("expected n.title in WHERE clause, got %q", result)
	}
	if !strings.Contains(result, "n.released") {
		t.Errorf("expected n.released in WHERE clause, got %q", result)
	}
}

// Test: Empty where argument returns empty string.
func TestBuildWhereClause_EmptyWhere(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{})

	result := tr.buildWhereClause(where, "n", node, scope)
	if result != "" {
		t.Errorf("expected empty WHERE clause for empty filter, got %q", result)
	}
}

// Test: Nil where argument returns empty string.
func TestBuildWhereClause_NilWhere(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()

	result := tr.buildWhereClause(nil, "n", node, scope)
	if result != "" {
		t.Errorf("expected empty WHERE clause for nil filter, got %q", result)
	}
}

// Test: Nested boolean composition — NOT containing OR.
func TestBuildWhereClause_NestedBooleanComposition(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()

	orList := listVal(
		makeWhereValue(map[string]*ast.Value{"title": strVal("Matrix")}),
		makeWhereValue(map[string]*ast.Value{"title": strVal("Inception")}),
	)
	notChild := makeWhereValue(map[string]*ast.Value{
		"OR": orList,
	})
	where := makeWhereValue(map[string]*ast.Value{
		"NOT": notChild,
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "NOT") {
		t.Errorf("expected 'NOT' in WHERE clause, got %q", result)
	}
	if !strings.Contains(result, "OR") {
		t.Errorf("expected 'OR' inside NOT clause, got %q", result)
	}
}

// Test: Parameterization — values are stored in scope, not inlined.
func TestBuildWhereClause_ParameterizesValues(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()
	where := makeWhereValue(map[string]*ast.Value{
		"title": strVal("Matrix"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	// The result should contain a parameter reference ($pN), not the literal value.
	if strings.Contains(result, "Matrix") {
		t.Errorf("WHERE clause should not contain literal value, got %q", result)
	}
	// The scope should have at least one parameter.
	params := scope.collect()
	if len(params) == 0 {
		t.Error("expected at least one parameter in scope after building WHERE clause")
	}
}
