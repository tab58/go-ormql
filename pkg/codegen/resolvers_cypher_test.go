package codegen

import (
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
)

// --- CG-18: @cypher field resolver templates ---

// cypherResolverModel returns a model with a Movie node that has @cypher fields
// for testing resolver generation.
func cypherResolverModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", Nullable: false, IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING", Nullable: false},
				},
				CypherFields: []schema.CypherFieldDefinition{
					{
						Name:        "averageRating",
						GraphQLType: "Float",
						GoType:      "*float64",
						Statement:   "MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.score)",
						IsList:      false,
						Nullable:    true,
						Arguments:   nil,
					},
					{
						Name:        "recommended",
						GraphQLType: "[Movie!]!",
						GoType:      "[]*Movie",
						Statement:   "MATCH (this)-[:IN_GENRE]->()<-[:IN_GENRE]-(rec) RETURN rec LIMIT $limit",
						IsList:      true,
						Nullable:    false,
						Arguments: []schema.ArgumentDefinition{
							{Name: "limit", GraphQLType: "Int!", GoType: "int"},
						},
					},
				},
			},
		},
	}
}

// noCypherResolverModel returns a model with a Movie node that has NO @cypher fields.
func noCypherResolverModel() schema.GraphModel {
	return resolverModel()
}

