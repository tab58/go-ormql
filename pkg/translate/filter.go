package translate

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tab58/gql-orm/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// operatorSuffixes maps GraphQL filter suffixes to Cypher operators.
var operatorSuffixes = map[string]string{
	"_gt":         ">",
	"_gte":        ">=",
	"_lt":         "<",
	"_lte":        "<=",
	"_not":        "<>",
	"_contains":   "CONTAINS",
	"_startsWith": "STARTS WITH",
	"_endsWith":   "ENDS WITH",
	"_regex":      "=~",
	"_in":         "IN",
	"_nin":        "NOT_IN",
	"_isNull":     "IS_NULL",
}

// buildWhereClause translates a GraphQL "where" argument into a Cypher WHERE clause string.
func (t *Translator) buildWhereClause(whereArg *ast.Value, variable string, node schema.NodeDefinition, scope *paramScope) string {
	if whereArg == nil || len(whereArg.Children) == 0 {
		return ""
	}

	var predicates []string

	for _, child := range whereArg.Children {
		name := child.Name
		val := child.Value

		switch name {
		case "AND":
			predicates = append(predicates, t.buildBooleanComposition("AND", val, variable, node, scope))
		case "OR":
			predicates = append(predicates, t.buildBooleanComposition("OR", val, variable, node, scope))
		case "NOT":
			inner := t.buildWhereClause(val, variable, node, scope)
			if inner != "" {
				predicates = append(predicates, fmt.Sprintf("NOT (%s)", inner))
			}
		default:
			pred := t.buildPredicate(name, val, variable, scope)
			if pred != "" {
				predicates = append(predicates, pred)
			}
		}
	}

	if len(predicates) == 0 {
		return ""
	}
	return strings.Join(predicates, " AND ")
}

// buildBooleanComposition builds an AND or OR composed clause.
func (t *Translator) buildBooleanComposition(op string, listVal *ast.Value, variable string, node schema.NodeDefinition, scope *paramScope) string {
	var clauses []string
	for _, child := range listVal.Children {
		clause := t.buildWhereClause(child.Value, variable, node, scope)
		if clause != "" {
			clauses = append(clauses, clause)
		}
	}
	if len(clauses) == 0 {
		return ""
	}
	return "(" + strings.Join(clauses, " "+op+" ") + ")"
}

// buildPredicate builds a single predicate from a field name (possibly with operator suffix) and value.
func (t *Translator) buildPredicate(name string, val *ast.Value, variable string, scope *paramScope) string {
	// Check for operator suffixes
	for suffix, cypherOp := range operatorSuffixes {
		if strings.HasSuffix(name, suffix) {
			fieldName := strings.TrimSuffix(name, suffix)

			if cypherOp == "IS_NULL" {
				boolStr := val.Raw
				isNull, _ := strconv.ParseBool(boolStr)
				if isNull {
					return fmt.Sprintf("%s.%s IS NULL", variable, fieldName)
				}
				return fmt.Sprintf("%s.%s IS NOT NULL", variable, fieldName)
			}

			if cypherOp == "NOT_IN" {
				param := scope.add(astValueToGo(val))
				return fmt.Sprintf("NOT %s.%s IN %s", variable, fieldName, param)
			}

			param := scope.add(astValueToGo(val))
			return fmt.Sprintf("%s.%s %s %s", variable, fieldName, cypherOp, param)
		}
	}

	// No suffix — equality: field = $param
	param := scope.add(astValueToGo(val))
	return fmt.Sprintf("%s.%s = %s", variable, name, param)
}

// astValueToGo converts an ast.Value to a Go native type for parameterization.
func astValueToGo(val *ast.Value) any {
	switch val.Kind {
	case ast.IntValue:
		n, _ := strconv.ParseInt(val.Raw, 10, 64)
		return n
	case ast.FloatValue:
		f, _ := strconv.ParseFloat(val.Raw, 64)
		return f
	case ast.BooleanValue:
		b, _ := strconv.ParseBool(val.Raw)
		return b
	case ast.StringValue, ast.EnumValue:
		return val.Raw
	case ast.ListValue:
		items := make([]any, 0, len(val.Children))
		for _, child := range val.Children {
			items = append(items, astValueToGo(child.Value))
		}
		return items
	case ast.ObjectValue:
		m := make(map[string]any, len(val.Children))
		for _, child := range val.Children {
			m[child.Name] = astValueToGo(child.Value)
		}
		return m
	default:
		return val.Raw
	}
}
