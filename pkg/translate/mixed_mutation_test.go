package translate

import (
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// === FE-5: Mixed mutation tests ===
//
// Comprehensive tests for mixed mutation operations via translateMutationSplit:
// - create+merge: 1 FOREACH write, read has CREATE CALL + MATCH CALL
// - merge+delete: 1 FOREACH write, read has MATCH CALL + DELETE CALL
// - merge+connect: 1 FOREACH write, read has MATCH CALL + connect UNWIND CALL
// - multiple merge fields: multiple FOREACH writes, read has multiple MATCH CALLs
//
// All tests verify: all writes produced before read, correct Cypher content.

// Test: create+merge produces 1 write and read with both CREATE and MATCH CALL blocks.
// Expected: writes[0] has FOREACH, read has CREATE and MATCH, both in AS data RETURN.
// FAILS RED: translateMutationSplit returns nil writes for merge fields.
func TestMixedMutation_CreatePlusMerge(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	createInput := listVal(
		makeWhereValue(map[string]*ast.Value{
			"title":    strVal("New Movie"),
			"released": intVal("2024"),
		}),
	)
	mergeInput := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("Old Movie"),
			}),
		}),
	)

	op := makeMutationOp(
		makeField("createMovies", ast.SelectionSet{
			&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "title"},
			}},
		}, makeArg("input", createInput)),
		makeField("mergeMovies", ast.SelectionSet{
			&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "title"},
			}},
		}, makeArg("input", mergeInput)),
	)

	writes, read, err := tr.translateMutationSplit(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1 FOREACH write from merge
	if len(writes) != 1 {
		t.Errorf("expected 1 write query, got %d", len(writes))
	}
	if len(writes) > 0 && !strings.Contains(writes[0], "FOREACH") {
		t.Errorf("writes[0] should contain FOREACH, got %q", writes[0])
	}

	// Read should have both CREATE CALL and MATCH CALL (for merge read)
	if !strings.Contains(read, "CREATE") {
		t.Errorf("read should contain CREATE CALL, got %q", read)
	}
	if !strings.Contains(read, "MATCH") {
		t.Errorf("read should contain MATCH CALL for merge, got %q", read)
	}
	if !strings.Contains(read, "AS data") {
		t.Errorf("read should contain 'AS data', got %q", read)
	}
}

// Test: merge+delete produces 1 write and read with MATCH + DELETE CALL blocks.
// Expected: writes[0] has FOREACH, read has MATCH (for merge) and DETACH DELETE.
// FAILS RED: translateMutationSplit returns nil writes for merge fields.
func TestMixedMutation_MergePlusDelete(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	mergeInput := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)
	deleteWhere := makeWhereValue(map[string]*ast.Value{
		"title": strVal("Bad Movie"),
	})

	op := makeMutationOp(
		makeField("mergeMovies", ast.SelectionSet{
			&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "title"},
			}},
		}, makeArg("input", mergeInput)),
		makeField("deleteMovies", ast.SelectionSet{
			&ast.Field{Name: "nodesDeleted"},
		}, makeArg("where", deleteWhere)),
	)

	writes, read, err := tr.translateMutationSplit(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(writes) != 1 {
		t.Errorf("expected 1 write query, got %d", len(writes))
	}
	if !strings.Contains(read, "DETACH DELETE") {
		t.Errorf("read should contain DETACH DELETE, got %q", read)
	}
}

// Test: merge+connect produces 2 writes and read with MATCH CALL + count CALL.
// Expected: writes[0] has FOREACH (merge), writes[1] has UNWIND+MERGE (connect).
// Read has MATCH (merge projection) and size() (connect count).
func TestMixedMutation_MergePlusConnect(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	mergeInput := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)
	connectInput := listVal(
		makeWhereValue(map[string]*ast.Value{
			"from": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
			"to": makeWhereValue(map[string]*ast.Value{
				"name": strVal("Keanu Reeves"),
			}),
		}),
	)

	op := makeMutationOp(
		makeField("mergeMovies", ast.SelectionSet{
			&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "title"},
			}},
		}, makeArg("input", mergeInput)),
		makeField("connectMovieActors", ast.SelectionSet{
			&ast.Field{Name: "relationshipsCreated"},
		}, makeArg("input", connectInput)),
	)

	writes, read, err := tr.translateMutationSplit(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 writes: FOREACH from merge + UNWIND+MERGE from connect
	if len(writes) != 2 {
		t.Errorf("expected 2 write queries (merge + connect), got %d", len(writes))
	}
	if len(writes) > 0 && !strings.Contains(writes[0], "FOREACH") {
		t.Errorf("writes[0] should contain FOREACH (merge), got %q", writes[0])
	}
	if len(writes) > 1 && !strings.Contains(writes[1], "MERGE") {
		t.Errorf("writes[1] should contain MERGE (connect), got %q", writes[1])
	}
	// Read should NOT contain UNWIND for connect (write is separate now)
	// Read should contain size() for the connect count
	if !strings.Contains(read, "size") {
		t.Errorf("read should contain size() for connect count, got %q", read)
	}
}

// Test: Multiple merge fields produce multiple write queries.
// Expected: 2 writes (one per merge field), read has 2 MATCH CALL blocks.
// FAILS RED: translateMutationSplit returns nil writes for merge fields.
func TestMixedMutation_MultipleMerges(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	mergeMoviesInput := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("Movie 1"),
			}),
		}),
	)
	mergeActorsInput := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"name": strVal("Actor 1"),
			}),
		}),
	)

	op := makeMutationOp(
		makeField("mergeMovies", ast.SelectionSet{
			&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "title"},
			}},
		}, makeArg("input", mergeMoviesInput)),
		makeField("mergeActors", ast.SelectionSet{
			&ast.Field{Name: "actors", SelectionSet: ast.SelectionSet{
				&ast.Field{Name: "id"},
				&ast.Field{Name: "name"},
			}},
		}, makeArg("input", mergeActorsInput)),
	)

	writes, read, err := tr.translateMutationSplit(op, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(writes) != 2 {
		t.Errorf("expected 2 write queries (one per merge), got %d", len(writes))
	}
	// Each write should contain FOREACH
	for i, w := range writes {
		if !strings.Contains(w, "FOREACH") {
			t.Errorf("writes[%d] should contain FOREACH, got %q", i, w)
		}
	}
	// Read should have 2 MATCH blocks (one per merge read projection)
	matchCount := strings.Count(read, "MATCH")
	if matchCount < 2 {
		t.Errorf("read should contain at least 2 MATCH blocks for 2 merge reads, got %d in %q", matchCount, read)
	}
	if !strings.Contains(read, "AS data") {
		t.Errorf("read should contain 'AS data', got %q", read)
	}
}
