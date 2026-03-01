package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
)

// === E2E-3: E2E compilation test for Tier 1 features ===

// tier1Schema returns a GraphQL schema string exercising ALL Tier 1 features:
// - Multiple node types (Movie, Actor, Genre)
// - @relationship with properties (ACTED_IN + ActedInProperties)
// - @relationship without properties (IN_GENRE)
// - @cypher field with arguments (recommended)
// - @cypher field without arguments (averageRating)
// - Multiple scalar types (String, Int, Float, Boolean, ID)
// This schema, when passed through the pipeline, must produce compilable Go code
// that uses filter operators, sorting, nested mutations, @cypher resolvers,
// and relationship connections.
func tier1Schema() string {
	return `type Movie @node {
	id: ID!
	title: String!
	released: Int
	rating: Float
	active: Boolean
	actors: [Actor!]! @relationship(type: "ACTED_IN", direction: IN, properties: "ActedInProperties")
	genres: [Genre!]! @relationship(type: "IN_GENRE", direction: OUT)
	averageRating: Float @cypher(statement: "MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.rating)")
	recommended(limit: Int!): [Movie!]! @cypher(statement: "MATCH (this)-[:ACTED_IN]->()<-[:ACTED_IN]-(rec) RETURN rec LIMIT $limit")
}

type Actor @node {
	id: ID!
	name: String!
	born: Int
	movies: [Movie!]! @relationship(type: "ACTED_IN", direction: OUT, properties: "ActedInProperties")
}

type Genre @node {
	id: ID!
	name: String!
}

type ActedInProperties @relationshipProperties {
	role: String!
	screenTime: Int
}
`
}

// tier1Model returns the GraphModel corresponding to tier1Schema(),
// including CypherFields for @cypher directive testing.
func tier1Model() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", Nullable: false, IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", Nullable: false},
					{Name: "released", GraphQLType: "Int", GoType: "*int", Nullable: true},
					{Name: "rating", GraphQLType: "Float", GoType: "*float64", Nullable: true},
					{Name: "active", GraphQLType: "Boolean", GoType: "*bool", Nullable: true},
				},
				CypherFields: []schema.CypherFieldDefinition{
					{
						Name:        "averageRating",
						GraphQLType: "Float",
						GoType:      "*float64",
						Statement:   "MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.rating)",
						IsList:      false,
						Nullable:    true,
						Arguments:   nil,
					},
					{
						Name:        "recommended",
						GraphQLType: "[Movie!]!",
						GoType:      "[]*Movie",
						Statement:   "MATCH (this)-[:ACTED_IN]->()<-[:ACTED_IN]-(rec) RETURN rec LIMIT $limit",
						IsList:      true,
						Nullable:    false,
						Arguments: []schema.ArgumentDefinition{
							{Name: "limit", GraphQLType: "Int!", GoType: "int"},
						},
					},
				},
			},
			{
				Name:   "Actor",
				Labels: []string{"Actor"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", Nullable: false, IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string", Nullable: false},
					{Name: "born", GraphQLType: "Int", GoType: "*int", Nullable: true},
				},
			},
			{
				Name:   "Genre",
				Labels: []string{"Genre"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", Nullable: false, IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string", Nullable: false},
				},
			},
		},
		Relationships: []schema.RelationshipDefinition{
			{
				FieldName: "actors",
				RelType:   "ACTED_IN",
				Direction: schema.DirectionIN,
				FromNode:  "Movie",
				ToNode:    "Actor",
				Properties: &schema.PropertiesDefinition{
					TypeName: "ActedInProperties",
					Fields: []schema.FieldDefinition{
						{Name: "role", GraphQLType: "String!", GoType: "string", Nullable: false},
						{Name: "screenTime", GraphQLType: "Int", GoType: "*int", Nullable: true},
					},
				},
			},
			{
				FieldName: "movies",
				RelType:   "ACTED_IN",
				Direction: schema.DirectionOUT,
				FromNode:  "Actor",
				ToNode:    "Movie",
				Properties: &schema.PropertiesDefinition{
					TypeName: "ActedInProperties",
					Fields: []schema.FieldDefinition{
						{Name: "role", GraphQLType: "String!", GoType: "string", Nullable: false},
						{Name: "screenTime", GraphQLType: "Int", GoType: "*int", Nullable: true},
					},
				},
			},
			{
				FieldName: "genres",
				RelType:   "IN_GENRE",
				Direction: schema.DirectionOUT,
				FromNode:  "Movie",
				ToNode:    "Genre",
			},
		},
	}
}

