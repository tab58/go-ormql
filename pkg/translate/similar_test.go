package translate

import (
	"strconv"
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// --- TR-9: translateSimilarField for vector similarity queries ---

// vectorModel returns a GraphModel with a Movie node that has a VectorField for testing.
func vectorModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
					{Name: "released", GraphQLType: "Int", GoType: "*int", CypherType: "INTEGER", Nullable: true},
					{Name: "embedding", GraphQLType: "[Float!]!", GoType: "[]float64", CypherType: "LIST<FLOAT>", IsList: true},
				},
				VectorField: &schema.VectorFieldDefinition{
					Name:       "embedding",
					IndexName:  "movie_embeddings",
					Dimensions: 1536,
					Similarity: "cosine",
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
	}
}

// Test: translateRootField dispatches to translateSimilarField for "moviesSimilar" field
// when the resolved node has a VectorField.
// Expected: Cypher contains "db.index.vector.queryNodes" and params include index name, first, vector.
func TestTranslateSimilarField_BasicQuery(t *testing.T) {
	tr := New(vectorModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("moviesSimilar", ast.SelectionSet{
					&ast.Field{Name: "score", Alias: "score"},
					&ast.Field{Name: "node", Alias: "node", SelectionSet: ast.SelectionSet{
						&ast.Field{Name: "title", Alias: "title"},
					}},
				},
					makeArg("vector", &ast.Value{
						Kind:     ast.ListValue,
						Children: makeFloatList(0.1, 0.2, 0.3),
					}),
					makeArg("first", intVal("5")),
				),
			),
		},
	}
	op := doc.Operations[0]

	stmt, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain vector query procedure call
	if !strings.Contains(stmt.Query, "db.index.vector.queryNodes") {
		t.Errorf("expected 'db.index.vector.queryNodes' in query, got:\n%s", stmt.Query)
	}

	// Should contain YIELD node AS n, score
	if !strings.Contains(stmt.Query, "YIELD node AS n, score") {
		t.Errorf("expected 'YIELD node AS n, score' in query, got:\n%s", stmt.Query)
	}

	// Should contain collect with score and node projection
	if !strings.Contains(stmt.Query, "score: score") {
		t.Errorf("expected 'score: score' in projection, got:\n%s", stmt.Query)
	}

	// Parameters should include index name, first count, and vector
	params := stmt.Params
	foundIndexName := false
	foundFirst := false
	foundVector := false
	for _, v := range params {
		switch val := v.(type) {
		case string:
			if val == "movie_embeddings" {
				foundIndexName = true
			}
		case int:
			if val == 5 {
				foundFirst = true
			}
		case []float64:
			if len(val) == 3 {
				foundVector = true
			}
		}
	}
	if !foundIndexName {
		t.Error("params missing index name 'movie_embeddings'")
	}
	if !foundFirst {
		t.Error("params missing first count 5")
	}
	if !foundVector {
		t.Error("params missing vector []float64")
	}
}

// Test: translateSimilarField defaults first to 10 when not provided.
// Expected: params include first=10.
func TestTranslateSimilarField_DefaultFirst(t *testing.T) {
	tr := New(vectorModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("moviesSimilar", ast.SelectionSet{
					&ast.Field{Name: "score", Alias: "score"},
				},
					makeArg("vector", &ast.Value{
						Kind:     ast.ListValue,
						Children: makeFloatList(0.1, 0.2),
					}),
					// No "first" argument — should default to 10
				),
			),
		},
	}
	op := doc.Operations[0]

	stmt, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have first=10 in params
	foundFirst10 := false
	for _, v := range stmt.Params {
		if n, ok := v.(int); ok && n == 10 {
			foundFirst10 = true
		}
	}
	if !foundFirst10 {
		t.Errorf("expected params to contain first=10 (default), params: %v", stmt.Params)
	}
}

// Test: translateSimilarField builds correct node projection with selected fields.
// Expected: projection includes ".title, .released" but not ".id" (only selected fields).
func TestTranslateSimilarField_NodeProjection(t *testing.T) {
	tr := New(vectorModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("moviesSimilar", ast.SelectionSet{
					&ast.Field{Name: "score", Alias: "score"},
					&ast.Field{Name: "node", Alias: "node", SelectionSet: ast.SelectionSet{
						&ast.Field{Name: "title", Alias: "title"},
						&ast.Field{Name: "released", Alias: "released"},
					}},
				},
					makeArg("vector", &ast.Value{Kind: ast.ListValue, Children: makeFloatList(0.1)}),
					makeArg("first", intVal("3")),
				),
			),
		},
	}
	op := doc.Operations[0]

	stmt, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Projection should include selected fields
	if !strings.Contains(stmt.Query, ".title") {
		t.Errorf("expected '.title' in node projection, got:\n%s", stmt.Query)
	}
	if !strings.Contains(stmt.Query, ".released") {
		t.Errorf("expected '.released' in node projection, got:\n%s", stmt.Query)
	}
}

