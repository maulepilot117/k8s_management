---
status: complete
priority: p2
issue_id: "034"
tags: [code-review, api-design]
dependencies: []
---

# Create Response Envelope Inconsistent with Get/Update

## Problem Statement
Create handlers use `writeJSON(w, 201, created)` which writes the raw Kubernetes object directly. Get and Update handlers use `writeData` which wraps the response in a `{data: ...}` envelope. API consumers see different response shapes for Create vs Get/Update operations on the same resource type, making client-side parsing inconsistent. Additional inconsistencies exist in nodes.go (drain returns raw map) and tasks.go (manual wrapping).

## Findings
- All Create handlers (e.g., `configmaps.go`, `services.go`, `ingresses.go`) use `writeJSON(201, created)` — raw object
- Get/Update handlers use `writeData(w, obj)` — wrapped in `{data: ...}` envelope
- `nodes.go` drain handler returns a raw map
- `tasks.go` wraps manually with ad-hoc structure
- API Response Format in CLAUDE.md specifies `{data: ...}` envelope for all responses

Flagged by: Pattern Recognition (Finding 4).

## Proposed Solutions
### Option A: Add writeCreated helper
Create a `writeCreated` helper that wraps the response in `api.Response{Data: created}` with a 201 status code, and replace all `writeJSON(w, 201, created)` calls.
- **Pros:** Consistent envelope, single helper, easy to audit
- **Cons:** Touches ~15 locations
- **Effort:** Small (~15 locations + 1 new helper)
- **Risk:** Low (may require frontend adjustments if clients already handle raw responses)

### Option B: Make writeJSON always wrap in envelope
Modify `writeJSON` to always wrap in the standard envelope, with status code passed through.
- **Pros:** Prevents future inconsistencies
- **Cons:** May break non-resource responses (health check, etc.) that intentionally bypass the envelope
- **Effort:** Small
- **Risk:** Medium (broader impact)

## Recommended Action


## Technical Details
- **Affected files:** All Create handlers in `backend/internal/k8s/resources/`, `handler.go` (add `writeCreated`)
- **Components:** API response serialization, response envelope format

## Acceptance Criteria
- [ ] POST (Create) responses use the same `{data: ...}` envelope as GET and PUT responses
- [ ] 201 status code is preserved for Create responses
- [ ] Drain and task status responses also use the standard envelope
- [ ] Frontend API client handles the consistent response format

## Work Log
| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources
- PR: #3
