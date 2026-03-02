package translate

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// =============================================================================
// VAR-4: Migrate 5 .Value.Raw sites to resolveValue (pagination + sort)
// =============================================================================

// --- Site 1: query.go translateConnectionField() — first pagination ---

// Test: Root connection field with variable `first: $limit` resolves to correct page size.
// query.go line ~146: strconv.ParseInt(firstArg.Value.Raw, 10, 64)
// Expected: first: $limit with vars {"limit": float64(5)} → LIMIT uses 5, not default 10.
// FAILS: ParseInt("limit") fails silently, first stays at default (10).
func TestTranslateConnectionField_VariableFirst(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	scope.variables = map[string]any{"limit": float64(5)}

	node := movieNode()
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := &ast.Field{
		Alias: "moviesConnection",
		Name:  "moviesConnection",
		Arguments: ast.ArgumentList{
			{Name: "first", Value: varVal("limit")},
		},
		SelectionSet: ast.SelectionSet{
			&ast.Field{Alias: "edges", Name: "edges", SelectionSet: ast.SelectionSet{
				&ast.Field{Alias: "node", Name: "node", SelectionSet: ast.SelectionSet{
					&ast.Field{Alias: "title", Name: "title"},
				}},
			}},
		},
	}

	callBlock, _, err := tr.translateConnectionField(field, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The generated Cypher should use the resolved value (5), not default (10).
	// Look for the first param value in scope — it should be 5.
	params := scope.collect()
	found := false
	for k, v := range params {
		if strings.HasSuffix(k, "_first") {
			if v == 5 || v == int(5) || v == int64(5) {
				found = true
				break
			}
			t.Errorf("expected first=5, got %v (%T) for key %q", v, v, k)
		}
	}
	if !found {
		t.Errorf("expected resolved first=5 in params, got %v. CALL block: %s", params, callBlock)
	}
}

// --- Site 2: query.go translateConnectionField() — after cursor ---

// Test: Root connection field with variable `after: $cursor` resolves the cursor.
// query.go line ~150: decodeCursor(afterArg.Value.Raw)
// Expected: after: $cursor with vars {"cursor": "Y3Vyc29yOjU="} → offset = 6 (decoded 5 + 1).
// FAILS: decodeCursor("cursor") returns 0 (can't decode the variable name).
func TestTranslateConnectionField_VariableAfter(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	cursor := base64.StdEncoding.EncodeToString([]byte("cursor:5"))
	scope.variables = map[string]any{"cursor": cursor}

	node := movieNode()
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := &ast.Field{
		Alias: "moviesConnection",
		Name:  "moviesConnection",
		Arguments: ast.ArgumentList{
			{Name: "after", Value: varVal("cursor")},
		},
		SelectionSet: ast.SelectionSet{
			&ast.Field{Alias: "edges", Name: "edges", SelectionSet: ast.SelectionSet{
				&ast.Field{Alias: "node", Name: "node", SelectionSet: ast.SelectionSet{
					&ast.Field{Alias: "title", Name: "title"},
				}},
			}},
		},
	}

	_, _, err := tr.translateConnectionField(field, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The offset param should be 6 (decoded cursor "cursor:5" → 5, then +1 = 6)
	params := scope.collect()
	found := false
	for k, v := range params {
		if strings.HasSuffix(k, "_offset") {
			if v == 6 || v == int(6) || v == int64(6) {
				found = true
				break
			}
			t.Errorf("expected offset=6, got %v (%T) for key %q", v, v, k)
		}
	}
	if !found {
		t.Errorf("expected resolved offset=6 in params, got %v", params)
	}
}

// --- Site 3: subquery.go buildConnectionSubquery() — nested first pagination ---

// Test: Nested connection field with variable `first: $limit` resolves correctly.
// subquery.go line ~145: strconv.ParseInt(firstArg.Value.Raw, 10, 64)
// Expected: actorsConnection(first: $limit) with vars {"limit": float64(3)} → LIMIT uses 3.
// FAILS: ParseInt("limit") fails silently, first stays at default (10).
func TestBuildConnectionSubquery_VariableFirst(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	scope.variables = map[string]any{"limit": float64(3)}

	rel := testModel().Relationships[0] // actors rel
	fc := fieldContext{
		node:     movieNode(),
		variable: "n",
		depth:    0,
	}

	field := &ast.Field{
		Alias: "actorsConnection",
		Name:  "actorsConnection",
		Arguments: ast.ArgumentList{
			{Name: "first", Value: varVal("limit")},
		},
		SelectionSet: ast.SelectionSet{
			&ast.Field{Alias: "edges", Name: "edges", SelectionSet: ast.SelectionSet{
				&ast.Field{Alias: "node", Name: "node", SelectionSet: ast.SelectionSet{
					&ast.Field{Alias: "name", Name: "name"},
				}},
			}},
		},
	}

	_, _, err := tr.buildConnectionSubquery(field, rel, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The first param should be 3, not default 10
	params := scope.collect()
	found := false
	for k, v := range params {
		if strings.HasSuffix(k, "_first") {
			if v == 3 || v == int(3) || v == int64(3) {
				found = true
				break
			}
			t.Errorf("expected first=3, got %v (%T) for key %q", v, v, k)
		}
	}
	if !found {
		t.Errorf("expected resolved first=3 in params, got %v", params)
	}
}

// --- Site 4: subquery.go buildConnectionSubquery() — nested after cursor ---

// Test: Nested connection field with variable `after: $cursor` resolves the cursor.
// subquery.go line ~149: decodeCursor(afterArg.Value.Raw)
// Expected: actorsConnection(after: $cursor) with vars {"cursor": "Y3Vyc29yOjI="} → offset=3.
// FAILS: decodeCursor("cursor") returns 0.
func TestBuildConnectionSubquery_VariableAfter(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	cursor := base64.StdEncoding.EncodeToString([]byte("cursor:2"))
	scope.variables = map[string]any{"cursor": cursor}

	rel := testModel().Relationships[0]
	fc := fieldContext{
		node:     movieNode(),
		variable: "n",
		depth:    0,
	}

	field := &ast.Field{
		Alias: "actorsConnection",
		Name:  "actorsConnection",
		Arguments: ast.ArgumentList{
			{Name: "after", Value: varVal("cursor")},
		},
		SelectionSet: ast.SelectionSet{
			&ast.Field{Alias: "edges", Name: "edges", SelectionSet: ast.SelectionSet{
				&ast.Field{Alias: "node", Name: "node", SelectionSet: ast.SelectionSet{
					&ast.Field{Alias: "name", Name: "name"},
				}},
			}},
		},
	}

	_, _, err := tr.buildConnectionSubquery(field, rel, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Offset should be 3 (decoded cursor "cursor:2" → 2, then +1 = 3)
	params := scope.collect()
	found := false
	for k, v := range params {
		if strings.HasSuffix(k, "_offset") {
			if v == 3 || v == int(3) || v == int64(3) {
				found = true
				break
			}
			t.Errorf("expected offset=3, got %v (%T) for key %q", v, v, k)
		}
	}
	if !found {
		t.Errorf("expected resolved offset=3 in params, got %v", params)
	}
}

// --- Site 5: sort.go buildOrderBy() — sort direction ---

// Test: Sort argument with variable sort input resolves the direction.
// sort.go line ~29: direction := strings.ToUpper(field.Value.Raw)
// Expected: sort: $sortInput with vars {"sortInput": [{"released": "DESC"}]} → ORDER BY n.released DESC.
// FAILS: The sort arg is a Variable kind, field.Value.Raw is "sortInput", not ObjectValue children.
func TestBuildOrderBy_VariableSort(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	scope.variables = map[string]any{
		"sortInput": []any{map[string]any{"released": "DESC"}},
	}

	// When sort is a variable, the AST value is Kind=Variable, Raw="sortInput"
	// The function needs to resolveValue first to get the actual sort data.
	sortArg := varVal("sortInput")

	result := tr.buildOrderBy(sortArg, "n", scope)

	// Should produce "n.released DESC"
	if !strings.Contains(result, "n.released") {
		t.Errorf("expected 'n.released' in ORDER BY, got %q", result)
	}
	if !strings.Contains(result, "DESC") {
		t.Errorf("expected 'DESC' in ORDER BY, got %q", result)
	}
}

// --- Regression guards ---

// Test: Literal pagination values still work correctly.
// Expected: first: 10 (literal IntValue) → page size 10.
func TestTranslateConnectionField_LiteralFirstStillWorks(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	node := movieNode()
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := &ast.Field{
		Alias: "moviesConnection",
		Name:  "moviesConnection",
		Arguments: ast.ArgumentList{
			{Name: "first", Value: intVal("7")},
		},
		SelectionSet: ast.SelectionSet{
			&ast.Field{Alias: "edges", Name: "edges", SelectionSet: ast.SelectionSet{
				&ast.Field{Alias: "node", Name: "node", SelectionSet: ast.SelectionSet{
					&ast.Field{Alias: "title", Name: "title"},
				}},
			}},
		},
	}

	_, _, err := tr.translateConnectionField(field, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	params := scope.collect()
	found := false
	for k, v := range params {
		if strings.HasSuffix(k, "_first") {
			if v == 7 || v == int(7) || v == int64(7) {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("expected literal first=7 in params, got %v", params)
	}
}

// Test: Literal sort direction still works correctly.
// Expected: sort: [{released: DESC}] (literal AST) → ORDER BY n.released DESC.
func TestBuildOrderBy_LiteralSortStillWorks(t *testing.T) {
	tr := New(testModel())

	sortArg := &ast.Value{
		Kind: ast.ListValue,
		Children: ast.ChildValueList{
			{Value: &ast.Value{
				Kind: ast.ObjectValue,
				Children: ast.ChildValueList{
					{Name: "released", Value: &ast.Value{Kind: ast.EnumValue, Raw: "DESC"}},
				},
			}},
		},
	}

	result := tr.buildOrderBy(sortArg, "n", nil)
	expected := "n.released DESC"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test: Both first and after as variables in the same connection field.
// Expected: first: $limit, after: $cursor → both resolve correctly.
// FAILS: Both .Value.Raw reads return variable names.
func TestTranslateConnectionField_BothVariables(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()
	cursor := base64.StdEncoding.EncodeToString([]byte("cursor:4"))
	scope.variables = map[string]any{
		"limit":  float64(20),
		"cursor": cursor,
	}

	node := movieNode()
	fc := fieldContext{node: node, variable: "n", depth: 0}

	field := &ast.Field{
		Alias: "moviesConnection",
		Name:  "moviesConnection",
		Arguments: ast.ArgumentList{
			{Name: "first", Value: varVal("limit")},
			{Name: "after", Value: varVal("cursor")},
		},
		SelectionSet: ast.SelectionSet{
			&ast.Field{Alias: "edges", Name: "edges", SelectionSet: ast.SelectionSet{
				&ast.Field{Alias: "node", Name: "node", SelectionSet: ast.SelectionSet{
					&ast.Field{Alias: "title", Name: "title"},
				}},
			}},
		},
	}

	_, _, err := tr.translateConnectionField(field, fc, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	params := scope.collect()
	firstOk := false
	offsetOk := false
	for k, v := range params {
		if strings.HasSuffix(k, "_first") {
			if v == 20 || v == int(20) || v == int64(20) {
				firstOk = true
			} else {
				t.Errorf("expected first=20, got %v (%T)", v, v)
			}
		}
		if strings.HasSuffix(k, "_offset") {
			// cursor:4 → decoded 4, +1 = 5
			if v == 5 || v == int(5) || v == int64(5) {
				offsetOk = true
			} else {
				t.Errorf("expected offset=5, got %v (%T)", v, v)
			}
		}
	}
	if !firstOk {
		t.Errorf("first not resolved correctly, params: %v", params)
	}
	if !offsetOk {
		t.Errorf("offset not resolved correctly, params: %v", params)
	}
}
