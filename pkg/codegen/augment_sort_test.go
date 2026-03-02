package codegen

import (
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
)

// --- CG-12: Sort inputs in augmented schema ---

// TestAugmentSchema_SortDirection_GeneratedOnce verifies that a shared SortDirection
// enum is generated exactly once, even with multiple nodes.
func TestAugmentSchema_SortDirection_GeneratedOnce(t *testing.T) {
	model := schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{Name: "Movie", Labels: []string{"Movie"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", IsID: true},
				{Name: "title", GraphQLType: "String!"},
			}},
			{Name: "Actor", Labels: []string{"Actor"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", IsID: true},
				{Name: "name", GraphQLType: "String!"},
			}},
		},
	}

	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "SortDirection") {
		t.Errorf("augmented schema missing 'SortDirection' enum:\n%s", sdl)
	}

	count := strings.Count(sdl, "enum SortDirection")
	if count != 1 {
		t.Errorf("'enum SortDirection' appears %d times, want exactly 1:\n%s", count, sdl)
	}
}

// TestAugmentSchema_SortDirection_HasAscDesc verifies that SortDirection enum
// contains ASC and DESC values.
func TestAugmentSchema_SortDirection_HasAscDesc(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	sortDir := s.Types["SortDirection"]
	if sortDir == nil {
		t.Fatal("SortDirection type not found in parsed schema")
	}

	vals := map[string]bool{}
	for _, v := range sortDir.EnumValues {
		vals[v.Name] = true
	}
	if !vals["ASC"] {
		t.Error("SortDirection missing 'ASC' value")
	}
	if !vals["DESC"] {
		t.Error("SortDirection missing 'DESC' value")
	}
}

// TestAugmentSchema_SortInput_PerNode verifies that each node gets a {NodeName}Sort
// input type with one optional SortDirection field per scalar field.
func TestAugmentSchema_SortInput_PerNode(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieSort") {
		t.Errorf("augmented schema missing 'MovieSort' input:\n%s", sdl)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	movieSort := s.Types["MovieSort"]
	if movieSort == nil {
		t.Fatal("MovieSort type not found in parsed schema")
	}

	fieldNames := map[string]bool{}
	for _, f := range movieSort.Fields {
		fieldNames[f.Name] = true
	}

	// All scalar fields should have sort entries
	for _, name := range []string{"id", "title", "released"} {
		if !fieldNames[name] {
			t.Errorf("MovieSort missing field %q", name)
		}
	}
}

// TestAugmentSchema_SortInput_FieldsAreSortDirection verifies that each field in
// the Sort input has type SortDirection (optional).
func TestAugmentSchema_SortInput_FieldsAreSortDirection(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	movieSort := s.Types["MovieSort"]
	if movieSort == nil {
		t.Fatal("MovieSort type not found in parsed schema")
	}

	for _, f := range movieSort.Fields {
		if f.Type.Name() != "SortDirection" {
			t.Errorf("MovieSort.%s type = %q, want SortDirection", f.Name, f.Type.Name())
		}
	}
}

// TestAugmentSchema_SortInput_MultiNode verifies that each node gets its own Sort input.
func TestAugmentSchema_SortInput_MultiNode(t *testing.T) {
	sdl, err := AugmentSchema(multiNodeModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "MovieSort") {
		t.Errorf("augmented schema missing 'MovieSort':\n%s", sdl)
	}
	if !strings.Contains(sdl, "ActorSort") {
		t.Errorf("augmented schema missing 'ActorSort':\n%s", sdl)
	}
}

// TestAugmentSchema_SortParam_OnListQuery verifies that the list query (e.g., movies)
// has a sort parameter: movies(where: MovieWhere, sort: [MovieSort!]): [Movie!]!
func TestAugmentSchema_SortParam_OnListQuery(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	queryType := s.Types["Query"]
	if queryType == nil {
		t.Fatal("Query type not found")
	}

	moviesField := queryType.Fields.ForName("movies")
	if moviesField == nil {
		t.Fatal("movies field not found on Query type")
	}

	// Check for sort argument
	sortArg := moviesField.Arguments.ForName("sort")
	if sortArg == nil {
		t.Errorf("movies query missing 'sort' parameter:\n%s", sdl)
	}
}

// TestAugmentSchema_SortParam_OnConnection verifies that the connection query
// has a sort parameter: moviesConnection(first: Int, after: String, where: MovieWhere, sort: [MovieSort!])
func TestAugmentSchema_SortParam_OnConnection(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	s, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema failed parse: %v\nSDL:\n%s", parseErr, sdl)
	}

	queryType := s.Types["Query"]
	if queryType == nil {
		t.Fatal("Query type not found")
	}

	connField := queryType.Fields.ForName("moviesConnection")
	if connField == nil {
		t.Fatal("moviesConnection field not found on Query type")
	}

	sortArg := connField.Arguments.ForName("sort")
	if sortArg == nil {
		t.Errorf("moviesConnection query missing 'sort' parameter:\n%s", sdl)
	}
}

// TestAugmentSchema_SortInput_ValidSDL verifies that the augmented schema with
// sort inputs is still valid GraphQL SDL.
func TestAugmentSchema_SortInput_ValidSDL(t *testing.T) {
	sdl, err := AugmentSchema(movieModel())
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}
	if sdl == "" {
		t.Skip("AugmentSchema returned empty")
	}
	_, parseErr := parseSDL(sdl)
	if parseErr != nil {
		t.Fatalf("augmented schema with sort inputs failed gqlparser validation: %v\nSDL:\n%s", parseErr, sdl)
	}
}
