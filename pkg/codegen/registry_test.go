package codegen

import (
	"go/format"
	"strings"
	"testing"

	"github.com/tab58/go-ormql/pkg/schema"
)

// --- CG-21: GraphModel registry generator tests ---

const testAugSchemaSDL = `type Query {
  movies: [Movie!]!
}
type Movie {
  id: ID!
  title: String!
}`

// Test: GenerateGraphModelRegistry returns non-nil output for a valid model.
// Expected: non-nil, non-empty byte slice.
func TestGenerateGraphModelRegistry_ReturnsOutput(t *testing.T) {
	out, err := GenerateGraphModelRegistry(fullModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || len(out) == 0 {
		t.Fatal("GenerateGraphModelRegistry should return non-nil, non-empty output")
	}
}

// Test: Output starts with correct package declaration.
// Expected: output contains "package generated".
func TestGenerateGraphModelRegistry_PackageDeclaration(t *testing.T) {
	out, err := GenerateGraphModelRegistry(fullModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(out), "package generated") {
		t.Errorf("expected 'package generated' in output, got:\n%s", string(out))
	}
}

// Test: Output contains var GraphModel declaration with schema.GraphModel type.
// Expected: output contains "var GraphModel = schema.GraphModel{".
func TestGenerateGraphModelRegistry_GraphModelVar(t *testing.T) {
	out, err := GenerateGraphModelRegistry(fullModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "var GraphModel") {
		t.Error("expected 'var GraphModel' declaration in output")
	}
	if !strings.Contains(src, "schema.GraphModel") {
		t.Error("expected 'schema.GraphModel' type reference in output")
	}
}

// Test: Output serializes all node definitions with names and labels.
// Expected: output contains "Movie" and "Actor" node names and labels.
func TestGenerateGraphModelRegistry_SerializesNodes(t *testing.T) {
	out, err := GenerateGraphModelRegistry(fullModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, `"Movie"`) {
		t.Error("expected 'Movie' node name in serialized GraphModel")
	}
	if !strings.Contains(src, `"Actor"`) {
		t.Error("expected 'Actor' node name in serialized GraphModel")
	}
}

// Test: Output serializes field definitions including GraphQLType and GoType.
// Expected: output contains field properties like "ID!", "string", etc.
func TestGenerateGraphModelRegistry_SerializesFields(t *testing.T) {
	out, err := GenerateGraphModelRegistry(fullModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, `"title"`) {
		t.Error("expected field name 'title' in serialized GraphModel")
	}
	if !strings.Contains(src, `"String!"`) {
		t.Error("expected GraphQLType 'String!' in serialized fields")
	}
}

// Test: Output serializes relationship definitions with type, direction, and properties.
// Expected: output contains ACTED_IN relationship with Properties reference.
func TestGenerateGraphModelRegistry_SerializesRelationships(t *testing.T) {
	out, err := GenerateGraphModelRegistry(fullModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, `"ACTED_IN"`) {
		t.Error("expected 'ACTED_IN' rel type in serialized GraphModel")
	}
	if !strings.Contains(src, "DirectionIN") || !strings.Contains(src, "schema.DirectionIN") {
		t.Error("expected schema.DirectionIN in serialized relationship")
	}
	if !strings.Contains(src, "ActedInProperties") {
		t.Error("expected 'ActedInProperties' in serialized properties")
	}
}

// Test: Output serializes @cypher field definitions.
// Expected: output contains cypher statement and field metadata.
func TestGenerateGraphModelRegistry_SerializesCypherFields(t *testing.T) {
	out, err := GenerateGraphModelRegistry(fullModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, `"averageRating"`) {
		t.Error("expected 'averageRating' cypher field in serialized GraphModel")
	}
	if !strings.Contains(src, "MATCH (this)") {
		t.Error("expected cypher statement content in serialized GraphModel")
	}
}

// Test: Output serializes enum definitions.
// Expected: output contains Genre enum with its values.
func TestGenerateGraphModelRegistry_SerializesEnums(t *testing.T) {
	out, err := GenerateGraphModelRegistry(fullModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, `"Genre"`) {
		t.Error("expected 'Genre' enum name in serialized GraphModel")
	}
	if !strings.Contains(src, `"ACTION"`) || !strings.Contains(src, `"COMEDY"`) || !strings.Contains(src, `"DRAMA"`) {
		t.Error("expected enum values ACTION, COMEDY, DRAMA in serialized GraphModel")
	}
}

// Test: Output contains var AugmentedSchemaSDL with the augmented schema.
// Expected: output contains "var AugmentedSchemaSDL" and the schema content.
func TestGenerateGraphModelRegistry_AugmentedSchemaSDL(t *testing.T) {
	out, err := GenerateGraphModelRegistry(fullModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "var AugmentedSchemaSDL") {
		t.Error("expected 'var AugmentedSchemaSDL' declaration in output")
	}
	if !strings.Contains(src, "Movie") {
		t.Error("expected schema content containing 'Movie' in AugmentedSchemaSDL")
	}
}

// Test: Output imports the schema package.
// Expected: output contains import of the schema package path.
func TestGenerateGraphModelRegistry_ImportsSchemaPackage(t *testing.T) {
	out, err := GenerateGraphModelRegistry(fullModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, `"github.com/tab58/go-ormql/pkg/schema"`) {
		t.Error("expected schema package import in output")
	}
}

// Test: Output passes gofmt (well-formatted Go source).
// Expected: gofmt.Source returns no error.
func TestGenerateGraphModelRegistry_PassesGofmt(t *testing.T) {
	out, err := GenerateGraphModelRegistry(fullModel(), testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil, cannot check gofmt")
	}
	_, fmtErr := format.Source(out)
	if fmtErr != nil {
		t.Errorf("output does not pass gofmt: %v\nOutput:\n%s", fmtErr, string(out))
	}
}

// Test: Output handles augmented schema SDL with backtick characters.
// Expected: backticks in SDL are properly escaped (quoted string or escaped).
func TestGenerateGraphModelRegistry_EscapesBackticks(t *testing.T) {
	sdlWithBacktick := "type Query {\n  movies: [Movie!]! # uses `Movie` type\n}"
	out, err := GenerateGraphModelRegistry(fullModel(), sdlWithBacktick, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("output is nil for SDL with backticks")
	}
	// The generated code should compile — backticks must be escaped
	_, fmtErr := format.Source(out)
	if fmtErr != nil {
		t.Errorf("output with backtick SDL does not pass gofmt: %v", fmtErr)
	}
}

// Test: GenerateGraphModelRegistry with empty model (zero nodes) returns valid output.
// Expected: output contains empty Nodes slice.
func TestGenerateGraphModelRegistry_EmptyModel(t *testing.T) {
	out, err := GenerateGraphModelRegistry(schema.GraphModel{}, testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || len(out) == 0 {
		t.Fatal("expected non-nil output even for empty model")
	}
}

// Test: GenerateGraphModelRegistry with empty packageName returns error.
// Expected: non-nil error.
func TestGenerateGraphModelRegistry_EmptyPackageName(t *testing.T) {
	_, err := GenerateGraphModelRegistry(fullModel(), testAugSchemaSDL, "")
	if err == nil {
		t.Error("GenerateGraphModelRegistry with empty packageName should return error")
	}
}

// === CG-35: IsList serialization in registry tests ===

// TestGenerateGraphModelRegistry_IsListTrue verifies that IsList: true is serialized
// for to-many relationships in the GraphModel registry.
// Expected: generated output contains "IsList: true".
func TestGenerateGraphModelRegistry_IsListTrue(t *testing.T) {
	model := schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{Name: "Movie", Labels: []string{"Movie"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
			}},
			{Name: "Actor", Labels: []string{"Actor"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
			}},
		},
		Relationships: []schema.RelationshipDefinition{
			{FieldName: "actors", RelType: "ACTED_IN", Direction: schema.DirectionIN, FromNode: "Movie", ToNode: "Actor", IsList: true},
		},
	}

	out, err := GenerateGraphModelRegistry(model, testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "IsList: true") {
		t.Errorf("registry output missing 'IsList: true' for to-many relationship:\n%s", src)
	}
}

