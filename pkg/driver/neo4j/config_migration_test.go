package neo4j

import (
	"testing"

	"github.com/tab58/go-ormql/pkg/driver"
)

// === CFG-3: Config migration guardrail tests ===
// These tests verify that the Config refactor (URI → Host/Port/Scheme) is
// correctly reflected in all test code that constructs driver.Config.

// Test: parseBoltURL extracts Host, Port, Scheme from a bolt:// URL.
// This helper is needed by integration tests that get a bolt URL from testcontainers.
// Expected: parseBoltURL("bolt://localhost:7687") returns ("bolt", "localhost", 7687, nil).
func TestParseBoltURL(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantScheme string
		wantHost   string
		wantPort   int
		wantErr    bool
	}{
		{
			name:       "bolt default",
			url:        "bolt://localhost:7687",
			wantScheme: "bolt",
			wantHost:   "localhost",
			wantPort:   7687,
		},
		{
			name:       "bolt+s with IP",
			url:        "bolt+s://10.0.0.1:7688",
			wantScheme: "bolt+s",
			wantHost:   "10.0.0.1",
			wantPort:   7688,
		},
		{
			name:       "neo4j scheme",
			url:        "neo4j://db.example.com:7687",
			wantScheme: "neo4j",
			wantHost:   "db.example.com",
			wantPort:   7687,
		},
		{
			name:    "invalid URL",
			url:     "not-a-url",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme, host, port, err := ParseBoltURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatal("parseBoltURL should return error for invalid URL")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseBoltURL(%q) returned error: %v", tt.url, err)
			}
			if scheme != tt.wantScheme {
				t.Errorf("scheme = %q, want %q", scheme, tt.wantScheme)
			}
			if host != tt.wantHost {
				t.Errorf("host = %q, want %q", host, tt.wantHost)
			}
			if port != tt.wantPort {
				t.Errorf("port = %d, want %d", port, tt.wantPort)
			}
		})
	}
}

// Test: driver.Config can be constructed with Host/Port/Scheme (not URI).
// This is a compile-time guardrail ensuring CFG-1 is applied before CFG-3.
// Expected: compiles without error.
func TestConfig_NewShapeCompiles(t *testing.T) {
	_ = driver.Config{
		Host:     "localhost",
		Port:     7687,
		Scheme:   "bolt",
		Username: "neo4j",
		Password: "",
		Database: "neo4j",
	}
}
