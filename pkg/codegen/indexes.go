package codegen

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/tab58/go-ormql/pkg/schema"
)

// vectorIndexDDLFormat is the Neo4j Cypher DDL template for creating a vector index.
// Arguments: indexName, label, fieldName, dimensions, similarity.
const vectorIndexDDLFormat = "CREATE VECTOR INDEX %s IF NOT EXISTS FOR (n:%s) ON (n.%s) OPTIONS {indexConfig: {`vector.dimensions`: %d, `vector.similarity_function`: '%s'}}"

// falkorDBVectorIndexDDLFormat is the FalkorDB Cypher DDL template for creating a vector index.
// Arguments: label, fieldName, dimensions, similarity.
const falkorDBVectorIndexDDLFormat = "CREATE VECTOR INDEX FOR (n:%s) ON (n.%s) OPTIONS {dimension: %d, similarityFunction: '%s'}"

// GenerateIndexes produces Go source code containing a CreateIndexes function
// that creates vector indexes using driver.ExecuteWrite.
// Returns nil, nil when no nodes have a VectorField (no indexes needed).
// The target parameter controls the DDL dialect (Neo4j vs FalkorDB).
func GenerateIndexes(model schema.GraphModel, packageName string, target Target) ([]byte, error) {
	if !model.HasVectorField() {
		return nil, nil
	}

	// Collect nodes with VectorField
	type vectorIndex struct {
		label      string
		fieldName  string
		indexName  string
		dimensions int
		similarity string
	}
	var indexes []vectorIndex
	for _, n := range model.Nodes {
		if n.VectorField != nil {
			indexes = append(indexes, vectorIndex{
				label:      n.Labels[0],
				fieldName:  n.VectorField.Name,
				indexName:  n.VectorField.IndexName,
				dimensions: n.VectorField.Dimensions,
				similarity: n.VectorField.Similarity,
			})
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\t\"fmt\"\n\n")
	sb.WriteString("\t\"github.com/tab58/go-ormql/pkg/cypher\"\n")
	sb.WriteString("\t\"github.com/tab58/go-ormql/pkg/driver\"\n")
	sb.WriteString(")\n\n")

	// FalkorDB: generate VectorIndexes var for driver-level vector query rewrite
	if target == TargetFalkorDB {
		sb.WriteString("// VectorIndexes maps index names to their label/property for FalkorDB vector query rewrite.\n")
		sb.WriteString("var VectorIndexes = map[string]driver.VectorIndex{\n")
		for _, idx := range indexes {
			sb.WriteString(fmt.Sprintf("\t%q: {Label: %q, Property: %q},\n", idx.indexName, idx.label, idx.fieldName))
		}
		sb.WriteString("}\n\n")
	}

	sb.WriteString("// CreateIndexes creates vector indexes for nodes with @vector directives.\n")
	sb.WriteString("func CreateIndexes(ctx context.Context, drv driver.Driver) error {\n")

	for _, idx := range indexes {
		var ddl string
		switch target {
		case TargetFalkorDB:
			ddl = fmt.Sprintf(falkorDBVectorIndexDDLFormat,
				idx.label, idx.fieldName, idx.dimensions, idx.similarity,
			)
		default:
			ddl = fmt.Sprintf(vectorIndexDDLFormat,
				idx.indexName, idx.label, idx.fieldName, idx.dimensions, idx.similarity,
			)
		}
		sb.WriteString(fmt.Sprintf("\tif _, err := drv.ExecuteWrite(ctx, cypher.Statement{Query: %q}); err != nil {\n", ddl))
		sb.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"failed to create vector index %s: %%w\", err)\n", idx.indexName))
		sb.WriteString("\t}\n")
	}

	sb.WriteString("\treturn nil\n")
	sb.WriteString("}\n")

	src := sb.String()
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return []byte(src), nil
	}
	return formatted, nil
}
