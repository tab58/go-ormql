package translate

import (
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// === FE-1: TranslationPlan type + Translate() signature change ===

// Test: Translate() returns TranslationPlan with empty WriteStatements for queries.
// Expected: plan.WriteStatements is nil/empty for query operations.
func TestTranslationPlan_QueryHasEmptyWriteStatements(t *testing.T) {
	tr := New(testModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Name: "title", Alias: "title"},
				}),
			),
		},
	}
	op := doc.Operations[0]

	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.WriteStatements) != 0 {
		t.Errorf("expected empty WriteStatements for query, got %d", len(plan.WriteStatements))
	}
}

// Test: Translate() returns TranslationPlan with non-empty ReadStatement for queries.
// Expected: plan.ReadStatement.Query is non-empty for query operations.
func TestTranslationPlan_QueryHasNonEmptyReadStatement(t *testing.T) {
	tr := New(testModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("movies", ast.SelectionSet{
					&ast.Field{Name: "title", Alias: "title"},
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
		t.Error("expected non-empty ReadStatement.Query for query operation")
	}
}

// Test: Translate() returns TranslationPlan with non-empty WriteStatements for merge mutations.
// Expected: plan.WriteStatements has one entry per merge field (FOREACH write).
// FAILS RED: current stub does not populate WriteStatements for merge mutations.
func TestTranslationPlan_MergeMutationHasWriteStatements(t *testing.T) {
	tr := New(mergeTestModel())

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	op := makeMutationOp(
		makeField("mergeMovies", ast.SelectionSet{
			&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "title"},
			}},
		}, makeArg("input", inputVal)),
	)

	doc := &ast.QueryDocument{Operations: ast.OperationList{op}}
	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.WriteStatements) == 0 {
		t.Error("expected non-empty WriteStatements for merge mutation, got 0")
	}
}

// Test: Translate() for merge mutation produces WriteStatements with FOREACH Cypher.
// Expected: each WriteStatement.Query contains FOREACH.
// FAILS RED: current stub does not produce FOREACH writes.
func TestTranslationPlan_MergeWriteStatementContainsForeach(t *testing.T) {
	tr := New(mergeTestModel())

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	op := makeMutationOp(
		makeField("mergeMovies", ast.SelectionSet{
			&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "title"},
			}},
		}, makeArg("input", inputVal)),
	)

	doc := &ast.QueryDocument{Operations: ast.OperationList{op}}
	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.WriteStatements) == 0 {
		t.Fatal("expected WriteStatements for merge mutation")
	}
	for i, ws := range plan.WriteStatements {
		if !strings.Contains(ws.Query, "FOREACH") {
			t.Errorf("WriteStatements[%d] expected FOREACH, got %q", i, ws.Query)
		}
	}
}

// Test: Translate() for merge mutation still has a ReadStatement with MATCH for projection.
// Expected: plan.ReadStatement.Query is non-empty and contains MATCH.
// FAILS RED: current stub puts everything in ReadStatement (UNWIND+MERGE), not split.
func TestTranslationPlan_MergeReadStatementContainsMatch(t *testing.T) {
	tr := New(mergeTestModel())

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	op := makeMutationOp(
		makeField("mergeMovies", ast.SelectionSet{
			&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "title"},
			}},
		}, makeArg("input", inputVal)),
	)

	doc := &ast.QueryDocument{Operations: ast.OperationList{op}}
	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.ReadStatement.Query == "" {
		t.Error("expected non-empty ReadStatement.Query for merge mutation")
	}
	// The read statement should use MATCH to read back merged nodes, not UNWIND+MERGE
	if strings.Contains(plan.ReadStatement.Query, "MERGE") {
		t.Error("ReadStatement should NOT contain MERGE — merge writes belong in WriteStatements")
	}
}

// Test: Translate() for non-merge mutation (create) has empty WriteStatements.
// Expected: create-only mutations don't produce WriteStatements.
func TestTranslationPlan_CreateMutationHasEmptyWriteStatements(t *testing.T) {
	tr := New(testModel())

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

	doc := &ast.QueryDocument{Operations: ast.OperationList{op}}
	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.WriteStatements) != 0 {
		t.Errorf("expected empty WriteStatements for create mutation, got %d", len(plan.WriteStatements))
	}
}

// Test: Translate() still returns error for subscriptions after TranslationPlan change.
// Expected: error for subscription operation.
func TestTranslationPlan_SubscriptionStillErrors(t *testing.T) {
	tr := New(testModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			{Operation: ast.Subscription, Name: "OnMovieCreated"},
		},
	}
	op := doc.Operations[0]

	_, err := tr.Translate(doc, op, nil)
	if err == nil {
		t.Error("expected error for subscription operation after TranslationPlan change")
	}
}
