---
status: pending
priority: p3
issue_id: "099"
tags: [code-review, quality, step-6]
dependencies: []
---

# CodeBlock Copy Fallback Uses Deprecated `document.execCommand`

## Problem Statement
The CodeBlock component's copy-to-clipboard fallback uses `document.execCommand("copy")`, which is deprecated. This fallback is only triggered in non-secure contexts (HTTP without TLS), which shouldn't occur in production (TLS required per CLAUDE.md), but could affect local dev.

## Findings
- **Agent**: code-simplicity-reviewer (PR #6 review)
- **Location**: `frontend/components/ui/CodeBlock.tsx:27-36`

## Proposed Solutions

### Option A: Remove Fallback Entirely (Recommended)
Since KubeCenter requires TLS in production, `navigator.clipboard.writeText` will always be available. Remove the fallback and let the catch block show a user-friendly error.
- **Effort**: Small
- **Risk**: Low (only affects insecure local dev)

## Acceptance Criteria
- [ ] Copy button works in secure contexts
- [ ] Graceful error handling if clipboard API unavailable
