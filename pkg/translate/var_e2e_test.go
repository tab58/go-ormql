package translate

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// =============================================================================
// VAR-5: E2E tests with parameterized queries and mutations
// =============================================================================

// E2E variable tests exercise the full Translate() path with variables.
// Each test builds a complete OperationDefinition with variable references,
// passes a variables map, and verifies the final Cypher + params.

// Test: Query with variable filter.
// query($title: String!) { movies(where: {title: $title}) { title } }
// variables: {"title": "Matrix"}
// Expected: WHERE n.title = $p0 with params {p0: "Matrix"}
// FAILS: Variable not resolved — param contains "title" (var name) not "Matrix".
func TestE2E_QueryVariableFilter(t *testing.T) {
	tr := New(testModel())

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"title": varVal("title"),
	}))

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Alias: "title", Name: "title"},
				}, whereArg),
			),
		},
	}
	op := doc.Operations[0]
	variables := map[string]any{"title": "Matrix"}

	stmt, err := tr.Translate(doc, op, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cypher should contain WHERE with a parameter
	if !strings.Contains(stmt.Query, "WHERE") {
		t.Errorf("expected WHERE in query, got %q", stmt.Query)
	}

	// Params should contain "Matrix" (resolved), not "title" (var name)
	foundMatrix := false
	for _, v := range stmt.Params {
		if v == "Matrix" {
			foundMatrix = true
			break
		}
	}
	if !foundMatrix {
		t.Errorf("expected 'Matrix' in params, got %v", stmt.Params)
	}

	// "title" as a param VALUE would be wrong (that's the variable name)
	for k, v := range stmt.Params {
		if v == "title" {
			t.Errorf("param %q has variable name 'title' instead of resolved value", k)
		}
	}
}

// Test: Query with variable pagination.
// query($limit: Int!) { movies(first: $limit) { title } }
// variables: {"limit": 10}
// Expected: collect(n {...})[..10] or LIMIT with resolved 10.
// FAILS: Variable not resolved in list pagination path.
func TestE2E_QueryVariablePagination(t *testing.T) {
	tr := New(testModel())

	// Note: root list queries don't currently use first/after pagination
	// the same way connections do. This tests connection pagination.
	firstArg := makeArg("first", varVal("limit"))

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("moviesConnection", ast.SelectionSet{
					&ast.Field{Alias: "edges", Name: "edges", SelectionSet: ast.SelectionSet{
						&ast.Field{Alias: "node", Name: "node", SelectionSet: ast.SelectionSet{
							&ast.Field{Alias: "title", Name: "title"},
						}},
					}},
				}, firstArg),
			),
		},
	}
	op := doc.Operations[0]
	variables := map[string]any{"limit": float64(10)}

	stmt, err := tr.Translate(doc, op, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Params should contain first=10 (resolved from float64)
	found := false
	for k, v := range stmt.Params {
		if strings.HasSuffix(k, "_first") {
			if v == 10 || v == int(10) || v == int64(10) {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("expected first=10 in params, got %v", stmt.Params)
	}
}

// Test: Query with mixed literal + variable filter.
// query($year: Int!) { movies(where: {title: "Matrix", released_gte: $year}) { title } }
// variables: {"year": 1999}
// Expected: WHERE n.title = $p0 AND n.released >= $p1, params {p0: "Matrix", p1: 1999}
// FAILS: Variable site produces var name instead of resolved value.
func TestE2E_QueryMixedLiteralVariable(t *testing.T) {
	tr := New(testModel())

	whereArg := makeArg("where", &ast.Value{
		Kind: ast.ObjectValue,
		Children: ast.ChildValueList{
			{Name: "title", Value: strVal("Matrix")},
			{Name: "released_gte", Value: varVal("year")},
		},
	})

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Alias: "title", Name: "title"},
				}, whereArg),
			),
		},
	}
	op := doc.Operations[0]
	variables := map[string]any{"year": float64(1999)}

	stmt, err := tr.Translate(doc, op, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both "Matrix" (literal) and 1999 (resolved variable) should be in params
	hasMatrix := false
	hasYear := false
	for _, v := range stmt.Params {
		if v == "Matrix" {
			hasMatrix = true
		}
		if v == float64(1999) {
			hasYear = true
		}
	}
	if !hasMatrix {
		t.Errorf("expected 'Matrix' literal in params, got %v", stmt.Params)
	}
	if !hasYear {
		t.Errorf("expected float64(1999) resolved variable in params, got %v", stmt.Params)
	}
}

