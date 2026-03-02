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

	result := tr.buildOrderBy(sort, "n")
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

	result := tr.buildOrderBy(sort, "n")
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

	result := tr.buildOrderBy(sort, "n")
	if !strings.Contains(result, "ASC") {
		t.Errorf("expected ASC in ORDER BY, got %q", result)
	}
}

// Test: Nil sort argument returns empty string.
func TestBuildOrderBy_NilSort(t *testing.T) {
	tr := New(testModel())

	result := tr.buildOrderBy(nil, "n")
	if result != "" {
		t.Errorf("expected empty ORDER BY for nil sort, got %q", result)
	}
}

// Test: Empty sort list returns empty string.
func TestBuildOrderBy_EmptyList(t *testing.T) {
	tr := New(testModel())
	sort := &ast.Value{Kind: ast.ListValue, Children: nil}

	result := tr.buildOrderBy(sort, "n")
	if result != "" {
		t.Errorf("expected empty ORDER BY for empty list, got %q", result)
	}
}

// Test: Different variable name is used in property access.
func TestBuildOrderBy_UsesVariable(t *testing.T) {
	tr := New(testModel())
	sort := makeSortValue(map[string]string{"title": "ASC"})

	result := tr.buildOrderBy(sort, "a")
	if !strings.Contains(result, "a.title") {
		t.Errorf("expected a.title in ORDER BY, got %q", result)
	}
}
