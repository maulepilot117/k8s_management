---
status: complete
priority: p2
issue_id: "029"
tags: [code-review, security, rbac]
dependencies: []
---

# Access Cache Key Omits Groups, Allowing Stale RBAC Decisions

## Problem Statement
The `AccessChecker` cache key in `access.go:17-22` includes only the username, not the user's groups. This means if group membership changes within the 60-second TTL, stale permissions persist. Additionally, two users with the same Kubernetes username but different group memberships would share cache entries, potentially granting or denying access incorrectly.

## Findings
- `access.go:17-22` — `accessCacheKey` struct contains only username, namespace, resource, and verb
- `access.go:53` — `CanAccess` method accepts groups parameter but the cache lookup ignores them entirely
- Kubernetes RBAC frequently grants permissions via ClusterRoleBindings to groups, making this a significant gap

Flagged by: Security Sentinel (Finding 9), Architecture Strategist (Finding 6), Pattern Recognition (Finding 6).

## Proposed Solutions
### Option A: Include sorted groups in cache key
Sort the groups slice, join with a delimiter, and include the resulting string in `accessCacheKey`.
- **Pros:** Correct cache behavior, minimal code change
- **Cons:** Increases cache cardinality slightly (same user with different group sets gets separate entries)
- **Effort:** Small (~5 lines)
- **Risk:** Low

### Option B: Hash groups into cache key
Sort and hash the groups into a fixed-length string to keep cache keys compact.
- **Pros:** Bounded key size regardless of group count
- **Cons:** Slightly more complex, hash collision risk (negligible in practice)
- **Effort:** Small (~8 lines)
- **Risk:** Low

## Recommended Action


## Technical Details
- **Affected files:** `backend/internal/k8s/resources/access.go`
- **Components:** RBAC access checking, cache key generation

## Acceptance Criteria
- [ ] Cache key includes user groups in a deterministic order
- [ ] Test verifies that different groups for the same user produce different cache entries
- [ ] Test verifies that same groups in different order produce the same cache entry

## Work Log
| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources
- PR: #3
