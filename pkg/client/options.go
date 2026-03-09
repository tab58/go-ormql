package client

import "log/slog"

// defaultBatchSize is the default maximum number of items per mutation input chunk.
const defaultBatchSize = 50

// clientOptions holds optional configuration for the Client.
type clientOptions struct {
	logger    *slog.Logger
	batchSize int
}

// Option configures the Client. Use functional options with New().
type Option func(*clientOptions)

// WithLogger sets a structured logger for debug logging.
// When set, the client logs query and variable information before dispatch.
// When nil (default), no logging overhead occurs.
func WithLogger(logger *slog.Logger) Option {
	return func(o *clientOptions) {
		o.logger = logger
	}
}

// WithBatchSize sets the maximum number of items per mutation input chunk.
// Must be > 0, panics otherwise.
func WithBatchSize(n int) Option {
	if n <= 0 {
		panic("gormql: batch size must be > 0")
	}
	return func(o *clientOptions) {
		o.batchSize = n
	}
}
