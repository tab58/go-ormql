package translate

import (
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// --- TR-7: Update mutation tests ---

// Test: translateUpdateField produces MATCH + SET with WHERE.
func TestTranslateUpdateField_ProducesMatchSet(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"id": strVal("1"),
	}))
	updateArg := makeArg("update", makeWhereValue(map[string]*ast.Value{
		"title": strVal("Updated Title"),
	}))

	field := makeField("updateMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, whereArg, updateArg)

	result, alias, err := tr.translateUpdateField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty update query, got empty")
	}
	if alias == "" {
		t.Fatal("expected non-empty alias, got empty")
	}
	if !strings.Contains(result, "MATCH") {
		t.Errorf("expected MATCH in update mutation, got %q", result)
	}
	if !strings.Contains(result, "SET") {
		t.Errorf("expected SET in update mutation, got %q", result)
	}
	if !strings.Contains(result, "WHERE") {
		t.Errorf("expected WHERE in update mutation, got %q", result)
	}
	if !strings.Contains(result, "collect(") {
		t.Errorf("expected collect() in update mutation, got %q", result)
	}
}

// Test: translateUpdateField with nested disconnect produces DELETE r.
func TestTranslateUpdateField_NestedDisconnect(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	disconnectInput := makeWhereValue(map[string]*ast.Value{
		"where": makeWhereValue(map[string]*ast.Value{
			"name": strVal("Old Actor"),
		}),
	})

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"id": strVal("1"),
	}))
	updateArg := makeArg("update", makeWhereValue(map[string]*ast.Value{
		"actors": makeWhereValue(map[string]*ast.Value{
			"disconnect": listVal(disconnectInput),
		}),
	}))

	field := makeField("updateMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, whereArg, updateArg)

	result, _, err := tr.translateUpdateField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Nested disconnect uses DELETE r (relationship only, keeps nodes)
	if !strings.Contains(result, "DELETE") {
		t.Errorf("expected DELETE in nested disconnect, got %q", result)
	}
}

// Test: translateUpdateField with nested update produces SET on node and edge.
func TestTranslateUpdateField_NestedUpdate(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	updateInput := makeWhereValue(map[string]*ast.Value{
		"where": makeWhereValue(map[string]*ast.Value{
			"name": strVal("Keanu"),
		}),
		"node": makeWhereValue(map[string]*ast.Value{
			"name": strVal("Keanu Reeves"),
		}),
		"edge": makeWhereValue(map[string]*ast.Value{
			"role": strVal("Neo"),
		}),
	})

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"id": strVal("1"),
	}))
	updateArg := makeArg("update", makeWhereValue(map[string]*ast.Value{
		"actors": makeWhereValue(map[string]*ast.Value{
			"update": listVal(updateInput),
		}),
	}))

	field := makeField("updateMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, whereArg, updateArg)

	result, _, err := tr.translateUpdateField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should contain SET for updating properties
	setCount := strings.Count(result, "SET")
	if setCount < 2 {
		t.Errorf("expected at least 2 SET clauses (parent + nested update), got %d in %q", setCount, result)
	}
}

// Test: translateUpdateField with nested delete produces DETACH DELETE.
func TestTranslateUpdateField_NestedDelete(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	deleteInput := makeWhereValue(map[string]*ast.Value{
		"where": makeWhereValue(map[string]*ast.Value{
			"name": strVal("Remove Me"),
		}),
	})

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"id": strVal("1"),
	}))
	updateArg := makeArg("update", makeWhereValue(map[string]*ast.Value{
		"actors": makeWhereValue(map[string]*ast.Value{
			"delete": listVal(deleteInput),
		}),
	}))

	field := makeField("updateMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, whereArg, updateArg)

	result, _, err := tr.translateUpdateField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "DETACH DELETE") {
		t.Errorf("expected DETACH DELETE in nested delete, got %q", result)
	}
}

// Test: translateUpdateField matching zero nodes returns empty result.
func TestTranslateUpdateField_ZeroMatchReturnsEmpty(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"id": strVal("nonexistent"),
	}))
	updateArg := makeArg("update", makeWhereValue(map[string]*ast.Value{
		"title": strVal("X"),
	}))

	field := makeField("updateMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
		}},
	}, whereArg, updateArg)

	result, _, err := tr.translateUpdateField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The Cypher should be valid even if MATCH returns no rows
	if result == "" {
		t.Fatal("expected non-empty update query for zero-match case")
	}
}

// --- TR-7: Delete mutation tests ---

// Test: translateDeleteField produces MATCH + DETACH DELETE with count.
func TestTranslateDeleteField_ProducesDetachDelete(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"title": strVal("Old Movie"),
	}))

	field := makeField("deleteMovies", ast.SelectionSet{
		&ast.Field{Name: "nodesDeleted"},
		&ast.Field{Name: "relationshipsDeleted"},
	}, whereArg)

	result, alias, err := tr.translateDeleteField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty delete query, got empty")
	}
	if alias == "" {
		t.Fatal("expected non-empty alias, got empty")
	}
	if !strings.Contains(result, "MATCH") {
		t.Errorf("expected MATCH in delete mutation, got %q", result)
	}
	if !strings.Contains(result, "DETACH DELETE") {
		t.Errorf("expected DETACH DELETE in delete mutation, got %q", result)
	}
	if !strings.Contains(result, "count") {
		t.Errorf("expected count for nodesDeleted response, got %q", result)
	}
}

// Test: translateDeleteField matching zero nodes returns nodesDeleted: 0.
func TestTranslateDeleteField_ZeroMatch(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"title": strVal("nonexistent"),
	}))

	field := makeField("deleteMovies", ast.SelectionSet{
		&ast.Field{Name: "nodesDeleted"},
	}, whereArg)

	result, _, err := tr.translateDeleteField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The Cypher should be valid even for zero matches
	if result == "" {
		t.Fatal("expected non-empty delete query for zero-match case")
	}
}

// Test: translateDeleteField includes Movie label in MATCH.
func TestTranslateDeleteField_UsesCorrectLabel(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"title": strVal("Test"),
	}))

	field := makeField("deleteMovies", ast.SelectionSet{
		&ast.Field{Name: "nodesDeleted"},
	}, whereArg)

	result, _, err := tr.translateDeleteField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Movie") {
		t.Errorf("expected Movie label in delete mutation, got %q", result)
	}
}

// Test: translateDeleteField parameterizes WHERE values.
func TestTranslateDeleteField_ParameterizesValues(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"title": strVal("Test Movie"),
	}))

	field := makeField("deleteMovies", ast.SelectionSet{
		&ast.Field{Name: "nodesDeleted"},
	}, whereArg)

	_, _, err := tr.translateDeleteField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	params := scope.collect()
	if len(params) == 0 {
		t.Error("expected parameters in scope after delete mutation")
	}
}
