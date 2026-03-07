package translate

import (
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// === TR-10: translateMergeField tests ===

// mergeTestModel returns a model for merge/connect tests with Movie and Actor.
// Movie has: id, title, released. Actor has: id, name.
// Relationship: Movie.actors ACTED_IN(IN) Actor with ActedInProperties(role).
func mergeTestModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
					{Name: "released", GraphQLType: "Int", GoType: "*int", CypherType: "INTEGER", Nullable: true},
				},
			},
			{
				Name:   "Actor",
				Labels: []string{"Actor"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
				},
			},
		},
		Relationships: []schema.RelationshipDefinition{
			{
				FieldName: "actors",
				RelType:   "ACTED_IN",
				Direction: schema.DirectionIN,
				FromNode:  "Movie",
				ToNode:    "Actor",
				IsList:    true,
				Properties: &schema.PropertiesDefinition{
					TypeName: "ActedInProperties",
					Fields: []schema.FieldDefinition{
						{Name: "role", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
					},
				},
			},
		},
	}
}

// Test: translateMergeField produces UNWIND for batched processing.
// Expected: output contains UNWIND $p0 AS item
func TestTranslateMergeField_ProducesUnwind(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	field := makeField("mergeMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateMergeField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty merge query, got empty")
	}
	if !strings.Contains(result, "UNWIND") {
		t.Errorf("expected UNWIND in merge mutation, got %q", result)
	}
}

// Test: translateMergeField produces MERGE with match keys.
// Expected: output contains MERGE (n:Movie {title: item.match.title})
func TestTranslateMergeField_ProducesMergeWithMatchKeys(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	field := makeField("mergeMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateMergeField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "MERGE") {
		t.Errorf("expected MERGE in merge mutation, got %q", result)
	}
	if !strings.Contains(result, "Movie") {
		t.Errorf("expected Movie label in merge mutation, got %q", result)
	}
}

// Test: translateMergeField produces ON CREATE SET with randomUUID() for id.
// Expected: output contains ON CREATE SET and randomUUID()
func TestTranslateMergeField_OnCreateSetWithRandomUUID(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("New Movie"),
			}),
			"onCreate": makeWhereValue(map[string]*ast.Value{
				"released": intVal("2024"),
			}),
		}),
	)

	field := makeField("mergeMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateMergeField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "ON CREATE SET") {
		t.Errorf("expected ON CREATE SET in merge mutation, got %q", result)
	}
	if !strings.Contains(result, "randomUUID()") {
		t.Errorf("expected randomUUID() for id in ON CREATE SET, got %q", result)
	}
}

// Test: translateMergeField produces ON MATCH SET with COALESCE for non-null update.
// Expected: output contains ON MATCH SET and COALESCE
func TestTranslateMergeField_OnMatchSetWithCoalesce(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
			"onMatch": makeWhereValue(map[string]*ast.Value{
				"released": intVal("1999"),
			}),
		}),
	)

	field := makeField("mergeMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateMergeField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "ON MATCH SET") {
		t.Errorf("expected ON MATCH SET in merge mutation, got %q", result)
	}
	if !strings.Contains(result, "COALESCE") {
		t.Errorf("expected COALESCE in ON MATCH SET, got %q", result)
	}
}

// Test: translateMergeField returns collect() projection.
// Expected: output contains collect() for batched result aggregation.
func TestTranslateMergeField_CollectProjection(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	field := makeField("mergeMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateMergeField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "collect") {
		t.Errorf("expected collect() in merge mutation, got %q", result)
	}
}

// === TR-11: translateConnectField tests ===

// Test: translateConnectField produces UNWIND with double MATCH for from/to nodes.
// Expected: output contains UNWIND, two MATCH clauses
func TestTranslateConnectField_ProducesUnwindDoubleMatch(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"from": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
			"to": makeWhereValue(map[string]*ast.Value{
				"name": strVal("Keanu Reeves"),
			}),
		}),
	)

	field := makeField("connectMovieActors", ast.SelectionSet{
		&ast.Field{Name: "relationshipsCreated"},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateConnectField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty connect query, got empty")
	}
	if !strings.Contains(result, "UNWIND") {
		t.Errorf("expected UNWIND in connect mutation, got %q", result)
	}
	// Should have two MATCH clauses (one for from, one for to)
	if strings.Count(result, "MATCH") < 2 {
		t.Errorf("expected at least 2 MATCH in connect mutation, got %q", result)
	}
}

