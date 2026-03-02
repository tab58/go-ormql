package translate

import (
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// makeSortValue creates a sort argument ast.Value from a list of field→direction pairs.
// Each pair is a map like {"released": "DESC"}.
func makeSortValue(pairs ...map[string]string) *ast.Value {
	items := make(ast.ChildValueList, 0, len(pairs))
	for _, pair := range pairs {
		children := make(ast.ChildValueList, 0, len(pair))
		for field, direction := range pair {
			children = append(children, &ast.ChildValue{
				Name:  field,
				Value: &ast.Value{Kind: ast.EnumValue, Raw: direction},
			})
		}
		items = append(items, &ast.ChildValue{
			Value: &ast.Value{Kind: ast.ObjectValue, Children: children},
		})
	}
	return &ast.Value{Kind: ast.ListValue, Children: items}
}

// Test: Single field sort produces "n.released DESC".
func TestBuildOrderBy_SingleField(t *testing.T) {
	tr := New(testModel())
	sort := makeSortValue(map[string]string{"released": "DESC"})

	result := tr.buildOrderBy(sort, "n", nil)
	if result == "" {
		t.Fatal("expected non-empty ORDER BY clause, got empty")
	}
	if !strings.Contains(result, "n.released") {
		t.Errorf("expected n.released in ORDER BY, got %q", result)
	}
	if !strings.Contains(result, "DESC") {
		t.Errorf("expected DESC in ORDER BY, got %q", result)
	}
}

// Test: Multiple fields in sort produce comma-separated "n.released DESC, n.title ASC".
func TestBuildOrderBy_MultipleFields(t *testing.T) {
	tr := New(testModel())
	sort := makeSortValue(
		map[string]string{"released": "DESC"},
		map[string]string{"title": "ASC"},
	)

	result := tr.buildOrderBy(sort, "n", nil)
	if !strings.Contains(result, "n.released") {
		t.Errorf("expected n.released in ORDER BY, got %q", result)
	}
	if !strings.Contains(result, "n.title") {
		t.Errorf("expected n.title in ORDER BY, got %q", result)
	}
}

// Test: ASC direction produces ascending sort.
func TestBuildOrderBy_ASC(t *testing.T) {
	tr := New(testModel())
	sort := makeSortValue(map[string]string{"title": "ASC"})

	result := tr.buildOrderBy(sort, "n", nil)
	if !strings.Contains(result, "ASC") {
		t.Errorf("expected ASC in ORDER BY, got %q", result)
	}
}

// Test: Nil sort argument returns empty string.
func TestBuildOrderBy_NilSort(t *testing.T) {
	tr := New(testModel())

	result := tr.buildOrderBy(nil, "n", nil)
	if result != "" {
		t.Errorf("expected empty ORDER BY for nil sort, got %q", result)
	}
}

// Test: Empty sort list returns empty string.
func TestBuildOrderBy_EmptyList(t *testing.T) {
	tr := New(testModel())
	sort := &ast.Value{Kind: ast.ListValue, Children: nil}

	result := tr.buildOrderBy(sort, "n", nil)
	if result != "" {
		t.Errorf("expected empty ORDER BY for empty list, got %q", result)
	}
}

// Test: Different variable name is used in property access.
func TestBuildOrderBy_UsesVariable(t *testing.T) {
	tr := New(testModel())
	sort := makeSortValue(map[string]string{"title": "ASC"})

	result := tr.buildOrderBy(sort, "a", nil)
	if !strings.Contains(result, "a.title") {
		t.Errorf("expected a.title in ORDER BY, got %q", result)
	}
}

// Test: buildOrderByFromGo with multi-key sort object produces deterministic ordering.
// Expected: {"released": "DESC", "title": "ASC"} → "n.released DESC, n.title ASC"
// (alphabetical key order: released < title)
func TestBuildOrderByFromGo_MultiKeyDeterministic(t *testing.T) {
	resolved := []any{map[string]any{"released": "DESC", "title": "ASC"}}

	// Run multiple times to verify determinism
	first := buildOrderByFromGo(resolved, "n")
	for i := 0; i < 20; i++ {
		result := buildOrderByFromGo(resolved, "n")
		if result != first {
			t.Errorf("non-deterministic ordering on iteration %d: got %q, first was %q", i, result, first)
		}
	}

	// Verify alphabetical ordering: released < title
	expected := "n.released DESC, n.title ASC"
	if first != expected {
		t.Errorf("expected %q, got %q", expected, first)
	}
}
