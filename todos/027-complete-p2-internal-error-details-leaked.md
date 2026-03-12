---
status: complete
priority: p2
issue_id: "027"
tags: [code-review, security]
dependencies: []
---

# Internal Error Details Leaked in API Responses

## Problem Statement
Approximately 35+ locations pass `err.Error()` as the `detail` field in API error responses, leaking internal infrastructure details such as API server addresses, Go type names, and certificate paths. CLAUDE.md explicitly states "never expose internal errors to users." This creates an information disclosure vulnerability that could aid attackers in mapping internal infrastructure.

## Findings
- `handler.go:75` — `checkAccess` error path passes raw error to client
- `errors.go:54-58` — fallback default case in error mapper exposes raw error string
- Every `impersonatingClient` error path across all resource handlers leaks connection details
- Raw Kubernetes API server errors contain internal hostnames, port numbers, and certificate paths

Flagged by: Security Sentinel (Finding 5), Architecture Strategist (Finding 4), Pattern Recognition (Finding 5).

## Proposed Solutions
### Option A: Sanitize all writeError calls
Log the full error server-side via `h.Logger.Error`, replace `detail` with a generic string (e.g., "An internal error occurred. Check server logs for details." or a request-ID-based reference).
- **Pros:** Comprehensive fix, aligns with CLAUDE.md security rules
- **Cons:** Touches ~35 locations, requires careful review of each call site
- **Effort:** Medium
- **Risk:** Low

### Option B: Sanitize at the writeError level
Modify `writeError` itself to strip internal details from the `detail` field when not in dev mode, logging the original error server-side.
- **Pros:** Single point of change, catches future violations automatically
- **Cons:** May lose useful context in some error messages that are already safe to expose
- **Effort:** Small
- **Risk:** Low (with dev-mode escape hatch)

## Recommended Action


## Technical Details
- **Affected files:** `handler.go`, `errors.go`, all resource handler files in `backend/internal/k8s/resources/`
- **Components:** Error handling, API response serialization, logging

## Acceptance Criteria
- [ ] No `err.Error()` passed as `detail` in any `writeError` call
- [ ] All raw errors logged server-side with request context (request ID, user, resource)
- [ ] API error responses contain generic detail strings or request-ID references
- [ ] Dev mode optionally preserves detailed errors for debugging

## Work Log
| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources
- PR: #3
