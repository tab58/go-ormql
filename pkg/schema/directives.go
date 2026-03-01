package schema

import (
	"fmt"

	"github.com/vektah/gqlparser/v2/ast"
)

// Directive name constants for @node, @relationship, @relationshipProperties, and @cypher.
const (
	directiveNode                   = "node"
	directiveRelationship           = "relationship"
	directiveRelationshipProperties = "relationshipProperties"
	directiveCypher                 = "cypher"
)

// Directive argument name constants for @relationship and @cypher.
const (
	argType       = "type"
	argDirection  = "direction"
	argProperties = "properties"
	argStatement  = "statement"
)

// BuiltinDirectiveDefs returns the GraphQL SDL string for the built-in
// gql-orm directives (@node, @relationship, @relationshipProperties).
// This should be prepended to user schemas before parsing.
func BuiltinDirectiveDefs() string {
	return `
enum RelationshipDirection {
  IN
  OUT
}

directive @node on OBJECT
directive @relationship(type: String!, direction: RelationshipDirection!, properties: String) on FIELD_DEFINITION
directive @relationshipProperties on OBJECT
directive @cypher(statement: String!) on FIELD_DEFINITION
`
}

// NodeDirectiveInfo holds extraction results for @node on a type definition.
type NodeDirectiveInfo struct {
	HasDirective bool
}

// RelationshipDirectiveInfo holds extraction results for @relationship on a field definition.
type RelationshipDirectiveInfo struct {
	HasDirective bool
	RelType      string
	Direction    Direction
	Properties   string // name of the @relationshipProperties type, or empty
}

// CypherDirectiveInfo holds extraction results for @cypher on a field definition.
type CypherDirectiveInfo struct {
	HasDirective bool
	Statement    string
}

// ExtractCypherDirective extracts @cypher directive arguments from a field definition.
// Returns HasDirective=false if the field does not have the directive.
func ExtractCypherDirective(field *ast.FieldDefinition) CypherDirectiveInfo {
	if field == nil {
		return CypherDirectiveInfo{}
	}
	for _, d := range field.Directives {
		if d.Name == directiveCypher {
			info := CypherDirectiveInfo{HasDirective: true}
			stmtArg := d.Arguments.ForName(argStatement)
			if stmtArg != nil {
				info.Statement = stmtArg.Value.Raw
			}
			return info
		}
	}
	return CypherDirectiveInfo{}
}

// ExtractNodeDirective checks whether the given type definition has the @node directive.
func ExtractNodeDirective(def *ast.Definition) NodeDirectiveInfo {
	if def == nil {
		return NodeDirectiveInfo{}
	}
	for _, d := range def.Directives {
		if d.Name == directiveNode {
			return NodeDirectiveInfo{HasDirective: true}
		}
	}
	return NodeDirectiveInfo{}
}

// ExtractRelationshipDirective extracts @relationship directive arguments from a field definition.
// Returns HasDirective=false if the field does not have the directive.
func ExtractRelationshipDirective(field *ast.FieldDefinition) RelationshipDirectiveInfo {
	if field == nil {
		return RelationshipDirectiveInfo{}
	}
	for _, d := range field.Directives {
		if d.Name == directiveRelationship {
			info := RelationshipDirectiveInfo{HasDirective: true}
			for _, arg := range d.Arguments {
				switch arg.Name {
				case argType:
					info.RelType = arg.Value.Raw
				case argDirection:
					info.Direction = Direction(arg.Value.Raw)
				case argProperties:
					info.Properties = arg.Value.Raw
				}
			}
			return info
		}
	}
	return RelationshipDirectiveInfo{}
}

// HasRelationshipPropertiesDirective checks whether the given type definition
// has the @relationshipProperties directive.
func HasRelationshipPropertiesDirective(def *ast.Definition) bool {
	if def == nil {
		return false
	}
	for _, d := range def.Directives {
		if d.Name == directiveRelationshipProperties {
			return true
		}
	}
	return false
}

// ValidateDirectives validates all directive usage in the schema document.
// Checks: missing required args on @relationship, unknown direction values,
// @relationship properties referencing nonexistent types.
// Returns a slice of errors with position information (file/line/column).
func ValidateDirectives(doc *ast.SchemaDocument) []error {
	if doc == nil {
		return nil
	}

	// Collect all @relationshipProperties type names for reference checking.
	propsTypes := map[string]bool{}
	for _, def := range doc.Definitions {
		if HasRelationshipPropertiesDirective(def) {
			propsTypes[def.Name] = true
		}
	}

	var errs []error
	for _, def := range doc.Definitions {
		for _, field := range def.Fields {
			errs = append(errs, validateRelationshipField(field, propsTypes)...)
			errs = append(errs, validateCypherField(field)...)
		}
	}

	return errs
}

// validateRelationshipField validates @relationship directives on a single field.
// Returns errors for missing required args, unknown direction values,
// and properties referencing nonexistent types.
func validateRelationshipField(field *ast.FieldDefinition, propsTypes map[string]bool) []error {
	var errs []error
	for _, d := range field.Directives {
		if d.Name != directiveRelationship {
			continue
		}

		loc := formatDirectiveLocation(d.Position)

		// Check required "type" arg
		typeArg := d.Arguments.ForName(argType)
		if typeArg == nil || typeArg.Value.Raw == "" {
			errs = append(errs, fmt.Errorf("%s@relationship on field %q is missing required argument %q", loc, field.Name, argType))
		}

		// Check required "direction" arg
		dirArg := d.Arguments.ForName(argDirection)
		if dirArg == nil || dirArg.Value.Raw == "" {
			errs = append(errs, fmt.Errorf("%s@relationship on field %q is missing required argument %q", loc, field.Name, argDirection))
		} else {
			dir := dirArg.Value.Raw
			if dir != string(DirectionIN) && dir != string(DirectionOUT) {
				errs = append(errs, fmt.Errorf("%s@relationship on field %q has unknown direction %q (must be IN or OUT)", loc, field.Name, dir))
			}
		}

		// Check optional "properties" arg references an existing type
		propsArg := d.Arguments.ForName(argProperties)
		if propsArg != nil && propsArg.Value.Raw != "" {
			propsName := propsArg.Value.Raw
			if !propsTypes[propsName] {
				errs = append(errs, fmt.Errorf("%s@relationship on field %q references properties type %q which does not exist or is not annotated with @relationshipProperties", loc, field.Name, propsName))
			}
		}
	}
	return errs
}

// validateCypherField validates @cypher directives on a single field.
// Returns errors for mutual exclusivity with @relationship and empty statement.
func validateCypherField(field *ast.FieldDefinition) []error {
	var errs []error
	hasCypher := false
	hasRelationship := false
	for _, d := range field.Directives {
		if d.Name == directiveCypher {
			hasCypher = true
		}
		if d.Name == directiveRelationship {
			hasRelationship = true
		}
	}
	if hasCypher && hasRelationship {
		errs = append(errs, fmt.Errorf("field %q has both @cypher and @relationship directives (mutually exclusive)", field.Name))
	}
	if hasCypher {
		info := ExtractCypherDirective(field)
		if info.Statement == "" {
			errs = append(errs, fmt.Errorf("@cypher on field %q has empty statement", field.Name))
		}
	}
	return errs
}

// formatDirectiveLocation formats a position as "file:line:col: " for error messages.
// Returns empty string if position is nil.
func formatDirectiveLocation(pos *ast.Position) string {
	if pos == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d:%d: ", pos.Src.Name, pos.Line, pos.Column)
}