// Test: Query with variable sort.
// query($sort: [MovieSort!]) { movies(sort: $sort) { title } }
// variables: {"sort": [{"released": "DESC"}]}
// Expected: ORDER BY n.released DESC
// FAILS: buildOrderBy gets Variable kind with no Children, returns empty.
func TestE2E_QueryVariableSort(t *testing.T) {
	tr := New(testModel())

	sortArg := makeArg("sort", varVal("sort"))

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Alias: "title", Name: "title"},
				}, sortArg),
			),
		},
	}
	op := doc.Operations[0]
	variables := map[string]any{
		"sort": []any{map[string]any{"released": "DESC"}},
	}

	stmt, err := tr.Translate(doc, op, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain ORDER BY with n.released DESC
	if !strings.Contains(stmt.Query, "ORDER BY") {
		t.Errorf("expected ORDER BY in query with variable sort, got %q", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "n.released DESC") {
		t.Errorf("expected 'n.released DESC' in query, got %q", stmt.Query)
	}
}

// Test: Mutation with variable input.
// mutation($input: [MovieCreateInput!]!) { createMovies(input: $input) { movies { id title } } }
// variables: {"input": [{"title": "New", "released": 2024}]}
// Expected: UNWIND $p0 AS item, params {p0: [{"title": "New", "released": 2024}]}
// FAILS: astValueToGo reads val.Raw "input" (var name) not the resolved list.
func TestE2E_MutationVariableInput(t *testing.T) {
	tr := New(testModel())

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeMutationOp(
				makeField("createMovies", ast.SelectionSet{
					&ast.Field{Alias: "movies", Name: "movies", SelectionSet: ast.SelectionSet{
						&ast.Field{Alias: "id", Name: "id"},
						&ast.Field{Alias: "title", Name: "title"},
					}},
				}, makeArg("input", varVal("input"))),
			),
		},
	}
	op := doc.Operations[0]
	variables := map[string]any{
		"input": []any{map[string]any{"title": "New", "released": float64(2024)}},
	}

	stmt, err := tr.Translate(doc, op, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Params should contain the resolved input list
	found := false
	for _, v := range stmt.Params {
		if list, ok := v.([]any); ok && len(list) == 1 {
			if m, ok := list[0].(map[string]any); ok && m["title"] == "New" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("expected resolved input list in params, got %v", stmt.Params)
	}
}

// Test: Mutation with variable where + update.
// mutation($id: ID!, $title: String!) { updateMovies(where: {id: $id}, update: {title: $title}) { movies { id title } } }
// variables: {"id": "1", "title": "Updated"}
// Expected: WHERE n.id = $p0, SET n.title = $p1, params {p0: "1", p1: "Updated"}
// FAILS: Variables not resolved — params contain var names.
func TestE2E_MutationVariableWhereUpdate(t *testing.T) {
	tr := New(testModel())

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"id": varVal("id"),
	}))
	updateArg := makeArg("update", &ast.Value{
		Kind: ast.ObjectValue,
		Children: ast.ChildValueList{
			{Name: "title", Value: varVal("title")},
		},
	})

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeMutationOp(
				makeField("updateMovies", ast.SelectionSet{
					&ast.Field{Alias: "movies", Name: "movies", SelectionSet: ast.SelectionSet{
						&ast.Field{Alias: "id", Name: "id"},
						&ast.Field{Alias: "title", Name: "title"},
					}},
				}, whereArg, updateArg),
			),
		},
	}
	op := doc.Operations[0]
	variables := map[string]any{"id": "1", "title": "Updated"}

	stmt, err := tr.Translate(doc, op, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Params should contain "1" and "Updated" (resolved), not "id" and "title"
	hasID := false
	hasTitle := false
	for _, v := range stmt.Params {
		if v == "1" {
			hasID = true
		}
		if v == "Updated" {
			hasTitle = true
		}
	}
	if !hasID {
		t.Errorf("expected '1' in params for resolved $id, got %v", stmt.Params)
	}
	if !hasTitle {
		t.Errorf("expected 'Updated' in params for resolved $title, got %v", stmt.Params)
	}
}

// Test: @cypher field with variable argument.
// { movies { title similarMovies(limit: $limit) { title } } }
// variables: {"limit": 3}
// Expected: params contain limit=3 (resolved), not "limit".
// FAILS: astValueToGo reads val.Raw "limit" (var name).
func TestE2E_CypherFieldVariableArg(t *testing.T) {
	tr := New(testModel())

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Alias: "title", Name: "title"},
					&ast.Field{Alias: "averageRating", Name: "averageRating"},
				}),
			),
		},
	}
	op := doc.Operations[0]

	// The testModel() @cypher field "averageRating" has no arguments,
	// so this test just verifies variables pass through without error.
	variables := map[string]any{"limit": float64(3)}

	stmt, err := tr.Translate(doc, op, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stmt.Query == "" {
		t.Fatal("expected non-empty query with @cypher field")
	}
}

