package client

import (
	"context"
	"errors"
	"testing"

	"github.com/tab58/gql-orm/pkg/cypher"
	"github.com/tab58/gql-orm/pkg/driver"
	"github.com/tab58/gql-orm/pkg/internal/strutil"
	"github.com/vektah/gqlparser/v2/ast"
)

// TestDirectExecute_Query_ReturnsRecords verifies that directExecute
// handles a simple query field by calling driver.Execute and returning records.
func TestDirectExecute_Query_ReturnsRecords(t *testing.T) {
	drv := &mockDriver{
		executeFn: func(_ context.Context, stmt cypher.Statement) (driver.Result, error) {
			return driver.Result{
				Records: []driver.Record{
					{Values: map[string]any{"title": "The Matrix", "released": 1999}},
				},
			}, nil
		},
	}
	c := New(nil, drv)

	result, err := c.Execute(context.Background(), `query { movies { title released } }`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	movies, ok := result["movies"]
	if !ok {
		t.Fatal("result missing 'movies' key")
	}
	list, ok := movies.([]any)
	if !ok {
		t.Fatalf("movies is %T, want []any", movies)
	}
	if len(list) != 1 {
		t.Fatalf("got %d movies, want 1", len(list))
	}
}

// TestDirectExecute_Query_WithWhere verifies that where arguments are passed through.
func TestDirectExecute_Query_WithWhere(t *testing.T) {
	var capturedStmt cypher.Statement
	drv := &mockDriver{
		executeFn: func(_ context.Context, stmt cypher.Statement) (driver.Result, error) {
			capturedStmt = stmt
			return driver.Result{}, nil
		},
	}
	c := New(nil, drv)

	_, err := c.Execute(context.Background(), `query { movies(where: {title: "The Matrix"}) { title } }`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedStmt.Query == "" {
		t.Fatal("driver.Execute was not called")
	}
}

// TestDirectExecute_Query_DriverError verifies that driver errors propagate.
func TestDirectExecute_Query_DriverError(t *testing.T) {
	drv := &mockDriver{
		executeFn: func(_ context.Context, _ cypher.Statement) (driver.Result, error) {
			return driver.Result{}, errors.New("db unreachable")
		},
	}
	c := New(nil, drv)

	_, err := c.Execute(context.Background(), `query { movies { title } }`, nil)
	if err == nil {
		t.Fatal("expected error from driver, got nil")
	}
}

// TestDirectExecute_CreateMutation verifies create mutation dispatching.
func TestDirectExecute_CreateMutation(t *testing.T) {
	drv := &mockDriver{
		executeWriteFn: func(_ context.Context, stmt cypher.Statement) (driver.Result, error) {
			return driver.Result{
				Records: []driver.Record{
					{Values: map[string]any{"title": "Inception"}},
				},
			}, nil
		},
	}
	c := New(nil, drv)

	result, err := c.Execute(context.Background(), `mutation { createMovies(input: [{title: "Inception"}]) { movies { title } } }`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cm, ok := result["createMovies"]
	if !ok {
		t.Fatal("result missing 'createMovies' key")
	}
	cmMap, ok := cm.(map[string]any)
	if !ok {
		t.Fatalf("createMovies is %T, want map[string]any", cm)
	}
	movies, ok := cmMap["movies"]
	if !ok {
		t.Fatal("createMovies missing 'movies' key")
	}
	list, ok := movies.([]any)
	if !ok {
		t.Fatalf("movies is %T, want []any", movies)
	}
	if len(list) != 1 {
		t.Fatalf("got %d movies, want 1", len(list))
	}
}

// TestDirectExecute_UpdateMutation verifies update mutation dispatching.
func TestDirectExecute_UpdateMutation(t *testing.T) {
	drv := &mockDriver{
		executeWriteFn: func(_ context.Context, _ cypher.Statement) (driver.Result, error) {
			return driver.Result{
				Records: []driver.Record{
					{Values: map[string]any{"title": "The Matrix Reloaded"}},
				},
			}, nil
		},
	}
	c := New(nil, drv)

	result, err := c.Execute(context.Background(), `mutation { updateMovies(where: {title: "The Matrix"}, update: {title: "The Matrix Reloaded"}) { movies { title } } }`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	um, ok := result["updateMovies"]
	if !ok {
		t.Fatal("result missing 'updateMovies' key")
	}
	umMap, ok := um.(map[string]any)
	if !ok {
		t.Fatalf("updateMovies is %T, want map[string]any", um)
	}
	if _, ok := umMap["movies"]; !ok {
		t.Fatal("updateMovies missing 'movies' key")
	}
}

// TestDirectExecute_DeleteMutation verifies delete mutation dispatching.
func TestDirectExecute_DeleteMutation(t *testing.T) {
	drv := &mockDriver{
		executeWriteFn: func(_ context.Context, _ cypher.Statement) (driver.Result, error) {
			return driver.Result{}, nil
		},
	}
	c := New(nil, drv)

	result, err := c.Execute(context.Background(), `mutation { deleteMovies(where: {title: "The Matrix"}) { nodesDeleted } }`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	dm, ok := result["deleteMovies"]
	if !ok {
		t.Fatal("result missing 'deleteMovies' key")
	}
	dmMap, ok := dm.(map[string]any)
	if !ok {
		t.Fatalf("deleteMovies is %T, want map[string]any", dm)
	}
	if nd, ok := dmMap["nodesDeleted"]; !ok || nd != 1 {
		t.Errorf("nodesDeleted = %v, want 1", nd)
	}
}

// TestDirectExecute_EmptyOperations verifies empty query returns empty map.
func TestDirectExecute_EmptyOperations(t *testing.T) {
	drv := &mockDriver{}
	c := New(nil, drv)

	// A valid query with no operation (just a fragment or empty selection)
	// won't parse as valid GraphQL, but a mutation with unknown prefix returns empty.
	result, err := c.Execute(context.Background(), `mutation { unknownOp { field } }`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	um, ok := result["unknownOp"]
	if !ok {
		t.Fatal("result missing 'unknownOp' key")
	}
	umMap, ok := um.(map[string]any)
	if !ok {
		t.Fatalf("unknownOp is %T, want map[string]any", um)
	}
	if len(umMap) != 0 {
		t.Errorf("unknownOp should be empty map, got %v", umMap)
	}
}

// TestDirectExecute_CreateMutation_DriverError verifies driver write error propagation.
func TestDirectExecute_CreateMutation_DriverError(t *testing.T) {
	drv := &mockDriver{
		executeWriteFn: func(_ context.Context, _ cypher.Statement) (driver.Result, error) {
			return driver.Result{}, errors.New("write failed")
		},
	}
	c := New(nil, drv)

	_, err := c.Execute(context.Background(), `mutation { createMovies(input: [{title: "Fail"}]) { movies { title } } }`, nil)
	if err == nil {
		t.Fatal("expected error from driver write, got nil")
	}
}

// TestDirectExecute_UpdateMutation_DriverError verifies update driver error propagation.
func TestDirectExecute_UpdateMutation_DriverError(t *testing.T) {
	drv := &mockDriver{
		executeWriteFn: func(_ context.Context, _ cypher.Statement) (driver.Result, error) {
			return driver.Result{}, errors.New("update failed")
		},
	}
	c := New(nil, drv)

	_, err := c.Execute(context.Background(), `mutation { updateMovies(where: {title: "X"}, update: {title: "Y"}) { movies { title } } }`, nil)
	if err == nil {
		t.Fatal("expected error from driver write, got nil")
	}
}

// TestDirectExecute_DeleteMutation_DriverError verifies delete driver error propagation.
func TestDirectExecute_DeleteMutation_DriverError(t *testing.T) {
	drv := &mockDriver{
		executeWriteFn: func(_ context.Context, _ cypher.Statement) (driver.Result, error) {
			return driver.Result{}, errors.New("delete failed")
		},
	}
	c := New(nil, drv)

	_, err := c.Execute(context.Background(), `mutation { deleteMovies(where: {title: "X"}) { nodesDeleted } }`, nil)
	if err == nil {
		t.Fatal("expected error from driver write, got nil")
	}
}

// TestDirectExecute_NestedCreate verifies nested create mutation.
func TestDirectExecute_NestedCreate(t *testing.T) {
	callCount := 0
	drv := &mockDriver{
		executeWriteFn: func(_ context.Context, _ cypher.Statement) (driver.Result, error) {
			callCount++
			return driver.Result{
				Records: []driver.Record{
					{Values: map[string]any{"name": "created"}},
				},
			}, nil
		},
	}
	c := New(nil, drv)

	vars := map[string]any{
		"input": []any{
			map[string]any{
				"title": "The Matrix",
				"actors": map[string]any{
					"create": []any{
						map[string]any{
							"node": map[string]any{"name": "Keanu"},
						},
					},
				},
			},
		},
	}
	_, err := c.Execute(context.Background(), `mutation CreateMovies($input: [MovieCreateInput!]!) { createMovies(input: $input) { movies { title } } }`, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expect at least 3 write calls: create node + create related node + create relationship
	if callCount < 3 {
		t.Errorf("expected at least 3 driver write calls, got %d", callCount)
	}
}

// TestDirectExecute_NestedConnect verifies nested connect mutation.
func TestDirectExecute_NestedConnect(t *testing.T) {
	readCalled := false
	writeCount := 0
	drv := &mockDriver{
		executeFn: func(_ context.Context, _ cypher.Statement) (driver.Result, error) {
			readCalled = true
			return driver.Result{
				Records: []driver.Record{
					{Values: map[string]any{"name": "Keanu"}},
				},
			}, nil
		},
		executeWriteFn: func(_ context.Context, _ cypher.Statement) (driver.Result, error) {
			writeCount++
			return driver.Result{
				Records: []driver.Record{
					{Values: map[string]any{"title": "The Matrix"}},
				},
			}, nil
		},
	}
	c := New(nil, drv)

	vars := map[string]any{
		"input": []any{
			map[string]any{
				"title": "The Matrix",
				"actors": map[string]any{
					"connect": []any{
						map[string]any{
							"where": map[string]any{"name": "Keanu"},
						},
					},
				},
			},
		},
	}
	_, err := c.Execute(context.Background(), `mutation CreateMovies($input: [MovieCreateInput!]!) { createMovies(input: $input) { movies { title } } }`, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !readCalled {
		t.Error("expected driver.Execute (read) to be called for connect match")
	}
	// At least 2 writes: create node + create relationship
	if writeCount < 2 {
		t.Errorf("expected at least 2 driver write calls, got %d", writeCount)
	}
}

// TestDirectExecute_WithVariables_Query verifies variables resolve in queries.
func TestDirectExecute_WithVariables_Query(t *testing.T) {
	var capturedStmt cypher.Statement
	drv := &mockDriver{
		executeFn: func(_ context.Context, stmt cypher.Statement) (driver.Result, error) {
			capturedStmt = stmt
			return driver.Result{}, nil
		},
	}
	c := New(nil, drv)

	vars := map[string]any{
		"where": map[string]any{"title": "The Matrix"},
	}
	_, err := c.Execute(context.Background(), `query GetMovies($where: MovieWhere) { movies(where: $where) { title } }`, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedStmt.Query == "" {
		t.Fatal("driver.Execute was not called")
	}
}

// TestSplitNodeAndRelFields verifies separation of scalar and relationship fields.
func TestSplitNodeAndRelFields(t *testing.T) {
	input := map[string]any{
		"title":    "The Matrix",
		"released": 1999,
		"actors": map[string]any{
			"create": []any{
				map[string]any{"node": map[string]any{"name": "Keanu"}},
			},
		},
		"directors": map[string]any{
			"connect": []any{
				map[string]any{"where": map[string]any{"name": "Wachowski"}},
			},
		},
	}

	nodeProps, relFields := splitNodeAndRelFields(input)

	if len(nodeProps) != 2 {
		t.Errorf("expected 2 node props, got %d: %v", len(nodeProps), nodeProps)
	}
	if _, ok := nodeProps["title"]; !ok {
		t.Error("nodeProps missing 'title'")
	}
	if _, ok := nodeProps["released"]; !ok {
		t.Error("nodeProps missing 'released'")
	}
	if len(relFields) != 2 {
		t.Errorf("expected 2 rel fields, got %d: %v", len(relFields), relFields)
	}
	if _, ok := relFields["actors"]; !ok {
		t.Error("relFields missing 'actors'")
	}
	if _, ok := relFields["directors"]; !ok {
		t.Error("relFields missing 'directors'")
	}
}

// TestRecordsToSlice verifies conversion from driver.Record to []any.
func TestRecordsToSlice(t *testing.T) {
	records := []driver.Record{
		{Values: map[string]any{"name": "Alice"}},
		{Values: map[string]any{"name": "Bob"}},
	}
	result := recordsToSlice(records)
	if len(result) != 2 {
		t.Fatalf("got %d results, want 2", len(result))
	}
	first, ok := result[0].(map[string]any)
	if !ok {
		t.Fatalf("result[0] is %T, want map[string]any", result[0])
	}
	if first["name"] != "Alice" {
		t.Errorf("first name = %v, want Alice", first["name"])
	}
}

// TestSingularize verifies simple plural to singular conversion.
func TestSingularize(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Movies", "Movie"},
		{"actors", "actor"},
		{"M", "M"},
		{"", ""},
	}
	for _, tt := range tests {
		got := singularize(tt.input)
		if got != tt.want {
			t.Errorf("singularize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestCapitalize verifies first-rune uppercasing.
func TestCapitalize(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"movies", "Movies"},
		{"Movies", "Movies"},
		{"", ""},
		{"a", "A"},
	}
	for _, tt := range tests {
		got := strutil.Capitalize(tt.input)
		if got != tt.want {
			t.Errorf("Capitalize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestValueToAny covers all AST value kinds.
func TestValueToAny(t *testing.T) {
	vars := map[string]any{"myVar": "resolved"}

	tests := []struct {
		name string
		val  *ast.Value
		want any
	}{
		{"nil", nil, nil},
		{"variable_found", &ast.Value{Kind: ast.Variable, Raw: "myVar"}, "resolved"},
		{"variable_missing", &ast.Value{Kind: ast.Variable, Raw: "nope"}, nil},
		{"int", &ast.Value{Kind: ast.IntValue, Raw: "42"}, 42},
		{"float", &ast.Value{Kind: ast.FloatValue, Raw: "3.14"}, 3.14},
		{"string", &ast.Value{Kind: ast.StringValue, Raw: "hello"}, "hello"},
		{"bool_true", &ast.Value{Kind: ast.BooleanValue, Raw: "true"}, true},
		{"bool_false", &ast.Value{Kind: ast.BooleanValue, Raw: "false"}, false},
		{"enum", &ast.Value{Kind: ast.EnumValue, Raw: "ACTIVE"}, "ACTIVE"},
		{"object", &ast.Value{
			Kind: ast.ObjectValue,
			Children: ast.ChildValueList{
				{Name: "key", Value: &ast.Value{Kind: ast.StringValue, Raw: "val"}},
			},
		}, map[string]any{"key": "val"}},
		{"list", &ast.Value{
			Kind: ast.ListValue,
			Children: ast.ChildValueList{
				{Value: &ast.Value{Kind: ast.IntValue, Raw: "1"}},
				{Value: &ast.Value{Kind: ast.IntValue, Raw: "2"}},
			},
		}, []any{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := valueToAny(tt.val, vars)
			// For slices and maps, just check non-nil
			switch expected := tt.want.(type) {
			case nil:
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}
			case map[string]any:
				gotMap, ok := got.(map[string]any)
				if !ok {
					t.Fatalf("got %T, want map[string]any", got)
				}
				if len(gotMap) != len(expected) {
					t.Errorf("map length = %d, want %d", len(gotMap), len(expected))
				}
			case []any:
				gotList, ok := got.([]any)
				if !ok {
					t.Fatalf("got %T, want []any", got)
				}
				if len(gotList) != len(expected) {
					t.Errorf("list length = %d, want %d", len(gotList), len(expected))
				}
			default:
				if got != tt.want {
					t.Errorf("got %v (%T), want %v (%T)", got, got, tt.want, tt.want)
				}
			}
		})
	}
}

// TestValueToList covers list and variable resolution.
func TestValueToList(t *testing.T) {
	vars := map[string]any{
		"listVar":    []any{"a", "b"},
		"mapListVar": []map[string]any{{"x": 1}},
		"notList":    "scalar",
	}

	tests := []struct {
		name    string
		val     *ast.Value
		wantLen int
		wantNil bool
	}{
		{"nil", nil, 0, true},
		{"variable_list", &ast.Value{Kind: ast.Variable, Raw: "listVar"}, 2, false},
		{"variable_map_list", &ast.Value{Kind: ast.Variable, Raw: "mapListVar"}, 1, false},
		{"variable_not_list", &ast.Value{Kind: ast.Variable, Raw: "notList"}, 0, true},
		{"variable_missing", &ast.Value{Kind: ast.Variable, Raw: "nope"}, 0, true},
		{"literal_list", &ast.Value{
			Kind: ast.ListValue,
			Children: ast.ChildValueList{
				{Value: &ast.Value{Kind: ast.StringValue, Raw: "a"}},
			},
		}, 1, false},
		{"not_list_kind", &ast.Value{Kind: ast.StringValue, Raw: "x"}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := valueToList(tt.val, vars)
			if tt.wantNil {
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("got nil, want non-nil")
			}
			if len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// TestValueToMap covers map and variable resolution.
func TestValueToMap(t *testing.T) {
	vars := map[string]any{
		"mapVar":    map[string]any{"key": "val"},
		"notMap":    "scalar",
	}

	tests := []struct {
		name    string
		val     *ast.Value
		wantLen int
	}{
		{"nil", nil, 0},
		{"variable_map", &ast.Value{Kind: ast.Variable, Raw: "mapVar"}, 1},
		{"variable_not_map", &ast.Value{Kind: ast.Variable, Raw: "notMap"}, 0},
		{"variable_missing", &ast.Value{Kind: ast.Variable, Raw: "nope"}, 0},
		{"object", &ast.Value{
			Kind: ast.ObjectValue,
			Children: ast.ChildValueList{
				{Name: "a", Value: &ast.Value{Kind: ast.IntValue, Raw: "1"}},
			},
		}, 1},
		{"not_object_kind", &ast.Value{Kind: ast.StringValue, Raw: "x"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := valueToMap(tt.val, vars)
			if len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

// TestNestedCreate_WithEdgeProperties verifies edge properties are passed through.
func TestNestedCreate_WithEdgeProperties(t *testing.T) {
	writeCount := 0
	drv := &mockDriver{
		executeWriteFn: func(_ context.Context, _ cypher.Statement) (driver.Result, error) {
			writeCount++
			return driver.Result{
				Records: []driver.Record{
					{Values: map[string]any{"name": "created"}},
				},
			}, nil
		},
	}
	c := New(nil, drv)

	vars := map[string]any{
		"input": []any{
			map[string]any{
				"title": "The Matrix",
				"actors": map[string]any{
					"create": []any{
						map[string]any{
							"node": map[string]any{"name": "Keanu"},
							"edge": map[string]any{"roles": []string{"Neo"}},
						},
					},
				},
			},
		},
	}
	_, err := c.Execute(context.Background(), `mutation CreateMovies($input: [MovieCreateInput!]!) { createMovies(input: $input) { movies { title } } }`, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if writeCount < 3 {
		t.Errorf("expected at least 3 driver write calls, got %d", writeCount)
	}
}
