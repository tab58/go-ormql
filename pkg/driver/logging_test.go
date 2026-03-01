package driver

import (
	"log/slog"
	"testing"
)

// --- LOG-1: slog debug logging in driver ---

// TestConfig_LoggerField verifies that Config has a Logger field of type *slog.Logger.
// Expected: Config{Logger: logger} compiles and logger is accessible.
func TestConfig_LoggerField(t *testing.T) {
	logger := slog.Default()
	cfg := Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
		Logger:   logger,
	}
	if cfg.Logger == nil {
		t.Error("Config.Logger should be set when provided")
	}
}

// TestConfig_LoggerNilDefault verifies that Config.Logger defaults to nil
// when not explicitly set (zero overhead — no logging).
// Expected: Config{}.Logger == nil.
func TestConfig_LoggerNilDefault(t *testing.T) {
	cfg := Config{}
	if cfg.Logger != nil {
		t.Error("Config.Logger should default to nil")
	}
}
