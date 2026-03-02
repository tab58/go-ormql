package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tab58/go-ormql/pkg/driver"
	"github.com/tab58/go-ormql/pkg/driver/neo4j"
	"github.com/tab58/go-ormql/cmd/example/generated"
)

func main() {
	ctx := context.Background()
	drv, err := neo4j.NewNeo4jDriver(driver.Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "password",
	})
	if err != nil {
		log.Fatalf("failed to create driver: %v", err)
	}
	defer drv.Close(ctx)

	c := generated.NewClient(drv)
	defer c.Close(ctx)

	result, err := c.Execute(ctx, `
		mutation {
			createMovies(input: [{ title: "The Matrix" }]) {
				movies { title }
			}
		}
	`, nil)
	if err != nil {
		log.Fatalf("failed to execute query: %v", err)
	}
	fmt.Println(result)
}