// writeTier1Schema writes the Tier 1 schema to a temp directory and returns paths.
func writeTier1Schema(t *testing.T) (schemaPath, outputDir string) {
	t.Helper()
	baseDir := t.TempDir()
	schemaPath = filepath.Join(baseDir, "schema.graphql")
	if err := os.WriteFile(schemaPath, []byte(tier1Schema()), 0644); err != nil {
		t.Fatalf("failed to write schema file: %v", err)
	}
	outputDir = filepath.Join(baseDir, "generated")
	return schemaPath, outputDir
}

// --- Augmented schema content tests ---

// TestE2ETier1_AugmentedSchema_FilterOperators verifies that the augmented schema
// includes filter operator fields in Where inputs (e.g., title_contains, released_gt).
// Expected: MovieWhere contains operator-suffixed fields like title_contains, released_gte.
func TestE2ETier1_AugmentedSchema_FilterOperators(t *testing.T) {
	model := tier1Model()
	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// String operators
	for _, op := range []string{"title_contains", "title_startsWith", "title_endsWith", "title_in"} {
		if !strings.Contains(sdl, op) {
			t.Errorf("augmented schema should contain filter operator %q in MovieWhere, got:\n%s", op, sdl)
		}
	}

	// Int/Float comparison operators
	for _, op := range []string{"released_gt", "released_gte", "released_lt", "released_lte"} {
		if !strings.Contains(sdl, op) {
			t.Errorf("augmented schema should contain filter operator %q in MovieWhere, got:\n%s", op, sdl)
		}
	}

	// Boolean composition
	for _, op := range []string{"AND: [MovieWhere!]", "OR: [MovieWhere!]", "NOT: MovieWhere"} {
		if !strings.Contains(sdl, op) {
			t.Errorf("augmented schema should contain boolean composition %q, got:\n%s", op, sdl)
		}
	}
}

// TestE2ETier1_AugmentedSchema_SortInputs verifies that the augmented schema
// includes Sort input types with SortDirection enum.
// Expected: MovieSort input type exists with fields using SortDirection.
func TestE2ETier1_AugmentedSchema_SortInputs(t *testing.T) {
	model := tier1Model()
	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	if !strings.Contains(sdl, "enum SortDirection") {
		t.Errorf("augmented schema should contain SortDirection enum, got:\n%s", sdl)
	}
	if !strings.Contains(sdl, "input MovieSort") {
		t.Errorf("augmented schema should contain MovieSort input, got:\n%s", sdl)
	}
	if !strings.Contains(sdl, "title: SortDirection") {
		t.Errorf("MovieSort should have title: SortDirection field, got:\n%s", sdl)
	}
	// Query should accept sort parameter
	if !strings.Contains(sdl, "sort: [MovieSort!]") {
		t.Errorf("query should accept sort parameter, got:\n%s", sdl)
	}
}

// TestE2ETier1_AugmentedSchema_NestedMutationInputs verifies that the augmented schema
// includes disconnect/update/delete fields in UpdateFieldInput types.
// Expected: MovieActorsUpdateFieldInput has disconnect, update, delete fields.
func TestE2ETier1_AugmentedSchema_NestedMutationInputs(t *testing.T) {
	model := tier1Model()
	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// UpdateFieldInput should exist with all 5 operations
	if !strings.Contains(sdl, "MovieActorsUpdateFieldInput") {
		t.Errorf("augmented schema should contain MovieActorsUpdateFieldInput, got:\n%s", sdl)
	}

	// Disconnect/update/delete field input types
	if !strings.Contains(sdl, "MovieActorsDisconnectFieldInput") {
		t.Errorf("augmented schema should contain MovieActorsDisconnectFieldInput, got:\n%s", sdl)
	}
	if !strings.Contains(sdl, "MovieActorsDeleteFieldInput") {
		t.Errorf("augmented schema should contain MovieActorsDeleteFieldInput, got:\n%s", sdl)
	}

	// UpdateInput should reference UpdateFieldInput for relationship fields
	if !strings.Contains(sdl, "actors: MovieActorsUpdateFieldInput") {
		t.Errorf("MovieUpdateInput should reference UpdateFieldInput for actors, got:\n%s", sdl)
	}
}

// TestE2ETier1_AugmentedSchema_CypherFields verifies that the augmented schema
// includes @cypher fields in the node object types.
// Expected: Movie type has averageRating: Float and recommended(limit: Int!): [Movie!]!
func TestE2ETier1_AugmentedSchema_CypherFields(t *testing.T) {
	model := tier1Model()
	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// @cypher field without args
	if !strings.Contains(sdl, "averageRating: Float") {
		t.Errorf("augmented schema Movie type should contain averageRating field, got:\n%s", sdl)
	}

	// @cypher field with args
	if !strings.Contains(sdl, "recommended") {
		t.Errorf("augmented schema Movie type should contain recommended field, got:\n%s", sdl)
	}
	if !strings.Contains(sdl, "limit: Int!") {
		t.Errorf("augmented schema recommended field should have limit argument, got:\n%s", sdl)
	}
}

