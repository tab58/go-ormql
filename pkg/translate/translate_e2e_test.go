package translate

import (
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// --- TR-8: Translate entry point + multi-operation tests ---

// Test: Translate() for a query operation produces a valid cypher.Statement
// with non-empty Query and Params.
func TestTranslate_QueryProducesStatement(t *testing.T) {
	tr := New(testModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Name: "title"},
				}),
			),
		},
	}
	op := doc.Operations[0]

	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.ReadStatement.Query == "" {
		t.Fatal("expected non-empty Query in Statement, got empty")
	}
	if !strings.Contains(plan.ReadStatement.Query, "CALL") {
		t.Errorf("expected CALL in Query, got %q", plan.ReadStatement.Query)
	}
	if !strings.Contains(plan.ReadStatement.Query, "AS data") {
		t.Errorf("expected 'AS data' in Query, got %q", plan.ReadStatement.Query)
	}
}

// Test: Translate() for a mutation operation produces a valid cypher.Statement.
func TestTranslate_MutationProducesStatement(t *testing.T) {
	tr := New(testModel())

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"title": strVal("New Movie"),
		}),
	)

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeMutationOp(
				makeField("createMovies", ast.SelectionSet{
					&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
						&ast.Field{Name: "id"},
						&ast.Field{Name: "title"},
					}},
				}, makeArg("input", inputVal)),
			),
		},
	}
	op := doc.Operations[0]

	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.ReadStatement.Query == "" {
		t.Fatal("expected non-empty Query in Statement for mutation, got empty")
	}
}

// Test: Translate() returns error for subscription operation.
func TestTranslate_SubscriptionError(t *testing.T) {
	tr := New(testModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			{Operation: ast.Subscription, Name: "OnMovieCreated"},
		},
	}
	op := doc.Operations[0]

	_, err := tr.Translate(doc, op, nil)
	if err == nil {
		t.Fatal("expected error for subscription, got nil")
	}
	if !strings.Contains(err.Error(), "subscription") && !strings.Contains(err.Error(), "Subscription") {
		t.Errorf("expected error message to mention subscription, got %q", err.Error())
	}
}

// Test: Translate() collects all parameters from scope into Statement.Params.
func TestTranslate_CollectsParams(t *testing.T) {
	tr := New(testModel())

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"title": strVal("Matrix"),
	}))

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Name: "title"},
				}, whereArg),
			),
		},
	}
	op := doc.Operations[0]

	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.ReadStatement.Params == nil {
		t.Fatal("expected non-nil Params in Statement")
	}
	if len(plan.ReadStatement.Params) == 0 {
		t.Error("expected at least one parameter in Statement.Params for filtered query")
	}
}

// Test: Translate() with nil variables treats as empty map (no panic).
func TestTranslate_NilVariables(t *testing.T) {
	tr := New(testModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Name: "title"},
				}),
			),
		},
	}
	op := doc.Operations[0]

	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error with nil variables: %v", err)
	}
	if plan.ReadStatement.Query == "" {
		t.Fatal("expected non-empty Query with nil variables")
	}
}

// Test: Translate() with multiple root query fields produces combined RETURN.
func TestTranslate_MultipleRootFields(t *testing.T) {
	tr := New(testModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Name: "title"},
				}),
				makeField("actors", ast.SelectionSet{
					&ast.Field{Name: "name"},
				}),
			),
		},
	}
	op := doc.Operations[0]

	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Count(plan.ReadStatement.Query, "CALL") < 2 {
		t.Errorf("expected at least 2 CALL subqueries for 2 root fields, got %q", plan.ReadStatement.Query)
	}
	if !strings.Contains(plan.ReadStatement.Query, "movies") {
		t.Errorf("expected 'movies' in combined RETURN, got %q", plan.ReadStatement.Query)
	}
	if !strings.Contains(plan.ReadStatement.Query, "actors") {
		t.Errorf("expected 'actors' in combined RETURN, got %q", plan.ReadStatement.Query)
	}
}

// Test: End-to-end query with filter + sort + nested relationship.
func TestTranslate_E2E_QueryFilterSortNested(t *testing.T) {
	tr := New(testModel())

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"title_contains": strVal("Matrix"),
		"released_gte":   intVal("1999"),
	}))
	sortArg := makeArg("sort", makeSortValue(map[string]string{"released": "DESC"}))

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Name: "title"},
					&ast.Field{Name: "released"},
					&ast.Field{Name: "actors", SelectionSet: ast.SelectionSet{
						&ast.Field{Name: "name"},
					}},
				}, whereArg, sortArg),
			),
		},
	}
	op := doc.Operations[0]

	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.ReadStatement.Query == "" {
		t.Fatal("expected non-empty E2E query")
	}
	if !strings.Contains(plan.ReadStatement.Query, "WHERE") {
		t.Errorf("expected WHERE in E2E query, got %q", plan.ReadStatement.Query)
	}
	if !strings.Contains(plan.ReadStatement.Query, "ORDER BY") {
		t.Errorf("expected ORDER BY in E2E query, got %q", plan.ReadStatement.Query)
	}
	if !strings.Contains(plan.ReadStatement.Query, "AS data") {
		t.Errorf("expected 'AS data' in E2E query, got %q", plan.ReadStatement.Query)
	}
}

// Test: End-to-end create mutation with nested create.
func TestTranslate_E2E_CreateWithNested(t *testing.T) {
	tr := New(testModel())

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

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeMutationOp(
				makeField("createMovies", ast.SelectionSet{
					&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
						&ast.Field{Name: "id"},
						&ast.Field{Name: "title"},
					}},
				}, makeArg("input", inputVal)),
			),
		},
	}
	op := doc.Operations[0]

	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.ReadStatement.Query == "" {
		t.Fatal("expected non-empty E2E create mutation")
	}
	if !strings.Contains(plan.ReadStatement.Query, "CREATE") {
		t.Errorf("expected CREATE in E2E mutation, got %q", plan.ReadStatement.Query)
	}
	if !strings.Contains(plan.ReadStatement.Query, "AS data") {
		t.Errorf("expected 'AS data' in E2E mutation, got %q", plan.ReadStatement.Query)
	}
	if plan.ReadStatement.Params == nil {
		t.Fatal("expected non-nil Params in E2E mutation Statement")
	}
}

// Test: End-to-end delete mutation.
func TestTranslate_E2E_Delete(t *testing.T) {
	tr := New(testModel())

	whereArg := makeArg("where", makeWhereValue(map[string]*ast.Value{
		"title": strVal("Old Movie"),
	}))

	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeMutationOp(
				makeField("deleteMovies", ast.SelectionSet{
					&ast.Field{Name: "nodesDeleted"},
					&ast.Field{Name: "relationshipsDeleted"},
				}, whereArg),
			),
		},
	}
	op := doc.Operations[0]

	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.ReadStatement.Query == "" {
		t.Fatal("expected non-empty E2E delete mutation")
	}
	if !strings.Contains(plan.ReadStatement.Query, "DETACH DELETE") {
		t.Errorf("expected DETACH DELETE in E2E delete mutation, got %q", plan.ReadStatement.Query)
	}
}
