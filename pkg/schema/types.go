package schema

import "strings"

// ScalarGoType holds the Go type and import path for a known custom scalar.
type ScalarGoType struct {
	GoType     string
	ImportPath string // empty string means no import needed
}

// knownScalarGoTypes maps well-known custom GraphQL scalars to Go types.
var knownScalarGoTypes = map[string]ScalarGoType{
	"DateTime": {GoType: "time.Time", ImportPath: "time"},
	"Date":     {GoType: "time.Time", ImportPath: "time"},
	"Time":     {GoType: "time.Time", ImportPath: "time"},
	"JSON":     {GoType: "map[string]any", ImportPath: ""},
	"BigInt":   {GoType: "int64", ImportPath: ""},
}

// knownScalarCypherTypes maps well-known custom GraphQL scalars to Cypher types.
var knownScalarCypherTypes = map[string]string{
	"DateTime": "LOCAL DATETIME",
	"Date":     "DATE",
	"Time":     "LOCAL TIME",
	"JSON":     "STRING",
	"BigInt":   "INTEGER",
}

// LookupScalarGoType returns the Go type mapping for a known custom scalar.
// Returns the mapping and true if found, zero value and false otherwise.
func LookupScalarGoType(name string) (ScalarGoType, bool) {
	v, ok := knownScalarGoTypes[name]
	return v, ok
}

// LookupScalarCypherType returns the Cypher type mapping for a known custom scalar.
// Returns the mapping and true if found, empty string and false otherwise.
func LookupScalarCypherType(name string) (string, bool) {
	v, ok := knownScalarCypherTypes[name]
	return v, ok
}

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
// Known custom scalars (DateTime, Date, etc.) map to their Go equivalents.
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

	// Check known custom scalars before falling through to object inference.
	baseName := stripModifiers(graphqlType)
	if scalar, ok := knownScalarGoTypes[baseName]; ok {
		return applyGoNullability(scalar.GoType, graphqlType)
	}

	return inferObjectGoType(graphqlType)
}

// inferObjectGoType derives Go type for non-scalar GraphQL types.
// "[Movie!]!" → "[]*Movie", "[Movie!]" → "[]*Movie", "Movie!" → "*Movie", "Movie" → "*Movie".
func inferObjectGoType(graphqlType string) string {
	base := stripModifiers(graphqlType)
	if isListType(graphqlType) {
		return "[]*" + base
	}
	return "*" + base
}

// GraphQLToCypherType converts a GraphQL type string to its Cypher type equivalent.
// Enum types map to STRING. Known custom scalars map to their Cypher equivalents.
// Returns empty string if the type is unknown.
func GraphQLToCypherType(graphqlType string, isEnum bool) string {
	if isEnum {
		return "STRING"
	}
	if cypherType, ok := cypherTypeMap[graphqlType]; ok {
		return cypherType
	}

	baseName := stripModifiers(graphqlType)
	if cypherType, ok := knownScalarCypherTypes[baseName]; ok {
		if isListType(graphqlType) {
			return "LIST<" + cypherType + ">"
		}
		return cypherType
	}

	return ""
}

// stripModifiers extracts the base type name from a GraphQL type string.
// "DateTime!" → "DateTime", "[DateTime!]!" → "DateTime", "[DateTime!]" → "DateTime".
func stripModifiers(graphqlType string) string {
	s := strings.TrimSuffix(graphqlType, "!")
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		s = s[1 : len(s)-1]
		s = strings.TrimSuffix(s, "!")
	}
	return s
}

// isListType checks if a GraphQL type string is a list type.
func isListType(graphqlType string) bool {
	s := strings.TrimSuffix(graphqlType, "!")
	return strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")
}

// applyGoNullability wraps a Go type based on GraphQL nullability and list modifiers.
// List types → "[]GoType", nullable non-list → "*GoType", non-nullable non-list → "GoType".
func applyGoNullability(goType string, graphqlType string) string {
	if isListType(graphqlType) {
		return "[]" + goType
	}
	if !strings.HasSuffix(graphqlType, "!") {
		return "*" + goType
	}
	return goType
}
