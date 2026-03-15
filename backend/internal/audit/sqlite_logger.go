package audit

import (
	"context"
	"log/slog"
)

// PostgresLogger implements the Logger interface with persistent PostgreSQL storage.
// It dual-writes to both PostgreSQL (for querying) and slog (for log aggregators).
type PostgresLogger struct {
	store  *PostgresStore
	slog   *SlogLogger
	logger *slog.Logger
}

// NewPostgresLogger creates an audit logger that persists entries in PostgreSQL
// and also writes to structured log output.
func NewPostgresLogger(store *PostgresStore, logger *slog.Logger) *PostgresLogger {
	return &PostgresLogger{
		store:  store,
		slog:   NewSlogLogger(logger),
		logger: logger,
	}
}

// Log writes an audit entry to both PostgreSQL and slog.
// PostgreSQL errors are logged but do not fail the caller.
func (l *PostgresLogger) Log(ctx context.Context, e Entry) error {
	// Always write to slog (structured log output for aggregators)
	l.slog.Log(ctx, e)

	// Persist to PostgreSQL
	if err := l.store.Insert(ctx, e); err != nil {
		l.logger.Error("failed to persist audit entry", "error", err, "action", e.Action, "user", e.User)
	}

	return nil
}

// Query delegates to the underlying PostgresStore for audit log queries.
func (l *PostgresLogger) Query(ctx context.Context, params QueryParams) ([]Entry, int, error) {
	return l.store.Query(ctx, params)
}

// Cleanup delegates to the underlying PostgresStore for retention cleanup.
func (l *PostgresLogger) Cleanup(ctx context.Context, retentionDays int) (int64, error) {
	return l.store.Cleanup(ctx, retentionDays)
}

// Queryable is implemented by Logger implementations that support audit log queries.
type Queryable interface {
	Query(ctx context.Context, params QueryParams) ([]Entry, int, error)
}