// Test: translateSimilarField with variable vector and first arguments.
// Expected: variables resolved via resolveValue before parameterization.
func TestTranslateSimilarField_VariableArguments(t *testing.T) {
	tr := New(vectorModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("moviesSimilar", ast.SelectionSet{
					&ast.Field{Name: "score", Alias: "score"},
				},
					makeArg("vector", &ast.Value{Kind: ast.Variable, Raw: "queryVector"}),
					makeArg("first", &ast.Value{Kind: ast.Variable, Raw: "k"}),
				),
			),
		},
	}
	op := doc.Operations[0]

	variables := map[string]any{
		"queryVector": []float64{0.5, 0.6, 0.7},
		"k":           float64(8), // JSON deserializes as float64
	}

	stmt, err := tr.Translate(doc, op, variables)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Params should contain the resolved vector
	foundVector := false
	for _, v := range stmt.Params {
		if vec, ok := v.([]float64); ok && len(vec) == 3 {
			foundVector = true
		}
	}
	if !foundVector {
		t.Errorf("params missing resolved vector from variable, params: %v", stmt.Params)
	}
}

// Test: translateSimilarField with only "score" selected (no "node").
// Expected: collect({score: score}) without node projection.
func TestTranslateSimilarField_ScoreOnlySelection(t *testing.T) {
	tr := New(vectorModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("moviesSimilar", ast.SelectionSet{
					&ast.Field{Name: "score", Alias: "score"},
				},
					makeArg("vector", &ast.Value{Kind: ast.ListValue, Children: makeFloatList(0.1)}),
				),
			),
		},
	}
	op := doc.Operations[0]

	stmt, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stmt.Query, "score: score") {
		t.Errorf("expected 'score: score' in projection, got:\n%s", stmt.Query)
	}
}

// Test: translateSimilarField alongside other root fields produces separate CALL subqueries.
// Expected: multiple CALL blocks with moviesSimilar and movies aliases.
func TestTranslateSimilarField_AlongsideOtherFields(t *testing.T) {
	tr := New(vectorModel())
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("moviesSimilar", ast.SelectionSet{
					&ast.Field{Name: "score", Alias: "score"},
				},
					makeArg("vector", &ast.Value{Kind: ast.ListValue, Children: makeFloatList(0.1)}),
				),
				makeField("movies", ast.SelectionSet{
					&ast.Field{Name: "title", Alias: "title"},
				}),
			),
		},
	}
	op := doc.Operations[0]

	stmt, err := tr.Translate(doc, op, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have both aliases in the RETURN clause
	if !strings.Contains(stmt.Query, "moviesSimilar:") {
		t.Errorf("expected 'moviesSimilar:' in RETURN, got:\n%s", stmt.Query)
	}
	if !strings.Contains(stmt.Query, "movies:") {
		t.Errorf("expected 'movies:' in RETURN, got:\n%s", stmt.Query)
	}
}

// Test: Translate() returns error for {node}Similar on a node without VectorField.
// Expected: error about missing vector index.
func TestTranslateSimilarField_NoVectorField_ReturnsError(t *testing.T) {
	tr := New(vectorModel()) // Actor has no VectorField
	doc := &ast.QueryDocument{
		Operations: ast.OperationList{
			makeQueryOp(
				makeField("actorsSimilar", ast.SelectionSet{
					&ast.Field{Name: "score", Alias: "score"},
				},
					makeArg("vector", &ast.Value{Kind: ast.ListValue, Children: makeFloatList(0.1)}),
				),
			),
		},
	}
	op := doc.Operations[0]

	_, err := tr.Translate(doc, op, nil)
	if err == nil {
		t.Fatal("expected error for Similar query on node without VectorField, got nil")
	}
}

// --- Test Helpers ---

// makeFloatList creates ast.ChildValue children for a ListValue from float64 values.
func makeFloatList(vals ...float64) ast.ChildValueList {
	var children ast.ChildValueList
	for _, v := range vals {
		children = append(children, &ast.ChildValue{
			Value: &ast.Value{
				Kind: ast.FloatValue,
				Raw:  formatFloat(v),
			},
		})
	}
	return children
}

// formatFloat converts a float64 to a string for AST FloatValue.
func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
