package translate

import (
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// --- TR-3: Root list query translation tests ---

// Test: translateQuery with a single root list field produces a CALL subquery
// with MATCH, collect(), and RETURN ... AS data.
func TestTranslateQuery_SingleRootField(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	op := makeQueryOp(
		makeField("movies", ast.SelectionSet{
			&ast.Field{Name: "title"},
			&ast.Field{Name: "released"},
		}),
	)

	result, err := tr.translateQuery(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty query, got empty")
	}
	if !strings.Contains(result, "CALL") {
		t.Errorf("expected CALL subquery, got %q", result)
	}
	if !strings.Contains(result, "MATCH") {
		t.Errorf("expected MATCH in query, got %q", result)
	}
	if !strings.Contains(result, "Movie") {
		t.Errorf("expected Movie label in query, got %q", result)
	}
	if !strings.Contains(result, "collect(") {
		t.Errorf("expected collect() in query, got %q", result)
	}
	if !strings.Contains(result, "AS data") {
		t.Errorf("expected 'AS data' in RETURN, got %q", result)
	}
}

// Test: translateQuery with multiple root fields produces multiple CALL subqueries
// combined in RETURN {movies: __movies, actors: __actors} AS data.
func TestTranslateQuery_MultipleRootFields(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	op := makeQueryOp(
		makeField("movies", ast.SelectionSet{
			&ast.Field{Name: "title"},
		}),
		makeField("actors", ast.SelectionSet{
			&ast.Field{Name: "name"},
		}),
	)

	result, err := tr.translateQuery(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have two CALL blocks
	if strings.Count(result, "CALL") < 2 {
		t.Errorf("expected at least 2 CALL subqueries, got %q", result)
	}
	// RETURN should reference both aliases
	if !strings.Contains(result, "movies") {
		t.Errorf("expected 'movies' in RETURN map, got %q", result)
	}
	if !strings.Contains(result, "actors") {
		t.Errorf("expected 'actors' in RETURN map, got %q", result)
	}
}

// Test: translateQuery with filters passes WHERE clause.
func TestTranslateQuery_WithFilter(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"title": strVal("Matrix"),
	}))
	op := makeQueryOp(
		makeField("movies", ast.SelectionSet{
			&ast.Field{Name: "title"},
		}, whereArg),
	)

	result, err := tr.translateQuery(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "WHERE") {
		t.Errorf("expected WHERE clause in filtered query, got %q", result)
	}
}

// Test: translateQuery with sort passes ORDER BY clause.
func TestTranslateQuery_WithSort(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	sortArg := makeArg("sort", makeSortValue(map[string]string{"released": "DESC"}))
	op := makeQueryOp(
		makeField("movies", ast.SelectionSet{
			&ast.Field{Name: "title"},
		}, sortArg),
	)

	result, err := tr.translateQuery(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "ORDER BY") {
		t.Errorf("expected ORDER BY in sorted query, got %q", result)
	}
}

// Test: translateQuery with empty selection set returns minimal data response.
func TestTranslateQuery_EmptyOperation(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	op := makeQueryOp()

	result, err := tr.translateQuery(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return something like "RETURN {} AS data"
	if !strings.Contains(result, "AS data") {
		t.Errorf("expected 'AS data' even for empty operation, got %q", result)
	}
}

// Test: translateRootField produces a CALL subquery with the correct label.
func TestTranslateRootField_ProducesCALLSubquery(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	field := makeField("movies", ast.SelectionSet{
		&ast.Field{Name: "title"},
	})

	cypher, alias, err := tr.translateRootField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cypher == "" {
		t.Fatal("expected non-empty CALL subquery, got empty")
	}
	if alias == "" {
		t.Fatal("expected non-empty alias, got empty")
	}
	if !strings.Contains(cypher, "CALL") {
		t.Errorf("expected CALL in subquery, got %q", cypher)
	}
	if !strings.Contains(cypher, "MATCH") {
		t.Errorf("expected MATCH in subquery, got %q", cypher)
	}
	if !strings.Contains(cypher, "collect(") {
		t.Errorf("expected collect() in subquery, got %q", cypher)
	}
}

// Test: translateRootField returns alias like "__movies".
func TestTranslateRootField_ReturnsAlias(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	field := makeField("movies", ast.SelectionSet{
		&ast.Field{Name: "title"},
	})

	_, alias, err := tr.translateRootField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(alias, "__") {
		t.Errorf("expected alias to start with '__', got %q", alias)
	}
}
