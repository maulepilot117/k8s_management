// Package store provides PostgreSQL database access for k8sCenter.
// It manages connection pooling, schema migrations, and provides
// the pgxpool.Pool for use by other packages (audit, settings, clusters).
package store

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps a pgxpool.Pool with migration support.
type DB struct {
	Pool   *pgxpool.Pool
	logger *slog.Logger
}

// New creates a new database connection pool and runs migrations.
// The connString should be a PostgreSQL connection URL:
// postgresql://user:pass@host:5432/dbname?sslmode=require
// maxConns/minConns of 0 use defaults (10/2).
func New(ctx context.Context, connString string, maxConns, minConns int32, logger *slog.Logger) (*DB, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	// Apply pool sizing from config (overrides connection string defaults)
	if maxConns > 0 {
		config.MaxConns = maxConns
	} else if config.MaxConns < 10 {
		config.MaxConns = 10
	}
	if minConns > 0 {
		config.MinConns = minConns
	} else if config.MinConns < 2 {
		config.MinConns = 2
	}

	// Retry connection with backoff — PostgreSQL may still be starting
	var pool *pgxpool.Pool
	const maxRetries = 10
	for attempt := range maxRetries {
		pool, err = pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			if attempt < maxRetries-1 {
				delay := time.Duration(attempt+1) * 2 * time.Second
				logger.Warn("database connection failed, retrying", "attempt", attempt+1, "delay", delay, "error", err)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			return nil, fmt.Errorf("creating connection pool: %w", err)
		}

		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			if attempt < maxRetries-1 {
				delay := time.Duration(attempt+1) * 2 * time.Second
				logger.Warn("database ping failed, retrying", "attempt", attempt+1, "delay", delay, "error", err)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			return nil, fmt.Errorf("database ping failed: %w", err)
		}
		break // connected successfully
	}

	db := &DB{Pool: pool, logger: logger}

	// Run migrations
	if err := db.migrate(connString); err != nil {
		pool.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	logger.Info("database connected", "host", config.ConnConfig.Host, "database", config.ConnConfig.Database)

	return db, nil
}

// Close closes the connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}

// Ping checks database connectivity (used by readiness probe).
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}
