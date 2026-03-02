package translate

import (
	"fmt"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

// buildProjection builds a map projection from a GraphQL selection set.
// Produces: n { .title, .released, actors: __sub0, averageRating: __cypher0 }
// Only includes fields present in the selection set (no over-fetching).
//
// Returns the projection string and a list of subquery CALL blocks that must be
// placed before this projection in the Cypher query.
func (t *Translator) buildProjection(selSet ast.SelectionSet, fc fieldContext, scope *paramScope) (string, []string, error) {
	if len(selSet) == 0 {
		return fmt.Sprintf("%s {}", fc.variable), nil, nil
	}

	// Classify fields: scalar fields get .prop access, relationship/cypher fields get subquery aliases
	var projParts []string
	var subqueries []string

	// Build a set of scalar field names from the node definition
	scalarFields := make(map[string]bool, len(fc.node.Fields))
	for _, f := range fc.node.Fields {
		scalarFields[f.Name] = true
	}

	// Build a set of cypher field names
	cypherFields := make(map[string]bool, len(fc.node.CypherFields))
	for _, cf := range fc.node.CypherFields {
		cypherFields[cf.Name] = true
	}

	// Build a map of relationship field names
	rels := t.model.RelationshipsForNode(fc.node.Name)
	relFields := make(map[string]bool, len(rels))
	for _, rel := range rels {
		relFields[rel.FieldName] = true
	}

	for _, sel := range selSet {
		field, ok := sel.(*ast.Field)
		if !ok {
			continue
		}
		name := field.Name

		if scalarFields[name] {
			projParts = append(projParts, "."+name)
		} else if cypherFields[name] {
			// @cypher field — will be resolved as a subquery alias
			alias := fmt.Sprintf("__cypher%d", fc.depth)
			fc.depth++

			// Find the cypher field definition
			for _, cf := range fc.node.CypherFields {
				if cf.Name == name {
					sq, sqAlias, err := t.buildCypherSubquery(field, cf, fc, scope)
					if err != nil {
						return "", nil, err
					}
					alias = sqAlias
					subqueries = append(subqueries, sq)
					break
				}
			}
			projParts = append(projParts, fmt.Sprintf("%s: %s", name, alias))
		} else if relFields[name] {
			// Regular relationship subquery (relFields contains base names like "actors",
			// not connection suffixed names like "actorsConnection")
			for _, rel := range rels {
				if rel.FieldName == name {
					sq, sqAlias, err := t.buildSubquery(field, rel, fc, scope)
					if err != nil {
						return "", nil, err
					}
					subqueries = append(subqueries, sq)
					projParts = append(projParts, fmt.Sprintf("%s: %s", name, sqAlias))
					fc.depth++
					break
				}
			}
		} else {
			// Check if it's a connection field name (e.g., actorsConnection)
			for _, rel := range rels {
				connFieldName := rel.FieldName + "Connection"
				if connFieldName == name {
					sq, sqAlias, err := t.buildConnectionSubquery(field, rel, fc, scope)
					if err != nil {
						return "", nil, err
					}
					subqueries = append(subqueries, sq)
					projParts = append(projParts, fmt.Sprintf("%s: %s", name, sqAlias))
					fc.depth++
					break
				}
			}
		}
	}

	proj := fmt.Sprintf("%s { %s }", fc.variable, strings.Join(projParts, ", "))
	return proj, subqueries, nil
}
