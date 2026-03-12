---
status: complete
priority: p2
issue_id: "035"
tags: [code-review, correctness]
dependencies: []
---

# Pagination Returns Nondeterministic Results from Unsorted Cache

## Problem Statement
The `paginate[T]` function operates on unsorted informer cache results. Go map iteration order is nondeterministic, so consecutive page requests may return overlapping or missing items if the cache changes between requests. This means a user paging through a resource list could see duplicate items or miss items entirely, undermining the reliability of the pagination feature.

## Findings
- `deployments.go:329-350` — `paginate` function slices directly from unordered cache results
- Informer cache `List()` returns items in arbitrary order
- Problem worsens with larger result sets and higher page numbers

Flagged by: Performance Oracle (Finding 4), Architecture Strategist (Finding 10).

## Proposed Solutions
### Option A: Sort items by namespace+name before paginating
Add a sort step before slicing for pagination. Sort by `namespace/name` to produce deterministic ordering.
- **Pros:** Deterministic results, simple implementation
- **Cons:** Needs a sort function per type or a generic `Name()` accessor; O(n log n) on each request
- **Effort:** Small (~5-10 lines, potentially with a generic sort helper)
- **Risk:** Low

### Option B: Sort by creation timestamp + name as tiebreaker
Use creation timestamp as primary sort key with namespace+name as tiebreaker, giving users a natural chronological default.
- **Pros:** More intuitive default ordering
- **Cons:** Slightly more complex accessor
- **Effort:** Small (~10 lines)
- **Risk:** Low

## Recommended Action


## Technical Details
- **Affected files:** `backend/internal/k8s/resources/deployments.go` (paginate function and all callers)
- **Components:** Pagination, informer cache, resource listing

## Acceptance Criteria
- [ ] Paginated results are in deterministic order across consecutive requests
- [ ] Test verifies stable pagination: page 1 + page 2 items equal the full sorted list
- [ ] Sort key is documented (namespace+name or creation timestamp)

## Work Log
| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources
- PR: #3
