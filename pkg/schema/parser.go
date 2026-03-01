package schema

import (
	"fmt"
	"os"
	"strings"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

// ParseSchema reads one or more .graphql files, parses them using gqlparser
// (with built-in directive definitions prepended), walks the AST, and returns
// an immutable GraphModel. Returns an error with position info for invalid schemas.
func ParseSchema(paths []string) (GraphModel, error) {
	var sources []*ast.Source

	// Add built-in directive definitions as the first source.
	sources = append(sources, &ast.Source{
		Name:  "builtins.graphql",
		Input: BuiltinDirectiveDefs(),
	})

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return GraphModel{}, fmt.Errorf("failed to read schema file %s: %w", path, err)
		}
		sources = append(sources, &ast.Source{
			Name:  path,
			Input: string(data),
		})
	}

	return parseFromSources(sources)
}

// ParseSchemaString parses a GraphQL SDL string into a GraphModel.
// Convenience function for testing — equivalent to ParseSchema with inline content.
func ParseSchemaString(sdl string) (GraphModel, error) {
	sources := []*ast.Source{
		{Name: "builtins.graphql", Input: BuiltinDirectiveDefs()},
		{Name: "schema.graphql", Input: sdl},
	}
	return parseFromSources(sources)
}

// parseFromSources parses ast.Source slices into a GraphModel.
func parseFromSources(sources []*ast.Source) (GraphModel, error) {
	schema, err := gqlparser.LoadSchema(sources...)
	if err != nil {
		return GraphModel{}, fmt.Errorf("schema parse error: %w", err)
	}

	// Validate directives on the underlying document.
	// gqlparser merges sources into schema.Types, but we need the raw doc
	// for directive validation. Build a SchemaDocument from schema.Types.
	doc := buildSchemaDocument(schema)
	if validationErrs := ValidateDirectives(doc); len(validationErrs) > 0 {
		msgs := make([]string, len(validationErrs))
		for i, e := range validationErrs {
			msgs[i] = e.Error()
		}
		return GraphModel{}, fmt.Errorf("directive validation failed: %s", strings.Join(msgs, "; "))
	}

	// Collect enum names for type mapping.
	enumNames := map[string]bool{}
	var enums []EnumDefinition
	for _, t := range schema.Types {
		if t.Kind == ast.Enum && !isBuiltinEnum(t.Name) {
			vals := make([]string, len(t.EnumValues))
			for i, v := range t.EnumValues {
				vals[i] = v.Name
			}
			enums = append(enums, EnumDefinition{Name: t.Name, Values: vals})
			enumNames[t.Name] = true
		}
	}

	// Collect @relationshipProperties types.
	propsMap := map[string]*PropertiesDefinition{}
	for _, t := range schema.Types {
		if t.Kind != ast.Object {
			continue
		}
		if !HasRelationshipPropertiesDirective(t) {
			continue
		}
		fields := extractScalarFields(t, enumNames)
		propsMap[t.Name] = &PropertiesDefinition{
			TypeName: t.Name,
			Fields:   fields,
		}
	}

	// Build nodes and relationships.
	var nodes []NodeDefinition
	var relationships []RelationshipDefinition

	for _, t := range schema.Types {
		if t.Kind != ast.Object {
			continue
		}
		nodeInfo := ExtractNodeDirective(t)
		if !nodeInfo.HasDirective {
			continue
		}

		var scalarFields []FieldDefinition
		var cypherFields []CypherFieldDefinition
		for _, f := range t.Fields {
			relInfo := ExtractRelationshipDirective(f)
			if relInfo.HasDirective {
				// Build RelationshipDefinition
				toNode := f.Type.Name()
				rel := RelationshipDefinition{
					FieldName: f.Name,
					RelType:   relInfo.RelType,
					Direction: relInfo.Direction,
					FromNode:  t.Name,
					ToNode:    toNode,
				}
				if relInfo.Properties != "" {
					if props, ok := propsMap[relInfo.Properties]; ok {
						rel.Properties = props
					}
				}
				relationships = append(relationships, rel)
				continue
			}

			cypherInfo := ExtractCypherDirective(f)
			if cypherInfo.HasDirective {
				cypherFields = append(cypherFields, buildCypherFieldDefinition(f, cypherInfo, enumNames))
				continue
			}

			// Scalar field
			fd := buildFieldDefinition(f, enumNames)
			scalarFields = append(scalarFields, fd)
		}

		nodes = append(nodes, NodeDefinition{
			Name:         t.Name,
			Labels:       []string{t.Name},
			Fields:       scalarFields,
			CypherFields: cypherFields,
		})
	}

	return GraphModel{
		Nodes:         nodes,
		Relationships: relationships,
		Enums:         enums,
	}, nil
}

