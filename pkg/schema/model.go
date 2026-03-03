package schema

// Direction represents the traversal direction for a relationship.
type Direction string

const (
	DirectionIN  Direction = "IN"
	DirectionOUT Direction = "OUT"
)

// FieldDefinition represents a scalar field on a node or relationship properties type.
type FieldDefinition struct {
	Name        string
	GraphQLType string
	GoType      string
	CypherType  string
	Nullable    bool
	IsList      bool
	IsID        bool
}

// ArgumentDefinition represents an argument on a @cypher field.
// DefaultValue is nil if no default is specified in the schema.
type ArgumentDefinition struct {
	Name        string
	GraphQLType string
	GoType      string
	DefaultValue any
}

// CypherFieldDefinition represents a field annotated with @cypher(statement).
// These are read-only computed fields, separate from stored Fields.
// Arguments are the field's GraphQL arguments, which become $paramName parameters
// in the Cypher statement.
type CypherFieldDefinition struct {
	Name        string
	GraphQLType string
	GoType      string
	Statement   string
	IsList      bool
	Nullable    bool
	Arguments   []ArgumentDefinition
}

// VectorFieldDefinition represents a field annotated with @vector(indexName, dimensions, similarity)
// on a @node type. The field must be of GraphQL type [Float!]!.
type VectorFieldDefinition struct {
	Name       string
	IndexName  string
	Dimensions int
	Similarity string
}

// NodeDefinition represents a GraphQL type annotated with @node.
type NodeDefinition struct {
	Name         string
	Labels       []string
	Fields       []FieldDefinition
	CypherFields []CypherFieldDefinition
	VectorField  *VectorFieldDefinition
}

// PropertiesDefinition represents a type annotated with @relationshipProperties.
type PropertiesDefinition struct {
	TypeName string
	Fields   []FieldDefinition
}

// RelationshipDefinition represents a field annotated with @relationship.
type RelationshipDefinition struct {
	FieldName  string
	RelType    string
	Direction  Direction
	FromNode   string
	ToNode     string
	Properties *PropertiesDefinition
}

// EnumDefinition represents a GraphQL enum type.
type EnumDefinition struct {
	Name   string
	Values []string
}

// GraphModel is the complete graph model inferred from a parsed schema.
// Immutable after construction — all accessor methods return copies.
type GraphModel struct {
	Nodes         []NodeDefinition
	Relationships []RelationshipDefinition
	Enums         []EnumDefinition
	CustomScalars []string // custom scalar names from the schema (e.g. "DateTime", "JSON")
}

// NodeByName looks up a node by its GraphQL type name.
// Returns a copy of the NodeDefinition and true if found, zero value and false otherwise.
func (m GraphModel) NodeByName(name string) (NodeDefinition, bool) {
	for _, n := range m.Nodes {
		if n.Name == name {
			// Return a deep copy to preserve immutability
			labels := make([]string, len(n.Labels))
			copy(labels, n.Labels)
			fields := make([]FieldDefinition, len(n.Fields))
			copy(fields, n.Fields)
			cypherFields := make([]CypherFieldDefinition, len(n.CypherFields))
			copy(cypherFields, n.CypherFields)
			var vf *VectorFieldDefinition
			if n.VectorField != nil {
				vfCopy := *n.VectorField
				vf = &vfCopy
			}
			return NodeDefinition{
				Name:         n.Name,
				Labels:       labels,
				Fields:       fields,
				CypherFields: cypherFields,
				VectorField:  vf,
			}, true
		}
	}
	return NodeDefinition{}, false
}

// HasVectorField returns true if any node in the model has a @vector field.
func (m GraphModel) HasVectorField() bool {
	for _, n := range m.Nodes {
		if n.VectorField != nil {
			return true
		}
	}
	return false
}

// RelationshipsForNode returns all relationships where FromNode or ToNode matches the given name.
// Returns deep copies of the RelationshipDefinitions, including Properties.
func (m GraphModel) RelationshipsForNode(nodeName string) []RelationshipDefinition {
	var result []RelationshipDefinition
	for _, r := range m.Relationships {
		if r.FromNode == nodeName || r.ToNode == nodeName {
			if r.Properties != nil {
				fields := make([]FieldDefinition, len(r.Properties.Fields))
				copy(fields, r.Properties.Fields)
				r.Properties = &PropertiesDefinition{
					TypeName: r.Properties.TypeName,
					Fields:   fields,
				}
			}
			result = append(result, r)
		}
	}
	return result
}
