# gql-orm

A Go code generator that bridges GraphQL and graph databases. Write an annotated `.graphql` schema, run `gql-orm generate`, and get a type-safe Go client that translates GraphQL operations into optimized Cypher queries — one database round-trip per operation, regardless of nesting depth.

Built for Go developers working with Neo4j (or any Cypher-compatible database). Inspired by [`@neo4j/graphql`](https://neo4j.com/docs/graphql/current/) but using compile-time code generation instead of runtime schema construction.

## Features

- **Single-roundtrip execution** — every GraphQL query or mutation becomes exactly one Cypher query using CALL subqueries and map projections
- **Schema-first** — the `.graphql` schema with directives is the single source of truth
- **Full CRUD** — auto-generated queries and mutations for all node types
- **Advanced filtering** — 14 operators (comparison, string, list, null, regex) with AND/OR/NOT boolean composition
- **Multi-field sorting** — ASC/DESC on any scalar field
- **Relay connections** — cursor-based pagination at root level and on relationships, with `totalCount` and `pageInfo`
- **Nested mutations** — create, connect, disconnect, update, and delete related nodes in a single operation
- **Relationship properties** — first-class support for typed edge data
- **`@cypher` directive** — custom Cypher for computed fields
- **GraphQL variables** — full support for parameterized queries and mutations
- **Type-safe results** — `Result.Decode()` unmarshals into generated Go structs
- **Debug logging** — optional `log/slog` integration for GraphQL and Cypher query visibility
- **Driver-agnostic** — abstract driver interface with Neo4j as the first implementation

## Requirements

- Go 1.24+
- Neo4j 4.1+ (required for CALL subqueries)

## Installation

```bash
go install github.com/tab58/gql-orm/cmd/gql-orm@latest
```

Or build from source:

```bash
go build ./cmd/gql-orm/...
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
gql-orm generate \
  --schema schema.graphql \
  --output ./generated \
  --package generated
```

This produces 4 files:

| File | Purpose |
|------|---------|
| `schema.graphql` | Augmented schema with auto-generated CRUD operations, filter inputs, sort inputs, connection types, and nested mutation input types |
| `models_gen.go` | Go structs for all GraphQL types — nodes (with relationship fields), inputs, enums, connections, and response types |
| `graphmodel_gen.go` | Serialized graph model and augmented schema SDL embedded as Go code (no schema files needed at runtime) |
| `client_gen.go` | `NewClient(drv, opts...)` constructor wiring the embedded model and schema to the client |

### 3. Use the client

```go
package main

import (
    "context"
    "fmt"
    "log"

    "your/project/generated"
    "github.com/tab58/gql-orm/pkg/driver"
    "github.com/tab58/gql-orm/pkg/driver/neo4j"
)

func main() {
    ctx := context.Background()

    drv, err := neo4j.NewNeo4jDriver(driver.Config{
        URI:      "bolt://localhost:7687",
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
- `type` (required) — the Neo4j relationship type (e.g., `"ACTED_IN"`)
- `direction` (required) — `IN` or `OUT`
- `properties` (optional) — name of a `@relationshipProperties` type for edge data

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

## Auto-Generated API

For each `@node` type (e.g., `Movie`), the generator creates:

**Queries:**
- `movies(where: MovieWhere, sort: [MovieSort!]): [Movie!]!`
- `moviesConnection(first: Int, after: String, where: MovieWhere, sort: [MovieSort!]): MoviesConnection!`

**Mutations:**
- `createMovies(input: [MovieCreateInput!]!): CreateMoviesMutationResponse!`
- `updateMovies(where: MovieWhere, update: MovieUpdateInput): UpdateMoviesMutationResponse!`
- `deleteMovies(where: MovieWhere): DeleteInfo!`

**Input types:** `MovieCreateInput`, `MovieUpdateInput`, `MovieWhere`, `MovieSort`, and nested mutation input types for each relationship.

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
    query($search: String!, $minYear: Int, $limit: Int!) {
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
    "limit":   20,
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

## Debug Logging

Enable debug logging with `log/slog` to see both the GraphQL query and the generated Cypher:

```go
import (
    "log/slog"
    "os"
    "github.com/tab58/gql-orm/pkg/client"
)

logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

// Pass logger to both driver and client
drv, _ := neo4j.NewNeo4jDriver(driver.Config{
    URI:      "bolt://localhost:7687",
    Username: "neo4j",
    Password: "password",
    Logger:   logger,
})

c := generated.NewClient(drv, client.WithLogger(logger))
```

This logs `graphql.execute` (with query and variables) at the client level and `cypher.execute` (with the Cypher query and parameters) at the driver level. When no logger is set, there is zero overhead.

## Architecture

```
schema.graphql (user input)
        │
        ▼
  gql-orm generate           Build time
  ┌─────────────────┐
  │ 1. Parse schema │
  │ 2. Augment CRUD │
  │ 3. Gen models   │
  │ 4. Gen registry │
  │ 5. Gen client   │
  └────────┬────────┘
           │
           ▼
  generated/ (4 Go files)
           │
           │                  Runtime
           ▼
  client.Execute(ctx, query, vars)
           │
           ▼
  gqlparser: parse + validate
           │
           ▼
  translator: GraphQL AST → single Cypher
           │
           ▼
  driver: one database round-trip
           │
           ▼
  Result.Decode(&resp)
```

Every GraphQL operation — no matter how deeply nested — translates into exactly **one Cypher query** using CALL subqueries, `collect()`, and map projections. A query for 100 movies with their actors and connection counts is still one database call.

## Package Structure

```
cmd/
  gql-orm/          CLI entry point (generate subcommand)
pkg/
  schema/           GraphQL schema parsing, directive extraction, GraphModel
  cypher/           Cypher statement builder library (public utility API)
  translate/        GraphQL-to-Cypher translator (CALL subqueries, map projections)
  driver/           Abstract Cypher driver interface
    neo4j/          Neo4j driver implementation
  codegen/          Code generation pipeline
  client/           Programmatic GraphQL client (translator + validation)
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

Nullable GraphQL types map to Go pointer types. Enum types map to `string`.

## License

MIT
