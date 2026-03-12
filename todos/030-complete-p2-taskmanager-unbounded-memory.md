---
status: complete
priority: p2
issue_id: "030"
tags: [code-review, performance, memory]
dependencies: []
---

# TaskManager and AccessChecker Have Unbounded Memory Growth

## Problem Statement
`TaskManager.tasks` map grows without bound — tasks are created but never deleted. Over time in a long-running cluster, this constitutes a memory leak proportional to the number of drain operations performed. Similarly, `AccessChecker.cache` (sync.Map) has no eviction for entries that are never re-read after expiry — the `delete` call only occurs on cache miss (line 69), so entries never queried again remain in memory forever.

## Findings
- `tasks.go` — no cleanup mechanism for completed tasks; map grows monotonically
- `access.go` — expired entries deleted only on cache miss (line 69); entries never re-read persist indefinitely
- Both follow the same anti-pattern: time-based expiry checked at read time with no background sweep

Flagged by: Performance Oracle (Finding 1, Finding 2), Architecture Strategist (Finding 5).

## Proposed Solutions
### Option A: Add reaper goroutine for both caches
Implement a background goroutine (same pattern as `ClientFactory.StartCacheSweeper`) that periodically sweeps both stores — deleting completed tasks older than 1 hour and expired access entries every 60 seconds.
- **Pros:** Consistent with existing codebase pattern, bounded memory growth
- **Cons:** Adds background goroutine lifecycle management
- **Effort:** Small (~30 lines total)
- **Risk:** Low

### Option B: Use a bounded LRU cache
Replace sync.Map with a bounded LRU cache (e.g., `hashicorp/golang-lru`) that automatically evicts oldest entries when capacity is reached.
- **Pros:** Hard memory bound, no background goroutine needed
- **Cons:** Adds dependency, may evict still-valid entries under pressure
- **Effort:** Medium
- **Risk:** Low

## Recommended Action


## Technical Details
- **Affected files:** `tasks.go`, `access.go`
- **Components:** Task management, RBAC access cache, memory management

## Acceptance Criteria
- [ ] Completed tasks are evicted after a configurable TTL (default 1 hour)
- [ ] Expired access cache entries are swept periodically (every 60 seconds)
- [ ] Memory usage stays bounded under sustained load testing
- [ ] Reaper goroutines respect server shutdown context

## Work Log
| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources
- PR: #3
