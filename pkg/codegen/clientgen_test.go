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

// TestGenerateClient_ReferencesResolver verifies that the generated NewClient
// function creates a Resolver instance (the generated resolver struct).
// Expected: "Resolver" referenced in the output.
func TestGenerateClient_ReferencesResolver(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if !strings.Contains(string(src), "Resolver") {
		t.Errorf("output missing reference to 'Resolver':\n%s", string(src))
	}
}

// TestGenerateClient_ReferencesNewExecutableSchema verifies that the generated
// NewClient function calls gqlgen's NewExecutableSchema to create the schema.
// Expected: "NewExecutableSchema" referenced in the output.
func TestGenerateClient_ReferencesNewExecutableSchema(t *testing.T) {
	src, err := GenerateClient(movieModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	if !strings.Contains(string(src), "NewExecutableSchema") {
		t.Errorf("output missing reference to 'NewExecutableSchema':\n%s", string(src))
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
