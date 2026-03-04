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

// === TR-13: Dispatch wiring tests ===

// Test: translateMutation dispatches mergeMovies to translateMergeField.
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

	_, err := tr.translateMutation(op, scope)
	if err != nil && strings.Contains(err.Error(), "unknown mutation field") {
		t.Fatalf("translateMutation did not dispatch mergeMovies — got unknown mutation field error: %v", err)
	}
}

// Test: translateMutation dispatches connectMovieActors to translateConnectField.
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

	_, err := tr.translateMutation(op, scope)
	if err != nil && strings.Contains(err.Error(), "unknown mutation field") {
		t.Fatalf("translateMutation did not dispatch connectMovieActors — got unknown mutation field error: %v", err)
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
	stmt, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if stmt.Query == "" {
		t.Fatal("expected non-empty Cypher query from Translate")
	}
	if !strings.Contains(stmt.Query, "MERGE") {
		t.Errorf("expected MERGE in Cypher output, got %q", stmt.Query)
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
	stmt, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if stmt.Query == "" {
		t.Fatal("expected non-empty Cypher query from Translate")
	}
}
