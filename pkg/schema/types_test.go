package schema

import "testing"

// TestGraphQLToGoType verifies that every GraphQL scalar type maps to the correct Go type.
// Covers nullable, non-nullable, list, ID, and enum variants per the spec type mapping table.
func TestGraphQLToGoType(t *testing.T) {
	tests := []struct {
		name        string
		graphqlType string
		isEnum      bool
		expected    string
	}{
		// Nullable scalars → pointer types
		{"nullable String", "String", false, "*string"},
		{"nullable Int", "Int", false, "*int"},
		{"nullable Float", "Float", false, "*float64"},
		{"nullable Boolean", "Boolean", false, "*bool"},
		{"nullable ID", "ID", false, "*string"},

		// Non-nullable scalars → value types
		{"non-nullable String", "String!", false, "string"},
		{"non-nullable Int", "Int!", false, "int"},
		{"non-nullable Float", "Float!", false, "float64"},
		{"non-nullable Boolean", "Boolean!", false, "bool"},
		{"non-nullable ID", "ID!", false, "string"},

		// List types
		{"list of String", "[String!]", false, "[]string"},

		// Enum types → string
		{"nullable enum", "Status", true, "*string"},
		{"non-nullable enum", "Status!", true, "string"},

		// Object types → pointer types
		{"object type", "Unknown", false, "*Unknown"},
		{"non-nullable object", "Movie!", false, "*Movie"},
		{"list of objects", "[Movie!]!", false, "[]*Movie"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GraphQLToGoType(tt.graphqlType, tt.isEnum)
			if got != tt.expected {
				t.Errorf("GraphQLToGoType(%q, %v) = %q, want %q", tt.graphqlType, tt.isEnum, got, tt.expected)
			}
		})
	}
}

// TestGraphQLToCypherType verifies that every GraphQL scalar type maps to the correct Cypher type.
// Covers nullable, non-nullable, list, ID, and enum variants per the spec type mapping table.
func TestGraphQLToCypherType(t *testing.T) {
	tests := []struct {
		name        string
		graphqlType string
		isEnum      bool
		expected    string
	}{
		// Nullable scalars
		{"nullable String", "String", false, "STRING"},
		{"nullable Int", "Int", false, "INTEGER"},
		{"nullable Float", "Float", false, "FLOAT"},
		{"nullable Boolean", "Boolean", false, "BOOLEAN"},
		{"nullable ID", "ID", false, "STRING"},

		// Non-nullable scalars (same Cypher type as nullable)
		{"non-nullable String", "String!", false, "STRING"},
		{"non-nullable Int", "Int!", false, "INTEGER"},
		{"non-nullable Float", "Float!", false, "FLOAT"},
		{"non-nullable Boolean", "Boolean!", false, "BOOLEAN"},
		{"non-nullable ID", "ID!", false, "STRING"},

		// List types
		{"list of String", "[String!]", false, "LIST<STRING>"},

		// Enum types → STRING
		{"nullable enum", "Status", true, "STRING"},
		{"non-nullable enum", "Status!", true, "STRING"},

		// Unknown type → empty string
		{"unknown type", "Unknown", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GraphQLToCypherType(tt.graphqlType, tt.isEnum)
			if got != tt.expected {
				t.Errorf("GraphQLToCypherType(%q, %v) = %q, want %q", tt.graphqlType, tt.isEnum, got, tt.expected)
			}
		})
	}
}
