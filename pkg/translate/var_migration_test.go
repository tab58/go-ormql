package translate

import (
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// =============================================================================
// VAR-3: Migrate 7 astValueToGo call sites to resolveValue
// =============================================================================

// --- Site 1-3: filter.go buildPredicate() — 3 astValueToGo calls ---

// Test: Equality filter with variable produces correct param value.
// filter.go line ~106: param := scope.add(astValueToGo(val))
// Expected: where: {title: $title} with vars {"title": "Matrix"} → params contain "Matrix" not "title".
// FAILS: astValueToGo reads val.Raw which is "title" (the variable name), not "Matrix".
func TestBuildPredicate_Equality_WithVariable(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	scope.variables = map[string]any{"title": "Matrix"}
	node := movieNode()

	where := makeWhereValue(map[string]*ast.Value{
		"title": varVal("title"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if result == "" {
		t.Fatal("expected non-empty WHERE clause")
	}
	if !strings.Contains(result, "n.title = $") {
		t.Errorf("expected 'n.title = $pN' in WHERE clause, got %q", result)
	}

	// The parameter value should be "Matrix" (resolved), not "title" (variable name)
	params := scope.collect()
	foundMatrix := false
	for _, v := range params {
		if v == "Matrix" {
			foundMatrix = true
			break
		}
	}
	if !foundMatrix {
		t.Errorf("expected resolved variable value 'Matrix' in params, got %v", params)
	}
}

// Test: Comparison operator (_gte) with variable produces correct param value.
// filter.go line ~100: param := scope.add(astValueToGo(val))
// Expected: where: {released_gte: $year} with vars {"year": float64(1999)} → params contain 1999.
// FAILS: astValueToGo reads val.Raw which is "year" (the variable name).
func TestBuildPredicate_ComparisonOp_WithVariable(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	scope.variables = map[string]any{"year": float64(1999)}
	node := movieNode()

	where := makeWhereValue(map[string]*ast.Value{
		"released_gte": varVal("year"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, ">= $") {
		t.Errorf("expected '>= $pN' in WHERE clause, got %q", result)
	}

	// The parameter value should be float64(1999) (resolved), not "year"
	params := scope.collect()
	found := false
	for _, v := range params {
		if v == float64(1999) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected resolved variable value float64(1999) in params, got %v", params)
	}
}

// Test: NOT_IN operator (_nin) with variable list produces correct param value.
// filter.go line ~96: param := scope.add(astValueToGo(val))
// Expected: where: {title_nin: $excluded} with vars {"excluded": []any{"Bad","Ugly"}} → params contain the list.
// FAILS: astValueToGo reads val.Raw which is "excluded" (the variable name).
func TestBuildPredicate_NotIn_WithVariable(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	scope.variables = map[string]any{"excluded": []any{"Bad", "Ugly"}}
	node := movieNode()

	where := makeWhereValue(map[string]*ast.Value{
		"title_nin": varVal("excluded"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "NOT") || !strings.Contains(result, "IN $") {
		t.Errorf("expected 'NOT n.title IN $pN' in WHERE clause, got %q", result)
	}

	// The parameter value should be the resolved list, not "excluded"
	params := scope.collect()
	found := false
	for _, v := range params {
		if list, ok := v.([]any); ok && len(list) == 2 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected resolved variable list in params, got %v", params)
	}
}

// Test: Mixed literal + variable in same where clause.
// Expected: where: {title: "Matrix", released_gte: $year} → both resolve correctly.
// FAILS: The variable site still uses astValueToGo.
func TestBuildWhereClause_MixedLiteralVariable(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	scope.variables = map[string]any{"year": float64(1999)}
	node := movieNode()

	where := &ast.Value{
		Kind: ast.ObjectValue,
		Children: ast.ChildValueList{
			{Name: "title", Value: strVal("Matrix")},
			{Name: "released_gte", Value: varVal("year")},
		},
	}

	result := tr.buildWhereClause(where, "n", node, scope)
	if result == "" {
		t.Fatal("expected non-empty WHERE clause")
	}

	params := scope.collect()
	// Should have "Matrix" (literal) and float64(1999) (resolved variable)
	hasMatrix := false
	hasYear := false
	for _, v := range params {
		if v == "Matrix" {
			hasMatrix = true
		}
		if v == float64(1999) {
			hasYear = true
		}
	}
	if !hasMatrix {
		t.Errorf("expected 'Matrix' literal in params, got %v", params)
	}
	if !hasYear {
		t.Errorf("expected float64(1999) resolved variable in params, got %v", params)
	}
}

// --- Site 4: mutation.go translateCreateField() — input argument ---

// Test: Create mutation with variable input resolves the variable value.
// mutation.go line ~88: inputParam := scope.add(astValueToGo(inputArg.Value))
// Expected: createMovies(input: $input) with vars {"input": [...]} → params contain the resolved list.
// FAILS: astValueToGo reads val.Raw which is "input" (variable name).
func TestTranslateCreateField_VariableInput(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	scope.variables = map[string]any{
		"input": []any{map[string]any{"title": "New Movie", "released": float64(2024)}},
	}

	field := &ast.Field{
		Alias: "createMovies",
		Name:  "createMovies",
		Arguments: ast.ArgumentList{
			{Name: "input", Value: varVal("input")},
		},
		SelectionSet: ast.SelectionSet{
			&ast.Field{
				Alias: "movies",
				Name:  "movies",
				SelectionSet: ast.SelectionSet{
					&ast.Field{Alias: "id", Name: "id"},
					&ast.Field{Alias: "title", Name: "title"},
				},
			},
		},
	}

	_, _, err := tr.translateCreateField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The input param should be the resolved list value, not the string "input"
	params := scope.collect()
	found := false
	for _, v := range params {
		if list, ok := v.([]any); ok && len(list) == 1 {
			if m, ok := list[0].(map[string]any); ok {
				if m["title"] == "New Movie" {
					found = true
					break
				}
			}
		}
	}
	if !found {
		t.Errorf("expected resolved input variable in params, got %v", params)
	}
}

// --- Site 5: mutation.go translateUpdateField() — update scalar SET ---

// Test: Update mutation with variable in update argument resolves correctly.
// mutation.go line ~191: param := scope.add(astValueToGo(child.Value))
// Expected: updateMovies(where: {...}, update: {title: $newTitle}) → params contain "Updated Title".
// FAILS: astValueToGo reads val.Raw which is "newTitle" (variable name).
func TestTranslateUpdateField_VariableUpdateArg(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	scope.variables = map[string]any{"newTitle": "Updated Title"}

	field := &ast.Field{
		Alias: "updateMovies",
		Name:  "updateMovies",
		Arguments: ast.ArgumentList{
			{Name: "where", Value: makeWhereValue(map[string]*ast.Value{
				"id": strVal("1"),
			})},
			{Name: "update", Value: &ast.Value{
				Kind: ast.ObjectValue,
				Children: ast.ChildValueList{
					{Name: "title", Value: varVal("newTitle")},
				},
			}},
		},
		SelectionSet: ast.SelectionSet{
			&ast.Field{
				Alias: "movies",
				Name:  "movies",
				SelectionSet: ast.SelectionSet{
					&ast.Field{Alias: "id", Name: "id"},
					&ast.Field{Alias: "title", Name: "title"},
				},
			},
		},
	}

	_, _, err := tr.translateUpdateField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The SET param should be "Updated Title" (resolved), not "newTitle"
	params := scope.collect()
	found := false
	for _, v := range params {
		if v == "Updated Title" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected resolved 'Updated Title' in params, got %v", params)
	}
}

// --- Site 6: subquery.go buildCypherSubquery() — @cypher field arguments ---

// Test: @cypher field with variable argument resolves the variable value.
// subquery.go line ~97: scope.addNamed(arg.Name, astValueToGo(arg.Value))
// Expected: similarMovies(limit: $limit) with vars {"limit": float64(3)} → params contain float64(3).
// FAILS: astValueToGo reads val.Raw which is "limit" (variable name).
func TestBuildCypherSubquery_VariableArg(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	scope.variables = map[string]any{"limit": float64(3)}

	cf := schema.CypherFieldDefinition{
		Name:        "similarMovies",
		GraphQLType: "[Movie]",
		GoType:      "[]Movie",
		Statement:   "MATCH (this)-[:IN_GENRE]->()<-[:IN_GENRE]-(rec) RETURN rec LIMIT $limit",
		IsList:      true,
		Arguments: []schema.ArgumentDefinition{
			{Name: "limit", GraphQLType: "Int"},
		},
	}
	fc := fieldContext{
		node:     movieNode(),
		variable: "n",
		depth:    0,
	}
	field := &ast.Field{
		Alias: "similarMovies",
		Name:  "similarMovies",
		Arguments: ast.ArgumentList{
			{Name: "limit", Value: varVal("limit")},
		},
	}

	_, _, err := tr.buildCypherSubquery(field, cf, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The "limit" param should be float64(3) (resolved), not "limit"
	params := scope.collect()
	val, ok := params["limit"]
	if !ok {
		t.Fatalf("expected 'limit' key in params, got %v", params)
	}
	if val != float64(3) {
		t.Errorf("expected float64(3) for limit param, got %v (%T)", val, val)
	}
}

// --- Site 7: query.go buildFieldAssignments() — nested mutation helper ---

// Test: buildFieldAssignments with variable value resolves it.
// query.go line ~335: param := scope.add(astValueToGo(child.Value))
// Expected: field data with {name: $name} and vars {"name": "Keanu"} → params contain "Keanu".
// FAILS: astValueToGo reads val.Raw which is "name" (variable name).
func TestBuildFieldAssignments_WithVariable(t *testing.T) {
	scope := newParamScope()
	scope.variables = map[string]any{"name": "Keanu"}

	data := &ast.Value{
		Kind: ast.ObjectValue,
		Children: ast.ChildValueList{
			{Name: "name", Value: varVal("name")},
		},
	}

	parts := buildFieldAssignments(data, "target", scope)
	if len(parts) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(parts))
	}
	if !strings.Contains(parts[0], "target.name = $") {
		t.Errorf("expected 'target.name = $pN', got %q", parts[0])
	}

	// The param value should be "Keanu" (resolved), not "name"
	params := scope.collect()
	found := false
	for _, v := range params {
		if v == "Keanu" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected resolved 'Keanu' in params, got %v", params)
	}
}

// Test: buildFieldAssignments with mixed literal + variable fields resolves both.
// Expected: {name: "Literal", born: $year} → params contain "Literal" and float64(1964).
// FAILS: variable site still uses astValueToGo.
func TestBuildFieldAssignments_MixedLiteralVariable(t *testing.T) {
	scope := newParamScope()
	scope.variables = map[string]any{"year": float64(1964)}

	data := &ast.Value{
		Kind: ast.ObjectValue,
		Children: ast.ChildValueList{
			{Name: "name", Value: strVal("Keanu")},
			{Name: "born", Value: varVal("year")},
		},
	}

	parts := buildFieldAssignments(data, "a", scope)
	if len(parts) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(parts))
	}

	params := scope.collect()
	hasKeanu := false
	hasYear := false
	for _, v := range params {
		if v == "Keanu" {
			hasKeanu = true
		}
		if v == float64(1964) {
			hasYear = true
		}
	}
	if !hasKeanu {
		t.Errorf("expected 'Keanu' literal in params, got %v", params)
	}
	if !hasYear {
		t.Errorf("expected float64(1964) resolved variable in params, got %v", params)
	}
}

// --- Regression guard: Existing literal-only calls continue to work ---

// Test: buildPredicate with literal values still works correctly (regression guard).
// Expected: All existing literal-only tests continue to pass.
func TestBuildPredicate_LiteralStillWorks(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	node := movieNode()

	where := makeWhereValue(map[string]*ast.Value{
		"title": strVal("Matrix"),
	})

	result := tr.buildWhereClause(where, "n", node, scope)
	if !strings.Contains(result, "n.title = $") {
		t.Errorf("literal equality should still work, got %q", result)
	}

	params := scope.collect()
	found := false
	for _, v := range params {
		if v == "Matrix" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("literal value 'Matrix' should be in params, got %v", params)
	}
}
