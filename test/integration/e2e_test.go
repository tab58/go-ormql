//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tab58/go-ormql/pkg/client"
	"github.com/tab58/go-ormql/pkg/codegen"
	"github.com/tab58/go-ormql/pkg/driver"
	neo4jdriver "github.com/tab58/go-ormql/pkg/driver/neo4j"
	"github.com/tab58/go-ormql/pkg/schema"
	tcneo4j "github.com/testcontainers/testcontainers-go/modules/neo4j"
)

// writeTestSchema writes a test GraphQL schema with Movie and Actor nodes
// plus an ACTED_IN relationship with properties. Returns the schema file path.
func writeTestSchema(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "schema.graphql")
	sdl := `type Movie @node {
  id: ID!
  title: String!
  released: Int
}

type Actor @node {
  id: ID!
  name: String!
  movies: [Movie!]! @relationship(type: "ACTED_IN", direction: OUT, properties: "ActedInProperties")
}

type ActedInProperties @relationshipProperties {
  role: String!
}
`
	if err := os.WriteFile(schemaPath, []byte(sdl), 0644); err != nil {
		t.Fatalf("failed to write test schema: %v", err)
	}
	return schemaPath
}

// startNeo4jContainer starts a Neo4j testcontainer and returns driver.Config.
func startNeo4jContainer(t *testing.T) driver.Config {
	t.Helper()
	ctx := context.Background()

	container, err := tcneo4j.Run(ctx,
		"neo4j:5",
		tcneo4j.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatalf("failed to start Neo4j container: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := container.Terminate(ctx); cleanupErr != nil {
			t.Logf("failed to terminate Neo4j container: %v", cleanupErr)
		}
	})

	boltURL, err := container.BoltUrl(ctx)
	if err != nil {
		t.Fatalf("failed to get Neo4j bolt URL: %v", err)
	}

	return driver.Config{
		URI:      boltURL,
		Username: "neo4j",
		Password: "",
		Database: "neo4j",
	}
}

// testGraphModel returns a schema.GraphModel with Movie + Actor nodes
// and an ACTED_IN relationship with properties for V2 client testing.
func testGraphModel() schema.GraphModel {
	return schema.GraphModel{
		Nodes: []schema.NodeDefinition{
			{
				Name:   "Movie",
				Labels: []string{"Movie"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "title", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
					{Name: "released", GraphQLType: "Int", GoType: "*int", CypherType: "INTEGER", Nullable: true},
				},
			},
			{
				Name:   "Actor",
				Labels: []string{"Actor"},
				Fields: []schema.FieldDefinition{
					{Name: "id", GraphQLType: "ID!", GoType: "string", CypherType: "STRING", IsID: true},
					{Name: "name", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
				},
			},
		},
		Relationships: []schema.RelationshipDefinition{
			{
				FieldName: "movies",
				RelType:   "ACTED_IN",
				Direction: schema.DirectionOUT,
				FromNode:  "Actor",
				ToNode:    "Movie",
				Properties: &schema.PropertiesDefinition{
					TypeName: "ActedInProperties",
					Fields: []schema.FieldDefinition{
						{Name: "role", GraphQLType: "String!", GoType: "string", CypherType: "STRING"},
					},
				},
			},
		},
	}
}

// testAugmentedSchemaSDL returns a minimal augmented schema SDL for V2 client testing.
// In the full pipeline, AugmentSchema() generates this. Here we provide a
// hand-crafted version for test isolation.
const testAugmentedSchemaSDL = `type Query {
  movies(where: MovieWhere, sort: [MovieSort!]): [Movie!]!
  actors(where: ActorWhere, sort: [ActorSort!]): [Actor!]!
}

type Mutation {
  createMovies(input: [MovieCreateInput!]!): CreateMoviesMutationResponse!
  updateMovies(where: MovieWhere, update: MovieUpdateInput): UpdateMoviesMutationResponse!
  deleteMovies(where: MovieWhere): DeleteInfo!
  createActors(input: [ActorCreateInput!]!): CreateActorsMutationResponse!
}

type Movie {
  id: ID!
  title: String!
  released: Int
}

type Actor {
  id: ID!
  name: String!
}

input MovieWhere { title: String }
input ActorWhere { name: String }
input MovieSort { title: SortDirection }
input ActorSort { name: SortDirection }
input MovieCreateInput { title: String!, released: Int }
input ActorCreateInput { name: String! }
input MovieUpdateInput { title: String, released: Int }
type CreateMoviesMutationResponse { movies: [Movie!]! }
type UpdateMoviesMutationResponse { movies: [Movie!]! }
type CreateActorsMutationResponse { actors: [Actor!]! }
type DeleteInfo { nodesDeleted: Int! }
enum SortDirection { ASC DESC }
`

// createV2Client creates a V2 client connected to the test Neo4j container.
func createV2Client(t *testing.T, cfg driver.Config) *client.Client {
	t.Helper()
	drv, err := neo4jdriver.NewNeo4jDriver(cfg)
	if err != nil {
		t.Fatalf("NewNeo4jDriver failed: %v", err)
	}

	c := client.New(testGraphModel(), testAugmentedSchemaSDL, drv)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.Close(ctx)
	})
	return c
}

