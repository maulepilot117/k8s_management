---
status: complete
priority: p2
issue_id: "032"
tags: [code-review, reliability]
dependencies: []
---

# Drain Goroutine Lifecycle Issues: Orphans, Duplicates, Unbounded Timeout

## Problem Statement
`executeDrain` creates a `context.Background()` detached from the server's lifecycle. On SIGTERM, drain goroutines continue running as orphans, potentially interfering with graceful shutdown. There is no deduplication — multiple concurrent drain requests for the same node can race against each other. Additionally, `DrainRequest.Timeout` has no upper bound, allowing a user to set an indefinite timeout that holds resources forever.

## Findings
- `nodes.go:144` — `context.Background()` used instead of server shutdown context
- `nodes.go:120-130` — no validation of timeout upper bound
- No mechanism to check if a drain is already in progress for a given node
- On server shutdown, orphaned drain goroutines may hold API server connections

Flagged by: Security Sentinel (Finding 7, Finding 8), Performance Oracle (Finding 5).

## Proposed Solutions
### Option A: Fix all three issues together
(a) Pass server shutdown context as parent to `executeDrain`. (b) Track active drains in `TaskManager` by node name, reject duplicates with 409 Conflict. (c) Clamp timeout to a maximum of 30 minutes.
- **Pros:** Comprehensive fix, all three issues resolved together
- **Cons:** Touches multiple code paths
- **Effort:** Medium (~25 lines)
- **Risk:** Low

### Option B: Incremental fixes
Fix context and timeout cap first (small, low risk), defer deduplication to a separate issue.
- **Pros:** Faster to ship the critical fixes
- **Cons:** Leaves race condition for duplicate drains
- **Effort:** Small (~10 lines for context + timeout)
- **Risk:** Low

## Recommended Action


## Technical Details
- **Affected files:** `backend/internal/k8s/resources/nodes.go`, `tasks.go`
- **Components:** Node drain, goroutine lifecycle, task management

## Acceptance Criteria
- [ ] Drain goroutines are cancelled on server shutdown (use server context as parent)
- [ ] Duplicate drain request for the same node returns 409 Conflict
- [ ] Timeout values greater than 30 minutes are clamped to 30 minutes
- [ ] Test verifies drain cancellation on context cancellation

## Work Log
| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources
- PR: #3
