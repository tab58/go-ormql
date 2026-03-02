package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/tab58/gql-orm/pkg/codegen"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run parses CLI args and dispatches to subcommands.
// Supported subcommands: generate.
func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("subcommand required: generate")
	}
	switch args[0] {
	case "generate":
		return runGenerate(args[1:])
	default:
		return fmt.Errorf("unknown subcommand: %s", args[0])
	}
}

// runGenerate handles the "generate" subcommand.
// Required flags: --schema (comma-separated .graphql files), --output (output directory).
// Optional flags: --package (Go package name, default "generated").
func runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	schemaFlag := fs.String("schema", "", "comma-separated .graphql schema files")
	outputFlag := fs.String("output", "", "output directory for generated code")
	packageFlag := fs.String("package", "generated", "Go package name for generated code")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *schemaFlag == "" {
		return fmt.Errorf("--schema is required: provide comma-separated .graphql file paths")
	}
	if *outputFlag == "" {
		return fmt.Errorf("--output is required: provide output directory path")
	}

	schemaFiles := strings.Split(*schemaFlag, ",")

	return codegen.Generate(codegen.Config{
		SchemaFiles: schemaFiles,
		OutputDir:   *outputFlag,
		PackageName: *packageFlag,
		Stderr:      os.Stderr,
	})
}
