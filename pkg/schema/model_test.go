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

// TestRelationshipsForNode_PropertiesDeepCopy_FieldsMutation verifies that mutating
// Properties.Fields on a returned RelationshipDefinition does NOT affect the original
// GraphModel. This is the core M1 fix — Properties *PropertiesDefinition must be deep-copied.
// Expected: original model's Properties.Fields remains unchanged after mutation.
func TestRelationshipsForNode_PropertiesDeepCopy_FieldsMutation(t *testing.T) {
	model := GraphModel{
		Relationships: []RelationshipDefinition{
			{
				FieldName: "actedIn",
				RelType:   "ACTED_IN",
				Direction: DirectionOUT,
				FromNode:  "Actor",
				ToNode:    "Movie",
				Properties: &PropertiesDefinition{
					TypeName: "ActedInProperties",
					Fields: []FieldDefinition{
						{Name: "role", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
						{Name: "year", GraphQLType: "Int!", GoType: "int", CypherType: "INTEGER"},
					},
				},
			},
		},
	}

	// Get relationships and mutate the Properties.Fields slice
	rels := model.RelationshipsForNode("Actor")
	if len(rels) != 1 {
		t.Fatalf("RelationshipsForNode(\"Actor\") returned %d rels, want 1", len(rels))
	}
	if rels[0].Properties == nil {
		t.Fatal("returned relationship has nil Properties, want non-nil")
	}

	// Mutate the returned Properties.Fields — should NOT affect the original
	rels[0].Properties.Fields[0].Name = "MUTATED_FIELD"
	rels[0].Properties.Fields = append(rels[0].Properties.Fields, FieldDefinition{Name: "extra"})

	// Verify original is unchanged
	originalRels := model.RelationshipsForNode("Actor")
	if originalRels[0].Properties.Fields[0].Name != "role" {
		t.Errorf("original Properties.Fields[0].Name = %q, want %q — deep copy violated",
			originalRels[0].Properties.Fields[0].Name, "role")
	}
	if len(originalRels[0].Properties.Fields) != 2 {
		t.Errorf("original Properties.Fields has %d entries, want 2 — append leaked through shared slice",
			len(originalRels[0].Properties.Fields))
	}
}

// TestRelationshipsForNode_PropertiesDeepCopy_TypeNameMutation verifies that mutating
// Properties.TypeName on a returned RelationshipDefinition does NOT affect the original model.
// Expected: original model's Properties.TypeName remains unchanged.
func TestRelationshipsForNode_PropertiesDeepCopy_TypeNameMutation(t *testing.T) {
	model := GraphModel{
		Relationships: []RelationshipDefinition{
			{
				FieldName: "actedIn",
				RelType:   "ACTED_IN",
				Direction: DirectionOUT,
				FromNode:  "Actor",
				ToNode:    "Movie",
				Properties: &PropertiesDefinition{
					TypeName: "ActedInProperties",
					Fields:   []FieldDefinition{{Name: "role", GraphQLType: "String!"}},
				},
			},
		},
	}

	rels := model.RelationshipsForNode("Actor")
	rels[0].Properties.TypeName = "MUTATED_TYPE"

	// Verify original is unchanged
	originalRels := model.RelationshipsForNode("Actor")
	if originalRels[0].Properties.TypeName != "ActedInProperties" {
		t.Errorf("original Properties.TypeName = %q, want %q — pointer shared with model",
			originalRels[0].Properties.TypeName, "ActedInProperties")
	}
}

// TestRelationshipsForNode_NilProperties_NoPanic verifies that relationships with
// nil Properties are returned without panic or error. Properties should remain nil.
// Expected: Properties is nil on the returned relationship.
func TestRelationshipsForNode_NilProperties_NoPanic(t *testing.T) {
	model := GraphModel{
		Relationships: []RelationshipDefinition{
			{
				FieldName: "actedIn",
				RelType:   "ACTED_IN",
				Direction: DirectionOUT,
				FromNode:  "Actor",
				ToNode:    "Movie",
				Properties: nil, // no @relationshipProperties
			},
		},
	}

	rels := model.RelationshipsForNode("Actor")
	if len(rels) != 1 {
		t.Fatalf("RelationshipsForNode(\"Actor\") returned %d rels, want 1", len(rels))
	}
	if rels[0].Properties != nil {
		t.Errorf("Properties = %v, want nil for relationship without @relationshipProperties", rels[0].Properties)
	}
}

// TestRelationshipsForNode_MultiplePropertiesIndependent verifies that when multiple
// relationships with Properties are returned, each Properties is an independent copy.
// Expected: mutating one relationship's Properties does not affect another.
func TestRelationshipsForNode_MultiplePropertiesIndependent(t *testing.T) {
	model := GraphModel{
		Relationships: []RelationshipDefinition{
			{
				FieldName: "actedIn",
				RelType:   "ACTED_IN",
				Direction: DirectionOUT,
				FromNode:  "Actor",
				ToNode:    "Movie",
				Properties: &PropertiesDefinition{
					TypeName: "ActedInProperties",
					Fields:   []FieldDefinition{{Name: "role", GraphQLType: "String!"}},
				},
			},
			{
				FieldName: "directed",
				RelType:   "DIRECTED",
				Direction: DirectionOUT,
				FromNode:  "Actor",
				ToNode:    "Movie",
				Properties: &PropertiesDefinition{
					TypeName: "DirectedProperties",
					Fields:   []FieldDefinition{{Name: "year", GraphQLType: "Int!"}},
				},
			},
		},
	}

	rels := model.RelationshipsForNode("Actor")
	if len(rels) != 2 {
		t.Fatalf("RelationshipsForNode(\"Actor\") returned %d rels, want 2", len(rels))
	}

	// Mutate first relationship's Properties
	rels[0].Properties.TypeName = "MUTATED"

	// Verify second relationship's Properties is unaffected
	freshRels := model.RelationshipsForNode("Actor")
	if freshRels[0].Properties.TypeName != "ActedInProperties" {
		t.Errorf("freshRels[0].Properties.TypeName = %q, want %q",
			freshRels[0].Properties.TypeName, "ActedInProperties")
	}
	if freshRels[1].Properties.TypeName != "DirectedProperties" {
		t.Errorf("freshRels[1].Properties.TypeName = %q, want %q",
			freshRels[1].Properties.TypeName, "DirectedProperties")
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
