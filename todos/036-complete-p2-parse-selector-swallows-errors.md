---
status: complete
priority: p2
issue_id: "036"
tags: [code-review, correctness]
dependencies: []
---

# parseSelector Silently Swallows Errors, Returns All Resources

## Problem Statement
`parseSelector` in `deployments.go:315-324` silently returns `labels.Everything()` on parse error. When a user provides an invalid label selector, they believe they are filtering results but actually receive all resources in the namespace. There is no feedback that their selector was invalid, leading to confusing and incorrect results.

## Findings
- `deployments.go:315-324` — on error, returns `labels.Everything()` with no error propagation
- Users submitting malformed selectors (e.g., typos, invalid operators) get unfiltered results silently
- Pattern may be copied to other resource handlers as the codebase grows

Flagged by: Architecture Strategist (Finding 9).

## Proposed Solutions
### Option A: Return error from parseSelector
Change `parseSelector` to return `(labels.Selector, error)`. In List handlers, when the error is non-nil, return a 400 Bad Request with a descriptive error message including the invalid selector string.
- **Pros:** Clear feedback to users, prevents silent data correctness issues
- **Cons:** Changes function signature, requires updating all callers
- **Effort:** Small (~10 lines across parseSelector + callers)
- **Risk:** Low

### Option B: Log warning and return Everything()
Keep the current behavior but log a warning server-side, and add a response header indicating the selector was invalid.
- **Pros:** Non-breaking change
- **Cons:** Users still get wrong results, header is easy to miss
- **Effort:** Small (~3 lines)
- **Risk:** Low but does not fully solve the problem

## Recommended Action


## Technical Details
- **Affected files:** `backend/internal/k8s/resources/deployments.go` (`parseSelector` and all callers)
- **Components:** Label selector parsing, resource listing, input validation

## Acceptance Criteria
- [ ] Invalid `labelSelector` query parameter returns 400 Bad Request
- [ ] Error message includes the invalid selector string and parse error description
- [ ] Valid selectors continue to work as before
- [ ] Test verifies 400 response for malformed selectors (e.g., `=`, `!!`, `key==value`)

## Work Log
| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources
- PR: #3
