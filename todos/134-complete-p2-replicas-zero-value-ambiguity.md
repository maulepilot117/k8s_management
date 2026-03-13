---
status: pending
priority: p2
issue_id: "134"
tags: [code-review, backend, validation]
dependencies: []
---

# Replicas Zero-Value Ambiguity

## Problem Statement
`DeploymentInput.Replicas` is `int32`. A client omitting `replicas` from JSON gets 0 replicas (no pods). While the frontend defaults to 1, the backend API permits this edge case silently.

## Proposed Solutions

### Option A: Default 0 to 1 in ToDeployment()
- **Effort:** Trivial
- **Risk:** None

### Option B: Validate replicas >= 1
- **Effort:** Trivial
- **Risk:** Low — prevents scale-to-zero via wizard

## Acceptance Criteria
- [ ] 0 replicas either defaulted to 1 or rejected with validation error
- [ ] Decision documented

## Work Log
- 2026-03-13: Created from PR #14 code review

## Resources
- PR: #14 | Agent: architecture-strategist