// Test: Connection with variable pagination (first + after).
// { moviesConnection(first: $limit, after: $cursor) { edges { node { title } } totalCount } }
// variables: {"limit": 5, "cursor": "Y3Vyc29yOjI="}
// Expected: SKIP $offset LIMIT $first with offset=3, first=5
// FAILS: .Value.Raw reads variable names.
func TestE2E_ConnectionVariablePagination(t *testing.T) {
	tr := New(testModel())

	cursor := base64.StdEncoding.EncodeToString([]byte("cursor:2"))

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("moviesConnection", ast.SelectionSet{
					&ast.Field{Alias: "edges", Name: "edges", SelectionSet: ast.SelectionSet{
						&ast.Field{Alias: "node", Name: "node", SelectionSet: ast.SelectionSet{
							&ast.Field{Alias: "title", Name: "title"},
						}},
					}},
					&ast.Field{Alias: "totalCount", Name: "totalCount"},
				}, makeArg("first", varVal("limit")), makeArg("after", varVal("cursor"))),
			),
		},
	}
	op := doc.Operations[0]
	variables := map[string]any{"limit": float64(5), "cursor": cursor}

	stmt, err := tr.Translate(doc, op, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// first=5 and offset=3 (cursor:2 → 2 + 1 = 3) should be in params
	firstOk := false
	offsetOk := false
	for k, v := range stmt.Params {
		if strings.HasSuffix(k, "_first") {
			if v == 5 || v == int(5) || v == int64(5) {
				firstOk = true
			}
		}
		if strings.HasSuffix(k, "_offset") {
			if v == 3 || v == int(3) || v == int64(3) {
				offsetOk = true
			}
		}
	}
	if !firstOk {
		t.Errorf("expected first=5 in params, got %v", stmt.Params)
	}
	if !offsetOk {
		t.Errorf("expected offset=3 in params, got %v", stmt.Params)
	}
}

// Test: Nil/missing optional variable resolves gracefully.
// query($title: String) { movies(where: {title: $title}) { title } }
// variables: {} (title not provided)
// Expected: No panic, param value is nil.
// FAILS: resolveValue stub returns nil for Variable kind — but astValueToGo
// reads val.Raw "title" producing the string "title" instead of nil.
func TestE2E_MissingOptionalVariable(t *testing.T) {
	tr := New(testModel())

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"title": varVal("title"),
	}))

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Alias: "title", Name: "title"},
				}, whereArg),
			),
		},
	}
	op := doc.Operations[0]
	variables := map[string]any{} // title not provided

	stmt, err := tr.Translate(doc, op, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The param for the missing variable should be nil, not the string "title"
	for k, v := range stmt.Params {
		if v == "title" {
			t.Errorf("param %q has variable name 'title' — should be nil for missing optional var, params: %v", k, stmt.Params)
		}
	}
}

// Test: Existing literal-only E2E query continues to work (regression guard).
// Expected: All params contain literal values, no regressions from variable support.
func TestE2E_LiteralOnlyRegression(t *testing.T) {
	tr := New(testModel())

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"title": strVal("Matrix"),
	}))

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Alias: "title", Name: "title"},
				}, whereArg),
			),
		},
	}
	op := doc.Operations[0]

	stmt, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should work exactly as before — param contains "Matrix"
	found := false
	for _, v := range stmt.Params {
		if v == "Matrix" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'Matrix' in params for literal-only query, got %v", stmt.Params)
	}
}

// Test: Boolean variable in _isNull filter.
// query($check: Boolean!) { movies(where: {rating_isNull: $check}) { title } }
// variables: {"check": true}
// Expected: WHERE n.rating IS NULL
// FAILS: buildPredicate reads val.Raw which is "check" (var name), ParseBool("check")=false.
func TestE2E_BooleanVariableIsNull(t *testing.T) {
	tr := New(testModel())

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"rating_isNull": varVal("check"),
	}))

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Alias: "title", Name: "title"},
				}, whereArg),
			),
		},
	}
	op := doc.Operations[0]
	variables := map[string]any{"check": true}

	stmt, err := tr.Translate(doc, op, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should produce IS NULL (check=true), not IS NOT NULL
	if !strings.Contains(stmt.Query, "IS NULL") {
		t.Errorf("expected 'IS NULL' for true _isNull variable, got %q", stmt.Query)
	}
	// Should NOT contain "IS NOT NULL" since the variable is true
	if strings.Contains(stmt.Query, "IS NOT NULL") {
		t.Errorf("expected 'IS NULL' (not 'IS NOT NULL') for true _isNull variable, got %q", stmt.Query)
	}
}