// TestGenerateGraphModelRegistry_IsListFalse verifies that IsList: false is serialized
// for to-one relationships in the GraphModel registry.
// Expected: generated output contains "IsList: false".
func TestGenerateGraphModelRegistry_IsListFalse(t *testing.T) {
	model := schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{Name: "Movie", Labels: []string{"Movie"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
			}},
			{Name: "Repository", Labels: []string{"Repository"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
			}},
		},
		Relationships: []schema.RelationshipDefinition{
			{FieldName: "repository", RelType: "BELONGS_TO", Direction: schema.DirectionOUT, FromNode: "Movie", ToNode: "Repository", IsList: false},
		},
	}

	out, err := GenerateGraphModelRegistry(model, testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "IsList: false") {
		t.Errorf("registry output missing 'IsList: false' for to-one relationship:\n%s", src)
	}
}

// TestGenerateGraphModelRegistry_IsListMixed verifies that both IsList values appear
// when the model has both to-one and to-many relationships.
func TestGenerateGraphModelRegistry_IsListMixed(t *testing.T) {
	model := schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{Name: "Movie", Labels: []string{"Movie"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
			}},
			{Name: "Actor", Labels: []string{"Actor"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
			}},
			{Name: "Repository", Labels: []string{"Repository"}, Fields: []schema.FieldDefinition{
				{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
			}},
		},
		Relationships: []schema.RelationshipDefinition{
			{FieldName: "actors", RelType: "ACTED_IN", Direction: schema.DirectionIN, FromNode: "Movie", ToNode: "Actor", IsList: true},
			{FieldName: "repository", RelType: "BELONGS_TO", Direction: schema.DirectionOUT, FromNode: "Movie", ToNode: "Repository", IsList: false},
		},
	}

	out, err := GenerateGraphModelRegistry(model, testAugSchemaSDL, "generated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := string(out)
	if !strings.Contains(src, "IsList: true") {
		t.Errorf("registry output missing 'IsList: true':\n%s", src)
	}
	if !strings.Contains(src, "IsList: false") {
		t.Errorf("registry output missing 'IsList: false':\n%s", src)
	}
}
