package codegen

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tab58/go-ormql/pkg/schema"
)

// Config controls the code generation pipeline.
type Config struct {
	SchemaFiles []string  // paths to .graphql schema files
	OutputDir   string    // directory for all generated output
	PackageName string    // Go package name for generated code
	Target      Target    // graph database target (default: TargetNeo4j)
	Stderr      io.Writer // optional writer for warnings (e.g., os.Stderr)
}

// Generate runs the full V2 code generation pipeline in sequence:
//  1. ParseSchema(cfg.SchemaFiles) → GraphModel
//  2. AugmentSchema(model) → write <OutputDir>/schema.graphql
//  3. GenerateModels(model) → write <OutputDir>/models_gen.go
//  4. GenerateGraphModelRegistry(model, augSDL) → write <OutputDir>/graphmodel_gen.go
//  5. GenerateClient(model) → write <OutputDir>/client_gen.go
//  6. GenerateIndexes(model) → write <OutputDir>/indexes_gen.go (conditional — only when @vector present)
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

	// Clean stale generated files from previous runs.
	cleanGeneratedGoFiles(cfg.OutputDir)

	// 1. Parse schema.
	model, err := schema.ParseSchema(cfg.SchemaFiles)
	if err != nil {
		return fmt.Errorf("schema parse failed: %w", err)
	}

	// Validate and normalize target.
	target, err := validateTarget(cfg.Target)
	if err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}

	// Emit @vector warning if any node uses the directive.
	if cfg.Stderr != nil && model.HasVectorField() {
		fmt.Fprintln(cfg.Stderr, vectorWarningForTarget(target))
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

	// 3. Generate models and write to output.
	modelsSrc, err := GenerateModels(model, packageName)
	if err != nil {
		return fmt.Errorf("failed to generate models: %w", err)
	}
	if err := writeGeneratedFile(filepath.Join(cfg.OutputDir, "models_gen.go"), modelsSrc); err != nil {
		return fmt.Errorf("failed to write models: %w", err)
	}

	// 4. Generate GraphModel registry and write to output.
	registrySrc, err := GenerateGraphModelRegistry(model, augmented, packageName)
	if err != nil {
		return fmt.Errorf("failed to generate graph model registry: %w", err)
	}
	if err := writeGeneratedFile(filepath.Join(cfg.OutputDir, "graphmodel_gen.go"), registrySrc); err != nil {
		return fmt.Errorf("failed to write graph model registry: %w", err)
	}

	// 5. Generate client constructor and write to output.
	clientSrc, err := GenerateClient(model, packageName)
	if err != nil {
		return fmt.Errorf("failed to generate client: %w", err)
	}
	if err := writeGeneratedFile(filepath.Join(cfg.OutputDir, "client_gen.go"), clientSrc); err != nil {
		return fmt.Errorf("failed to write client: %w", err)
	}

	// 6. Generate vector indexes (conditional — only when @vector is present).
	indexesSrc, err := GenerateIndexes(model, packageName, target)
	if err != nil {
		return fmt.Errorf("failed to generate indexes: %w", err)
	}
	if indexesSrc != nil {
		if err := writeGeneratedFile(filepath.Join(cfg.OutputDir, "indexes_gen.go"), indexesSrc); err != nil {
			return fmt.Errorf("failed to write indexes: %w", err)
		}
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
// This prevents stale files from causing errors on re-runs.
// Removes: *_gen.go (our naming convention), resolver.go and *.resolvers.go
// (V1 scaffold files), gqlgen.yml (V1 config).
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
		if strings.HasSuffix(name, "_gen.go") || name == "resolver.go" || strings.HasSuffix(name, ".resolvers.go") || name == "gqlgen.yml" {
			os.Remove(filepath.Join(dir, name))
		}
	}
}
