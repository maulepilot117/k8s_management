---
status: pending
priority: p3
issue_id: "043"
tags: [code-review, quality, consistency]
dependencies: []
---

# RBAC Viewer Uses Inline String Literals Instead of Constants

## Problem Statement

rbac_viewer.go passes inline string literals ("roles", "clusterroles", "rolebindings", "clusterrolebindings") to checkAccess instead of using kind* constants like every other handler file.

## Findings

- All other handler files use kind* constants (e.g., kindDeployment, kindPod) for checkAccess calls
- rbac_viewer.go uses raw string literals for resource kinds
- Inconsistency makes it easy to introduce typos
- Breaks the established pattern used across the codebase

## Proposed Solutions

Add const declarations (kindRole, kindClusterRole, kindRoleBinding, kindClusterRoleBinding) and use them in all checkAccess calls within rbac_viewer.go.

## Recommended Action


## Technical Details

- Affected files: rbac_viewer.go
- Effort: Tiny

## Acceptance Criteria

- All checkAccess calls in rbac_viewer.go use constants
- No inline string literals for resource kinds remain
- Constants follow the same naming pattern as other handler files

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources

- PR: #3