// Test: translateConnectField produces MERGE for the relationship.
// Expected: output contains MERGE with relationship type and correct direction.
func TestTranslateConnectField_ProducesMergeRelationship(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"from": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
			"to": makeWhereValue(map[string]*ast.Value{
				"name": strVal("Keanu Reeves"),
			}),
		}),
	)

	field := makeField("connectMovieActors", ast.SelectionSet{
		&ast.Field{Name: "relationshipsCreated"},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateConnectField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "MERGE") {
		t.Errorf("expected MERGE in connect mutation, got %q", result)
	}
	if !strings.Contains(result, "ACTED_IN") {
		t.Errorf("expected ACTED_IN relationship type, got %q", result)
	}
}

// Test: translateConnectField with edge properties produces SET for edge props.
// Expected: output contains SET r.role when edge properties provided.
func TestTranslateConnectField_EdgePropertiesSet(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"from": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
			"to": makeWhereValue(map[string]*ast.Value{
				"name": strVal("Keanu Reeves"),
			}),
			"edge": makeWhereValue(map[string]*ast.Value{
				"role": strVal("Neo"),
			}),
		}),
	)

	field := makeField("connectMovieActors", ast.SelectionSet{
		&ast.Field{Name: "relationshipsCreated"},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateConnectField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "SET") {
		t.Errorf("expected SET for edge properties in connect mutation, got %q", result)
	}
}

// Test: translateConnectField returns size($input) for count.
// Expected: output contains size() for relationshipsCreated count.
func TestTranslateConnectField_SizeReturn(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"from": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
			"to": makeWhereValue(map[string]*ast.Value{
				"name": strVal("Keanu Reeves"),
			}),
		}),
	)

	field := makeField("connectMovieActors", ast.SelectionSet{
		&ast.Field{Name: "relationshipsCreated"},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateConnectField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "size") {
		t.Errorf("expected size() for count return in connect mutation, got %q", result)
	}
}

// === TR-12: translateConnectFieldSplit tests ===

// Test: translateConnectFieldSplit write has UNWIND+MATCH+MERGE but no RETURN.
func TestTranslateConnectFieldSplit_WriteHasNoReturn(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"from": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
			"to": makeWhereValue(map[string]*ast.Value{
				"name": strVal("Keanu Reeves"),
			}),
		}),
	)

	field := makeField("connectMovieActors", ast.SelectionSet{
		&ast.Field{Name: "relationshipsCreated"},
	}, makeArg("input", inputVal))

	writeQuery, _, _, err := tr.translateConnectFieldSplit(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(writeQuery, "UNWIND") {
		t.Errorf("write should contain UNWIND, got %q", writeQuery)
	}
	if !strings.Contains(writeQuery, "MERGE") {
		t.Errorf("write should contain MERGE, got %q", writeQuery)
	}
	if strings.Contains(writeQuery, "RETURN") {
		t.Errorf("write should NOT contain RETURN, got %q", writeQuery)
	}
	if strings.Contains(writeQuery, "CALL") {
		t.Errorf("write should NOT be wrapped in CALL, got %q", writeQuery)
	}
}

// Test: translateConnectFieldSplit read returns only size() count.
func TestTranslateConnectFieldSplit_ReadHasSizeOnly(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"from": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
			"to": makeWhereValue(map[string]*ast.Value{
				"name": strVal("Keanu Reeves"),
			}),
		}),
	)

	field := makeField("connectMovieActors", ast.SelectionSet{
		&ast.Field{Name: "relationshipsCreated"},
	}, makeArg("input", inputVal))

	_, readBlock, _, err := tr.translateConnectFieldSplit(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(readBlock, "size") {
		t.Errorf("read should contain size(), got %q", readBlock)
	}
	if !strings.Contains(readBlock, "CALL") {
		t.Errorf("read should be wrapped in CALL, got %q", readBlock)
	}
	if strings.Contains(readBlock, "MATCH") {
		t.Errorf("read should NOT contain MATCH (write is separate), got %q", readBlock)
	}
	if strings.Contains(readBlock, "MERGE") {
		t.Errorf("read should NOT contain MERGE (write is separate), got %q", readBlock)
	}
}

