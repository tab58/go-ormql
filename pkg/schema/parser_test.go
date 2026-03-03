package schema

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseSchemaString_BasicNode verifies that a simple @node type produces a
// NodeDefinition with the correct name, default labels, and fields.
func TestParseSchemaString_BasicNode(t *testing.T) {
	sdl := `
		type Movie @node {
			id: ID!
			title: String!
			released: Int
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.Nodes) != 1 {
		t.Fatalf("len(Nodes) = %d, want 1", len(model.Nodes))
	}

	node := model.Nodes[0]
	if node.Name != "Movie" {
		t.Errorf("node.Name = %q, want %q", node.Name, "Movie")
	}
	if len(node.Labels) != 1 || node.Labels[0] != "Movie" {
		t.Errorf("node.Labels = %v, want [\"Movie\"]", node.Labels)
	}
	if len(node.Fields) != 3 {
		t.Fatalf("len(Fields) = %d, want 3", len(node.Fields))
	}
}

// TestParseSchemaString_FieldTypeMapping verifies that scalar fields are mapped
// to the correct Go and Cypher types per the type mapping table.
func TestParseSchemaString_FieldTypeMapping(t *testing.T) {
	sdl := `
		type Movie @node {
			id: ID!
			title: String!
			released: Int
			rating: Float
			active: Boolean!
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.Nodes) == 0 {
		t.Fatal("len(Nodes) = 0, want 1")
	}

	fields := model.Nodes[0].Fields
	expected := []struct {
		name       string
		goType     string
		cypherType string
		nullable   bool
		isID       bool
	}{
		{"id", "string", "STRING", false, true},
		{"title", "string", "STRING", false, false},
		{"released", "*int", "INTEGER", true, false},
		{"rating", "*float64", "FLOAT", true, false},
		{"active", "bool", "BOOLEAN", false, false},
	}

	if len(fields) != len(expected) {
		t.Fatalf("len(fields) = %d, want %d", len(fields), len(expected))
	}
	for i, exp := range expected {
		f := fields[i]
		if f.Name != exp.name {
			t.Errorf("fields[%d].Name = %q, want %q", i, f.Name, exp.name)
		}
		if f.GoType != exp.goType {
			t.Errorf("fields[%d].GoType = %q, want %q", i, f.GoType, exp.goType)
		}
		if f.CypherType != exp.cypherType {
			t.Errorf("fields[%d].CypherType = %q, want %q", i, f.CypherType, exp.cypherType)
		}
		if f.Nullable != exp.nullable {
			t.Errorf("fields[%d].Nullable = %v, want %v", i, f.Nullable, exp.nullable)
		}
		if f.IsID != exp.isID {
			t.Errorf("fields[%d].IsID = %v, want %v", i, f.IsID, exp.isID)
		}
	}
}

// TestParseSchemaString_NodeWithRelationship verifies that a @relationship field
// produces a RelationshipDefinition with correct FromNode, ToNode, RelType, Direction.
func TestParseSchemaString_NodeWithRelationship(t *testing.T) {
	sdl := `
		type Actor @node {
			name: String!
			movies: [Movie!]! @relationship(type: "ACTED_IN", direction: OUT)
		}
		type Movie @node {
			title: String!
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.Relationships) != 1 {
		t.Fatalf("len(Relationships) = %d, want 1", len(model.Relationships))
	}

	rel := model.Relationships[0]
	if rel.FieldName != "movies" {
		t.Errorf("rel.FieldName = %q, want %q", rel.FieldName, "movies")
	}
	if rel.RelType != "ACTED_IN" {
		t.Errorf("rel.RelType = %q, want %q", rel.RelType, "ACTED_IN")
	}
	if rel.Direction != DirectionOUT {
		t.Errorf("rel.Direction = %q, want %q", rel.Direction, DirectionOUT)
	}
	if rel.FromNode != "Actor" {
		t.Errorf("rel.FromNode = %q, want %q", rel.FromNode, "Actor")
	}
	if rel.ToNode != "Movie" {
		t.Errorf("rel.ToNode = %q, want %q", rel.ToNode, "Movie")
	}
	if rel.Properties != nil {
		t.Errorf("rel.Properties = %v, want nil", rel.Properties)
	}
}

// TestParseSchemaString_RelationshipProperties verifies that @relationshipProperties types
// are captured and linked to the @relationship that references them.
func TestParseSchemaString_RelationshipProperties(t *testing.T) {
	sdl := `
		type Actor @node {
			name: String!
			movies: [Movie!]! @relationship(type: "ACTED_IN", direction: OUT, properties: "ActedInProperties")
		}
		type Movie @node {
			title: String!
		}
		type ActedInProperties @relationshipProperties {
			role: String!
			year: Int
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.Relationships) != 1 {
		t.Fatalf("len(Relationships) = %d, want 1", len(model.Relationships))
	}

	rel := model.Relationships[0]
	if rel.Properties == nil {
		t.Fatal("rel.Properties is nil, want non-nil PropertiesDefinition")
	}
	if rel.Properties.TypeName != "ActedInProperties" {
		t.Errorf("Properties.TypeName = %q, want %q", rel.Properties.TypeName, "ActedInProperties")
	}
	if len(rel.Properties.Fields) != 2 {
		t.Fatalf("len(Properties.Fields) = %d, want 2", len(rel.Properties.Fields))
	}
	if rel.Properties.Fields[0].Name != "role" {
		t.Errorf("Properties.Fields[0].Name = %q, want %q", rel.Properties.Fields[0].Name, "role")
	}
}

