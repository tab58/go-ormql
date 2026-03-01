package codegen

import (
	"strings"
	"text/template"

	"github.com/tab58/gql-orm/pkg/schema"
)

// GenerateResolvers produces Go source code for resolver implementations.
// Generates typed query and mutation resolvers that implement the gqlgen
// ResolverRoot interface with proper queryResolver and mutationResolver
// dispatch types.
func GenerateResolvers(model schema.GraphModel, packageName string) ([]byte, error) {
	rels := model.Relationships
	nodes := model.Nodes
	funcMap := template.FuncMap{
		"plural":     plural,
		"pluralCap":  pluralCapitalized,
		"lowerFirst": lowerFirst,
		"goField":    goFieldName,
		"baseType":   baseTypeName,
		"zeroValue":  goZeroValue,
		"hasCypherFields": func(nodeName string) bool {
			for _, n := range nodes {
				if n.Name == nodeName {
					return len(n.CypherFields) > 0
				}
			}
			return false
		},
		"hasObjectResolver": func(nodeName string) bool {
			for _, n := range nodes {
				if n.Name == nodeName && len(n.CypherFields) > 0 {
					return true
				}
			}
			for _, r := range rels {
				if r.FromNode == nodeName {
					return true
				}
			}
			return false
		},
		"hasRels": func(nodeName string) bool {
			for _, r := range rels {
				if r.FromNode == nodeName {
					return true
				}
			}
			return false
		},
		"relsForNode": func(nodeName string) []schema.RelationshipDefinition {
			var result []schema.RelationshipDefinition
			for _, r := range rels {
				if r.FromNode == nodeName {
					result = append(result, r)
				}
			}
			return result
		},
		"cypherHelper":  cypherHelperFuncName,
		"listElemType":  listElementType,
		"cypherHelpers": collectCypherHelpers,
	}
	return executeTemplate("resolvers", resolverTemplate, funcMap, model, packageName)
}

// lowerFirst returns the string with its first character lowered.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// goFieldName converts a GraphQL field name to its Go struct field name.
// Handles Go initialisms (e.g., "id" → "ID").
func goFieldName(name string) string {
	if strings.EqualFold(name, "id") {
		return "ID"
	}
	if strings.EqualFold(name, "url") {
		return "URL"
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// goZeroValue returns the Go zero value expression for a type.
func goZeroValue(goType string) string {
	if strings.HasPrefix(goType, "*") || strings.HasPrefix(goType, "[]") {
		return "nil"
	}
	switch goType {
	case "string":
		return `""`
	case "int", "float64":
		return "0"
	case "bool":
		return "false"
	default:
		return "nil"
	}
}

// fromAnyFuncName returns the conversion function name for a non-nullable Go type.
func fromAnyFuncName(goType string) string {
	switch goType {
	case "string":
		return "stringFromAny"
	case "int":
		return "intFromAny"
	case "float64":
		return "float64FromAny"
	case "bool":
		return "boolFromAny"
	default:
		return "stringFromAny"
	}
}

// ptrFromAnyFuncName returns the conversion function name for a nullable Go type.
func ptrFromAnyFuncName(goType string) string {
	switch goType {
	case "*string":
		return "stringPtrFromAny"
	case "*int":
		return "intPtrFromAny"
	case "*float64":
		return "float64PtrFromAny"
	case "*bool":
		return "boolPtrFromAny"
	default:
		return "stringPtrFromAny"
	}
}

// cypherHelperInfo describes a typed helper function for converting @cypher results.
type cypherHelperInfo struct {
	FuncName string
	GoType   string
	BaseType string
	IsPtr    bool
}

// cypherHelperFuncName returns the helper function name for converting a @cypher
// result value to the given Go type. e.g. "*float64" → "cypherResultToFloat64Ptr".
func cypherHelperFuncName(goType string) string {
	if strings.HasPrefix(goType, "*") {
		base := strings.TrimPrefix(goType, "*")
		return "cypherResultTo" + goFieldName(base) + "Ptr"
	}
	return "cypherResultTo" + goFieldName(goType)
}

// listElementType extracts the element type from a Go slice type.
// e.g. "[]*Movie" → "Movie", "[]string" → "string".
func listElementType(goType string) string {
	t := strings.TrimPrefix(goType, "[]")
	t = strings.TrimPrefix(t, "*")
	return t
}

// collectCypherHelpers gathers unique scalar @cypher result types across all
// nodes and returns helper info for each. List types are excluded (they use
// existing recordsTo mappers).
func collectCypherHelpers(nodes []schema.NodeDefinition) []cypherHelperInfo {
	seen := map[string]bool{}
	var helpers []cypherHelperInfo
	for _, n := range nodes {
		for _, cf := range n.CypherFields {
			if cf.IsList {
				continue
			}
			name := cypherHelperFuncName(cf.GoType)
			if seen[name] {
				continue
			}
			seen[name] = true
			isPtr := strings.HasPrefix(cf.GoType, "*")
			baseType := strings.TrimPrefix(cf.GoType, "*")
			helpers = append(helpers, cypherHelperInfo{
				FuncName: name,
				GoType:   cf.GoType,
				BaseType: baseType,
				IsPtr:    isPtr,
			})
		}
	}
	return helpers
}
