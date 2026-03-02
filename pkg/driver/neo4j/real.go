package neo4j

import (
	"context"

	neo4jDriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/tab58/go-ormql/pkg/driver"
)

// realNeo4jDB wraps the real neo4j-go-driver to implement the neo4jDB interface.
type realNeo4jDB struct {
	drv neo4jDriver.DriverWithContext
}

// NewSession creates a session for the given database.
func (r *realNeo4jDB) NewSession(database string) sessionRunner {
	session := r.drv.NewSession(context.Background(), neo4jDriver.SessionConfig{
		DatabaseName: database,
	})
	return &realSession{session: session}
}

// Close closes the underlying neo4j driver.
func (r *realNeo4jDB) Close(ctx context.Context) error {
	return r.drv.Close(ctx)
}

// realSession wraps a neo4j session to implement sessionRunner.
type realSession struct {
	session neo4jDriver.SessionWithContext
}

// ExecuteRead runs a read transaction.
func (s *realSession) ExecuteRead(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	result, err := s.session.ExecuteRead(ctx, func(tx neo4jDriver.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		return collectRecords(ctx, records)
	})
	if err != nil {
		return nil, err
	}
	return result.([]map[string]any), nil
}

// ExecuteWrite runs a write transaction.
func (s *realSession) ExecuteWrite(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	result, err := s.session.ExecuteWrite(ctx, func(tx neo4jDriver.ManagedTransaction) (any, error) {
		records, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		return collectRecords(ctx, records)
	})
	if err != nil {
		return nil, err
	}
	return result.([]map[string]any), nil
}

// BeginTransaction starts an explicit transaction.
func (s *realSession) BeginTransaction(ctx context.Context) (transactionRunner, error) {
	tx, err := s.session.BeginTransaction(ctx)
	if err != nil {
		return nil, err
	}
	return &realTransaction{tx: tx}, nil
}

// Close closes the session.
func (s *realSession) Close(ctx context.Context) error {
	return s.session.Close(ctx)
}

// realTransaction wraps a neo4j explicit transaction to implement transactionRunner.
type realTransaction struct {
	tx neo4jDriver.ExplicitTransaction
}

// Run executes a query within the transaction.
func (t *realTransaction) Run(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	records, err := t.tx.Run(ctx, query, params)
	if err != nil {
		return nil, err
	}
	return collectRecords(ctx, records)
}

// Commit commits the transaction.
func (t *realTransaction) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

// Rollback rolls back the transaction.
func (t *realTransaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

// collectRecords collects all records from a neo4j result into []map[string]any.
func collectRecords(ctx context.Context, result neo4jDriver.ResultWithContext) ([]map[string]any, error) {
	var rows []map[string]any
	for result.Next(ctx) {
		record := result.Record()
		row := make(map[string]any)
		keys := record.Keys
		for i, key := range keys {
			row[key] = record.Values[i]
		}
		rows = append(rows, row)
	}
	if err := result.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

// newRealNeo4jDriver creates a Neo4jDriver connected to a real Neo4j instance.
func newRealNeo4jDriver(cfg driver.Config) (driver.Driver, error) {
	auth := neo4jDriver.NoAuth()
	if cfg.Password != "" {
		auth = neo4jDriver.BasicAuth(cfg.Username, cfg.Password, "")
	}

	drv, err := neo4jDriver.NewDriverWithContext(cfg.URI, auth)
	if err != nil {
		return nil, err
	}

	// Verify connectivity
	ctx := context.Background()
	if err := drv.VerifyConnectivity(ctx); err != nil {
		drv.Close(ctx)
		return nil, err
	}

	database := cfg.Database
	if database == "" {
		database = "neo4j"
	}

	return newFromDB(&realNeo4jDB{drv: drv}, database), nil
}
