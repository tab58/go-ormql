package translate

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tab58/go-ormql/pkg/schema"
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

	// Build a lookup for relationship filters from this node
	rels := t.model.RelationshipsForNode(node.Name)

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
			// Check for relationship filter fields before falling through to scalar predicate
			if pred := t.buildRelWherePredicate(name, val, variable, rels, scope); pred != "" {
				predicates = append(predicates, pred)
				continue
			}
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
				resolved := resolveValue(val, scope.variables)
				isNull := false
				switch v := resolved.(type) {
				case bool:
					isNull = v
				case string:
					isNull, _ = strconv.ParseBool(v)
				}
				if isNull {
					return fmt.Sprintf("%s.%s IS NULL", variable, fieldName)
				}
				return fmt.Sprintf("%s.%s IS NOT NULL", variable, fieldName)
			}

			if cypherOp == "NOT_IN" {
				param := scope.add(resolveValue(val, scope.variables))
				return fmt.Sprintf("NOT %s.%s IN %s", variable, fieldName, param)
			}

			param := scope.add(resolveValue(val, scope.variables))
			return fmt.Sprintf("%s.%s %s %s", variable, fieldName, cypherOp, param)
		}
	}

	// No suffix — equality: field = $param
	param := scope.add(resolveValue(val, scope.variables))
	return fmt.Sprintf("%s.%s = %s", variable, name, param)
}

// buildRelWherePredicate checks if a field name matches a relationship filter
// and builds the appropriate Cypher predicate.
//
// To-many (IsList=true): field name has "_some" suffix → EXISTS { MATCH pattern WHERE ... }
// To-one (IsList=false): field name matches directly → EXISTS { MATCH pattern WHERE ... }
//
// Returns empty string if the field name does not match any relationship filter.
func (t *Translator) buildRelWherePredicate(name string, val *ast.Value, variable string, rels []schema.RelationshipDefinition, scope *paramScope) string {
	for _, rel := range rels {
		if rel.FromNode == "" {
			continue
		}

		var matched bool
		if rel.IsList && name == rel.FieldName+"_some" {
			matched = true
		} else if !rel.IsList && name == rel.FieldName {
			matched = true
		}

		if !matched {
			continue
		}

		// Look up the target node for recursive WHERE
		targetNode, ok := t.model.NodeByName(rel.ToNode)
		if !ok {
			continue
		}

		// Build the relationship pattern with a unique child variable
		childVar := fmt.Sprintf("rel%d", scope.next)
		scope.next++

		pattern := buildRelPattern(variable, "", rel.RelType, childVar+":"+targetNode.Labels[0], rel.Direction)

		// Recursively build WHERE for the target node
		innerWhere := t.buildWhereClause(val, childVar, targetNode, scope)

		if innerWhere != "" {
			return fmt.Sprintf("EXISTS { MATCH %s WHERE %s }", pattern, innerWhere)
		}
		return fmt.Sprintf("EXISTS { MATCH %s }", pattern)
	}
	return ""
}

