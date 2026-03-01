package codegen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/99designs/gqlgen/api"
	gqlgenConfig "github.com/99designs/gqlgen/codegen/config"
	"gopkg.in/yaml.v3"
)

// GqlgenConfig holds the paths needed to generate a gqlgen.yml configuration.
type GqlgenConfig struct {
	SchemaPath  string // path to the augmented .graphql schema file
	OutputDir   string // directory for generated Go files
	PackageName string // Go package name for generated code
}

// gqlgenYAML represents the structure of a gqlgen.yml config file.
type gqlgenYAML struct {
	Schema   []string            `yaml:"schema"`
	Exec     gqlgenPackage       `yaml:"exec"`
	Model    gqlgenPackage       `yaml:"model"`
	Resolver gqlgenResolverYAML  `yaml:"resolver"`
}

type gqlgenPackage struct {
	Filename string `yaml:"filename"`
	Package  string `yaml:"package"`
}

// resolverLayoutFollowSchema tells gqlgen to generate per-schema resolver files.
const resolverLayoutFollowSchema = "follow-schema"

// gqlgenResolverYAML controls gqlgen's resolver scaffold generation.
type gqlgenResolverYAML struct {
	Layout           string `yaml:"layout"`
	Dir              string `yaml:"dir"`
	Package          string `yaml:"package"`
	FilenameTemplate string `yaml:"filename_template"`
}

// GenerateGqlgenConfig produces the contents of a gqlgen.yml configuration file
// pointing to the augmented schema and output directory.
// Returns the YAML content as a string.
func GenerateGqlgenConfig(cfg GqlgenConfig) (string, error) {
	config := gqlgenYAML{
		Schema: []string{filepath.Base(cfg.SchemaPath)},
		Exec: gqlgenPackage{
			Filename: "exec_gen.go",
			Package:  cfg.PackageName,
		},
		Model: gqlgenPackage{
			Filename: "models_gen.go",
			Package:  cfg.PackageName,
		},
		Resolver: gqlgenResolverYAML{
			Layout:           resolverLayoutFollowSchema,
			Dir:              ".",
			Package:          cfg.PackageName,
			FilenameTemplate: "{name}.resolvers.go",
		},
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal gqlgen config: %w", err)
	}

	return string(data), nil
}

// InvokeGqlgen runs gqlgen code generation programmatically using the given
// config file path. Produces Go model types and resolver interfaces.
func InvokeGqlgen(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("gqlgen config not found: %s", configPath)
	}

	// Read our YAML config to validate schema paths before invoking gqlgen.
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read gqlgen config: %w", err)
	}

	var yamlCfg gqlgenYAML
	if err := yaml.Unmarshal(data, &yamlCfg); err != nil {
		return fmt.Errorf("failed to parse gqlgen config: %w", err)
	}

	dir := filepath.Dir(configPath)

	for _, sp := range yamlCfg.Schema {
		absSchema := filepath.Join(dir, sp)
		if _, err := os.Stat(absSchema); os.IsNotExist(err) {
			return fmt.Errorf("schema file not found: %s", absSchema)
		}
	}
	var tempFiles []string

	// gqlgen needs a go.mod in the output directory to resolve Go package paths.
	// The go.mod must require gqlgen so go/packages can resolve its scalar types.
	goModPath := filepath.Join(dir, "go.mod")
	createdGoMod := false
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		moduleName := yamlCfg.Exec.Package
		goModContent := fmt.Sprintf("module %s\n\ngo 1.25\n\nrequire github.com/99designs/gqlgen v0.17.87\n", moduleName)

		// If the output dir contains .go files from a previous run that import
		// the main module, go mod tidy needs a replace directive to resolve them.
		goModContent += mainModuleReplace()

		if err := os.WriteFile(goModPath, []byte(goModContent), 0644); err != nil {
			return fmt.Errorf("failed to create go.mod for gqlgen: %w", err)
		}
		tempFiles = append(tempFiles, goModPath)
		createdGoMod = true
	}

	// gqlgen needs at least one .go file in the package. Import gqlgen/graphql
	// so go mod tidy keeps the gqlgen dependency.
	stubPath := filepath.Join(dir, "stub_gen.go")
	if _, err := os.Stat(stubPath); os.IsNotExist(err) {
		stubContent := fmt.Sprintf("package %s\n\nimport _ \"github.com/99designs/gqlgen/graphql\"\n", yamlCfg.Exec.Package)
		if err := os.WriteFile(stubPath, []byte(stubContent), 0644); err != nil {
			cleanupTempFiles(tempFiles)
			return fmt.Errorf("failed to create stub for gqlgen: %w", err)
		}
		tempFiles = append(tempFiles, stubPath)
	}

	// Resolve transitive dependencies so go/packages can find gqlgen types.
	if createdGoMod {
		cmd := exec.Command("go", "mod", "tidy")
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			cleanupTempFiles(tempFiles)
			return fmt.Errorf("failed to resolve dependencies: %s: %w", string(output), err)
		}
		tempFiles = append(tempFiles, filepath.Join(dir, "go.sum"))
	}

	// gqlgen uses go/packages which resolves from CWD. Change to the output
	// directory so it can find the go.mod and resolve the package.
	origDir, err := os.Getwd()
	if err != nil {
		cleanupTempFiles(tempFiles)
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Convert configPath to absolute before chdir, otherwise the relative
	// path resolves against the new working directory (e.g. out/out/gqlgen.yml).
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		cleanupTempFiles(tempFiles)
		return fmt.Errorf("failed to resolve config path: %w", err)
	}

	if err := os.Chdir(dir); err != nil {
		cleanupTempFiles(tempFiles)
		return fmt.Errorf("failed to chdir to output directory: %w", err)
	}

	cfg, loadErr := gqlgenConfig.LoadConfig(absConfigPath)
	if loadErr != nil {
		os.Chdir(origDir)
		cleanupTempFiles(tempFiles)
		return fmt.Errorf("gqlgen config load failed: %w", loadErr)
	}

	genErr := api.Generate(cfg)
	os.Chdir(origDir)
	cleanupTempFiles(tempFiles)
	if genErr != nil {
		return fmt.Errorf("gqlgen generate failed: %w", genErr)
	}

	return nil
}

// mainModuleReplace returns a go.mod replace directive pointing the main module
// to its local path. This allows go mod tidy to resolve imports in generated
// files that reference packages from the main module (e.g., pkg/driver, pkg/cypher).
// Returns an empty string if the main module cannot be determined.
func mainModuleReplace() string {
	goModBytes, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return ""
	}
	goModFile := strings.TrimSpace(string(goModBytes))
	if goModFile == "" || goModFile == os.DevNull {
		return ""
	}

	data, err := os.ReadFile(goModFile)
	if err != nil {
		return ""
	}

	// Extract module name from "module <name>" line.
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			moduleName := strings.TrimSpace(strings.TrimPrefix(line, "module"))
			moduleDir := filepath.Dir(goModFile)
			return fmt.Sprintf("\nreplace %s => %s\n", moduleName, moduleDir)
		}
	}
	return ""
}

// cleanupTempFiles removes temporary files created during gqlgen invocation.
func cleanupTempFiles(paths []string) {
	for _, p := range paths {
		os.Remove(p)
	}
}
