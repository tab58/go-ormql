package codegen

import (
	"strings"
	"testing"

	"github.com/tab58/gql-orm/pkg/schema"
)

// === CG-9: Client generator tests ===

// TestGenerateClient_NonEmpty verifies that GenerateClient returns non-empty
// output for a valid model.
// Expected: non-empty byte slice.
func TestGenerateClient_NonEmpty(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if len(src) == 0 {
		t.Fatal("GenerateClient returned empty output, want non-empty Go source")
	}
}

// TestGenerateClient_PackageDeclaration verifies that the generated source
// contains the correct package declaration.
// Expected: "package generated".
func TestGenerateClient_PackageDeclaration(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if !strings.Contains(string(src), "package generated") {
		t.Errorf("output missing 'package generated':\n%s", string(src))
	}
}

// TestGenerateClient_ContainsNewClientFunc verifies that the generated source
// defines a NewClient convenience constructor function.
// Expected: "func NewClient" present in output.
func TestGenerateClient_ContainsNewClientFunc(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if !strings.Contains(string(src), "func NewClient") {
		t.Errorf("output missing 'func NewClient':\n%s", string(src))
	}
}

// TestGenerateClient_ImportsClientPackage verifies that the generated source
// imports pkg/client to use client.New and client.Client.
// Expected: import path containing "pkg/client".
func TestGenerateClient_ImportsClientPackage(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if !strings.Contains(string(src), "pkg/client") {
		t.Errorf("output missing import of 'pkg/client':\n%s", string(src))
	}
}

// TestGenerateClient_ImportsDriverPackage verifies that the generated source
// imports pkg/driver for the driver.Driver parameter type.
// Expected: import path containing "pkg/driver".
func TestGenerateClient_ImportsDriverPackage(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if !strings.Contains(string(src), "pkg/driver") {
		t.Errorf("output missing import of 'pkg/driver':\n%s", string(src))
	}
}

// TestGenerateClient_ReferencesGraphModel verifies that the generated NewClient
// function references the GraphModel package variable (V2 architecture).
// Expected: "GraphModel" referenced in the output.
func TestGenerateClient_ReferencesGraphModel(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if !strings.Contains(string(src), "GraphModel") {
		t.Errorf("output missing reference to 'GraphModel':\n%s", string(src))
	}
}

// TestGenerateClient_ReferencesAugmentedSchemaSDL verifies that the generated
// NewClient function references the AugmentedSchemaSDL package variable (V2 architecture).
// Expected: "AugmentedSchemaSDL" referenced in the output.
func TestGenerateClient_ReferencesAugmentedSchemaSDL(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if !strings.Contains(string(src), "AugmentedSchemaSDL") {
		t.Errorf("output missing reference to 'AugmentedSchemaSDL':\n%s", string(src))
	}
}

// TestGenerateClient_ReferencesClientNew verifies that the generated NewClient
// function calls client.New to create the Client instance.
// Expected: "client.New" referenced in the output.
func TestGenerateClient_ReferencesClientNew(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if !strings.Contains(string(src), "client.New") {
		t.Errorf("output missing reference to 'client.New':\n%s", string(src))
	}
}

// TestGenerateClient_EmptyModel_NoError verifies that GenerateClient with
// an empty model does not return an error (graceful handling).
// Expected: no error returned.
func TestGenerateClient_EmptyModel_NoError(t *testing.T) {
	_, err := GenerateClient(schema.GraphModel{}, "generated")
	if err != nil {
		t.Fatalf("GenerateClient with empty model returned error: %v", err)
	}
}

// --- CG-22: V2 Client generator tests ---
// The V2 client generator passes GraphModel and AugmentedSchemaSDL to client.New()
// instead of using gqlgen's ExecutableSchema, Config, or Resolvers.

// Test: V2 generated NewClient references GraphModel package variable.
// Expected: output contains "GraphModel".
func TestGenerateClient_V2_ReferencesGraphModel(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if !strings.Contains(string(src), "GraphModel") {
		t.Error("V2 output should reference 'GraphModel' package variable")
	}
}

// Test: V2 generated NewClient references AugmentedSchemaSDL package variable.
// Expected: output contains "AugmentedSchemaSDL".
func TestGenerateClient_V2_ReferencesAugmentedSchemaSDL(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if !strings.Contains(string(src), "AugmentedSchemaSDL") {
		t.Error("V2 output should reference 'AugmentedSchemaSDL' package variable")
	}
}

// Test: V2 generated NewClient calls client.New with GraphModel + AugmentedSchemaSDL.
// Expected: output contains "client.New(GraphModel, AugmentedSchemaSDL".
func TestGenerateClient_V2_CallsClientNewWithModelAndSDL(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if !strings.Contains(string(src), "client.New(GraphModel, AugmentedSchemaSDL") {
		t.Error("V2 output should call 'client.New(GraphModel, AugmentedSchemaSDL, ...'")
	}
}

// Test: V2 generated output does NOT reference gqlgen Resolver.
// Expected: output does NOT contain "Resolver".
func TestGenerateClient_V2_NoResolver(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if strings.Contains(string(src), "Resolver") {
		t.Error("V2 output should NOT reference 'Resolver' (gqlgen removed)")
	}
}

// Test: V2 generated output does NOT reference gqlgen NewExecutableSchema.
// Expected: output does NOT contain "NewExecutableSchema".
func TestGenerateClient_V2_NoNewExecutableSchema(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if strings.Contains(string(src), "NewExecutableSchema") {
		t.Error("V2 output should NOT reference 'NewExecutableSchema' (gqlgen removed)")
	}
}

// Test: V2 generated output does NOT reference gqlgen Config.
// Expected: output does NOT contain "Config{".
func TestGenerateClient_V2_NoConfig(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if strings.Contains(string(src), "Config{") {
		t.Error("V2 output should NOT reference 'Config{' (gqlgen removed)")
	}
}

// Test: V2 generated output contains "DO NOT EDIT" header comment.
// Expected: output contains generated code marker.
func TestGenerateClient_V2_GeneratedHeader(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if !strings.Contains(string(src), "DO NOT EDIT") {
		t.Error("V2 output should contain 'DO NOT EDIT' generated code header")
	}
}
