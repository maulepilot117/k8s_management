---
status: pending
priority: p3
issue_id: "100"
tags: [code-review, quality, step-6]
dependencies: []
---

# Age Values Become Stale Without Periodic Re-Render

## Problem Statement
The `age()` function in `lib/format.ts` computes relative time from `Date.now()`. Once rendered, the displayed age (e.g., "5m") never updates until the component re-renders for another reason. A resource created 2 minutes ago will still show "2m" after 30 minutes if nothing triggers a re-render.

## Findings
- **Agent**: code-simplicity-reviewer (PR #6 review)
- **Location**: `frontend/islands/ResourceDetail.tsx:369` — `{age(resource.value.metadata.creationTimestamp)}`
- Also affects: `frontend/islands/ResourceDetail.tsx:472` (events last seen), MetadataSection

## Proposed Solutions

### Option A: Periodic Signal Tick (Recommended)
Add a `useSignal`-based tick that increments every 30-60 seconds, forcing age values to recompute. Simple and covers all age displays.
- **Effort**: Small
- **Risk**: Low

## Acceptance Criteria
- [ ] Age values update periodically without user interaction
- [ ] No excessive re-rendering (30-60s interval is sufficient)
