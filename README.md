# gql-orm

A GraphQL-based ORM for graph databases, such as Neo4j.

## Usage

1. Install the CLI: `go build ./cmd/gql-orm/...`

2. Write a GraphQL schema with custom directives

Create a `schema.graphql` file using three directives:

- `@node` — marks a type as a graph database node
- `@relationship(type, direction)` — defines edges between
  nodes
- `@relationshipProperties` — attaches properties to a
  relationship edge

```
  type Movie @node {
    id: ID!
    title: String!
    released: Int
  }

  type Actor @node {
    id: ID!
    name: String!
    movies: [Movie!]! @relationship(type: "ACTED_IN", direction: OUT)
  }

  type ActedInProperties @relationshipProperties {
    role: String!
  }
```

3. Run code generation

```
  gql-orm generate \
    --schema schema.graphql \
    --output ./generated \
    --package myapp
```

This produces 6 files in ./generated/:

```
  ┌──────────────────┬─────────────────────────────────────┐
  │       File       │               Purpose               │
  ├──────────────────┼─────────────────────────────────────┤
  │                  │ Augmented schema with               │
  │ schema.graphql   │ auto-generated CRUD                 │
  │                  │ queries/mutations, input types, and │
  │                  │  Relay connection types             │
  ├──────────────────┼─────────────────────────────────────┤
  │ gqlgen.yml       │ Config for the gqlgen code          │
  │                  │ generator                           │
  ├──────────────────┼─────────────────────────────────────┤
  │ models_gen.go    │ Go structs for all GraphQL types    │
  │                  │ (via gqlgen)                        │
  ├──────────────────┼─────────────────────────────────────┤
  │ exec_gen.go      │ Resolver interface definitions (via │
  │                  │  gqlgen)                            │
  ├──────────────────┼─────────────────────────────────────┤
  │ resolvers_gen.go │ Resolver implementations that build │
  │                  │  parameterized Cypher queries       │
  ├──────────────────┼─────────────────────────────────────┤
  │ mappers_gen.go   │ Database record to Go model         │
  │                  │ conversion functions                │
  └──────────────────┴─────────────────────────────────────┘
```

4. Start a GraphQL server (optional built-in server)

```
  gql-orm serve \
    --neo4j-uri bolt://localhost:7687 \
    --neo4j-user neo4j \
    --neo4j-password password \
    --port 8080
```

Or use env vars `NEO4J_URI`, `NEO4J_USERNAME`, `NEO4J_PASSWORD`.

What gets auto-generated from the schema

For each @node type, the generator creates:

- Query: `movies(where: MovieWhere): [Movie!]!` and
  `moviesConnection(...)` for Relay pagination
- Mutations: `createMovies`, `updateMovies`, `deleteMovies`
- Input types: `MovieWhere`, `MovieCreateInput`,
  `MovieUpdateInput`
- Resolvers: Go functions that build parameterized Cypher
  queries `(MATCH (n:Movie) WHERE ... RETURN n)` via the
  `pkg/cypher/` builder
- Mappers: Functions to convert Neo4j records into Go
  structs

The key design is schema-first — the GraphQL SDL with
`@node`/`@relationship` directives is the single source of
truth, and everything else (CRUD API, Cypher queries, Go
types, resolvers) is derived from it.
