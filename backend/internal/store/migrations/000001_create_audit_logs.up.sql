CREATE TABLE IF NOT EXISTS audit_logs (
    id                  BIGSERIAL PRIMARY KEY,
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cluster_id          TEXT NOT NULL,
    "user"              TEXT NOT NULL,
    source_ip           TEXT NOT NULL,
    action              TEXT NOT NULL,
    resource_kind       TEXT DEFAULT '',
    resource_namespace  TEXT DEFAULT '',
    resource_name       TEXT DEFAULT '',
    result              TEXT NOT NULL,
    detail              TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs("user");
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_cluster ON audit_logs(cluster_id, timestamp);
