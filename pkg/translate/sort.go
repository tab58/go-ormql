package translate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// buildOrderBy translates a GraphQL "sort" argument into a Cypher ORDER BY clause string.
// The sortArg is expected to be a list of objects, each with field names mapped to
// SortDirection values ("ASC" or "DESC").
//
// The variable parameter is the Cypher node variable (e.g., "n") used in property access.
// The scope provides access to GraphQL variables for resolving variable references.
//
// Returns the ORDER BY clause string (without the "ORDER BY" keyword) or empty string if no sort.
func (t *Translator) buildOrderBy(sortArg *ast.Value, variable string, scope *paramScope) string {
	if sortArg == nil {
		return ""
	}

	// If the sort arg is a variable, resolve it and work with the Go value
	if sortArg.Kind == ast.Variable && scope != nil {
		resolved := resolveValue(sortArg, scope.variables)
		return buildOrderByFromGo(resolved, variable)
	}

	if len(sortArg.Children) == 0 {
		return ""
	}

	var parts []string
	for _, item := range sortArg.Children {
		obj := item.Value
		if obj == nil || obj.Kind != ast.ObjectValue {
			continue
		}
		for _, field := range obj.Children {
			direction := strings.ToUpper(field.Value.Raw)
			if direction != "ASC" && direction != "DESC" {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s.%s %s", variable, field.Name, direction))
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

// buildOrderByFromGo handles sort data already resolved to Go values.
// Expected shape: []any of map[string]any where keys are field names and values are "ASC"/"DESC".
func buildOrderByFromGo(resolved any, variable string) string {
	list, ok := resolved.([]any)
	if !ok || len(list) == 0 {
		return ""
	}

	var parts []string
	for _, item := range list {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		// Sort map keys for deterministic ORDER BY clause ordering
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, fieldName := range keys {
			dirStr, ok := obj[fieldName].(string)
			if !ok {
				continue
			}
			direction := strings.ToUpper(dirStr)
			if direction != "ASC" && direction != "DESC" {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s.%s %s", variable, fieldName, direction))
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}
