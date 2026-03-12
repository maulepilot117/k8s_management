---
status: pending
priority: p3
issue_id: "041"
tags: [code-review, testing]
dependencies: []
---

# Test Coverage Gaps for Resource Handlers

## Problem Statement

14 tests cover only read paths (List/Get) for a few resource types. No tests exist for Create/Update/Delete, Scale, Rollback, Restart, Cordon, Drain, or Secret Reveal. No tests for access-denied paths. No tests for 10+ resource type handlers.

## Findings

- Existing tests cover only List and Get operations
- Write operations (Create, Update, Delete) have zero test coverage
- Specialized operations (Scale, Rollback, Restart, Cordon, Drain, Secret Reveal) untested
- Access-denied scenarios not tested despite AccessChecker being mockable
- 10+ resource type handlers have no tests at all

## Proposed Solutions

Add write operation tests using the fake clientset (already returned from testHandler). Add access-denied tests with a denying AccessChecker. Expand resource type coverage incrementally.

## Recommended Action


## Technical Details

- Affected files: resources_test.go
- Effort: Medium

## Acceptance Criteria

- Test coverage includes at least 1 Create test
- Test coverage includes at least 1 Update test
- Test coverage includes at least 1 Delete test
- Test coverage includes at least 1 access-denied scenario
- All tests pass

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources

- PR: #3
