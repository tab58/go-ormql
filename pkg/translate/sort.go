package translate

import (
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// buildOrderBy translates a GraphQL "sort" argument into a Cypher ORDER BY clause string.
// The sortArg is expected to be a list of objects, each with field names mapped to
// SortDirection values ("ASC" or "DESC").
//
// The variable parameter is the Cypher node variable (e.g., "n") used in property access.
//
// Returns the ORDER BY clause string (without the "ORDER BY" keyword) or empty string if no sort.
func (t *Translator) buildOrderBy(sortArg *ast.Value, variable string) string {
	if sortArg == nil || len(sortArg.Children) == 0 {
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