// TestParseSchemaString_Enums verifies that GraphQL enum types are captured as EnumDefinitions.
func TestParseSchemaString_Enums(t *testing.T) {
	sdl := `
		enum Status {
			ACTIVE
			INACTIVE
			PENDING
		}
		type Movie @node {
			title: String!
			status: Status!
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.Enums) != 1 {
		t.Fatalf("len(Enums) = %d, want 1", len(model.Enums))
	}

	enum := model.Enums[0]
	if enum.Name != "Status" {
		t.Errorf("enum.Name = %q, want %q", enum.Name, "Status")
	}
	if len(enum.Values) != 3 {
		t.Fatalf("len(Values) = %d, want 3", len(enum.Values))
	}
	expectedVals := []string{"ACTIVE", "INACTIVE", "PENDING"}
	for i, v := range expectedVals {
		if enum.Values[i] != v {
			t.Errorf("Values[%d] = %q, want %q", i, enum.Values[i], v)
		}
	}
}

// TestParseSchemaString_SelfReferencing verifies that self-referencing relationships
// (FromNode == ToNode) are valid and correctly modeled.
func TestParseSchemaString_SelfReferencing(t *testing.T) {
	sdl := `
		type User @node {
			name: String!
			friends: [User!]! @relationship(type: "FRIENDS_WITH", direction: OUT)
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.Relationships) != 1 {
		t.Fatalf("len(Relationships) = %d, want 1", len(model.Relationships))
	}

	rel := model.Relationships[0]
	if rel.FromNode != "User" || rel.ToNode != "User" {
		t.Errorf("rel FromNode=%q, ToNode=%q, want both \"User\"", rel.FromNode, rel.ToNode)
	}
}

// TestParseSchemaString_MultipleRelsBetweenSameTypes verifies that multiple relationships
// between the same two types each produce a separate RelationshipDefinition.
func TestParseSchemaString_MultipleRelsBetweenSameTypes(t *testing.T) {
	sdl := `
		type Person @node {
			name: String!
			actedIn: [Movie!]! @relationship(type: "ACTED_IN", direction: OUT)
			directed: [Movie!]! @relationship(type: "DIRECTED", direction: OUT)
		}
		type Movie @node {
			title: String!
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.Relationships) != 2 {
		t.Fatalf("len(Relationships) = %d, want 2", len(model.Relationships))
	}

	relTypes := map[string]bool{}
	for _, r := range model.Relationships {
		relTypes[r.RelType] = true
	}
	if !relTypes["ACTED_IN"] {
		t.Error("missing ACTED_IN relationship")
	}
	if !relTypes["DIRECTED"] {
		t.Error("missing DIRECTED relationship")
	}
}

// TestParseSchemaString_NoRelationships verifies that a schema with nodes but no
// relationships produces a valid GraphModel with empty Relationships slice.
func TestParseSchemaString_NoRelationships(t *testing.T) {
	sdl := `
		type Movie @node {
			title: String!
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.Relationships) != 0 {
		t.Errorf("len(Relationships) = %d, want 0", len(model.Relationships))
	}
}

