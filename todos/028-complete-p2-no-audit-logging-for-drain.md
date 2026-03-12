---
status: complete
priority: p2
issue_id: "028"
tags: [code-review, security, audit]
dependencies: []
---

# No Audit Logging for Node Drain Operations

## Problem Statement
`HandleDrainNode` and `executeDrain` never call `auditWrite`. Drain is one of the most destructive operations available — it cordons the node and evicts all pods — yet it produces no audit trail. This is inconsistent with cordon/uncordon operations which DO have audit logging, and violates the CLAUDE.md requirement that "all write operations" are audit-logged.

## Findings
- `nodes.go:110-141` — `HandleDrainNode` has no `auditWrite` call
- `nodes.go` (executeDrain) — async goroutine completes or fails with no audit entry
- `nodes.go:93/97` — cordon/uncordon correctly call `auditWrite`, highlighting the inconsistency

Flagged by: Security Sentinel (Finding 6), Pattern Recognition (Finding 3).

## Proposed Solutions
### Option A: Add audit entries at drain start and completion
Add `auditWrite` call in `HandleDrainNode` when the drain is initiated, and in `executeDrain` when it completes or fails. This captures the full lifecycle.
- **Pros:** Complete audit trail, captures both intent and outcome
- **Cons:** Minimal code change
- **Effort:** Small (~5 lines)
- **Risk:** Low

### Option B: Audit only at initiation
Add `auditWrite` only in `HandleDrainNode` at request time, treating it like other fire-and-forget operations.
- **Pros:** Simplest change
- **Cons:** Misses drain failures and completion status
- **Effort:** Small (~2 lines)
- **Risk:** Low, but incomplete

## Recommended Action


## Technical Details
- **Affected files:** `backend/internal/k8s/resources/nodes.go`
- **Components:** Node drain, audit logging

## Acceptance Criteria
- [ ] Drain initiation produces an audit log entry with user, node name, and timeout
- [ ] Drain completion (success or failure) produces an audit log entry with outcome
- [ ] Audit entries are consistent in format with cordon/uncordon audit entries

## Work Log
| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources
- PR: #3