// Test: translateConnectFieldSplit with edge properties includes SET in write.
func TestTranslateConnectFieldSplit_EdgePropertiesInWrite(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"from": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
			"to": makeWhereValue(map[string]*ast.Value{
				"name": strVal("Keanu Reeves"),
			}),
			"edge": makeWhereValue(map[string]*ast.Value{
				"role": strVal("Neo"),
			}),
		}),
	)

	field := makeField("connectMovieActors", ast.SelectionSet{
		&ast.Field{Name: "relationshipsCreated"},
	}, makeArg("input", inputVal))

	writeQuery, _, _, err := tr.translateConnectFieldSplit(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(writeQuery, "SET") {
		t.Errorf("write should contain SET for edge properties, got %q", writeQuery)
	}
}

// === TR-13: Dispatch wiring tests ===

// Test: translateMutationSplit dispatches mergeMovies to translateMergeFieldSplit.
// Expected: mergeMovies mutation field does not return "unknown mutation field" error.
func TestTranslateMutation_DispatchesMerge(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

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

	_, _, err := tr.translateMutationSplit(op, scope)
	if err != nil && strings.Contains(err.Error(), "unknown mutation field") {
		t.Fatalf("translateMutationSplit did not dispatch mergeMovies — got unknown mutation field error: %v", err)
	}
}

// Test: translateMutationSplit dispatches connectMovieActors to translateConnectField.
// Expected: connectMovieActors mutation field does not return "unknown mutation field" error.
func TestTranslateMutation_DispatchesConnect(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	inputVal := listVal(
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
		makeField("connectMovieActors", ast.SelectionSet{
			&ast.Field{Name: "relationshipsCreated"},
		}, makeArg("input", inputVal)),
	)

	_, _, err := tr.translateMutationSplit(op, scope)
	if err != nil && strings.Contains(err.Error(), "unknown mutation field") {
		t.Fatalf("translateMutationSplit did not dispatch connectMovieActors — got unknown mutation field error: %v", err)
	}
}

// Test: full Translate() call with mergeMovies produces valid Cypher.
// Expected: Translate() returns non-empty query with MERGE.
func TestTranslate_MergeMutation_E2E(t *testing.T) {
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
		t.Fatalf("Translate returned error: %v", err)
	}
	if plan.ReadStatement.Query == "" {
		t.Fatal("expected non-empty Cypher query from Translate")
	}
	// After FOREACH rewrite: MERGE is in WriteStatements, ReadStatement has MATCH only
	if len(plan.WriteStatements) == 0 {
		t.Fatal("expected non-empty WriteStatements for merge mutation")
	}
	if !strings.Contains(plan.WriteStatements[0].Query, "MERGE") {
		t.Errorf("expected MERGE in WriteStatements[0], got %q", plan.WriteStatements[0].Query)
	}
	if strings.Contains(plan.ReadStatement.Query, "MERGE") {
		t.Errorf("ReadStatement should NOT contain MERGE after FOREACH rewrite, got %q", plan.ReadStatement.Query)
	}
}

// === FIX-1: extractMergeMatchKeyNames tests ===

// Test: extractMergeMatchKeyNames with inline AST partial match returns only provided keys.
// Expected: when input has match: {title: "X"}, only ["title"] is returned (not ["title", "released"]).
func TestExtractMergeMatchKeyNames_InlinePartialMatch(t *testing.T) {
	node := mergeTestModel().Nodes[0] // Movie: id(isID), title, released
	scope := newParamScope()

	// Input: [{match: {title: "The Matrix"}}] — only "title" in match, not "released"
	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)
	arg := makeArg("input", inputVal)

	keys := extractMergeMatchKeyNames(arg, scope, node)
	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1 (only 'title')", len(keys))
	}
	if keys[0] != "title" {
		t.Errorf("keys[0] = %q, want %q", keys[0], "title")
	}
}

// Test: extractMergeMatchKeyNames with variable-based input resolves keys from variable map.
// Expected: when input is a $variable, keys are extracted from the resolved variable value.
func TestExtractMergeMatchKeyNames_VariableInput(t *testing.T) {
	node := mergeTestModel().Nodes[0] // Movie: id(isID), title, released
	scope := newParamScope()
	scope.variables = map[string]any{
		"input": []any{
			map[string]any{
				"match": map[string]any{
					"title": "The Matrix",
				},
			},
		},
	}

	// Input: $input (variable reference)
	arg := makeArg("input", &ast.Value{Kind: ast.Variable, Raw: "input"})

	keys := extractMergeMatchKeyNames(arg, scope, node)
	if len(keys) != 1 {
		t.Fatalf("len(keys) = %d, want 1 (only 'title')", len(keys))
	}
	if keys[0] != "title" {
		t.Errorf("keys[0] = %q, want %q", keys[0], "title")
	}
}

