package schema

import (
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
)

// --- Test helpers ---

// makeDirective creates an ast.Directive with the given name and arguments.
func makeDirective(name string, args ...[2]string) *ast.Directive {
	d := &ast.Directive{
		Name:     name,
		Position: &ast.Position{Src: &ast.Source{Name: "test.graphql"}, Line: 1, Column: 1},
	}
	for _, arg := range args {
		d.Arguments = append(d.Arguments, &ast.Argument{
			Name: arg[0],
			Value: &ast.Value{
				Raw:  arg[1],
				Kind: ast.EnumValue,
			},
		})
	}
	return d
}

// makeTypeDef creates an ast.Definition (OBJECT) with the given name and directives.
func makeTypeDef(name string, directives ...*ast.Directive) *ast.Definition {
	return &ast.Definition{
		Kind:       ast.Object,
		Name:       name,
		Directives: ast.DirectiveList(directives),
		Position:   &ast.Position{Src: &ast.Source{Name: "test.graphql"}, Line: 1, Column: 1},
	}
}

// makeFieldDef creates an ast.FieldDefinition with the given name, type, and directives.
func makeFieldDef(name string, typeName string, directives ...*ast.Directive) *ast.FieldDefinition {
	return &ast.FieldDefinition{
		Name: name,
		Type: &ast.Type{NamedType: typeName},
		Directives: ast.DirectiveList(directives),
		Position:   &ast.Position{Src: &ast.Source{Name: "test.graphql"}, Line: 1, Column: 1},
	}
}

// --- ExtractNodeDirective tests ---

// TestExtractNodeDirective_WithDirective verifies that a type with @node returns HasDirective=true.
func TestExtractNodeDirective_WithDirective(t *testing.T) {
	def := makeTypeDef("Movie", makeDirective("node"))

	info := ExtractNodeDirective(def)
	if !info.HasDirective {
		t.Error("ExtractNodeDirective returned HasDirective=false for type with @node, want true")
	}
}

// TestExtractNodeDirective_WithoutDirective verifies that a type without @node returns HasDirective=false.
func TestExtractNodeDirective_WithoutDirective(t *testing.T) {
	def := makeTypeDef("Movie")

	info := ExtractNodeDirective(def)
	if info.HasDirective {
		t.Error("ExtractNodeDirective returned HasDirective=true for type without @node, want false")
	}
}

// TestExtractNodeDirective_NilDef verifies that a nil definition returns HasDirective=false without panicking.
func TestExtractNodeDirective_NilDef(t *testing.T) {
	info := ExtractNodeDirective(nil)
	if info.HasDirective {
		t.Error("ExtractNodeDirective returned HasDirective=true for nil def, want false")
	}
}

// --- ExtractRelationshipDirective tests ---

// TestExtractRelationshipDirective_WithDirective verifies that a field with @relationship
// extracts type, direction correctly.
// Expected: HasDirective=true, RelType="ACTED_IN", Direction=DirectionOUT, Properties="".
func TestExtractRelationshipDirective_WithDirective(t *testing.T) {
	dir := makeDirective("relationship",
		[2]string{"type", "ACTED_IN"},
		[2]string{"direction", "OUT"},
	)
	field := makeFieldDef("movies", "Movie", dir)

	info := ExtractRelationshipDirective(field)
	if !info.HasDirective {
		t.Fatal("HasDirective = false, want true")
	}
	if info.RelType != "ACTED_IN" {
		t.Errorf("RelType = %q, want %q", info.RelType, "ACTED_IN")
	}
	if info.Direction != DirectionOUT {
		t.Errorf("Direction = %q, want %q", info.Direction, DirectionOUT)
	}
	if info.Properties != "" {
		t.Errorf("Properties = %q, want empty string", info.Properties)
	}
}

// TestExtractRelationshipDirective_WithProperties verifies that the optional properties arg is extracted.
// Expected: Properties="ActedInProperties".
func TestExtractRelationshipDirective_WithProperties(t *testing.T) {
	dir := makeDirective("relationship",
		[2]string{"type", "ACTED_IN"},
		[2]string{"direction", "OUT"},
		[2]string{"properties", "ActedInProperties"},
	)
	field := makeFieldDef("movies", "Movie", dir)

	info := ExtractRelationshipDirective(field)
	if !info.HasDirective {
		t.Fatal("HasDirective = false, want true")
	}
	if info.Properties != "ActedInProperties" {
		t.Errorf("Properties = %q, want %q", info.Properties, "ActedInProperties")
	}
}