// TestParseSchemaString_InvalidSyntax verifies that a schema with syntax errors returns an error.
func TestParseSchemaString_InvalidSyntax(t *testing.T) {
	sdl := `
		type Movie @node {
			title String!
		}
	`

	_, err := ParseSchemaString(sdl)
	if err == nil {
		t.Fatal("ParseSchemaString returned nil error for invalid syntax, want error")
	}
}

// TestParseSchemaString_InvalidDirective verifies that a schema with invalid directive
// usage (e.g., missing required @relationship args) returns an error.
func TestParseSchemaString_InvalidDirective(t *testing.T) {
	sdl := `
		type Actor @node {
			name: String!
			movies: [Movie!]! @relationship(direction: OUT)
		}
		type Movie @node {
			title: String!
		}
	`

	_, err := ParseSchemaString(sdl)
	if err == nil {
		t.Fatal("ParseSchemaString returned nil error for @relationship missing 'type' arg, want error")
	}
}

// TestParseSchemaString_NonAnnotatedTypesIgnored verifies that types without @node or
// @relationshipProperties are not included in the GraphModel.
func TestParseSchemaString_NonAnnotatedTypesIgnored(t *testing.T) {
	sdl := `
		type Movie @node {
			title: String!
		}
		type SomeUtilityType {
			value: String
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.Nodes) != 1 {
		t.Fatalf("len(Nodes) = %d, want 1 (non-annotated type should be ignored)", len(model.Nodes))
	}
	if model.Nodes[0].Name != "Movie" {
		t.Errorf("Nodes[0].Name = %q, want %q", model.Nodes[0].Name, "Movie")
	}
}

// TestParseSchemaString_MultipleNodes verifies that multiple @node types are all captured.
func TestParseSchemaString_MultipleNodes(t *testing.T) {
	sdl := `
		type Movie @node {
			title: String!
		}
		type Actor @node {
			name: String!
		}
		type Director @node {
			name: String!
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.Nodes) != 3 {
		t.Fatalf("len(Nodes) = %d, want 3", len(model.Nodes))
	}

	names := map[string]bool{}
	for _, n := range model.Nodes {
		names[n.Name] = true
	}
	for _, want := range []string{"Movie", "Actor", "Director"} {
		if !names[want] {
			t.Errorf("missing node %q", want)
		}
	}
}

// TestParseSchemaString_RelationshipFieldExcludedFromNodeFields verifies that fields
// annotated with @relationship are NOT included in the node's Fields slice
// (Fields should contain only scalar fields).
func TestParseSchemaString_RelationshipFieldExcludedFromNodeFields(t *testing.T) {
	sdl := `
		type Actor @node {
			name: String!
			movies: [Movie!]! @relationship(type: "ACTED_IN", direction: OUT)
		}
		type Movie @node {
			title: String!
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}

	// Find Actor node
	var actorNode *NodeDefinition
	for i := range model.Nodes {
		if model.Nodes[i].Name == "Actor" {
			actorNode = &model.Nodes[i]
			break
		}
	}
	if actorNode == nil {
		t.Fatal("Actor node not found")
	}

	// Actor should only have "name" field, not "movies" (that's a relationship)
	if len(actorNode.Fields) != 1 {
		t.Fatalf("len(Actor.Fields) = %d, want 1 (relationship fields excluded)", len(actorNode.Fields))
	}
	if actorNode.Fields[0].Name != "name" {
		t.Errorf("Actor.Fields[0].Name = %q, want %q", actorNode.Fields[0].Name, "name")
	}
}

// TestParseSchema_FilePaths verifies that ParseSchema reads from actual .graphql files.
func TestParseSchema_FilePaths(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.graphql")

	sdl := `type Movie @node {
	id: ID!
	title: String!
}
`
	if err := os.WriteFile(schemaPath, []byte(sdl), 0644); err != nil {
		t.Fatalf("failed to write temp schema file: %v", err)
	}

	model, err := ParseSchema([]string{schemaPath})
	if err != nil {
		t.Fatalf("ParseSchema returned error: %v", err)
	}
	if len(model.Nodes) != 1 {
		t.Fatalf("len(Nodes) = %d, want 1", len(model.Nodes))
	}
	if model.Nodes[0].Name != "Movie" {
		t.Errorf("Nodes[0].Name = %q, want %q", model.Nodes[0].Name, "Movie")
	}
}

// TestParseSchemaString_CustomScalars verifies that custom scalar declarations
// are collected into GraphModel.CustomScalars, sorted alphabetically.
func TestParseSchemaString_CustomScalars(t *testing.T) {
	sdl := `
		scalar DateTime
		scalar JSON
		type Movie @node {
			id: ID!
			title: String!
			createdAt: DateTime!
			metadata: JSON
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.CustomScalars) != 2 {
		t.Fatalf("len(CustomScalars) = %d, want 2", len(model.CustomScalars))
	}
	// Should be sorted alphabetically.
	if model.CustomScalars[0] != "DateTime" {
		t.Errorf("CustomScalars[0] = %q, want %q", model.CustomScalars[0], "DateTime")
	}
	if model.CustomScalars[1] != "JSON" {
		t.Errorf("CustomScalars[1] = %q, want %q", model.CustomScalars[1], "JSON")
	}
}

