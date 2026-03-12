---
status: pending
priority: p3
issue_id: "038"
tags: [code-review, quality]
dependencies: []
---

# Replace Custom itoa() with strconv.Itoa

## Problem Statement

tasks.go:109-113 has a hand-rolled recursive itoa() function that doesn't handle negatives and could stack overflow for large numbers. The standard library strconv.Itoa exists and handles all cases correctly.

## Findings

- Custom itoa() at tasks.go:109-113 is recursive without a depth guard
- Does not handle negative numbers
- strconv.Itoa is a drop-in replacement
- resources_test.go also uses itoa in TestPagination

## Proposed Solutions

Replace all usages of itoa() with strconv.Itoa(). Delete the custom itoa() function entirely.

## Recommended Action


## Technical Details

- Affected files: tasks.go, resources_test.go (uses itoa in TestPagination)
- Effort: Tiny (5 min)

## Acceptance Criteria

- Custom itoa function is removed
- strconv.Itoa is used everywhere itoa was previously called
- All existing tests pass

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources

- PR: #3
