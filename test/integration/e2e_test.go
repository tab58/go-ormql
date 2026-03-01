//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tab58/gql-orm/pkg/client"
	"github.com/tab58/gql-orm/pkg/codegen"
	"github.com/tab58/gql-orm/pkg/driver"
	neo4jdriver "github.com/tab58/gql-orm/pkg/driver/neo4j"
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

// === INT-2: End-to-end client integration tests ===

// TestE2E_PipelineProducesAllFiles verifies that the codegen pipeline
// produces all expected generated files, including client_gen.go.
// Expected: schema.graphql, gqlgen.yml, resolvers_gen.go, mappers_gen.go,
// and client_gen.go all exist in the output directory.
func TestE2E_PipelineProducesAllFiles(t *testing.T) {
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
		"gqlgen.yml",
		"resolvers_gen.go",
		"mappers_gen.go",
		"client_gen.go",
	}
	for _, name := range expectedFiles {
		path := filepath.Join(outputDir, name)
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			t.Errorf("pipeline did not produce %s", name)
		}
	}
}

// TestE2E_DriverConnectsToNeo4j verifies that NewNeo4jDriver can connect
// to a real Neo4j container instance.
// Expected: non-nil driver, nil error.
func TestE2E_DriverConnectsToNeo4j(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv, err := neo4jdriver.NewNeo4jDriver(cfg)
	if err != nil {
		t.Fatalf("NewNeo4jDriver should connect: %v", err)
	}
	if drv == nil {
		t.Fatal("NewNeo4jDriver returned nil driver")
	}
	defer drv.Close(context.Background())
}

// TestE2E_ClientExecuteQuery verifies that a Client can execute a simple
// GraphQL query against a real Neo4j database and return results.
// Expected: Execute returns non-nil map[string]any with query results.
func TestE2E_ClientExecuteQuery(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv, err := neo4jdriver.NewNeo4jDriver(cfg)
	if err != nil {
		t.Fatalf("NewNeo4jDriver failed: %v", err)
	}
	defer drv.Close(context.Background())

	// Create client with nil schema for now — will use generated schema once pipeline is wired
	c := client.New(nil, drv)
	defer c.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := c.Execute(ctx, `query { movies { title } }`, nil)
	if err != nil {
		t.Fatalf("client.Execute query failed: %v", err)
	}
	if result == nil {
		t.Fatal("client.Execute returned nil result")
	}
}

// TestE2E_ClientCreateMutation verifies that a Client can execute a create
// mutation and the created data is returned.
// Expected: mutation returns non-nil result with created movie data.
func TestE2E_ClientCreateMutation(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv, err := neo4jdriver.NewNeo4jDriver(cfg)
	if err != nil {
		t.Fatalf("NewNeo4jDriver failed: %v", err)
	}
	defer drv.Close(context.Background())

	c := client.New(nil, drv)
	defer c.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := c.Execute(ctx, `mutation {
		createMovies(input: [{ title: "The Matrix", released: 1999 }]) {
			movies { title released }
		}
	}`, nil)
	if err != nil {
		t.Fatalf("client.Execute create mutation failed: %v", err)
	}
	if result == nil {
		t.Fatal("create mutation returned nil result")
	}
}

// TestE2E_ClientNestedCreateMutation verifies that a Client can execute a
// nested create mutation (creating a node + related node + relationship).
// Expected: mutation succeeds and returns the created data.
func TestE2E_ClientNestedCreateMutation(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv, err := neo4jdriver.NewNeo4jDriver(cfg)
	if err != nil {
		t.Fatalf("NewNeo4jDriver failed: %v", err)
	}
	defer drv.Close(context.Background())

	c := client.New(nil, drv)
	defer c.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	vars := map[string]any{
		"input": []map[string]any{
			{
				"name": "Keanu Reeves",
				"movies": map[string]any{
					"create": []map[string]any{
						{
							"node": map[string]any{
								"title":    "The Matrix",
								"released": 1999,
							},
							"edge": map[string]any{
								"role": "Neo",
							},
						},
					},
				},
			},
		},
	}
	result, err := c.Execute(ctx, `mutation CreateActorWithMovie($input: [ActorCreateInput!]!) {
		createActors(input: $input) {
			actors { name movies { title } }
		}
	}`, vars)
	if err != nil {
		t.Fatalf("nested create mutation failed: %v", err)
	}
	if result == nil {
		t.Fatal("nested create mutation returned nil result")
	}
}

// TestE2E_ClientNestedConnectMutation verifies that a Client can execute a
// nested connect mutation (connecting an existing node to another via relationship).
// Expected: mutation succeeds and returns the connected data.
func TestE2E_ClientNestedConnectMutation(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv, err := neo4jdriver.NewNeo4jDriver(cfg)
	if err != nil {
		t.Fatalf("NewNeo4jDriver failed: %v", err)
	}
	defer drv.Close(context.Background())

	c := client.New(nil, drv)
	defer c.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First create a movie
	_, err = c.Execute(ctx, `mutation {
		createMovies(input: [{ title: "John Wick", released: 2014 }]) {
			movies { title }
		}
	}`, nil)
	if err != nil {
		t.Fatalf("create movie failed: %v", err)
	}

	// Then create an actor connected to the existing movie
	vars := map[string]any{
		"input": []map[string]any{
			{
				"name": "Keanu Reeves",
				"movies": map[string]any{
					"connect": []map[string]any{
						{
							"where": map[string]any{"title": "John Wick"},
							"edge":  map[string]any{"role": "John Wick"},
						},
					},
				},
			},
		},
	}
	result, err := c.Execute(ctx, `mutation ConnectActorToMovie($input: [ActorCreateInput!]!) {
		createActors(input: $input) {
			actors { name }
		}
	}`, vars)
	if err != nil {
		t.Fatalf("nested connect mutation failed: %v", err)
	}
	if result == nil {
		t.Fatal("nested connect mutation returned nil result")
	}
}

// TestE2E_RoundTrip_CreateThenQuery verifies the full round trip:
// create a node, then query it back and verify the result shape.
// Expected: query returns the created node with matching field values.
func TestE2E_RoundTrip_CreateThenQuery(t *testing.T) {
	cfg := startNeo4jContainer(t)
	drv, err := neo4jdriver.NewNeo4jDriver(cfg)
	if err != nil {
		t.Fatalf("NewNeo4jDriver failed: %v", err)
	}
	defer drv.Close(context.Background())

	c := client.New(nil, drv)
	defer c.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create
	_, err = c.Execute(ctx, `mutation {
		createMovies(input: [{ title: "Arrival", released: 2016 }]) {
			movies { title }
		}
	}`, nil)
	if err != nil {
		t.Fatalf("create mutation failed: %v", err)
	}

	// Query
	result, err := c.Execute(ctx, `query {
		movies(where: { title: "Arrival" }) { title released }
	}`, nil)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if result == nil {
		t.Fatal("query returned nil result")
	}

	// Verify shape
	movies, ok := result["movies"]
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
