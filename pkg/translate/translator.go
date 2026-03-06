package translate

import (
	"fmt"

	"github.com/tab58/go-ormql/pkg/cypher"
	"github.com/tab58/go-ormql/pkg/schema"
	"github.com/vektah/gqlparser/v2/ast"
)

// defaultConnectionPageSize is the default number of items returned per page
// when no "first" argument is provided in a connection or list query.
const defaultConnectionPageSize = 10

// Translator converts parsed GraphQL ASTs into single parameterized Cypher statements.
// Holds the GraphModel for runtime lookup of node labels, relationship types, directions,
// field types, and @cypher statements. Constructed once per client, reused across translations.
// Stateless — all per-translation state lives in paramScope.
type Translator struct {
	model schema.GraphModel
}

// New creates a Translator from a GraphModel.
// The model provides runtime knowledge of node labels, relationship types,
// directions, field types, and @cypher statements.
func New(model schema.GraphModel) *Translator {
	return &Translator{model: model}
}

// TranslationPlan holds the output of translating a GraphQL operation.
// For queries: WriteStatements is empty, ReadStatement contains the single query.
// For merge mutations: WriteStatements contains FOREACH write statements (one per merge field),
// ReadStatement contains the combined read query (MATCH projections + non-merge CALL blocks).
// For non-merge mutations: WriteStatements is empty, ReadStatement contains the full mutation.
type TranslationPlan struct {
	WriteStatements []cypher.Statement
	ReadStatement   cypher.Statement
}

// Translate converts a parsed GraphQL operation into a TranslationPlan.
//
// The ReadStatement produces a single record with a single column "data"
// whose value is a map matching the GraphQL response shape.
// WriteStatements (if any) are FOREACH writes for merge mutations that must
// be executed before the ReadStatement.
//
// Returns an error for unsupported operations (e.g., subscriptions),
// unknown types, or invalid field references.
func (t *Translator) Translate(
	doc *ast.QueryDocument,
	op *ast.OperationDefinition,
	variables map[string]any,
) (TranslationPlan, error) {
	if op.Operation == ast.Subscription {
		return TranslationPlan{}, fmt.Errorf("unsupported operation type: subscription")
	}

	scope := newParamScope()
	scope.variables = variables

	switch op.Operation {
	case ast.Query:
		cypherStr, err := t.translateQuery(op, scope)
		if err != nil {
			return TranslationPlan{}, err
		}
		return TranslationPlan{
			ReadStatement: cypher.Statement{
				Query:  cypherStr,
				Params: scope.collect(),
			},
		}, nil

	case ast.Mutation:
		writeQueries, readQuery, err := t.translateMutationSplit(op, scope)
		if err != nil {
			return TranslationPlan{}, err
		}
		params := scope.collect()
		var writeStatements []cypher.Statement
		for _, wq := range writeQueries {
			writeStatements = append(writeStatements, cypher.Statement{
				Query:  wq,
				Params: params,
			})
		}
		return TranslationPlan{
			WriteStatements: writeStatements,
			ReadStatement: cypher.Statement{
				Query:  readQuery,
				Params: params,
			},
		}, nil

	default:
		return TranslationPlan{}, fmt.Errorf("unsupported operation type: %s", op.Operation)
	}
}

// fieldContext carries context for translating a field within the AST walk.
// node is the GraphModel node being queried. variable is the Cypher variable name
// (e.g., "n", "a", "child"). depth tracks nesting for unique subquery aliases.
type fieldContext struct {
	node     schema.NodeDefinition
	variable string
	depth    int
}

// buildRelPattern builds a Cypher relationship pattern string based on direction.
// relVar is the relationship variable (e.g., "r", "r0"); use "" for anonymous.
// childExpr is the child node expression (e.g., "child:Movie", "target").
//
// Examples:
//
//	buildRelPattern("n", "r", "ACTED_IN", "a:Actor", DirectionOUT)  → "(n)-[r:ACTED_IN]->(a:Actor)"
//	buildRelPattern("n", "",  "ACTED_IN", "target:Actor", DirectionIN) → "(n)<-[:ACTED_IN]-(target:Actor)"
func buildRelPattern(parentVar, relVar, relType, childExpr string, direction schema.Direction) string {
	relPart := ":" + relType
	if relVar != "" {
		relPart = relVar + ":" + relType
	}
	switch direction {
	case schema.DirectionIN:
		return fmt.Sprintf("(%s)<-[%s]-(%s)", parentVar, relPart, childExpr)
	default:
		return fmt.Sprintf("(%s)-[%s]->(%s)", parentVar, relPart, childExpr)
	}
}
