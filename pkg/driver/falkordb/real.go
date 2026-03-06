package falkordb

import (
	"crypto/tls"
	"fmt"

	fdb "github.com/FalkorDB/falkordb-go/v2"
	"github.com/tab58/go-ormql/pkg/driver"
)

// realFalkorDB wraps the real FalkorDB client to implement the falkordbDB interface.
type realFalkorDB struct {
	db *fdb.FalkorDB
}

// SelectGraph returns a graph wrapper implementing graphRunner.
func (r *realFalkorDB) SelectGraph(name string) graphRunner {
	return &realGraph{graph: r.db.SelectGraph(name)}
}

// Close closes the underlying Redis connection.
func (r *realFalkorDB) Close() error {
	return r.db.Conn.Close()
}

// realGraph wraps a FalkorDB Graph to implement graphRunner.
type realGraph struct {
	graph *fdb.Graph
}

// Query runs a read-write query.
func (g *realGraph) Query(query string, params map[string]any) (resultIterator, error) {
	qr, err := g.graph.Query(query, params, nil)
	if err != nil {
		return nil, err
	}
	return &realResultIterator{qr: qr}, nil
}

// ROQuery runs a read-only query.
func (g *realGraph) ROQuery(query string, params map[string]any) (resultIterator, error) {
	qr, err := g.graph.ROQuery(query, params, nil)
	if err != nil {
		return nil, err
	}
	return &realResultIterator{qr: qr}, nil
}

// realResultIterator wraps a FalkorDB QueryResult to implement resultIterator.
type realResultIterator struct {
	qr *fdb.QueryResult
}

// Next advances to the next record.
func (r *realResultIterator) Next() bool {
	return r.qr.Next()
}

// Record returns the current record as a map.
func (r *realResultIterator) Record() map[string]any {
	rec := r.qr.Record()
	if rec == nil {
		return nil
	}
	row := make(map[string]any)
	keys := rec.Keys()
	values := rec.Values()
	for i, key := range keys {
		row[key] = values[i]
	}
	return row
}

// newRealFalkorDBDriver creates a FalkorDBDriver connected to a real FalkorDB instance.
func newRealFalkorDBDriver(cfg driver.Config) (driver.Driver, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	opts := &fdb.ConnectionOption{
		Addr:         addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	if cfg.Scheme == "rediss" {
		opts.TLSConfig = &tls.Config{
			ServerName: cfg.Host,
		}
	}

	db, err := fdb.FalkorDBNew(opts)
	if err != nil {
		return nil, err
	}

	realDB := &realFalkorDB{db: db}
	graph := realDB.SelectGraph(cfg.Database)

	drv := &FalkorDBDriver{
		db:            realDB,
		graph:         graph,
		graphName:     cfg.Database,
		logger:        cfg.Logger,
		vectorIndexes: cfg.VectorIndexes,
	}

	return drv, nil
}
