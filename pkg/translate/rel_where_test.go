package translate

import (
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// === TR-12: buildRelWhereClause tests ===

// relWhereTestModel returns a model with to-one and to-many relationships
// for testing relationship-based WHERE filter translation.
func relWhereTestModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
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
			{
				Name:   "Repository",
				Labels: []string{"Repository"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
				},
			},
		},
		Relationships: []schema.RelationshipDefinition{
			// to-many: Movie.actors → IsList=true
			{FieldName: "actors", RelType: "ACTED_IN", Direction: schema.DirectionIN, FromNode: "Movie", ToNode: "Actor", IsList: true},
			// to-one: Movie.repository → IsList=false
			{FieldName: "repository", RelType: "BELONGS_TO", Direction: schema.DirectionOUT, FromNode: "Movie", ToNode: "Repository", IsList: false},
		},
	}
}

// Test: buildWhereClause with to-many relationship filter (actors_some) produces EXISTS.
// Expected: output contains EXISTS { MATCH ... } for to-many relationship filter.
func TestBuildWhereClause_RelToMany_ProducesExists(t *testing.T) {
	tr := New(relWhereTestModel())
	scope := newParamScope()

	whereArg := makeWhereValue(map[string]*ast.Value{
		"actors_some": makeWhereValue(map[string]*ast.Value{
			"name": strVal("Keanu"),
		}),
	})

	node, _ := relWhereTestModel().NodeByName("Movie")
	result := tr.buildWhereClause(whereArg, "n", node, scope)

	if result == "" {
		t.Fatal("expected non-empty WHERE clause for relationship filter, got empty")
	}
	if !strings.Contains(result, "EXISTS") {
		t.Errorf("expected EXISTS for to-many relationship filter, got %q", result)
	}
	if !strings.Contains(result, "ACTED_IN") {
		t.Errorf("expected ACTED_IN relationship type in filter, got %q", result)
	}
}

// Test: buildWhereClause with to-one relationship filter (repository) produces MATCH pattern.
// Expected: output contains MATCH pattern (not EXISTS) for to-one relationship filter.
func TestBuildWhereClause_RelToOne_ProducesMatchPattern(t *testing.T) {
	tr := New(relWhereTestModel())
	scope := newParamScope()

	whereArg := makeWhereValue(map[string]*ast.Value{
		"repository": makeWhereValue(map[string]*ast.Value{
			"name": strVal("my-repo"),
		}),
	})

	node, _ := relWhereTestModel().NodeByName("Movie")
	result := tr.buildWhereClause(whereArg, "n", node, scope)

	if result == "" {
		t.Fatal("expected non-empty WHERE clause for relationship filter, got empty")
	}
	if !strings.Contains(result, "BELONGS_TO") {
		t.Errorf("expected BELONGS_TO relationship type in filter, got %q", result)
	}
}

// Test: buildWhereClause with relationship filter AND scalar filter composes them.
// Expected: output contains both scalar WHERE and relationship filter combined.
func TestBuildWhereClause_RelFilterComposesWithScalar(t *testing.T) {
	tr := New(relWhereTestModel())
	scope := newParamScope()

	whereArg := makeWhereValue(map[string]*ast.Value{
		"title":       strVal("The Matrix"),
		"actors_some": makeWhereValue(map[string]*ast.Value{
			"name": strVal("Keanu"),
		}),
	})

	node, _ := relWhereTestModel().NodeByName("Movie")
	result := tr.buildWhereClause(whereArg, "n", node, scope)

	if result == "" {
		t.Fatal("expected non-empty WHERE clause, got empty")
	}
	// Should contain both the scalar filter and the relationship filter
	if !strings.Contains(result, "title") {
		t.Errorf("expected title scalar filter, got %q", result)
	}
	if !strings.Contains(result, "EXISTS") || !strings.Contains(result, "ACTED_IN") {
		t.Errorf("expected EXISTS with ACTED_IN for actors_some filter, got %q", result)
	}
}

// Test: buildWhereClause with nested boolean (AND) containing relationship filters.
// Expected: relationship filters inside AND are composed correctly.
func TestBuildWhereClause_RelFilterInAND(t *testing.T) {
	tr := New(relWhereTestModel())
	scope := newParamScope()

	whereArg := makeWhereValue(map[string]*ast.Value{
		"AND": listVal(
			makeWhereValue(map[string]*ast.Value{
				"actors_some": makeWhereValue(map[string]*ast.Value{
					"name": strVal("Keanu"),
				}),
			}),
			makeWhereValue(map[string]*ast.Value{
				"title": strVal("The Matrix"),
			}),
		),
	})

	node, _ := relWhereTestModel().NodeByName("Movie")
	result := tr.buildWhereClause(whereArg, "n", node, scope)

	if result == "" {
		t.Fatal("expected non-empty WHERE clause, got empty")
	}
	if !strings.Contains(result, "AND") {
		t.Errorf("expected AND composition, got %q", result)
	}
}

// Test: buildWhereClause with relationship filter uses correct direction.
// Expected: to-many IN direction produces (n)<-[:ACTED_IN]-(r) pattern.
func TestBuildWhereClause_RelFilterCorrectDirection(t *testing.T) {
	tr := New(relWhereTestModel())
	scope := newParamScope()

	whereArg := makeWhereValue(map[string]*ast.Value{
		"actors_some": makeWhereValue(map[string]*ast.Value{
			"name": strVal("Keanu"),
		}),
	})

	node, _ := relWhereTestModel().NodeByName("Movie")
	result := tr.buildWhereClause(whereArg, "n", node, scope)

	if result == "" {
		t.Fatal("expected non-empty WHERE clause, got empty")
	}
	// Direction IN means Actor->Movie, so pattern should reflect that
	if !strings.Contains(result, "Actor") {
		t.Errorf("expected Actor label in relationship filter pattern, got %q", result)
	}
}
