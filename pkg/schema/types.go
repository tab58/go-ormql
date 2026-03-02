package schema

import "strings"

// goTypeMap maps GraphQL scalar types to Go types.
// Nullable scalars → pointer types, non-nullable → value types.
var goTypeMap = map[string]string{
	// Nullable scalars
	"String":  "*string",
	"Int":     "*int",
	"Float":   "*float64",
	"Boolean": "*bool",
	"ID":      "*string",
	// Non-nullable scalars
	"String!":  "string",
	"Int!":     "int",
	"Float!":   "float64",
	"Boolean!": "bool",
	"ID!":      "string",
	// List types (non-nullable and nullable outer)
	"[String!]":  "[]string",
	"[String!]!": "[]string",
	"[Int!]":     "[]int",
	"[Int!]!":    "[]int",
	"[Float!]":   "[]float64",
	"[Float!]!":  "[]float64",
	"[Boolean!]": "[]bool",
	"[Boolean!]!": "[]bool",
	"[ID!]":      "[]string",
	"[ID!]!":     "[]string",
}

// cypherTypeMap maps GraphQL scalar types to Cypher types.
// Nullability doesn't affect Cypher type (STRING is STRING either way).
var cypherTypeMap = map[string]string{
	"String":    "STRING",
	"String!":   "STRING",
	"Int":       "INTEGER",
	"Int!":      "INTEGER",
	"Float":     "FLOAT",
	"Float!":    "FLOAT",
	"Boolean":   "BOOLEAN",
	"Boolean!":  "BOOLEAN",
	"ID":        "STRING",
	"ID!":       "STRING",
	"[String!]":  "LIST<STRING>",
	"[String!]!": "LIST<STRING>",
	"[Int!]":     "LIST<INTEGER>",
	"[Int!]!":    "LIST<INTEGER>",
	"[Float!]":   "LIST<FLOAT>",
	"[Float!]!":  "LIST<FLOAT>",
	"[Boolean!]": "LIST<BOOLEAN>",
	"[Boolean!]!": "LIST<BOOLEAN>",
	"[ID!]":      "LIST<STRING>",
	"[ID!]!":     "LIST<STRING>",
}

// GraphQLToGoType converts a GraphQL type string to its Go type equivalent.
// Handles nullable (pointer), non-nullable, and list variants.
// Enum types map to *string (nullable) or string (non-nullable).
// Object types are inferred by name: "Movie!" → "*Movie", "[Movie!]!" → "[]*Movie".
func GraphQLToGoType(graphqlType string, isEnum bool) string {
	if isEnum {
		if strings.HasSuffix(graphqlType, "!") {
			return "string"
		}
		return "*string"
	}
	if goType, ok := goTypeMap[graphqlType]; ok {
		return goType
	}
	return inferObjectGoType(graphqlType)
}

// inferObjectGoType derives Go type for non-scalar GraphQL types.
// "[Movie!]!" → "[]*Movie", "[Movie!]" → "[]*Movie", "Movie!" → "*Movie", "Movie" → "*Movie".
func inferObjectGoType(graphqlType string) string {
	s := strings.TrimSuffix(graphqlType, "!")
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		inner := s[1 : len(s)-1]
		inner = strings.TrimSuffix(inner, "!")
		return "[]*" + inner
	}
	return "*" + s
}

// GraphQLToCypherType converts a GraphQL type string to its Cypher type equivalent.
// Enum types map to STRING.
// Returns empty string if the type is unknown.
func GraphQLToCypherType(graphqlType string, isEnum bool) string {
	if isEnum {
		return "STRING"
	}
	if cypherType, ok := cypherTypeMap[graphqlType]; ok {
		return cypherType
	}
	return ""
}
