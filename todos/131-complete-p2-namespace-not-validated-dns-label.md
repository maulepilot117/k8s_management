---
status: pending
priority: p2
issue_id: "131"
tags: [code-review, backend, validation]
dependencies: []
---

# Namespace Not Validated Against DNS Label Format

## Problem Statement
Both wizard `Validate()` functions check that Namespace is non-empty but do not validate it against `dnsLabelRegex`. The Name field uses this regex, but Namespace does not. Invalid namespace values would be rejected by k8s API, but error messages may leak internal information.

## Findings
- `deployment.go` lines 87-89 and `service.go` lines 37-39
- `dnsLabelRegex` already exists and is used for Name validation

## Proposed Solutions

### Option A: Apply dnsLabelRegex to namespace in both validators
- **Effort:** Trivial (5 min)
- **Risk:** None

## Acceptance Criteria
- [ ] Namespace validated with `dnsLabelRegex` in both deployment and service validators
- [ ] Tests added

## Work Log
- 2026-03-13: Created from PR #14 code review

## Resources
- PR: #14 | Agent: security-sentinel
