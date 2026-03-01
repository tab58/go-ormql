package codegen

import (
	"strings"
	"testing"
)

// --- LOG-1: Generated NewClient passthrough for client.Option ---

// TestGenerateClient_AcceptsOptions verifies that the generated NewClient
// function accepts variadic client.Option parameters.
// Expected: generated source contains "opts ...client.Option" in NewClient signature.
func TestGenerateClient_AcceptsOptions(t *testing.T) {
	src, err := GenerateClient(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "...client.Option") {
		t.Errorf("generated NewClient should accept variadic client.Option:\n%s", s)
	}
}

// TestGenerateClient_PassesOptionsToClientNew verifies that the generated
// NewClient passes the options through to client.New().
// Expected: generated source contains "client.New(es, drv, opts...)" or similar.
func TestGenerateClient_PassesOptionsToClientNew(t *testing.T) {
	src, err := GenerateClient(resolverModel(), "generated")
	if err != nil {
		t.Fatalf("GenerateClient returned error: %v", err)
	}
	s := string(src)
	if !strings.Contains(s, "opts...") {
		t.Errorf("generated NewClient should pass opts... to client.New:\n%s", s)
	}
}
