---
status: pending
priority: p2
issue_id: "136"
tags: [code-review, backend, validation]
dependencies: []
---

# Missing ConfigMapRef/SecretRef Mutual Exclusion

## Problem Statement
If a user sets both `configMapRef` and `secretRef` on the same env var, `ToDeployment()` silently picks ConfigMap (first case in switch). Validation should reject this as ambiguous.

## Proposed Solutions

### Option A: Add validation rejecting both refs set simultaneously
- **Effort:** Small (10 min)
- **Risk:** None

## Acceptance Criteria
- [ ] Validation error when both configMapRef and secretRef are set
- [ ] Test added

## Work Log
- 2026-03-13: Created from PR #14 code review

## Resources
- PR: #14 | Agent: pattern-recognition-specialist
