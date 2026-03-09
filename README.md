# gormql

A Go code generator that bridges GraphQL and graph databases. Write an annotated `.graphql` schema, run `gormql generate`, and get a type-safe Go client that translates GraphQL operations into optimized Cypher queries — one database round-trip per operation, regardless of nesting depth.

Built for Go developers working with Neo4j, FalkorDB, or any Cypher-compatible database. Inspired by [`@neo4j/graphql`](https://neo4j.com/docs/graphql/current/) but using compile-time code generation instead of runtime schema construction.

## Features

- **Single-roundtrip execution** — every GraphQL query or mutation becomes exactly one Cypher query using CALL subqueries and map projections
- **Schema-first** — the `.graphql` schema with directives is the single source of truth
- **Full CRUD** — auto-generated queries and mutations for all node types
- **Merge (upsert)** — atomic create-or-update via Cypher `MERGE` with `ON CREATE SET` / `ON MATCH SET`
- **Top-level connect** — standalone mutations to create relationships between existing nodes
- **Advanced filtering** — 14 operators (comparison, string, list, null, regex) with AND/OR/NOT boolean composition
- **Relationship filters** — filter nodes by related node properties (e.g., find actors who acted in a specific movie)
- **Multi-field sorting** — ASC/DESC on any scalar field
- **Relay connections** — cursor-based pagination at root level and on relationships, with `totalCount` and `pageInfo`
- **Nested mutations** — create, connect, disconnect, update, and delete related nodes in a single operation
- **Relationship properties** — first-class support for typed edge data
- **`@cypher` directive** — custom Cypher for computed fields
- **`@vector` directive** — similarity search with automatic index DDL generation
- **GraphQL variables** — full support for parameterized queries and mutations
- **Type-safe results** — `Result.Decode()` unmarshals into generated Go structs
- **Debug logging** — optional `log/slog` integration for GraphQL and Cypher query visibility
- **Auto-chunking** — bulk mutations automatically split into batches (default 50, configurable via `WithBatchSize`) with transparent result aggregation
- **Multi-database** — Neo4j and FalkorDB drivers included, with an abstract interface for others

## Requirements

- Go 1.24+
- **Neo4j** 4.1+ (required for CALL subqueries), 5.11+ for `@vector` similarity queries
- **FalkorDB** 4.0+ (required for CALL subqueries), 4.2+ for `@vector` similarity queries

## Installation

```bash
go install github.com/tab58/go-ormql/cmd/gormql@latest
```

Or build from source:

```bash
go build ./cmd/gormql/...
```

## Quick Start

### 1. Define your schema

Create a `schema.graphql` file:

```graphql
type Movie @node {
  id: ID!
  title: String!
  released: Int
}

type Actor @node {
  id: ID!
  name: String!
  movies: [Movie!]!
    @relationship(
      type: "ACTED_IN"
      direction: OUT
      properties: "ActedInProperties"
    )
}

type ActedInProperties @relationshipProperties {
  role: String!
}
```

### 2. Generate code

```bash
gormql generate \
  --schema schema.graphql \
  --output ./generated \
  --package generated
```

For FalkorDB, add `--target falkordb`:

```bash
gormql generate \
  --schema schema.graphql \
  --output ./generated \
  --package generated \
  --target falkordb
```

This produces 4 files (5 when `@vector` is present):

| File | Purpose |
|------|---------|
| `schema.graphql` | Augmented schema with auto-generated CRUD operations, filter inputs, sort inputs, connection types, and nested mutation input types |
| `models_gen.go` | Go structs for all GraphQL types — nodes (with relationship fields), inputs, enums, connections, and response types |
| `graphmodel_gen.go` | Serialized graph model and augmented schema SDL embedded as Go code (no schema files needed at runtime) |
| `client_gen.go` | `NewClient(drv, opts...)` constructor wiring the embedded model and schema to the client |
| `indexes_gen.go` | `CreateIndexes(ctx, drv)` function with vector index DDL (only generated when `@vector` is present) |

### 3. Use the client

```go
package main

import (
    "context"
    "fmt"
    "log"

    "your/project/generated"
    "github.com/tab58/go-ormql/pkg/driver"
    "github.com/tab58/go-ormql/pkg/driver/neo4j"
)

func main() {
    ctx := context.Background()

    drv, err := neo4j.NewNeo4jDriver(driver.Config{
        Host:     "localhost",
        Port:     7687,
        Scheme:   "bolt",
        Username: "neo4j",
        Password: "password",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer drv.Close(ctx)

    c := generated.NewClient(drv)
    defer c.Close(ctx)

    // Create a movie
    result, err := c.Execute(ctx, `
        mutation {
            createMovies(input: [{ title: "The Matrix", released: 1999 }]) {
                movies { id title released }
            }
        }
    `, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Type-safe decode into generated models
    var resp struct {
        CreateMovies generated.CreateMoviesMutationResponse `json:"createMovies"`
    }
    if err := result.Decode(&resp); err != nil {
        log.Fatal(err)
    }
    fmt.Println(resp.CreateMovies.Movies[0].Title) // "The Matrix"
}
```

## Schema Directives

### `@node`

Marks a GraphQL type as a graph database node. Every `@node` type gets auto-generated CRUD queries and mutations.

```graphql
type Movie @node {
  id: ID!
  title: String!
  released: Int
}
```

### `@relationship`

Defines an edge between two nodes. Placed on a field within a `@node` type.

```graphql
type Actor @node {
  id: ID!
  name: String!
  movies: [Movie!]!
    @relationship(type: "ACTED_IN", direction: OUT, properties: "ActedInProperties")
}
```

Arguments:
- `type` (required) — the relationship type in the database (e.g., `"ACTED_IN"`)
- `direction` (required) — `IN` or `OUT`, relative to the declaring node (see below)
- `properties` (optional) — name of a `@relationshipProperties` type for edge data

#### Understanding `direction`

The `direction` argument describes which way the relationship arrow points **from the perspective of the node type that declares the field**:

- **`OUT`** — the arrow points **away from** the declaring node: `(DeclaringNode)-[:REL]->(TargetNode)`
- **`IN`** — the arrow points **into** the declaring node: `(DeclaringNode)<-[:REL]-(TargetNode)`

For example, given the relationship `(Actor)-[:ACTED_IN]->(Movie)`, you can declare the same edge from either side:

```graphql
# From Actor's perspective: the arrow goes OUT of Actor toward Movie
type Actor @node {
  movies: [Movie!]! @relationship(type: "ACTED_IN", direction: OUT)
}

# From Movie's perspective: the arrow comes IN to Movie from Actor
type Movie @node {
  actors: [Actor!]! @relationship(type: "ACTED_IN", direction: IN)
}
```

Both declarations describe the same physical edge. The generated Cypher uses the direction to place the arrow correctly:

| Declaration | Generated Cypher pattern |
|-------------|--------------------------|
| `direction: OUT` on `Actor` | `(actor)-[:ACTED_IN]->(movie)` |
| `direction: IN` on `Movie` | `(movie)<-[:ACTED_IN]-(actor)` |

You can declare the relationship on one side or both — declaring on both sides lets you traverse and mutate the relationship from either node type.

### `@relationshipProperties`

Attaches typed properties to a relationship edge.

```graphql
type ActedInProperties @relationshipProperties {
  role: String!
  year: Int
}
```

### `@cypher`

Defines a computed field backed by a custom Cypher statement. The parent node is available as `this`.

```graphql
type Movie @node {
  id: ID!
  title: String!
  averageRating: Float
    @cypher(statement: "MATCH (this)<-[r:REVIEWED]-() RETURN avg(r.score)")
  similarMovies(limit: Int! = 3): [Movie!]!
    @cypher(statement: """
      MATCH (this)-[:IN_GENRE]->()<-[:IN_GENRE]-(rec)
      WHERE rec <> this
      RETURN rec LIMIT $limit
    """)
}
```

- `@cypher` fields are read-only — excluded from create/update inputs
- Field arguments become `$paramName` parameters in the Cypher statement
- Scalar return types are automatically limited to one result; list types return all results

### `@vector`

Enables similarity search on a node field backed by a vector index. The field must be of type `[Float!]!`. At most one `@vector` field per `@node` type.

```graphql
type Movie @node {
  id: ID!
  title: String!
  embedding: [Float!]! @vector(indexName: "movie_embedding", dimensions: 256, similarity: "cosine")
}
```

Arguments:
- `indexName` (required) — the vector index name
- `dimensions` (required) — vector dimensionality
- `similarity` (required) — similarity function (`"cosine"`, `"euclidean"`)

When `@vector` is present, the generator produces:
- A `moviesSimilar(embedding: [Float!]!, first: Int): [MovieSimilarResult!]!` query
- A `MovieSimilarResult` type with a `score` field and the node fields
- An `indexes_gen.go` file with `CreateIndexes(ctx, drv)` that runs the appropriate `CREATE VECTOR INDEX` DDL for your target database

```go
// Create vector indexes before first use
if err := generated.CreateIndexes(ctx, drv); err != nil {
    log.Fatal(err)
}

// Search for similar movies by embedding vector
result, _ := c.Execute(ctx, `
    query {
        moviesSimilar(embedding: [0.1, 0.2, ...], first: 10) {
            score
            title
        }
    }
`, nil)
```

## Auto-Generated API

For each `@node` type (e.g., `Movie`), the generator creates:

**Queries:**
- `movies(where: MovieWhere, sort: [MovieSort!]): [Movie!]!`
- `moviesConnection(first: Int, after: String, where: MovieWhere, sort: [MovieSort!]): MoviesConnection!`

**Mutations:**
- `createMovies(input: [MovieCreateInput!]!): CreateMoviesMutationResponse!`
- `updateMovies(where: MovieWhere, update: MovieUpdateInput): UpdateMoviesMutationResponse!`
- `deleteMovies(where: MovieWhere): DeleteInfo!`
- `mergeMovies(input: [MovieMergeInput!]!): MergeMoviesMutationResponse!`

**Per-relationship connect mutations** (e.g., for an `Actor` with a `movies` relationship field):
- `connectActorMovies(input: [ConnectActorMoviesInput!]!): ConnectInfo!`

**Input types:** `MovieCreateInput`, `MovieUpdateInput`, `MovieWhere`, `MovieSort`, `MovieMatchInput`, `MovieMergeInput`, and nested mutation input types for each relationship.

**Connection types:** `MoviesConnection`, `MovieEdge`, `PageInfo`, and relationship-level connection types (e.g., `MovieActorsConnection`, `MovieActorsEdge`).

## Queries

### Basic query

```go
result, _ := c.Execute(ctx, `
    query {
        movies { id title released }
    }
`, nil)
```

### Filtering

Filter with `where` using any combination of operators:

```go
result, _ := c.Execute(ctx, `
    query {
        movies(where: {
            title_contains: "Matrix"
            released_gte: 1990
        }) {
            title released
        }
    }
`, nil)
```

**Available filter operators:**

| Suffix | Operator | Applicable Types |
|--------|----------|------------------|
| *(none)* | equals | all |
| `_not` | not equals | all |
| `_gt` | greater than | Int, Float, String |
| `_gte` | greater than or equal | Int, Float, String |
| `_lt` | less than | Int, Float, String |
| `_lte` | less than or equal | Int, Float, String |
| `_contains` | string contains | String |
| `_startsWith` | string starts with | String |
| `_endsWith` | string ends with | String |
| `_regex` | regex match | String |
| `_in` | in list | all |
| `_nin` | not in list | all |
| `_isNull` | is null / is not null | all nullable |

**Boolean composition** with `AND`, `OR`, `NOT`:

```go
result, _ := c.Execute(ctx, `
    query {
        movies(where: {
            OR: [
                { title: "The Matrix" }
                { title: "Inception" }
            ]
            released_gte: 1990
            NOT: { title_contains: "Reloaded" }
        }) {
            title
        }
    }
`, nil)
```

### Relationship Filters

Filter nodes by properties of their related nodes. For to-many relationships, use the `_some` suffix. For to-one relationships, use the field name directly.

```go
// Find actors who acted in a movie titled "The Matrix"
result, _ := c.Execute(ctx, `
    query {
        actors(where: {
            movies_some: { title: "The Matrix" }
        }) {
            name
        }
    }
`, nil)
```

Relationship filters support the full set of scalar operators and can be nested:

```go
// Find movies where at least one actor's name starts with "Keanu"
result, _ := c.Execute(ctx, `
    query {
        movies(where: {
            actors_some: { name_startsWith: "Keanu" }
        }) {
            title
        }
    }
`, nil)
```

### Sorting

Sort results with the `sort` argument:

```go
result, _ := c.Execute(ctx, `
    query {
        movies(sort: [{ released: DESC }, { title: ASC }]) {
            title released
        }
    }
`, nil)
```

### Relay Connections

Cursor-based pagination at root level:

```go
result, _ := c.Execute(ctx, `
    query {
        moviesConnection(first: 10, after: "Y3Vyc29yOjU=", where: { released_gte: 2000 }) {
            edges {
                node { title released }
                cursor
            }
            totalCount
            pageInfo { hasNextPage hasPreviousPage startCursor endCursor }
        }
    }
`, nil)
```

### Nested Relationships

Query related nodes — all resolved in a single database call:

```go
result, _ := c.Execute(ctx, `
    query {
        movies {
            title
            actors { name }
        }
    }
`, nil)
```

### Relationship Connections

Paginate over relationships with optional edge properties:

```go
result, _ := c.Execute(ctx, `
    query {
        movies {
            title
            actorsConnection(first: 5, sort: [{ name: ASC }]) {
                edges {
                    node { name }
                    properties { role }
                }
                totalCount
                pageInfo { hasNextPage }
            }
        }
    }
`, nil)
```

### GraphQL Variables

All argument positions support variables — filters, pagination, sort, mutation inputs, and `@cypher` field arguments:

```go
result, _ := c.Execute(ctx, `
    query($search: String!, $minYear: Int) {
        movies(
            where: { title_contains: $search, released_gte: $minYear }
            sort: [{ released: DESC }]
        ) {
            title released
        }
    }
`, map[string]any{
    "search":  "Matrix",
    "minYear": 1999,
})
```

## Mutations

### Create

```go
result, _ := c.Execute(ctx, `
    mutation {
        createMovies(input: [
            { title: "The Matrix", released: 1999 }
            { title: "Inception", released: 2010 }
        ]) {
            movies { id title }
        }
    }
`, nil)
```

Node IDs with type `ID!` are auto-generated using `randomUUID()`.

### Update

```go
result, _ := c.Execute(ctx, `
    mutation {
        updateMovies(
            where: { title: "The Matrix" }
            update: { released: 1999 }
        ) {
            movies { id title released }
        }
    }
`, nil)
```

### Delete

```go
result, _ := c.Execute(ctx, `
    mutation {
        deleteMovies(where: { title: "Old Movie" }) {
            nodesDeleted
        }
    }
`, nil)
```

### Merge (Upsert)

Merge mutations use Cypher `MERGE` to atomically create-or-update nodes. The `match` fields identify the node; `onCreate` sets values only on creation; `onMatch` updates values only when the node already exists.

```go
result, _ := c.Execute(ctx, `
    mutation {
        mergeMovies(input: [
            {
                match: { title: "The Matrix" }
                onCreate: { title: "The Matrix", released: 1999 }
                onMatch: { released: 1999 }
            }
        ]) {
            movies { id title released }
        }
    }
`, nil)
```

- `match` — the identity fields used to find or create the node (all non-ID, non-vector scalar fields)
- `onCreate` — set these fields only when creating a new node (IDs are auto-generated with `randomUUID()`)
- `onMatch` — update these fields only when the node already exists (uses `COALESCE` to preserve existing values when a field is null)
- Batched with `UNWIND` — pass multiple items in the input array for efficient bulk upserts

### Top-Level Connect

Create relationships between existing nodes without modifying the nodes themselves. The mutation name follows the pattern `connect{SourceType}{FieldName}`:

```go
// Connect existing actors to existing movies
result, _ := c.Execute(ctx, `
    mutation {
        connectActorMovies(input: [
            {
                from: { name: "Keanu Reeves" }
                to: { title: "The Matrix" }
                edge: { role: "Neo" }
            }
        ]) {
            relationshipsCreated
        }
    }
`, nil)
```

- `from` — WHERE filter to match the source node
- `to` — WHERE filter to match the target node
- `edge` — optional relationship properties (only when `@relationshipProperties` is defined)
- Uses `MERGE` — safe to call multiple times without creating duplicate relationships
- Batched with `UNWIND` — pass multiple items for bulk relationship creation

### Nested Mutations

Create, connect, disconnect, update, and delete related nodes in a single atomic operation:

```go
// Create a movie with new actors and connect to existing ones
result, _ := c.Execute(ctx, `
    mutation {
        createMovies(input: [{
            title: "New Movie"
            actors: {
                create: [
                    { node: { name: "New Actor" }, edge: { role: "Lead" } }
                ]
                connect: [
                    { where: { name: "Keanu Reeves" }, edge: { role: "Support" } }
                ]
            }
        }]) {
            movies { id title actors { name } }
        }
    }
`, nil)
```

```go
// Update a movie: disconnect, update, and delete related actors
result, _ := c.Execute(ctx, `
    mutation {
        updateMovies(
            where: { id: "movie-1" }
            update: {
                title: "Updated Title"
                actors: {
                    disconnect: [{ where: { name: "Old Actor" } }]
                    update: [{
                        where: { name: "Keanu" }
                        node: { name: "Keanu Reeves" }
                        edge: { role: "Neo" }
                    }]
                    delete: [{ where: { name: "Remove Me" } }]
                }
            }
        ) {
            movies { id title }
        }
    }
`, nil)
```

- **create** — creates a new related node and relationship
- **connect** — matches an existing node and creates a relationship
- **disconnect** — removes the relationship (keeps both nodes)
- **update** — updates the related node and/or edge properties
- **delete** — removes the related node and all its relationships

## Result Handling

### Type-safe decode

Use `Result.Decode()` with generated model structs:

```go
var resp struct {
    Movies []generated.Movie `json:"movies"`
}
result.Decode(&resp)

for _, m := range resp.Movies {
    fmt.Println(m.Title, m.Released)
}
```

### Raw map access

Use `Result.Data()` for dynamic access:

```go
data := result.Data()
movies := data["movies"].([]any)
first := movies[0].(map[string]any)
fmt.Println(first["title"])
```

## Database Drivers

### Neo4j

```go
import (
    "github.com/tab58/go-ormql/pkg/driver"
    "github.com/tab58/go-ormql/pkg/driver/neo4j"
)

drv, err := neo4j.NewNeo4jDriver(driver.Config{
    Host:     "localhost",
    Port:     7687,
    Scheme:   "bolt",
    Username: "neo4j",
    Password: "password",
    Database: "neo4j",  // optional, defaults to "neo4j"
})
```

Supported schemes: `bolt`, `bolt+s`, `bolt+ssc`, `neo4j`, `neo4j+s`, `neo4j+ssc`.

### FalkorDB

```go
import (
    "github.com/tab58/go-ormql/pkg/driver"
    "github.com/tab58/go-ormql/pkg/driver/falkordb"
)

drv, err := falkordb.NewFalkorDBDriver(driver.Config{
    Host:     "localhost",
    Port:     6379,
    Scheme:   "redis",
    Username: "default",
    Password: "password",
    Database: "mygraph",  // required: the FalkorDB graph name
})
```

Supported schemes: `redis`, `rediss`.

When using `@vector` with FalkorDB, the generated `indexes_gen.go` exports a `VectorIndexes` variable. Pass it to the driver config for automatic vector query rewriting:

```go
drv, err := falkordb.NewFalkorDBDriver(driver.Config{
    Host:          "localhost",
    Port:          6379,
    Scheme:        "redis",
    Database:      "mygraph",
    VectorIndexes: generated.VectorIndexes,
})
```

### Driver Config

Both drivers use the same `driver.Config` struct:

```go
type Config struct {
    Host     string       // database host (required)
    Port     int          // database port (required)
    Scheme   string       // connection scheme (required, varies by driver)
    Username string       // authentication username
    Password string       // authentication password
    Database string       // database/graph name
    Logger   *slog.Logger // optional debug logging
    VectorIndexes map[string]driver.VectorIndex // FalkorDB vector query rewriting
}
```

## Debug Logging

Enable debug logging with `log/slog` to see both the GraphQL query and the generated Cypher:

```go
import (
    "log/slog"
    "os"
    "github.com/tab58/go-ormql/pkg/client"
)

logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

// Pass logger to both driver and client
drv, _ := neo4j.NewNeo4jDriver(driver.Config{
    Host:     "localhost",
    Port:     7687,
    Scheme:   "bolt",
    Username: "neo4j",
    Password: "password",
    Logger:   logger,
})

c := generated.NewClient(drv, client.WithLogger(logger))
```

This logs `graphql.execute` (with query and variables) at the client level and `cypher.execute` (with the Cypher query and parameters) at the driver level. When no logger is set, there is zero overhead.

## CLI Reference

```
gormql generate [flags]
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--schema` | yes | | Comma-separated `.graphql` schema file paths |
| `--output` | yes | | Output directory for generated code |
| `--package` | no | `generated` | Go package name for generated files |
| `--target` | no | `neo4j` | Target database: `neo4j` or `falkordb` |

The `--target` flag affects index DDL generation. When using `@vector`:
- `neo4j` generates `CREATE VECTOR INDEX IF NOT EXISTS` statements
- `falkordb` generates FalkorDB-specific vector index DDL and exports a `VectorIndexes` variable for driver configuration

## Architecture

```
schema.graphql (user input)
        |
        v
  gormql generate             Build time
  +----------------------+
  | 1. Parse schema      |
  | 2. Augment CRUD      |
  | 3. Gen models        |
  | 4. Gen registry      |
  | 5. Gen client        |
  | 6. Gen indexes       |
  |    (@vector only)    |
  +----------+-----------+
             |
             v
  generated/ (4-5 Go files)
           |
           |                  Runtime
           v
  client.Execute(ctx, query, vars)
           |
           v
  gqlparser: parse + validate
           |
           v
  translator: GraphQL AST -> single Cypher
           |
           v
  driver: one database round-trip
           |
           v
  Result.Decode(&resp)
```

Every GraphQL operation — no matter how deeply nested — translates into exactly **one Cypher query** using CALL subqueries, `collect()`, and map projections. A query for 100 movies with their actors and connection counts is still one database call.

## Package Structure

```
cmd/
  gormql/           CLI entry point (generate subcommand)
pkg/
  schema/           GraphQL schema parsing, directive extraction, GraphModel
  cypher/           Cypher statement builder library (parameterized queries, WhereClause, SortField)
  translate/        GraphQL-to-Cypher translator (AST walker, CALL subqueries, map projections)
  driver/           Abstract Cypher driver interface + Transaction support
    neo4j/          Neo4j driver implementation
    falkordb/       FalkorDB driver implementation
  codegen/          Code generation pipeline (augment, models, registry, client, indexes)
  client/           Programmatic GraphQL client (translator + gqlparser validation, Result with Decode)
  internal/
    strutil/        Shared string utilities
```

## Type Mapping

| GraphQL | Go | Cypher |
|---------|----|--------|
| `String!` | `string` | `STRING` |
| `String` | `*string` | `STRING` |
| `Int!` | `int` | `INTEGER` |
| `Int` | `*int` | `INTEGER` |
| `Float!` | `float64` | `FLOAT` |
| `Float` | `*float64` | `FLOAT` |
| `Boolean!` | `bool` | `BOOLEAN` |
| `Boolean` | `*bool` | `BOOLEAN` |
| `ID!` | `string` | `STRING` |
| `ID` | `*string` | `STRING` |
| `[Float!]!` | `[]float64` | `LIST<FLOAT>` |

Nullable GraphQL types map to Go pointer types. Enum types map to `string`.

## License

MIT
