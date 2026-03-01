package schema

import "testing"

// TestDirectionConstants verifies that Direction constants have the expected string values.
func TestDirectionConstants(t *testing.T) {
	tests := []struct {
		name     string
		dir      Direction
		expected string
	}{
		{"IN direction", DirectionIN, "IN"},
		{"OUT direction", DirectionOUT, "OUT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.dir) != tt.expected {
				t.Errorf("Direction = %q, want %q", tt.dir, tt.expected)
			}
		})
	}
}

// TestNodeByName_Found verifies that NodeByName returns the correct node and true when the node exists.
func TestNodeByName_Found(t *testing.T) {
	model := GraphModel{
		Nodes: []NodeDefinition{
			{Name: "Movie", Labels: []string{"Movie"}, Fields: []FieldDefinition{
				{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
			}},
			{Name: "Actor", Labels: []string{"Actor"}, Fields: []FieldDefinition{
				{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
			}},
		},
	}

	node, ok := model.NodeByName("Movie")
	if !ok {
		t.Fatal("NodeByName(\"Movie\") returned false, want true")
	}
	if node.Name != "Movie" {
		t.Errorf("node.Name = %q, want %q", node.Name, "Movie")
	}
	if len(node.Labels) != 1 || node.Labels[0] != "Movie" {
		t.Errorf("node.Labels = %v, want [\"Movie\"]", node.Labels)
	}
	if len(node.Fields) != 1 || node.Fields[0].Name != "title" {
		t.Errorf("node.Fields = %v, want single field named \"title\"", node.Fields)
	}
}

// TestNodeByName_NotFound verifies that NodeByName returns zero value and false for a non-existent node.
func TestNodeByName_NotFound(t *testing.T) {
	model := GraphModel{
		Nodes: []NodeDefinition{
			{Name: "Movie", Labels: []string{"Movie"}},
		},
	}

	node, ok := model.NodeByName("NonExistent")
	if ok {
		t.Fatal("NodeByName(\"NonExistent\") returned true, want false")
	}
	if node.Name != "" {
		t.Errorf("node.Name = %q, want empty string for zero value", node.Name)
	}
}

// TestNodeByName_EmptyModel verifies that NodeByName returns false on a model with no nodes.
func TestNodeByName_EmptyModel(t *testing.T) {
	model := GraphModel{}

	_, ok := model.NodeByName("Movie")
	if ok {
		t.Fatal("NodeByName on empty model returned true, want false")
	}
}

// TestNodeByName_ReturnsCopy verifies that the returned NodeDefinition is a copy,
// not a reference to internal state. Modifying the returned value must not affect the model.
func TestNodeByName_ReturnsCopy(t *testing.T) {
	model := GraphModel{
		Nodes: []NodeDefinition{
			{Name: "Movie", Labels: []string{"Movie"}, Fields: []FieldDefinition{
				{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
			}},
		},
	}

	// Get a copy and mutate it
	node, ok := model.NodeByName("Movie")
	if !ok {
		t.Fatal("NodeByName(\"Movie\") returned false, want true — cannot test copy semantics")
	}
	node.Name = "MUTATED"
	if len(node.Labels) > 0 {
		node.Labels[0] = "MUTATED"
	}

	// Verify original is unchanged
	original, _ := model.NodeByName("Movie")
	if original.Name != "Movie" {
		t.Errorf("original.Name = %q after mutation, want %q — copy semantics violated", original.Name, "Movie")
	}
	if len(original.Labels) == 0 || original.Labels[0] != "Movie" {
		t.Errorf("original.Labels = %v after mutation, want [\"Movie\"] — slice was shared", original.Labels)
	}
}

// TestRelationshipsForNode_FromNode verifies that relationships where FromNode matches are returned.
func TestRelationshipsForNode_FromNode(t *testing.T) {
	model := GraphModel{
		Relationships: []RelationshipDefinition{
			{FieldName: "movies", RelType: "ACTED_IN", Direction: DirectionOUT, FromNode: "Actor", ToNode: "Movie"},
			{FieldName: "directors", RelType: "DIRECTED", Direction: DirectionOUT, FromNode: "Director", ToNode: "Movie"},
		},
	}

	rels := model.RelationshipsForNode("Actor")
	if len(rels) != 1 {
		t.Fatalf("RelationshipsForNode(\"Actor\") returned %d rels, want 1", len(rels))
	}
	if rels[0].RelType != "ACTED_IN" {
		t.Errorf("rels[0].RelType = %q, want %q", rels[0].RelType, "ACTED_IN")
	}
}

// TestRelationshipsForNode_ToNode verifies that relationships where ToNode matches are returned.
func TestRelationshipsForNode_ToNode(t *testing.T) {
	model := GraphModel{
		Relationships: []RelationshipDefinition{
			{FieldName: "movies", RelType: "ACTED_IN", Direction: DirectionOUT, FromNode: "Actor", ToNode: "Movie"},
		},
	}

	rels := model.RelationshipsForNode("Movie")
	if len(rels) != 1 {
		t.Fatalf("RelationshipsForNode(\"Movie\") returned %d rels, want 1", len(rels))
	}
	if rels[0].RelType != "ACTED_IN" {
		t.Errorf("rels[0].RelType = %q, want %q", rels[0].RelType, "ACTED_IN")
	}
}

// TestRelationshipsForNode_BothDirections verifies that a self-referencing relationship
// (FromNode == ToNode) is returned when queried for that node.
func TestRelationshipsForNode_BothDirections(t *testing.T) {
	model := GraphModel{
		Relationships: []RelationshipDefinition{
			{FieldName: "friends", RelType: "FRIENDS_WITH", Direction: DirectionOUT, FromNode: "User", ToNode: "User"},
			{FieldName: "movies", RelType: "ACTED_IN", Direction: DirectionOUT, FromNode: "Actor", ToNode: "Movie"},
		},
	}

	rels := model.RelationshipsForNode("User")
	if len(rels) != 1 {
		t.Fatalf("RelationshipsForNode(\"User\") returned %d rels, want 1 (self-referencing should not duplicate)", len(rels))
	}
}

// TestRelationshipsForNode_MultipleMatches verifies that a node appearing in multiple
// relationships (as either from or to) returns all of them.
func TestRelationshipsForNode_MultipleMatches(t *testing.T) {
	model := GraphModel{
		Relationships: []RelationshipDefinition{
			{FieldName: "actedIn", RelType: "ACTED_IN", Direction: DirectionOUT, FromNode: "Actor", ToNode: "Movie"},
			{FieldName: "directed", RelType: "DIRECTED", Direction: DirectionOUT, FromNode: "Director", ToNode: "Movie"},
			{FieldName: "produced", RelType: "PRODUCED", Direction: DirectionOUT, FromNode: "Producer", ToNode: "Movie"},
		},
	}

	rels := model.RelationshipsForNode("Movie")
	if len(rels) != 3 {
		t.Fatalf("RelationshipsForNode(\"Movie\") returned %d rels, want 3", len(rels))
	}
}

// TestRelationshipsForNode_NoMatches verifies that an empty slice is returned for an unknown node.
func TestRelationshipsForNode_NoMatches(t *testing.T) {
	model := GraphModel{
		Relationships: []RelationshipDefinition{
			{FieldName: "movies", RelType: "ACTED_IN", Direction: DirectionOUT, FromNode: "Actor", ToNode: "Movie"},
		},
	}

	rels := model.RelationshipsForNode("NonExistent")
	if len(rels) != 0 {
		t.Fatalf("RelationshipsForNode(\"NonExistent\") returned %d rels, want 0", len(rels))
	}
}

// TestRelationshipsForNode_EmptyModel verifies that an empty (or nil) slice is returned on a model with no relationships.
func TestRelationshipsForNode_EmptyModel(t *testing.T) {
	model := GraphModel{}

	rels := model.RelationshipsForNode("Movie")
	if len(rels) != 0 {
		t.Fatalf("RelationshipsForNode on empty model returned %d rels, want 0", len(rels))
	}
}

// TestRelationshipsForNode_ReturnsCopies verifies that modifying returned relationships
// does not affect the model's internal state.
func TestRelationshipsForNode_ReturnsCopies(t *testing.T) {
	model := GraphModel{
		Relationships: []RelationshipDefinition{
			{FieldName: "movies", RelType: "ACTED_IN", Direction: DirectionOUT, FromNode: "Actor", ToNode: "Movie"},
		},
	}

	rels := model.RelationshipsForNode("Actor")
	if len(rels) == 0 {
		t.Fatal("RelationshipsForNode(\"Actor\") returned 0 rels, want 1 — cannot test copy semantics")
	}
	rels[0].RelType = "MUTATED"

	// Verify original is unchanged
	original := model.RelationshipsForNode("Actor")
	if len(original) == 0 {
		t.Fatal("RelationshipsForNode(\"Actor\") returned 0 rels on second call")
	}
	if original[0].RelType != "ACTED_IN" {
		t.Errorf("original RelType = %q after mutation, want %q — copy semantics violated", original[0].RelType, "ACTED_IN")
	}
}

// TestValueTypeSemantics verifies that struct types behave as value types —
// copying a NodeDefinition creates an independent copy.
func TestValueTypeSemantics(t *testing.T) {
	original := NodeDefinition{
		Name:   "Movie",
		Labels: []string{"Movie"},
		Fields: []FieldDefinition{
			{Name: "title", GraphQLType: "String!"},
		},
	}

	// Copy the struct
	copied := original
	copied.Name = "MUTATED"

	if original.Name != "Movie" {
		t.Errorf("original.Name = %q after copy mutation, want %q", original.Name, "Movie")
	}
}
