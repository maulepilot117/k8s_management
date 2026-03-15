# Step 14: Audit Logging — SQLite Persistence

## Overview

Swap the audit logger implementation from `SlogLogger` (structured log output) to `SQLiteLogger` (persistent database). Add a paginated, filterable audit log API endpoint and a frontend audit log viewer. The `audit.Logger` interface (defined in Step 2) remains unchanged — only the implementation is swapped.

## Problem Statement / Motivation

The current `SlogLogger` writes audit entries to stdout as structured JSON. These entries are ephemeral — lost on pod restart, hard to query, and impossible to filter by user/action/time range. Production environments need persistent, queryable audit logs for compliance, incident investigation, and security monitoring.

## Proposed Solution

Add `SQLiteLogger` implementing the existing `audit.Logger` interface. SQLite is operationally simple (no external database), stores on a PVC, and handles the expected write volume easily. Add `GET /api/v1/audit/logs` with pagination and filters, and a frontend audit log viewer page.

## Technical Considerations

- **Single-replica constraint**: SQLite requires `ReadWriteOnce` PVC. Backend must be single-replica when persistence is enabled. Document in Helm values.
- **CGO requirement**: `mattn/go-sqlite3` requires CGO. Alternative: `modernc.org/sqlite` is pure Go (no CGO) — use this to keep distroless container compatibility.
- **Schema migrations**: Embed SQL migration files. Run automatically on startup before accepting traffic.
- **Write performance**: Audit writes are low-volume (tens per minute, not thousands). WAL mode is sufficient.
- **Retention**: 90-day default with configurable retention. Daily cleanup goroutine.

## Files to Create

**`backend/internal/audit/store.go`** — SQLite store
- `SQLiteStore` struct wrapping `*sql.DB`
- `NewSQLiteStore(dbPath string) (*SQLiteStore, error)` — opens DB, runs migrations, enables WAL
- `Insert(ctx, Entry) error` — insert audit entry
- `Query(ctx, QueryParams) ([]Entry, int, error)` — paginated query with filters
- `Cleanup(ctx, retentionDays int) (int64, error)` — delete entries older than retention
- `Close() error`

**`backend/internal/audit/sqlite_logger.go`** — Logger implementation
- `SQLiteLogger` struct implementing `audit.Logger` interface
- Wraps `SQLiteStore.Insert` with error logging (never fails the caller)
- Also writes to slog for structured log output (dual-write for log aggregators)

**`backend/internal/audit/store_test.go`** — SQLite store tests
- Test: insert and query entries
- Test: pagination (offset + limit)
- Test: filter by user, action, resource kind, time range
- Test: retention cleanup deletes old entries
- Test: concurrent writes (WAL mode)

**`backend/internal/audit/migrations/001_create_audit_logs.sql`** — Initial schema
```sql
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TEXT NOT NULL,
    cluster_id TEXT NOT NULL,
    user TEXT NOT NULL,
    source_ip TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_kind TEXT,
    resource_namespace TEXT,
    resource_name TEXT,
    result TEXT NOT NULL,
    detail TEXT
);

CREATE INDEX idx_audit_timestamp ON audit_logs(timestamp);
CREATE INDEX idx_audit_user ON audit_logs(user);
CREATE INDEX idx_audit_action ON audit_logs(action);
```

**`backend/internal/audit/query.go`** — Query types
- `QueryParams` struct: `User`, `Action`, `ResourceKind`, `Namespace`, `Since`, `Until`, `Page`, `PageSize`
- Default page size: 50, max: 200

**`backend/internal/server/handle_audit.go`** — Audit API handler
- `GET /api/v1/audit/logs` — paginated, filterable audit log query
- Query params: `user`, `action`, `kind`, `namespace`, `since`, `until`, `page`, `pageSize`
- Returns standard `api.Response` envelope with `metadata.total`

**`frontend/routes/settings/audit.tsx`** — Audit log page route
- Server-rendered page hosting `AuditLogViewer` island

**`frontend/islands/AuditLogViewer.tsx`** — Audit log viewer island
- Fetches from `/api/v1/audit/logs` with query params
- Filter controls: user, action, resource kind, time range
- Sortable table with pagination
- Auto-refresh toggle

## Files to Modify

- `backend/cmd/kubecenter/main.go` — Conditionally create SQLiteLogger or SlogLogger based on config
- `backend/internal/config/config.go` — Add `AuditConfig` with `DBPath`, `RetentionDays`
- `backend/internal/config/defaults.go` — Add audit defaults
- `backend/internal/server/routes.go` — Add `GET /api/v1/audit/logs` route (authenticated, admin-only)
- `backend/internal/server/server.go` — Add `AuditStore` to Server/Deps (for query handler)
- `frontend/lib/constants.ts` — Add "Audit Log" to Settings nav section
- `helm/kubecenter/values.yaml` — Add persistence and audit config sections
- `helm/kubecenter/templates/deployment-backend.yaml` — Add PVC mount when persistence enabled
- `helm/kubecenter/templates/pvc-data.yaml` — Re-add PVC template (gated on persistence.enabled)

## Dependencies

| Dependency | Version | Purpose |
|---|---|---|
| `modernc.org/sqlite` | latest | Pure Go SQLite (no CGO, distroless compatible) |

## Acceptance Criteria

- [ ] All write operations persisted in SQLite audit log
- [ ] Secret reveals logged with key name (not value)
- [ ] `GET /api/v1/audit/logs` returns paginated, filterable results
- [ ] Filters work: user, action, kind, namespace, time range
- [ ] Retention cleanup runs daily, deletes entries older than configured days
- [ ] SlogLogger continues to work as fallback when persistence is disabled
- [ ] SQLiteLogger dual-writes to both SQLite and slog
- [ ] Frontend audit log viewer displays entries with filters
- [ ] Audit log route is admin-only
- [ ] `make test` passes (SQLite tests use temp files)
- [ ] Helm chart supports persistence toggle with PVC

## References

- Existing interface: `backend/internal/audit/logger.go` — `Logger` interface, `Entry` struct
- Existing implementation: `backend/internal/audit/logger.go` — `SlogLogger`
- CLAUDE.md decision D5: SQLite with PVC, fallback to in-memory with structured log output
- modernc.org/sqlite: https://pkg.go.dev/modernc.org/sqlite