// TestE2ETier1_AugmentedSchema_RelationshipConnections verifies that the augmented schema
// includes relationship-level Relay connection types on parent objects.
// Expected: Movie type has actorsConnection field returning ActorsConnection with edges.
func TestE2ETier1_AugmentedSchema_RelationshipConnections(t *testing.T) {
	model := tier1Model()
	sdl, err := AugmentSchema(model)
	if err != nil {
		t.Fatalf("AugmentSchema returned error: %v", err)
	}

	// Relationship connection field on parent type
	if !strings.Contains(sdl, "actorsConnection") {
		t.Errorf("augmented schema Movie type should contain actorsConnection field, got:\n%s", sdl)
	}

	// Relationship-level connection type (distinct from root-level)
	if !strings.Contains(sdl, "MovieActorsConnection") {
		t.Errorf("augmented schema should contain MovieActorsConnection type, got:\n%s", sdl)
	}

	// Edge type with relationship properties
	if !strings.Contains(sdl, "MovieActorsEdge") {
		t.Errorf("augmented schema should contain MovieActorsEdge type, got:\n%s", sdl)
	}

	// Edge properties (ActedInProperties fields on the edge)
	if !strings.Contains(sdl, "role: String!") {
		t.Errorf("augmented schema MovieActorsEdge should include role property, got:\n%s", sdl)
	}
}

// --- Pipeline + compilation tests ---

// TestE2ETier1_GenerateSucceeds verifies that the full Generate() pipeline
// completes without error on the Tier 1 schema.
// Expected: Generate returns nil error.
func TestE2ETier1_GenerateSucceeds(t *testing.T) {
	schemaPath, outputDir := writeTier1Schema(t)

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	err := Generate(cfg)
	if err != nil {
		t.Fatalf("Generate failed on Tier 1 schema: %v", err)
	}
}

// TestE2ETier1_AugmentedSchemaFile_ContainsTier1Types verifies that the written
// schema.graphql file in the output directory contains Tier 1 augmented types.
// Expected: the file contains filter operators, sort inputs, connection types.
func TestE2ETier1_AugmentedSchemaFile_ContainsTier1Types(t *testing.T) {
	schemaPath, outputDir := writeTier1Schema(t)

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	schemaContent, err := os.ReadFile(filepath.Join(outputDir, "schema.graphql"))
	if err != nil {
		t.Fatalf("failed to read schema.graphql: %v", err)
	}
	sdl := string(schemaContent)

	// Spot-check a few Tier 1 types in the written file
	checks := []string{
		"title_contains",          // filter operators
		"SortDirection",           // sort enum
		"MovieSort",              // sort input
		"MovieActorsUpdateFieldInput", // nested mutation
		"averageRating",          // @cypher field
		"MovieActorsConnection",  // relationship connection
	}
	for _, check := range checks {
		if !strings.Contains(sdl, check) {
			t.Errorf("written schema.graphql should contain %q but doesn't", check)
		}
	}
}

// TestE2ETier1_GoBuildSucceeds verifies that the generated output from the full
// pipeline (Tier 1 schema with all features) compiles successfully.
// This is the ultimate acceptance gate for Tier 1: generated Go code with filter
// operators, sorting, nested mutations, @cypher resolvers, and relationship
// connections must be valid and compilable.
// Expected: `go build ./...` on the output directory exits with code 0.
func TestE2ETier1_GoBuildSucceeds(t *testing.T) {
	schemaPath, outputDir := writeTier1Schema(t)

	cfg := Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	}

	if err := Generate(cfg); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// The generated package needs a go.mod to compile.
	goModContent := "module generated\n\ngo 1.25\n\nrequire (\n" +
		"\tgithub.com/99designs/gqlgen v0.17.87\n" +
		"\tgithub.com/tab58/gql-orm v0.0.0\n" +
		")\n\n" +
		"replace github.com/tab58/gql-orm => " + projectRoot(t) + "\n"
	goModPath := filepath.Join(outputDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go mod tidy to resolve transitive dependencies
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = outputDir
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed: %s\n%v", string(output), err)
	}

	// Run go build to verify the generated code compiles
	buildCmd := exec.Command("go", "build", "./...")
	buildCmd.Dir = outputDir
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed on Tier 1 generated output:\n%s\n%v", string(output), err)
	}
}