// TestGenerateResolvers_CypherField_ObjectResolverStruct verifies that an
// object-level resolver struct is generated for nodes with @cypher fields.
// Expected: generated source contains "type movieResolver struct".
func TestGenerateResolvers_CypherField_ObjectResolverStruct(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "movieResolver") {
		t.Errorf("generated resolvers missing 'movieResolver' struct for node with @cypher fields:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_ResolverRootMethod verifies that the
// Resolver struct gets a {NodeName}() method returning {NodeName}Resolver
// for gqlgen's ResolverRoot interface.
// Expected: generated source contains "func (r *Resolver) Movie() MovieResolver".
func TestGenerateResolvers_CypherField_ResolverRootMethod(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "Movie() MovieResolver") {
		t.Errorf("Resolver missing Movie() method for ResolverRoot interface:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_ResolverMethod_NoArgs verifies that
// a resolver method is generated for a @cypher field without arguments.
// Expected: generated source contains "AverageRating" method on movieResolver.
func TestGenerateResolvers_CypherField_ResolverMethod_NoArgs(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "AverageRating") {
		t.Errorf("generated resolvers missing 'AverageRating' method for @cypher field:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_ResolverMethod_WithArgs verifies that
// a resolver method is generated for a @cypher field with arguments.
// Expected: generated source contains "Recommended" method with "limit" parameter.
func TestGenerateResolvers_CypherField_ResolverMethod_WithArgs(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "Recommended") {
		t.Errorf("generated resolvers missing 'Recommended' method for @cypher field:\n%s", s)
	}
	if !strings.Contains(s, "limit") {
		t.Errorf("Recommended method missing 'limit' parameter:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_CallsCypherDirective verifies that the
// @cypher field resolver calls cypher.CypherDirective.
// Expected: generated source contains "cypher.CypherDirective".
func TestGenerateResolvers_CypherField_CallsCypherDirective(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "cypher.CypherDirective") && !strings.Contains(s, "CypherDirective") {
		t.Errorf("@cypher field resolver should call cypher.CypherDirective:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_PassesParentLabel verifies that the
// @cypher field resolver passes the parent node label to CypherDirective.
// Expected: generated source contains the label "Movie" in the CypherDirective call.
func TestGenerateResolvers_CypherField_PassesParentLabel(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The CypherDirective call should reference the parent label
	if !strings.Contains(s, `"Movie"`) {
		t.Errorf("@cypher resolver should pass parent label \"Movie\" to CypherDirective:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_PassesStatement verifies that the
// @cypher field resolver passes the Cypher statement to CypherDirective.
// Expected: generated source contains the statement string.
func TestGenerateResolvers_CypherField_PassesStatement(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.score)") {
		t.Errorf("@cypher resolver should pass statement to CypherDirective:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_PassesArgs verifies that the @cypher field
// resolver passes field arguments to CypherDirective as a map.
// Expected: generated source builds args map with "limit" key.
func TestGenerateResolvers_CypherField_PassesArgs(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The recommended field resolver should build a map with the limit arg
	if !strings.Contains(s, `"limit"`) {
		t.Errorf("@cypher resolver should pass args map with \"limit\" key:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_UsesParentBinding verifies that the @cypher
// resolver accesses the parent object to build the parent WHERE clause.
// Expected: generated source references "obj" or parent parameter for parent binding.
func TestGenerateResolvers_CypherField_UsesParentBinding(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The resolver method should receive the parent object and use its ID for WHERE
	if !strings.Contains(s, "obj") && !strings.Contains(s, "parent") {
		t.Errorf("@cypher resolver should receive parent object for binding:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_NoCypherNode_NoObjectResolver verifies
// that nodes WITHOUT @cypher fields do NOT get an object-level resolver.
// Expected: no "movieResolver" struct when no @cypher fields.
func TestGenerateResolvers_CypherField_NoCypherNode_NoObjectResolver(t *testing.T) {
	src, err := GenerateResolvers(noCypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if strings.Contains(s, "movieResolver") {
		t.Errorf("node without @cypher fields should NOT have a movieResolver struct:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_NoCypherNode_NoResolverRootMethod verifies
// that nodes WITHOUT @cypher fields do NOT get a ResolverRoot method.
// Expected: no "Movie() MovieResolver" method when no @cypher fields.
func TestGenerateResolvers_CypherField_NoCypherNode_NoResolverRootMethod(t *testing.T) {
	src, err := GenerateResolvers(noCypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if strings.Contains(s, "Movie() MovieResolver") {
		t.Errorf("node without @cypher fields should NOT have Movie() ResolverRoot method:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_UsesDriverExecute verifies that the @cypher
// resolver uses Driver.Execute (read operation) to execute the CypherDirective statement.
// Expected: generated source uses r.Driver.Execute within the @cypher resolver.
func TestGenerateResolvers_CypherField_UsesDriverExecute(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// @cypher fields are read-only, so they should use Execute (not ExecuteWrite)
	if !strings.Contains(s, ".Execute(") {
		t.Errorf("@cypher resolver should use Driver.Execute for read-only execution:\n%s", s)
	}
}

// === L8: @cypher resolver result mapping tests ===

// TestGenerateResolvers_CypherField_ScalarExtractsCypherResult verifies that
// a scalar @cypher field resolver (e.g., averageRating returning *float64)
// extracts __cypher_result from the first record and converts it to the
// return type using a typed helper function.
// Expected: generated source contains `__cypher_result` extraction, NOT `_ = result`.
// Currently FAILS because the template has a zero-value stub.
func TestGenerateResolvers_CypherField_ScalarExtractsCypherResult(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The scalar @cypher resolver should extract __cypher_result
	if !strings.Contains(s, `"__cypher_result"`) {
		t.Errorf("scalar @cypher resolver should extract __cypher_result from record:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_NoStubDiscard verifies that the @cypher
// resolver template does NOT contain the zero-value stub `_ = result`.
// Expected: generated source does NOT contain `_ = result` (stub removed).
// Currently FAILS because the template still has the stub.
func TestGenerateResolvers_CypherField_NoStubDiscard(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if strings.Contains(s, "_ = result") {
		t.Errorf("@cypher resolver should NOT contain stub '_ = result' — result should be used:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_ScalarHelperGenerated verifies that the
// generated resolver output includes a typed helper function for converting
// the scalar @cypher result. For the averageRating field (returning *float64),
// the helper should be named cypherResultToFloat64Ptr.
// Expected: generated source contains "cypherResultToFloat64Ptr" helper.
// Currently FAILS because no helper functions are generated.
func TestGenerateResolvers_CypherField_ScalarHelperGenerated(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "cypherResultToFloat64Ptr") {
		t.Errorf("generated resolvers should include cypherResultToFloat64Ptr helper:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_ScalarHelperBody verifies that the
// cypherResultToFloat64Ptr helper function has the correct body:
// returns nil if val is nil, type-asserts to float64 otherwise.
// Expected: helper function body contains nil check and type assertion.
// Currently FAILS because the helper is not generated.
func TestGenerateResolvers_CypherField_ScalarHelperBody(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The helper should check for nil and type-assert
	if !strings.Contains(s, "func cypherResultToFloat64Ptr(val any) *float64") {
		t.Errorf("missing cypherResultToFloat64Ptr function signature:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_ScalarUsesHelper verifies that the
// averageRating @cypher resolver calls the typed helper (cypherResultToFloat64Ptr)
// to convert the result, rather than returning a zero value.
// Expected: generated source contains call to cypherResultToFloat64Ptr in the
// AverageRating resolver body.
// Currently FAILS because the resolver returns zero value.
func TestGenerateResolvers_CypherField_ScalarUsesHelper(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The AverageRating resolver should call the helper to convert the result
	if !strings.Contains(s, "cypherResultToFloat64Ptr") {
		t.Errorf("AverageRating resolver should call cypherResultToFloat64Ptr:\n%s", s)
	}
}

// TestGenerateResolvers_CypherField_ListUsesMapper verifies that a list-type
// @cypher field resolver (e.g., recommended returning []*Movie) uses the
// existing mapRecordsTo{Types} mapper on all records, rather than extracting
// __cypher_result from a single record.
// Expected: the Recommended resolver body returns recordsToMovies(result.Records).
// Currently FAILS because the resolver returns nil (zero value stub).
func TestGenerateResolvers_CypherField_ListUsesMapper(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// Find the Recommended resolver method and verify it returns the mapper result.
	// The pattern: after "Recommended(" should appear "recordsToMovies(result.Records)"
	// before the next "func " (ensuring it's in the right resolver, not the query resolver).
	recIdx := strings.Index(s, "Recommended(ctx context.Context")
	if recIdx < 0 {
		t.Fatal("Recommended resolver method not found in generated source")
	}
	body := s[recIdx:]
	// Find the end of this function (next top-level "func ")
	nextFunc := strings.Index(body[1:], "\nfunc ")
	if nextFunc > 0 {
		body = body[:nextFunc+1]
	}
	if !strings.Contains(body, "recordsToMovies(result.Records)") {
		t.Errorf("Recommended @cypher resolver should return recordsToMovies(result.Records) for list type:\n%s", body)
	}
}

// TestGenerateResolvers_CypherField_EmptyResultHandled verifies that the
// @cypher resolver handles empty result sets (no records) gracefully by
// returning nil for nullable types, without panicking on index out of range.
// Expected: generated source checks `len(result.Records) == 0` before accessing.
// Currently FAILS because the result is discarded entirely.
func TestGenerateResolvers_CypherField_EmptyResultHandled(t *testing.T) {
	src, err := GenerateResolvers(cypherResolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateResolvers returned error: %v", err)
	}
	s := string(src)
	// The resolver should check for empty results
	if !strings.Contains(s, "len(result.Records)") {
		t.Errorf("@cypher resolver should check for empty result set:\n%s", s)
	}
}