// TestExtractRelationshipDirective_INDirection verifies that direction=IN is extracted correctly.
func TestExtractRelationshipDirective_INDirection(t *testing.T) {
	dir := makeDirective("relationship",
		[2]string{"type", "ACTED_IN"},
		[2]string{"direction", "IN"},
	)
	field := makeFieldDef("actors", "Actor", dir)

	info := ExtractRelationshipDirective(field)
	if !info.HasDirective {
		t.Fatal("HasDirective = false, want true")
	}
	if info.Direction != DirectionIN {
		t.Errorf("Direction = %q, want %q", info.Direction, DirectionIN)
	}
}

// TestExtractRelationshipDirective_WithoutDirective verifies that a field without @relationship
// returns HasDirective=false.
func TestExtractRelationshipDirective_WithoutDirective(t *testing.T) {
	field := makeFieldDef("title", "String")

	info := ExtractRelationshipDirective(field)
	if info.HasDirective {
		t.Error("HasDirective = true for field without @relationship, want false")
	}
}

// TestExtractRelationshipDirective_NilField verifies nil field returns HasDirective=false without panicking.
func TestExtractRelationshipDirective_NilField(t *testing.T) {
	info := ExtractRelationshipDirective(nil)
	if info.HasDirective {
		t.Error("HasDirective = true for nil field, want false")
	}
}

// --- HasRelationshipPropertiesDirective tests ---

// TestHasRelationshipPropertiesDirective_WithDirective verifies that a type with
// @relationshipProperties returns true.
func TestHasRelationshipPropertiesDirective_WithDirective(t *testing.T) {
	def := makeTypeDef("ActedInProperties", makeDirective("relationshipProperties"))

	if !HasRelationshipPropertiesDirective(def) {
		t.Error("HasRelationshipPropertiesDirective returned false for type with @relationshipProperties, want true")
	}
}

// TestHasRelationshipPropertiesDirective_WithoutDirective verifies that a type without
// @relationshipProperties returns false.
func TestHasRelationshipPropertiesDirective_WithoutDirective(t *testing.T) {
	def := makeTypeDef("Movie")

	if HasRelationshipPropertiesDirective(def) {
		t.Error("HasRelationshipPropertiesDirective returned true for type without directive, want false")
	}
}

// TestHasRelationshipPropertiesDirective_NilDef verifies nil definition returns false without panicking.
func TestHasRelationshipPropertiesDirective_NilDef(t *testing.T) {
	if HasRelationshipPropertiesDirective(nil) {
		t.Error("HasRelationshipPropertiesDirective returned true for nil def, want false")
	}
}

// --- ValidateDirectives tests ---

// TestValidateDirectives_ValidSchema verifies that a well-formed schema produces no errors.
func TestValidateDirectives_ValidSchema(t *testing.T) {
	propsType := makeTypeDef("ActedInProperties", makeDirective("relationshipProperties"))
	propsType.Fields = ast.FieldList{
		makeFieldDef("role", "String"),
	}

	relDir := makeDirective("relationship",
		[2]string{"type", "ACTED_IN"},
		[2]string{"direction", "OUT"},
		[2]string{"properties", "ActedInProperties"},
	)

	movieType := makeTypeDef("Movie", makeDirective("node"))
	actorType := makeTypeDef("Actor", makeDirective("node"))
	actorType.Fields = ast.FieldList{
		makeFieldDef("name", "String"),
		makeFieldDef("movies", "Movie", relDir),
	}

	doc := &ast.SchemaDocument{
		Definitions: []*ast.Definition{movieType, actorType, propsType},
	}

	errs := ValidateDirectives(doc)
	if len(errs) != 0 {
		t.Errorf("ValidateDirectives returned %d errors for valid schema, want 0: %v", len(errs), errs)
	}
}