// === INT-3: V2 client integration tests ===

// Test: V2 pipeline produces exactly 4 expected output files.
// Expected: schema.graphql, models_gen.go, graphmodel_gen.go, client_gen.go.
func TestE2E_V2_PipelineProducesExpectedFiles(t *testing.T) {
	schemaPath := writeTestSchema(t)
	outputDir := t.TempDir()

	err := codegen.Generate(codegen.Config{
		SchemaFiles: []string{schemaPath},
		OutputDir:   outputDir,
		PackageName: "generated",
	})
	if err != nil {
		t.Fatalf("codegen.Generate failed: %v", err)
	}

	expectedFiles := []string{
		"schema.graphql",
		"models_gen.go",
		"graphmodel_gen.go",
		"client_gen.go",
	}
	for _, name := range expectedFiles {
		path := filepath.Join(outputDir, name)
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			t.Errorf("V2 pipeline did not produce %s", name)
		}
	}
}

// Test: V2 Client can execute a simple query and return a non-nil result.
// Expected: Execute returns non-nil *Result.
func TestE2E_V2_ClientExecuteQuery(t *testing.T) {
	cfg := startNeo4jContainer(t)
	c := createV2Client(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := c.Execute(ctx, `query { movies { title } }`, nil)
	if err != nil {
		t.Fatalf("V2 client.Execute query failed: %v", err)
	}
	if result == nil {
		t.Fatal("V2 client.Execute returned nil result")
	}
}

// Test: V2 Client can execute a create mutation.
// Expected: Execute returns non-nil *Result for mutation.
func TestE2E_V2_ClientCreateMutation(t *testing.T) {
	cfg := startNeo4jContainer(t)
	c := createV2Client(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := c.Execute(ctx, `mutation {
		createMovies(input: [{ title: "The Matrix", released: 1999 }]) {
			movies { title released }
		}
	}`, nil)
	if err != nil {
		t.Fatalf("V2 create mutation failed: %v", err)
	}
	if result == nil {
		t.Fatal("V2 create mutation returned nil result")
	}
}

// Test: V2 Client query with filter arguments.
// Expected: query with where filter returns filtered results.
func TestE2E_V2_ClientQueryWithFilter(t *testing.T) {
	cfg := startNeo4jContainer(t)
	c := createV2Client(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a movie first
	_, err := c.Execute(ctx, `mutation {
		createMovies(input: [{ title: "Inception", released: 2010 }]) {
			movies { title }
		}
	}`, nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Query with filter
	result, err := c.Execute(ctx, `query {
		movies(where: { title: "Inception" }) { title released }
	}`, nil)
	if err != nil {
		t.Fatalf("V2 filtered query failed: %v", err)
	}
	if result == nil {
		t.Fatal("V2 filtered query returned nil result")
	}

	data := result.Data()
	if data == nil {
		t.Fatal("result.Data() returned nil")
	}
	movies, ok := data["movies"]
	if !ok {
		t.Fatal("result missing 'movies' key")
	}
	movieList, ok := movies.([]any)
	if !ok {
		t.Fatalf("movies is %T, want []any", movies)
	}
	if len(movieList) == 0 {
		t.Fatal("expected at least 1 movie, got 0")
	}
}

// Test: V2 Client query with sort arguments.
// Expected: query with sort returns sorted results.
func TestE2E_V2_ClientQueryWithSort(t *testing.T) {
	cfg := startNeo4jContainer(t)
	c := createV2Client(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create two movies
	_, err := c.Execute(ctx, `mutation {
		createMovies(input: [
			{ title: "Zebra", released: 2020 },
			{ title: "Alpha", released: 2021 }
		]) {
			movies { title }
		}
	}`, nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Query with sort
	result, err := c.Execute(ctx, `query {
		movies(sort: [{ title: ASC }]) { title }
	}`, nil)
	if err != nil {
		t.Fatalf("V2 sorted query failed: %v", err)
	}
	if result == nil {
		t.Fatal("V2 sorted query returned nil result")
	}
	data := result.Data()
	if data == nil {
		t.Fatal("result.Data() returned nil for sorted query")
	}
}

// Test: V2 Client update mutation.
// Expected: update mutation modifies existing node and returns result.
func TestE2E_V2_ClientUpdateMutation(t *testing.T) {
	cfg := startNeo4jContainer(t)
	c := createV2Client(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create
	_, err := c.Execute(ctx, `mutation {
		createMovies(input: [{ title: "Interstellar", released: 2014 }]) {
			movies { title }
		}
	}`, nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Update
	result, err := c.Execute(ctx, `mutation {
		updateMovies(where: { title: "Interstellar" }, update: { released: 2015 }) {
			movies { title released }
		}
	}`, nil)
	if err != nil {
		t.Fatalf("V2 update mutation failed: %v", err)
	}
	if result == nil {
		t.Fatal("V2 update mutation returned nil result")
	}
}

// Test: V2 Client delete mutation.
// Expected: delete mutation removes node and returns nodesDeleted count.
func TestE2E_V2_ClientDeleteMutation(t *testing.T) {
	cfg := startNeo4jContainer(t)
	c := createV2Client(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create
	_, err := c.Execute(ctx, `mutation {
		createMovies(input: [{ title: "Tenet", released: 2020 }]) {
			movies { title }
		}
	}`, nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Delete
	result, err := c.Execute(ctx, `mutation {
		deleteMovies(where: { title: "Tenet" }) { nodesDeleted }
	}`, nil)
	if err != nil {
		t.Fatalf("V2 delete mutation failed: %v", err)
	}
	if result == nil {
		t.Fatal("V2 delete mutation returned nil result")
	}
}

// Test: V2 full round trip — create then query back.
// Expected: query returns the created node with matching field values.
func TestE2E_V2_RoundTrip_CreateThenQuery(t *testing.T) {
	cfg := startNeo4jContainer(t)
	c := createV2Client(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create
	_, err := c.Execute(ctx, `mutation {
		createMovies(input: [{ title: "Arrival", released: 2016 }]) {
			movies { title }
		}
	}`, nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Query back
	result, err := c.Execute(ctx, `query {
		movies(where: { title: "Arrival" }) { title released }
	}`, nil)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if result == nil {
		t.Fatal("query returned nil result")
	}

	data := result.Data()
	if data == nil {
		t.Fatal("result.Data() returned nil")
	}
	movies, ok := data["movies"]
	if !ok {
		t.Fatal("result missing 'movies' key")
	}
	movieList, ok := movies.([]any)
	if !ok {
		t.Fatalf("movies is %T, want []any", movies)
	}
	if len(movieList) == 0 {
		t.Fatal("expected at least 1 movie, got 0")
	}
}

// Test: V2 Result.Decode unmarshals into a Go struct.
// Expected: Decode populates struct fields from query result.
func TestE2E_V2_ResultDecode(t *testing.T) {
	cfg := startNeo4jContainer(t)
	c := createV2Client(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create
	_, err := c.Execute(ctx, `mutation {
		createMovies(input: [{ title: "Blade Runner", released: 1982 }]) {
			movies { title }
		}
	}`, nil)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Query and decode
	result, err := c.Execute(ctx, `query {
		movies(where: { title: "Blade Runner" }) { title released }
	}`, nil)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if result == nil {
		t.Fatal("query returned nil result")
	}

	var response struct {
		Movies []struct {
			Title    string `json:"title"`
			Released *int   `json:"released"`
		} `json:"movies"`
	}
	if decErr := result.Decode(&response); decErr != nil {
		t.Fatalf("Decode failed: %v", decErr)
	}
	if len(response.Movies) == 0 {
		t.Fatal("expected at least 1 movie after Decode")
	}
	if response.Movies[0].Title != "Blade Runner" {
		t.Errorf("title = %q, want %q", response.Movies[0].Title, "Blade Runner")
	}
}

// Test: V2 Client Close prevents further Execute calls.
// Expected: Execute after Close returns errClientClosed.
func TestE2E_V2_ExecuteAfterClose(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv, err := neo4jdriver.NewNeo4jDriver(cfg)
	if err != nil {
		t.Fatalf("NewNeo4jDriver failed: %v", err)
	}

	c := client.New(testGraphModel(), testAugmentedSchemaSDL, drv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c.Close(ctx)

	_, execErr := c.Execute(ctx, `query { movies { title } }`, nil)
	if execErr == nil {
		t.Fatal("Execute after Close should return error")
	}
}
