---
status: pending
priority: p3
issue_id: "042"
tags: [code-review, quality]
dependencies: []
---

# Remove Duplicate Informer Factory in Test Helper

## Problem Statement

testHandler() creates a local SharedInformerFactory (lines 35-54), starts and syncs it, then creates an InformerManager via NewInformerManager which creates a second factory internally. The local factory is wasted work.

## Findings

- testHandler() creates and starts a SharedInformerFactory at lines 35-54
- NewInformerManager creates its own internal factory
- The locally created factory is never used after InformerManager takes over
- This doubles the informer setup work in every test

## Proposed Solutions

Remove lines 35-59 (local factory creation, start, and sync). Keep only the InformerManager path which handles its own factory lifecycle.

## Recommended Action


## Technical Details

- Affected files: resources_test.go
- Effort: Tiny

## Acceptance Criteria

- testHandler creates only one informer factory
- All existing tests pass without modification

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources

- PR: #3
