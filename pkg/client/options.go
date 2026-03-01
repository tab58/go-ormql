package client

import "log/slog"

// clientOptions holds optional configuration for the Client.
type clientOptions struct {
	logger *slog.Logger
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
