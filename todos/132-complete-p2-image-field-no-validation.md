---
status: pending
priority: p2
issue_id: "132"
tags: [code-review, backend, validation, security]
dependencies: []
---

# Image Field Accepts Arbitrary Strings Without Validation

## Problem Statement
The deployment wizard `Image` field is only checked for being non-empty. No validation on format or length. An attacker could submit extremely long image strings (up to 1MB minus other fields).

## Proposed Solutions

### Option A: Add basic image validation
- Max length 512 characters, no whitespace or control characters
- **Effort:** Trivial
- **Risk:** None

## Acceptance Criteria
- [ ] Image field validated for max length and no control characters
- [ ] Tests added

## Work Log
- 2026-03-13: Created from PR #14 code review

## Resources
- PR: #14 | Agent: security-sentinel
