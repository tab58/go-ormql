package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tab58/gql-orm/pkg/schema"
)

// Config controls the code generation pipeline.
type Config struct {
	SchemaFiles []string // paths to .graphql schema files
	OutputDir   string   // directory for all generated output
	PackageName string   // Go package name for generated code
}

// Generate runs the full code generation pipeline in sequence:
//  1. ParseSchema(cfg.SchemaFiles) → GraphModel
//  2. AugmentSchema(model) → write <OutputDir>/schema.graphql
//  3. GenerateGqlgenConfig → write <OutputDir>/gqlgen.yml
//  4. Clean stale generated Go files and scaffold from previous runs
//  5. InvokeGqlgen → generates Go model types + resolver interfaces
//  6. deleteResolverScaffold → removes freshly generated scaffold (resolver.go, *.resolvers.go)
//  7. GenerateResolvers(model) → write <OutputDir>/resolvers_gen.go
//  8. GenerateMappers(model) → write <OutputDir>/mappers_gen.go
//  9. GenerateClient(model) → write <OutputDir>/client_gen.go
//
// All output goes to cfg.OutputDir. Re-running overwrites all generated files.
func Generate(cfg Config) error {
	if len(cfg.SchemaFiles) == 0 {
		return fmt.Errorf("no schema files provided")
	}
	if cfg.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	packageName := cfg.PackageName
	if packageName == "" {
		packageName = "generated"
	}

	// Create output directory if it doesn't exist.
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 1. Parse schema.
	model, err := schema.ParseSchema(cfg.SchemaFiles)
	if err != nil {
		return fmt.Errorf("schema parse failed: %w", err)
	}

	// 2. Augment schema and write to output.
	augmented, err := AugmentSchema(model)
	if err != nil {
		return fmt.Errorf("schema augmentation failed: %w", err)
	}
	schemaPath := filepath.Join(cfg.OutputDir, "schema.graphql")
	if err := writeGeneratedFile(schemaPath, []byte(augmented)); err != nil {
		return fmt.Errorf("failed to write augmented schema: %w", err)
	}

	// 3. Generate gqlgen config and write to output.
	gqlgenCfg := GqlgenConfig{
		SchemaPath:  schemaPath,
		OutputDir:   cfg.OutputDir,
		PackageName: packageName,
	}
	configContent, err := GenerateGqlgenConfig(gqlgenCfg)
	if err != nil {
		return fmt.Errorf("failed to generate gqlgen config: %w", err)
	}
	configPath := filepath.Join(cfg.OutputDir, "gqlgen.yml")
	if err := writeGeneratedFile(configPath, []byte(configContent)); err != nil {
		return fmt.Errorf("failed to write gqlgen config: %w", err)
	}

	// 4. Clean up previously generated Go files (including scaffold) before
	// invoking gqlgen. On re-runs, stale files can cause validation errors.
	cleanGeneratedGoFiles(cfg.OutputDir)

	// 5. Invoke gqlgen to generate Go model types + resolver interfaces.
	if err := InvokeGqlgen(configPath); err != nil {
		return fmt.Errorf("gqlgen generation failed: %w", err)
	}

	// 6. Delete gqlgen resolver scaffold files to prevent conflicts.
	if err := deleteResolverScaffold(cfg.OutputDir); err != nil {
		return fmt.Errorf("failed to delete resolver scaffold: %w", err)
	}

	// 7. Generate resolvers and write to output.
	resolverSrc, err := GenerateResolvers(model, packageName)
	if err != nil {
		return fmt.Errorf("failed to generate resolvers: %w", err)
	}
	if err := writeGeneratedFile(filepath.Join(cfg.OutputDir, "resolvers_gen.go"), resolverSrc); err != nil {
		return fmt.Errorf("failed to write resolvers: %w", err)
	}

	// 8. Generate mappers and write to output.
	mapperSrc, err := GenerateMappers(model, packageName)
	if err != nil {
		return fmt.Errorf("failed to generate mappers: %w", err)
	}
	if err := writeGeneratedFile(filepath.Join(cfg.OutputDir, "mappers_gen.go"), mapperSrc); err != nil {
		return fmt.Errorf("failed to write mappers: %w", err)
	}

	// 9. Generate client constructor and write to output.
	clientSrc, err := GenerateClient(model, packageName)
	if err != nil {
		return fmt.Errorf("failed to generate client: %w", err)
	}
	if err := writeGeneratedFile(filepath.Join(cfg.OutputDir, "client_gen.go"), clientSrc); err != nil {
		return fmt.Errorf("failed to write client: %w", err)
	}

	return nil
}

// writeGeneratedFile writes content to a file with standard permissions for generated code.
func writeGeneratedFile(path string, content []byte) error {
	return os.WriteFile(path, content, 0644)
}

// deleteResolverScaffold removes gqlgen-generated resolver scaffold files
// (resolver.go and *.resolvers.go) from the output directory. These files
// conflict with our resolvers_gen.go and must be deleted after gqlgen runs.
func deleteResolverScaffold(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "resolver.go" || strings.HasSuffix(name, ".resolvers.go") {
			if err := os.Remove(filepath.Join(dir, name)); err != nil {
				return fmt.Errorf("failed to remove scaffold file %s: %w", name, err)
			}
		}
	}
	return nil
}

// cleanGeneratedGoFiles removes stale generated files from the output directory.
// This prevents stale files from causing gqlgen validation errors on re-runs.
// Removes: *_gen.go (our naming convention), resolver.go and *.resolvers.go
// (gqlgen scaffold files that conflict with our resolvers_gen.go).
func cleanGeneratedGoFiles(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, "_gen.go") || name == "resolver.go" || strings.HasSuffix(name, ".resolvers.go") {
			os.Remove(filepath.Join(dir, name))
		}
	}
}
