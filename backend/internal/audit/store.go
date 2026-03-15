package audit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore provides persistent audit log storage backed by PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates an audit store using an existing connection pool.
// The audit_logs table must already exist (created by store.DB migrations).
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Insert writes an audit entry to the database.
func (s *PostgresStore) Insert(ctx context.Context, e Entry) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO audit_logs (timestamp, cluster_id, "user", source_ip, action, resource_kind, resource_namespace, resource_name, result, detail)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		e.Timestamp.UTC(), e.ClusterID, e.User, e.SourceIP, string(e.Action),
		e.ResourceKind, e.ResourceNamespace, e.ResourceName,
		string(e.Result), e.Detail,
	)
	if err != nil {
		return fmt.Errorf("inserting audit entry: %w", err)
	}
	return nil
}

// Query returns audit entries matching the given filters with pagination.
// Returns entries, total count, and error.
func (s *PostgresStore) Query(ctx context.Context, params QueryParams) ([]Entry, int, error) {
	params.Normalize()

	var conditions []string
	var args []any
	argIdx := 1

	if params.User != "" {
		conditions = append(conditions, fmt.Sprintf(`"user" = $%d`, argIdx))
		args = append(args, params.User)
		argIdx++
	}
	if params.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, params.Action)
		argIdx++
	}
	if params.ResourceKind != "" {
		conditions = append(conditions, fmt.Sprintf("resource_kind = $%d", argIdx))
		args = append(args, params.ResourceKind)
		argIdx++
	}
	if params.Namespace != "" {
		conditions = append(conditions, fmt.Sprintf("resource_namespace = $%d", argIdx))
		args = append(args, params.Namespace)
		argIdx++
	}
	if !params.Since.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argIdx))
		args = append(args, params.Since.UTC())
		argIdx++
	}
	if !params.Until.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argIdx))
		args = append(args, params.Until.UTC())
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM audit_logs ` + whereClause
	if err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting audit entries: %w", err)
	}

	// Get paginated results (newest first)
	dataQuery := fmt.Sprintf(
		`SELECT timestamp, cluster_id, "user", source_ip, action, resource_kind, resource_namespace, resource_name, result, detail
		 FROM audit_logs %s ORDER BY timestamp DESC LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1,
	)
	dataArgs := make([]any, 0, len(args)+2)
	dataArgs = append(dataArgs, args...)
	dataArgs = append(dataArgs, params.PageSize, params.Offset())

	rows, err := s.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying audit entries: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var action, result string
		if err := rows.Scan(&e.Timestamp, &e.ClusterID, &e.User, &e.SourceIP, &action,
			&e.ResourceKind, &e.ResourceNamespace, &e.ResourceName, &result, &e.Detail); err != nil {
			return nil, 0, fmt.Errorf("scanning audit entry: %w", err)
		}
		e.Action = Action(action)
		e.Result = Result(result)
		entries = append(entries, e)
	}

	return entries, total, rows.Err()
}

// Cleanup deletes audit entries older than the given number of days.
// Returns the number of deleted rows. Rejects retentionDays < 1.
func (s *PostgresStore) Cleanup(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays < 1 {
		return 0, fmt.Errorf("retentionDays must be >= 1, got %d", retentionDays)
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	result, err := s.pool.Exec(ctx, `DELETE FROM audit_logs WHERE timestamp < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleaning up audit entries: %w", err)
	}
	return result.RowsAffected(), nil
}
