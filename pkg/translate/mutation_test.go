package translate

import (
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// --- TR-6: Create mutation translation tests ---

// makeMutationOp creates a mutation operation with root fields.
func makeMutationOp(fields ...*ast.Field) *ast.OperationDefinition {
	selSet := make(ast.SelectionSet, len(fields))
	for i, f := range fields {
		selSet[i] = f
	}
	return &ast.OperationDefinition{
		Operation:    ast.Mutation,
		SelectionSet: selSet,
	}
}

// Test: translateMutation with a single createMovies field produces a CALL subquery
// with UNWIND, CREATE, and collect().
func TestTranslateMutation_SingleCreate(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"title":    strVal("New Movie"),
			"released": intVal("2024"),
		}),
	)

	op := makeMutationOp(
		makeField("createMovies", ast.SelectionSet{
			&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "title"},
			}},
		}, makeArg("input", inputVal)),
	)

	result, err := tr.translateMutation(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty mutation query, got empty")
	}
	if !strings.Contains(result, "CALL") {
		t.Errorf("expected CALL in mutation, got %q", result)
	}
	if !strings.Contains(result, "AS data") {
		t.Errorf("expected 'AS data' in RETURN, got %q", result)
	}
}

// Test: translateCreateField produces UNWIND + CREATE with SET for properties.
func TestTranslateCreateField_ProducesUnwindCreate(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"title":    strVal("New Movie"),
			"released": intVal("2024"),
		}),
	)

	field := makeField("createMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	result, alias, err := tr.translateCreateField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty create query, got empty")
	}
	if alias == "" {
		t.Fatal("expected non-empty alias, got empty")
	}
	if !strings.Contains(result, "UNWIND") {
		t.Errorf("expected UNWIND in create mutation, got %q", result)
	}
	if !strings.Contains(result, "CREATE") {
		t.Errorf("expected CREATE in create mutation, got %q", result)
	}
	if !strings.Contains(result, "Movie") {
		t.Errorf("expected Movie label in create mutation, got %q", result)
	}
	if !strings.Contains(result, "SET") {
		t.Errorf("expected SET in create mutation, got %q", result)
	}
	if !strings.Contains(result, "collect(") {
		t.Errorf("expected collect() in create mutation, got %q", result)
	}
}

// Test: translateCreateField generates randomUUID() for ID! fields.
func TestTranslateCreateField_GeneratesUUID(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"title": strVal("New Movie"),
		}),
	)

	field := makeField("createMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateCreateField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "randomUUID()") {
		t.Errorf("expected randomUUID() for ID field in create mutation, got %q", result)
	}
}

// Test: translateCreateField with nested create produces nested CALL subquery.
func TestTranslateCreateField_NestedCreate(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	nestedCreate := makeWhereValue(map[string]*ast.Value{
		"node": makeWhereValue(map[string]*ast.Value{
			"name": strVal("New Actor"),
		}),
		"edge": makeWhereValue(map[string]*ast.Value{
			"role": strVal("Lead"),
		}),
	})

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"title": strVal("New Movie"),
			"actors": makeWhereValue(map[string]*ast.Value{
				"create": listVal(nestedCreate),
			}),
		}),
	)

	field := makeField("createMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateCreateField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should contain nested CALL for create
	callCount := strings.Count(result, "CALL")
	if callCount < 2 {
		t.Errorf("expected at least 2 CALL blocks for nested create, got %d in %q", callCount, result)
	}
	if !strings.Contains(result, "Actor") {
		t.Errorf("expected Actor label in nested create, got %q", result)
	}
}

// Test: translateCreateField with nested connect produces MATCH + MERGE.
func TestTranslateCreateField_NestedConnect(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	nestedConnect := makeWhereValue(map[string]*ast.Value{
		"where": makeWhereValue(map[string]*ast.Value{
			"name": strVal("Existing Actor"),
		}),
		"edge": makeWhereValue(map[string]*ast.Value{
			"role": strVal("Support"),
		}),
	})

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"title": strVal("New Movie"),
			"actors": makeWhereValue(map[string]*ast.Value{
				"connect": listVal(nestedConnect),
			}),
		}),
	)

	field := makeField("createMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateCreateField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "MERGE") || !strings.Contains(result, "MATCH") {
		t.Errorf("expected MATCH + MERGE in nested connect, got %q", result)
	}
}

// Test: translateCreateField with empty input array produces valid Cypher
// (UNWIND [] returns no rows, collect returns []).
func TestTranslateCreateField_EmptyInput(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	inputVal := listVal() // empty list

	field := makeField("createMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateCreateField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should still produce valid Cypher even with empty input
	if result == "" {
		t.Fatal("expected non-empty create query for empty input")
	}
	if !strings.Contains(result, "UNWIND") {
		t.Errorf("expected UNWIND even for empty input, got %q", result)
	}
}

// --- H3 regression: extractNodeName empty string guard ---

// Test: extractNodeName returns false for field name equal to the prefix (empty plural).
// Before H3 fix, this panicked with index out of range.
func TestExtractNodeName_EmptyPlural(t *testing.T) {
	tr := New(testModel())
	_, ok := tr.extractNodeName("create", "create")
	if ok {
		t.Error("expected false for empty plural after prefix removal")
	}
}

// Test: findNodeByPluralName skips nodes with empty Name.
func TestFindNodeByPluralName_EmptyNodeName(t *testing.T) {
	model := testModel()
	model.Nodes = append(model.Nodes, schema.NodeDefinition{Name: "", Labels: []string{}})
	tr := New(model)
	_, ok := tr.findNodeByPluralName("movies")
	if !ok {
		t.Error("expected to find Movie node even with empty-name node in model")
	}
}

// Test: translateCreateField parameterizes input values (not inlined).
func TestTranslateCreateField_ParameterizesInput(t *testing.T) {
	tr := New(testModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"title": strVal("New Movie"),
		}),
	)

	field := makeField("createMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	_, _, err := tr.translateCreateField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	params := scope.collect()
	if len(params) == 0 {
		t.Error("expected parameters in scope after create mutation")
	}
}