// TestParseSchemaString_NoCustomScalars verifies that schemas without custom
// scalars produce an empty CustomScalars slice.
func TestParseSchemaString_NoCustomScalars(t *testing.T) {
	sdl := `
		type Movie @node {
			id: ID!
			title: String!
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.CustomScalars) != 0 {
		t.Errorf("len(CustomScalars) = %d, want 0", len(model.CustomScalars))
	}
}

// TestParseSchemaString_CustomScalarFieldTypes verifies that fields using custom
// scalars get the correct Go and Cypher type mappings.
func TestParseSchemaString_CustomScalarFieldTypes(t *testing.T) {
	sdl := `
		scalar DateTime
		type Event @node {
			id: ID!
			startTime: DateTime!
			endTime: DateTime
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.Nodes) == 0 {
		t.Fatal("len(Nodes) = 0, want 1")
	}

	fields := model.Nodes[0].Fields
	expected := []struct {
		name       string
		goType     string
		cypherType string
		nullable   bool
	}{
		{"id", "string", "STRING", false},
		{"startTime", "time.Time", "LOCAL DATETIME", false},
		{"endTime", "*time.Time", "LOCAL DATETIME", true},
	}

	if len(fields) != len(expected) {
		t.Fatalf("len(fields) = %d, want %d", len(fields), len(expected))
	}
	for i, exp := range expected {
		f := fields[i]
		if f.Name != exp.name {
			t.Errorf("fields[%d].Name = %q, want %q", i, f.Name, exp.name)
		}
		if f.GoType != exp.goType {
			t.Errorf("fields[%d].GoType = %q, want %q", i, f.GoType, exp.goType)
		}
		if f.CypherType != exp.cypherType {
			t.Errorf("fields[%d].CypherType = %q, want %q", i, f.CypherType, exp.cypherType)
		}
		if f.Nullable != exp.nullable {
			t.Errorf("fields[%d].Nullable = %v, want %v", i, f.Nullable, exp.nullable)
		}
	}
}

// TestParseSchemaString_UnknownCustomScalar verifies that unknown custom scalars
// (not in the known map) are still collected and fields using them fall through
// to object type inference.
func TestParseSchemaString_UnknownCustomScalar(t *testing.T) {
	sdl := `
		scalar Money
		type Product @node {
			id: ID!
			price: Money!
		}
	`

	model, err := ParseSchemaString(sdl)
	if err != nil {
		t.Fatalf("ParseSchemaString returned error: %v", err)
	}
	if len(model.CustomScalars) != 1 || model.CustomScalars[0] != "Money" {
		t.Fatalf("CustomScalars = %v, want [Money]", model.CustomScalars)
	}
}

// TestParseSchema_FileNotFound verifies that ParseSchema returns an error for nonexistent files.
func TestParseSchema_FileNotFound(t *testing.T) {
	_, err := ParseSchema([]string{"/nonexistent/path/schema.graphql"})
	if err == nil {
		t.Fatal("ParseSchema returned nil error for nonexistent file, want error")
	}
}
