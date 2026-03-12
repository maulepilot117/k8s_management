---
status: complete
priority: p2
issue_id: "033"
tags: [code-review, architecture]
dependencies: []
---

# ClusterID Empty String in All Audit Entries

## Problem Statement
`handler.go:92` sets `ClusterID` to `""` in all audit entries. CLAUDE.md's multi-cluster preparation section states that the cluster ID should default to `"local"` in Phase 1, so that Phase 2 multi-cluster support does not need to backfill or migrate existing audit records. Empty strings also make audit log filtering and grouping unreliable.

## Findings
- `handler.go:92` — `ClusterID: ""` hardcoded in `auditWrite`
- CLAUDE.md multi-cluster prep: "All k8s client operations accept a clusterID parameter (defaults to `"local"` in Phase 1)"
- CLAUDE.md database prep: "include a `cluster_id` column from day one"

Flagged by: Architecture Strategist (Finding 7), Pattern Recognition (Finding 7).

## Proposed Solutions
### Option A: Add ClusterID field to Handler struct
Add a `ClusterID string` field to the `Handler` struct, set it from config during server initialization in `server.go`, and reference it in `auditWrite`. Default value in config: `"local"`.
- **Pros:** Clean, config-driven, easy to change in Phase 2
- **Cons:** Minimal code change
- **Effort:** Small (~5 lines)
- **Risk:** Low

### Option B: Hardcode "local" in auditWrite
Replace `""` with `"local"` directly in `auditWrite`.
- **Pros:** Simplest possible fix
- **Cons:** Not config-driven, harder to change in Phase 2
- **Effort:** Trivial (~1 line)
- **Risk:** Low

## Recommended Action


## Technical Details
- **Affected files:** `handler.go`, `server.go`, potentially `config.go`
- **Components:** Audit logging, multi-cluster preparation, configuration

## Acceptance Criteria
- [ ] All audit entries have `ClusterID` set to `"local"` (or configurable value)
- [ ] ClusterID is sourced from application configuration, not hardcoded
- [ ] Existing audit log queries/filters work correctly with the new value

## Work Log
| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources
- PR: #3