// buildSchemaDocument reconstructs an ast.SchemaDocument from a parsed schema
// for directive validation.
func buildSchemaDocument(schema *ast.Schema) *ast.SchemaDocument {
	var defs []*ast.Definition
	for _, t := range schema.Types {
		if t.BuiltIn {
			continue
		}
		defs = append(defs, t)
	}
	return &ast.SchemaDocument{Definitions: defs}
}

// extractScalarFields extracts scalar FieldDefinitions from a type (no @relationship fields).
func extractScalarFields(t *ast.Definition, enumNames map[string]bool) []FieldDefinition {
	var fields []FieldDefinition
	for _, f := range t.Fields {
		relInfo := ExtractRelationshipDirective(f)
		if relInfo.HasDirective {
			continue
		}
		fields = append(fields, buildFieldDefinition(f, enumNames))
	}
	return fields
}

// buildFieldDefinition converts an AST field to a FieldDefinition with type mappings.
func buildFieldDefinition(f *ast.FieldDefinition, enumNames map[string]bool) FieldDefinition {
	gqlType := formatGraphQLType(f.Type)
	baseName := f.Type.Name()
	isEnum := enumNames[baseName]
	nullable := !f.Type.NonNull
	isList := f.Type.Elem != nil
	isID := baseName == "ID" && !nullable

	return FieldDefinition{
		Name:        f.Name,
		GraphQLType: gqlType,
		GoType:      GraphQLToGoType(gqlType, isEnum),
		CypherType:  GraphQLToCypherType(gqlType, isEnum),
		Nullable:    nullable,
		IsList:      isList,
		IsID:        isID,
	}
}

// buildCypherFieldDefinition converts an AST field with @cypher to a CypherFieldDefinition.
func buildCypherFieldDefinition(f *ast.FieldDefinition, cypherInfo CypherDirectiveInfo, enumNames map[string]bool) CypherFieldDefinition {
	gqlType := formatGraphQLType(f.Type)
	isEnum := enumNames[f.Type.Name()]

	var args []ArgumentDefinition
	for _, a := range f.Arguments {
		argGQLType := formatGraphQLType(a.Type)
		argDef := ArgumentDefinition{
			Name:        a.Name,
			GraphQLType: argGQLType,
			GoType:      GraphQLToGoType(argGQLType, enumNames[a.Type.Name()]),
		}
		if a.DefaultValue != nil {
			argDef.DefaultValue = a.DefaultValue.Raw
		}
		args = append(args, argDef)
	}

	return CypherFieldDefinition{
		Name:        f.Name,
		GraphQLType: gqlType,
		GoType:      GraphQLToGoType(gqlType, isEnum),
		Statement:   cypherInfo.Statement,
		IsList:      f.Type.Elem != nil,
		Nullable:    !f.Type.NonNull,
		Arguments:   args,
	}
}

// formatGraphQLType converts an ast.Type to a string representation like "String!", "Int", "[String!]".
func formatGraphQLType(t *ast.Type) string {
	if t.Elem != nil {
		// List type: [ElementType]
		inner := formatGraphQLType(t.Elem)
		return "[" + inner + "]"
	}
	name := t.NamedType
	if t.NonNull {
		return name + "!"
	}
	return name
}

// isBuiltinEnum returns true for GraphQL built-in enum types that shouldn't be
// included in the user's EnumDefinitions.
func isBuiltinEnum(name string) bool {
	switch name {
	case "__DirectiveLocation", "__TypeKind", "RelationshipDirection":
		return true
	}
	return false
}
