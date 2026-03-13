---
status: pending
priority: p2
issue_id: "135"
tags: [code-review, backend, validation]
dependencies: []
---

# Missing Duplicate Port Validation for ServiceInput

## Problem Statement
`DeploymentInput.Validate()` checks for duplicate container ports, but `ServiceInput.Validate()` has no equivalent. Duplicate service ports cause k8s API rejection with an opaque SSA error instead of a clean 422 field error.

## Proposed Solutions

### Option A: Add seenPorts tracking matching deployment pattern
- **Effort:** Small (10 min)
- **Risk:** None

## Acceptance Criteria
- [ ] Service validator rejects duplicate port numbers
- [ ] Tests added

## Work Log
- 2026-03-13: Created from PR #14 code review

## Resources
- PR: #14 | Agent: pattern-recognition-specialist
