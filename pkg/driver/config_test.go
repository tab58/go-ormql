package driver

import (
	"testing"
	"time"
)

// === CFG-1: Config refactor tests ===

// Test: Config struct has Host, Port, Scheme fields instead of URI.
// Expected: Config{Host: "localhost", Port: 7687, Scheme: "bolt"} compiles.
func TestConfig_HasHostPortScheme(t *testing.T) {
	cfg := Config{
		Host:   "localhost",
		Port:   7687,
		Scheme: "bolt",
	}
	if cfg.Host != "localhost" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost")
	}
	if cfg.Port != 7687 {
		t.Errorf("Port = %d, want %d", cfg.Port, 7687)
	}
	if cfg.Scheme != "bolt" {
		t.Errorf("Scheme = %q, want %q", cfg.Scheme, "bolt")
	}
}

// Test: Config struct has VectorIndexes field of type map[string]VectorIndex.
// Expected: Config{VectorIndexes: map[string]VectorIndex{...}} compiles.
func TestConfig_HasVectorIndexes(t *testing.T) {
	cfg := Config{
		Host:   "localhost",
		Port:   7687,
		Scheme: "bolt",
		VectorIndexes: map[string]VectorIndex{
			"movie_embeddings": {Label: "Movie", Property: "embedding"},
		},
	}
	if cfg.VectorIndexes == nil {
		t.Fatal("VectorIndexes should not be nil")
	}
	vi, ok := cfg.VectorIndexes["movie_embeddings"]
	if !ok {
		t.Fatal("VectorIndexes missing 'movie_embeddings' key")
	}
	if vi.Label != "Movie" {
		t.Errorf("VectorIndex.Label = %q, want %q", vi.Label, "Movie")
	}
	if vi.Property != "embedding" {
		t.Errorf("VectorIndex.Property = %q, want %q", vi.Property, "embedding")
	}
}

// Test: VectorIndex type can be constructed with Label and Property.
// Expected: VectorIndex{Label: "Movie", Property: "embedding"} compiles.
func TestVectorIndex_Construction(t *testing.T) {
	vi := VectorIndex{
		Label:    "Movie",
		Property: "embedding",
	}
	if vi.Label != "Movie" {
		t.Errorf("Label = %q, want %q", vi.Label, "Movie")
	}
	if vi.Property != "embedding" {
		t.Errorf("Property = %q, want %q", vi.Property, "embedding")
	}
}

// Test: Config with nil VectorIndexes is valid (optional field).
// Expected: Config{VectorIndexes: nil} compiles, field is nil.
func TestConfig_NilVectorIndexes(t *testing.T) {
	cfg := Config{
		Host:   "localhost",
		Port:   7687,
		Scheme: "bolt",
	}
	if cfg.VectorIndexes != nil {
		t.Error("VectorIndexes should be nil by default")
	}
}

// === FIX-2: ReadTimeout/WriteTimeout tests ===

// Test: Config has ReadTimeout and WriteTimeout fields of type time.Duration.
// Expected: Config{ReadTimeout: 30s, WriteTimeout: 30s} compiles and stores values.
func TestConfig_HasReadWriteTimeout(t *testing.T) {
	cfg := Config{
		Host:         "localhost",
		Port:         7687,
		Scheme:       "bolt",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	if cfg.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout = %v, want %v", cfg.ReadTimeout, 30*time.Second)
	}
	if cfg.WriteTimeout != 60*time.Second {
		t.Errorf("WriteTimeout = %v, want %v", cfg.WriteTimeout, 60*time.Second)
	}
}

// Test: Config with zero ReadTimeout/WriteTimeout means use driver default.
// Expected: zero-value Config has zero ReadTimeout and WriteTimeout.
func TestConfig_ZeroTimeoutMeansDefault(t *testing.T) {
	cfg := Config{
		Host:   "localhost",
		Port:   7687,
		Scheme: "bolt",
	}
	if cfg.ReadTimeout != 0 {
		t.Errorf("ReadTimeout should be zero by default, got %v", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 0 {
		t.Errorf("WriteTimeout should be zero by default, got %v", cfg.WriteTimeout)
	}
}

// Test: Config retains Username, Password, Database, Logger fields.
// Expected: all existing fields still compile.
func TestConfig_RetainsExistingFields(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     7687,
		Scheme:   "bolt",
		Username: "neo4j",
		Password: "password",
		Database: "testdb",
	}
	if cfg.Username != "neo4j" {
		t.Errorf("Username = %q, want %q", cfg.Username, "neo4j")
	}
	if cfg.Password != "password" {
		t.Errorf("Password = %q, want %q", cfg.Password, "password")
	}
	if cfg.Database != "testdb" {
		t.Errorf("Database = %q, want %q", cfg.Database, "testdb")
	}
}