// TestValidateDirectives_MissingTypeArg verifies that @relationship without type arg produces an error.
func TestValidateDirectives_MissingTypeArg(t *testing.T) {
	dir := makeDirective("relationship",
		[2]string{"direction", "OUT"},
	)
	nodeType := makeTypeDef("Actor", makeDirective("node"))
	nodeType.Fields = ast.FieldList{
		makeFieldDef("movies", "Movie", dir),
	}

	doc := &ast.SchemaDocument{
		Definitions: []*ast.Definition{nodeType},
	}

	errs := ValidateDirectives(doc)
	if len(errs) == 0 {
		t.Fatal("ValidateDirectives returned 0 errors for @relationship missing 'type' arg, want >= 1")
	}
}

// TestValidateDirectives_MissingDirectionArg verifies that @relationship without direction arg produces an error.
func TestValidateDirectives_MissingDirectionArg(t *testing.T) {
	dir := makeDirective("relationship",
		[2]string{"type", "ACTED_IN"},
	)
	nodeType := makeTypeDef("Actor", makeDirective("node"))
	nodeType.Fields = ast.FieldList{
		makeFieldDef("movies", "Movie", dir),
	}

	doc := &ast.SchemaDocument{
		Definitions: []*ast.Definition{nodeType},
	}

	errs := ValidateDirectives(doc)
	if len(errs) == 0 {
		t.Fatal("ValidateDirectives returned 0 errors for @relationship missing 'direction' arg, want >= 1")
	}
}

// TestValidateDirectives_UnknownDirection verifies that @relationship with direction="BOTH"
// produces a validation error.
func TestValidateDirectives_UnknownDirection(t *testing.T) {
	dir := makeDirective("relationship",
		[2]string{"type", "ACTED_IN"},
		[2]string{"direction", "BOTH"},
	)
	nodeType := makeTypeDef("Actor", makeDirective("node"))
	nodeType.Fields = ast.FieldList{
		makeFieldDef("movies", "Movie", dir),
	}

	doc := &ast.SchemaDocument{
		Definitions: []*ast.Definition{nodeType},
	}

	errs := ValidateDirectives(doc)
	if len(errs) == 0 {
		t.Fatal("ValidateDirectives returned 0 errors for unknown direction 'BOTH', want >= 1")
	}
}

// TestValidateDirectives_PropertiesRefNonexistent verifies that @relationship referencing
// a properties type that does not exist in the schema produces an error.
func TestValidateDirectives_PropertiesRefNonexistent(t *testing.T) {
	dir := makeDirective("relationship",
		[2]string{"type", "ACTED_IN"},
		[2]string{"direction", "OUT"},
		[2]string{"properties", "NonExistentProps"},
	)
	nodeType := makeTypeDef("Actor", makeDirective("node"))
	nodeType.Fields = ast.FieldList{
		makeFieldDef("movies", "Movie", dir),
	}

	doc := &ast.SchemaDocument{
		Definitions: []*ast.Definition{nodeType},
	}

	errs := ValidateDirectives(doc)
	if len(errs) == 0 {
		t.Fatal("ValidateDirectives returned 0 errors for properties referencing nonexistent type, want >= 1")
	}
}

// TestValidateDirectives_EmptyDocument verifies that an empty schema document produces no errors.
func TestValidateDirectives_EmptyDocument(t *testing.T) {
	doc := &ast.SchemaDocument{}

	errs := ValidateDirectives(doc)
	if len(errs) != 0 {
		t.Errorf("ValidateDirectives returned %d errors for empty document, want 0: %v", len(errs), errs)
	}
}

// --- BuiltinDirectiveDefs tests ---

// TestBuiltinDirectiveDefs_NonEmpty verifies that BuiltinDirectiveDefs returns a non-empty string
// containing the directive definitions for @node, @relationship, and @relationshipProperties.
func TestBuiltinDirectiveDefs_NonEmpty(t *testing.T) {
	defs := BuiltinDirectiveDefs()
	if defs == "" {
		t.Fatal("BuiltinDirectiveDefs() returned empty string, want non-empty directive definitions")
	}
}

// TestBuiltinDirectiveDefs_ContainsDirectives verifies that the returned SDL string
// contains all three expected directive names.
func TestBuiltinDirectiveDefs_ContainsDirectives(t *testing.T) {
	defs := BuiltinDirectiveDefs()

	expected := []string{
		"directive @node",
		"directive @relationship",
		"directive @relationshipProperties",
	}
	for _, want := range expected {
		found := false
		for i := 0; i <= len(defs)-len(want); i++ {
			if defs[i:i+len(want)] == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("BuiltinDirectiveDefs() missing %q", want)
		}
	}
}