// Test: extractMergeMatchKeyNames falls back to all non-ID non-vector schema fields
// when neither AST children nor variables provide match structure.
// Expected: for Movie(id, title, released), returns ["title", "released"].
func TestExtractMergeMatchKeyNames_Fallback(t *testing.T) {
	node := mergeTestModel().Nodes[0] // Movie: id(isID), title, released
	scope := newParamScope()

	// Input with no match children (empty object) — triggers fallback
	arg := makeArg("input", &ast.Value{Kind: ast.ListValue})

	keys := extractMergeMatchKeyNames(arg, scope, node)
	if len(keys) != 2 {
		t.Fatalf("len(keys) = %d, want 2 (title, released)", len(keys))
	}
	// Should contain title and released (not id which is IsID)
	found := map[string]bool{}
	for _, k := range keys {
		found[k] = true
	}
	if !found["title"] {
		t.Error("fallback keys should contain 'title'")
	}
	if !found["released"] {
		t.Error("fallback keys should contain 'released'")
	}
	if found["id"] {
		t.Error("fallback keys should NOT contain 'id' (IsID field)")
	}
}

// Test: mergeMatchKeyNames excludes ID and vector fields from match keys.
// Expected: for a node with id(isID), title, released, embedding(@vector),
// returns only ["title", "released"].
func TestMergeMatchKeyNames_ExcludesIDAndVector(t *testing.T) {
	node := schema.NodeDefinition{
		Name:   "Movie",
		Labels: []string{"Movie"},
		Fields: []schema.FieldDefinition{
			{Name: "id", IsID: true},
			{Name: "title"},
			{Name: "released"},
			{Name: "embedding"},
		},
		VectorField: &schema.VectorFieldDefinition{
			Name: "embedding",
		},
	}

	keys := mergeMatchKeyNames(node)
	if len(keys) != 2 {
		t.Fatalf("len(keys) = %d, want 2 (title, released)", len(keys))
	}
	found := map[string]bool{}
	for _, k := range keys {
		found[k] = true
	}
	if !found["title"] {
		t.Error("keys should contain 'title'")
	}
	if !found["released"] {
		t.Error("keys should contain 'released'")
	}
	if found["id"] {
		t.Error("keys should NOT contain 'id'")
	}
	if found["embedding"] {
		t.Error("keys should NOT contain 'embedding' (vector field)")
	}
}

// Test: full translateMergeField with partial match input produces MERGE pattern
// with only the provided key, not all schema fields.
// Expected: MERGE (n:Movie {title: item.match.title}) — NOT {title: ..., released: ...}
func TestTranslateMergeField_PartialMatchUsesOnlyProvidedKeys(t *testing.T) {
	tr := New(mergeTestModel())
	scope := newParamScope()

	// Only provide "title" in match — "released" is NOT in the match input
	inputVal := listVal(
		makeWhereValue(map[string]*ast.Value{
			"match": makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		}),
	)

	field := makeField("mergeMovies", ast.SelectionSet{
		&ast.Field{Name: "movies", SelectionSet: ast.SelectionSet{
			&ast.Field{Name: "id"},
			&ast.Field{Name: "title"},
		}},
	}, makeArg("input", inputVal))

	result, _, err := tr.translateMergeField(field, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// MERGE pattern should only contain title, not released
	if !strings.Contains(result, "title: item.match.title") {
		t.Errorf("expected 'title: item.match.title' in MERGE pattern, got %q", result)
	}
	if strings.Contains(result, "released: item.match.released") {
		t.Errorf("MERGE pattern should NOT contain 'released: item.match.released' when only 'title' provided in match input, got %q", result)
	}
}

// Test: full Translate() call with connectMovieActors produces valid Cypher.
// Expected: Translate() returns non-empty query with MERGE for relationship.
func TestTranslate_ConnectMutation_E2E(t *testing.T) {
	tr := New(mergeTestModel())

	inputVal := listVal(
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
		makeField("connectMovieActors", ast.SelectionSet{
			&ast.Field{Name: "relationshipsCreated"},
		}, makeArg("input", inputVal)),
	)

	doc := &ast.QueryDocument{Operations: ast.OperationList{op}}
	plan, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if plan.ReadStatement.Query == "" {
		t.Fatal("expected non-empty Cypher query from Translate")
	}
}
