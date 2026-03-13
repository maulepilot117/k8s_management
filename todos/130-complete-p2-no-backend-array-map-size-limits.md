---
status: pending
priority: p2
issue_id: "130"
tags: [code-review, backend, security, validation]
dependencies: []
---

# No Backend Limits on Array/Map Sizes in Wizard Inputs

## Problem Statement
Frontend limits ports to 20 and env vars to 50, but backend `Validate()` functions impose no upper bound on array sizes for Ports, EnvVars, Labels, or Selector. An attacker bypassing the frontend could submit thousands of entries within the 1MB body limit.

## Findings
- Labels and Selector maps have no count, key length, or value length limits
- K8s label keys max 253 chars, values max 63 chars
- 1MB body can fit thousands of small entries

## Proposed Solutions

### Option A: Add backend validation limits matching/exceeding frontend
- Labels: max 50, key max 253 chars, value max 63 chars
- Selector: max 20, same length constraints
- Ports: max 20 (deployment), max 100 (service)
- EnvVars: max 100
- **Effort:** Small
- **Risk:** Low

## Acceptance Criteria
- [ ] Backend validates array sizes
- [ ] Label key/value length validated
- [ ] Tests added for limit enforcement

## Work Log
- 2026-03-13: Created from PR #14 code review

## Resources
- PR: #14 | Agent: security-sentinel
