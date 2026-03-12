---
status: pending
priority: p3
issue_id: "040"
tags: [code-review, quality, yagni]
dependencies: []
---

# Remove Unused Fields (YAGNI)

## Problem Statement

ListParams.FieldSelector (handler.go:31) is parsed but never used by any handler — informer listers don't support field selectors. Metadata.Page and Metadata.PageSize (types.go) are defined but never populated — pagination uses continue tokens, not page numbers.

## Findings

- ListParams.FieldSelector is parsed from query params but no handler uses it
- Informer-backed listers do not support field selectors, so the field is misleading
- Metadata.Page and Metadata.PageSize are defined in the response type but never set
- Pagination is implemented via continue tokens, making page/pageSize fields dead code

## Proposed Solutions

Remove FieldSelector from ListParams and from parseListParams(). Remove Page and PageSize from Metadata struct. Clean up any references.

## Recommended Action


## Technical Details

- Affected files: handler.go, pkg/api/types.go
- Effort: Tiny

## Acceptance Criteria

- No unused fields remain in ListParams or Metadata
- Existing tests pass without modification

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources

- PR: #3
