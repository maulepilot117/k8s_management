---
status: pending
priority: p3
issue_id: "039"
tags: [code-review, security, validation]
dependencies: []
---

# Input Validation for Kubernetes Resource Names

## Problem Statement

Namespace and name URL parameters are passed directly to the Kubernetes API without validation. While the k8s API rejects invalid names, we should validate first to prevent unnecessary API calls and avoid leaking internal API error messages to users.

## Findings

- Namespace and name params extracted from URL are used directly in k8s API calls
- No validation against RFC 1123 DNS label format
- Invalid names cause unnecessary round-trips to the API server
- K8s API error messages may leak internal details to the client

## Proposed Solutions

Add a validateK8sName() helper function that checks against the RFC 1123 DNS label regex with a max length of 253 characters. Call this helper in each handler after extracting URL params. Return a 400 Bad Request before the name reaches the k8s API.

## Recommended Action


## Technical Details

- Affected files: handler.go (add helper), all handler files (add validation call)
- Effort: Small

## Acceptance Criteria

- Invalid Kubernetes names return 400 before hitting the k8s API
- Validation uses RFC 1123 DNS label regex
- Max length of 253 characters enforced
- All existing tests pass

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources

- PR: #3
