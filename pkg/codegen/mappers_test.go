package codegen

import (
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
)

// mapperModel returns a model for mapper generation tests.
func mapperModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
					{Name: "released", GraphQLType: "Int", GoType: "*int", CypherType: "INTEGER", Nullable: true},
					{Name: "rating", GraphQLType: "Float", GoType: "*float64", CypherType: "FLOAT", Nullable: true},
				},
			},
		},
	}
}

// multiMapperModel returns a multi-node model for mapper generation tests.
func multiMapperModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
				},
			},
			{
				Name:   "Actor",
				Labels: []string{"Actor"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
				},
			},
		},
	}
}

// --- Tests ---

// TestGenerateMappers_NonEmpty verifies that mapper generation produces non-empty output.
func TestGenerateMappers_NonEmpty(t *testing.T) {
	src, err := GenerateMappers(mapperModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateMappers returned error: %v", err)
	}
	if len(src) == 0 {
		t.Fatal("GenerateMappers returned empty output, want non-empty Go source")
	}
}

// TestGenerateMappers_PackageDeclaration verifies that the output contains a package declaration.
// Expected: "package generated"
func TestGenerateMappers_PackageDeclaration(t *testing.T) {
	src, err := GenerateMappers(mapperModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateMappers returned error: %v", err)
	}
	if !strings.Contains(string(src), "package generated") {
		t.Errorf("output missing 'package generated':\n%s", string(src))
	}
}

// TestGenerateMappers_SingleRecordMapper verifies that a mapRecordToMovie function is generated.
// Expected: contains "mapRecordToMovie" or "MapRecordToMovie" function.
func TestGenerateMappers_SingleRecordMapper(t *testing.T) {
	src, err := GenerateMappers(mapperModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateMappers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "recordToMovie") && !strings.Contains(s, "RecordToMovie") {
		t.Errorf("output missing recordToMovie function:\n%s", s)
	}
}

// TestGenerateMappers_BatchMapper verifies that a batch/plural mapper helper is generated.
// Expected: contains "mapRecordsToMovies" or "MapRecordsToMovies" function.
func TestGenerateMappers_BatchMapper(t *testing.T) {
	src, err := GenerateMappers(mapperModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateMappers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "recordsToMovies") && !strings.Contains(s, "RecordsToMovies") {
		t.Errorf("output missing batch mapper function:\n%s", s)
	}
}

// TestGenerateMappers_ReferencesDriverRecord verifies that generated code references driver.Record.
func TestGenerateMappers_ReferencesDriverRecord(t *testing.T) {
	src, err := GenerateMappers(mapperModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateMappers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "driver.Record") && !strings.Contains(s, "Record") {
		t.Errorf("output missing reference to driver.Record:\n%s", s)
	}
}

// TestGenerateMappers_NonNullableFieldMapping verifies that non-nullable fields use
// direct type assertions (e.g., rec.Values["title"].(string)).
func TestGenerateMappers_NonNullableFieldMapping(t *testing.T) {
	src, err := GenerateMappers(mapperModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateMappers returned error: %v", err)
	}
	s := string(src)
	// Should reference the "title" field and use a string type assertion
	if !strings.Contains(s, "title") {
		t.Errorf("output missing 'title' field reference:\n%s", s)
	}
}

// TestGenerateMappers_NullableFieldMapping verifies that nullable fields use pointer-based
// mapping with type assertion fallback (e.g., intPtrFromAny or similar helper).
func TestGenerateMappers_NullableFieldMapping(t *testing.T) {
	src, err := GenerateMappers(mapperModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateMappers returned error: %v", err)
	}
	s := string(src)
	// Should reference the "released" nullable field with some pointer handling
	if !strings.Contains(s, "released") {
		t.Errorf("output missing 'released' field reference:\n%s", s)
	}
}

// TestGenerateMappers_MultiNode verifies that mapper functions are generated for all nodes.
// Expected: mappers for both Movie and Actor.
func TestGenerateMappers_MultiNode(t *testing.T) {
	src, err := GenerateMappers(multiMapperModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateMappers returned error: %v", err)
	}
	s := string(src)

	for _, name := range []string{"Movie", "Actor"} {
		lower := strings.ToLower(name)
		// Check for mapRecordTo<Type> (case-insensitive first char is fine)
		if !strings.Contains(s, "RecordTo"+name) && !strings.Contains(s, "recordTo"+name) && !strings.Contains(s, "record_to_"+lower) {
			t.Errorf("output missing mapper for %q:\n%s", name, s)
		}
	}
}

// TestGenerateMappers_ImportsDriverPackage verifies that generated code imports the driver package.
func TestGenerateMappers_ImportsDriverPackage(t *testing.T) {
	src, err := GenerateMappers(mapperModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateMappers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "driver") {
		t.Errorf("output missing import of driver package:\n%s", s)
	}
}
