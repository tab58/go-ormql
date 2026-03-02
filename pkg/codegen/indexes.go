package codegen

import (
	"fmt"
	"go/format"
	"strings"

	"github.com/tab58/gql-orm/pkg/schema"
)

// vectorIndexDDLFormat is the Cypher DDL template for creating a vector index.
// Arguments: indexName, label, fieldName, dimensions, similarity.
const vectorIndexDDLFormat = "CREATE VECTOR INDEX %s IF NOT EXISTS FOR (n:%s) ON (n.%s) OPTIONS {indexConfig: {`vector.dimensions`: %d, `vector.similarity_function`: '%s'}}"

// GenerateIndexes produces Go source code containing a CreateIndexes function
// that creates vector indexes using driver.ExecuteWrite.
// Returns nil, nil when no nodes have a VectorField (no indexes needed).
func GenerateIndexes(model schema.GraphModel, packageName string) ([]byte, error) {
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
	sb.WriteString("\t\"github.com/tab58/gql-orm/pkg/cypher\"\n")
	sb.WriteString("\t\"github.com/tab58/gql-orm/pkg/driver\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString("// CreateIndexes creates vector indexes for nodes with @vector directives.\n")
	sb.WriteString("func CreateIndexes(ctx context.Context, drv driver.Driver) error {\n")

	for _, idx := range indexes {
		ddl := fmt.Sprintf(vectorIndexDDLFormat,
			idx.indexName, idx.label, idx.fieldName, idx.dimensions, idx.similarity,
		)
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
