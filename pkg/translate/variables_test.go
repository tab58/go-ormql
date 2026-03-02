package translate

import (
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// =============================================================================
// VAR-1: paramScope.variables field + sub() inheritance + Translate() wiring
// =============================================================================

// Test: paramScope has a variables field that is nil by default for root scopes.
// Expected: newParamScope() creates a scope with nil variables.
func TestParamScope_Variables_NilByDefault(t *testing.T) {
	s := newParamScope()
	if s.variables != nil {
		t.Errorf("expected nil variables on new root scope, got %v", s.variables)
	}
}

// Test: sub() propagates the parent's variables map to the child scope (same pointer).
// Expected: child.variables == parent.variables when parent has non-nil variables.
// FAILS: sub() does not set child.variables = s.variables (not yet implemented).
func TestParamScope_Sub_InheritsVariables(t *testing.T) {
	s := newParamScope()
	s.variables = map[string]any{"title": "Matrix", "year": float64(1999)}

	child := s.sub("sub0")
	if child.variables == nil {
		t.Fatal("expected child scope to inherit parent's variables, got nil")
	}
	if child.variables["title"] != "Matrix" {
		t.Errorf("expected child.variables[title]='Matrix', got %v", child.variables["title"])
	}
	if child.variables["year"] != float64(1999) {
		t.Errorf("expected child.variables[year]=1999, got %v", child.variables["year"])
	}
}

// Test: sub() shares the same variables map reference (not a copy).
// Expected: Modifying parent.variables is visible through child.variables.
// FAILS: sub() does not set child.variables = s.variables (not yet implemented).
func TestParamScope_Sub_SharesVariablesReference(t *testing.T) {
	s := newParamScope()
	vars := map[string]any{"title": "Matrix"}
	s.variables = vars

	child := s.sub("sub0")

	// They should point to the same underlying map
	if child.variables == nil {
		t.Fatal("expected child scope to share parent's variables reference, got nil")
	}
	// Verify it's the same reference (not a copy) — reading the same data proves sharing.
	vars["newKey"] = "newVal"
	if child.variables["newKey"] != "newVal" {
		t.Error("expected child scope variables to be the same reference as parent")
	}
}

// Test: Deeply nested sub() scopes all inherit the same variables map.
// Expected: grandchild.variables == parent.variables.
// FAILS: sub() does not set child.variables = s.variables (not yet implemented).
func TestParamScope_DeepSub_InheritsVariables(t *testing.T) {
	s := newParamScope()
	s.variables = map[string]any{"limit": float64(10)}

	child := s.sub("sub0")
	grandchild := child.sub("sub1")

	if grandchild.variables == nil {
		t.Fatal("expected grandchild scope to inherit variables, got nil")
	}
	if grandchild.variables["limit"] != float64(10) {
		t.Errorf("expected grandchild.variables[limit]=10, got %v", grandchild.variables["limit"])
	}
}

// Test: Translate() sets variables on the root scope before dispatching.
// Expected: After Translate() with non-nil variables, the generated Cypher
// uses the variables map for variable resolution.
// We verify this indirectly: Translate() with an empty query operation and non-nil
// variables should succeed (regression guard — no behavioral change needed yet).
func TestTranslate_SetsVariablesOnScope(t *testing.T) {
	tr := New(testModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			{Operation: ast.Query, Name: "GetMovies"},
		},
	}
	op := doc.Operations[0]
	variables := map[string]any{"title": "Matrix"}

	// Should not error — variables are accepted but not yet used in the query
	_, err := tr.Translate(doc, op, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Test: collect() is unaffected by the variables field.
// Expected: collect() still returns only params (not variables).
// This is a regression guard — variables should not leak into params.
func TestParamScope_Collect_ExcludesVariables(t *testing.T) {
	s := newParamScope()
	s.variables = map[string]any{"title": "Matrix", "year": float64(1999)}
	s.add("paramValue")

	params := s.collect()
	if len(params) != 1 {
		t.Errorf("expected 1 param in collect(), got %d (variables may have leaked)", len(params))
	}
	if _, ok := params["title"]; ok {
		t.Error("variables should not appear in collect() output")
	}
}

// Test: sub() with nil variables on parent produces nil variables on child.
// Expected: If parent has no variables, child also has nil (not an empty map).
func TestParamScope_Sub_NilVariablesPropagatesNil(t *testing.T) {
	s := newParamScope()
	// Do NOT set variables — they're nil

	child := s.sub("sub0")
	if child.variables != nil {
		t.Errorf("expected nil variables on child when parent has nil, got %v", child.variables)
	}
}
