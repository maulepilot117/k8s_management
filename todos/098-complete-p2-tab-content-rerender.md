---
status: pending
priority: p2
issue_id: "098"
tags: [code-review, performance, step-6]
dependencies: []
---

# Tab Panels Re-Render on Unrelated Signal Changes

## Problem Statement
Tab panel content functions (`tab.content()`) are called on every render cycle. Since ResourceDetail uses Preact signals, any signal change (e.g., `loading`, `updated`, `deleted`) triggers re-execution of ALL mounted tab content functions, even when the tab is hidden. The YAML tab is particularly affected since it calls `stringify()`.

## Findings
- **Agent**: performance-oracle, code-simplicity-reviewer (PR #6 review)
- **Location**: `frontend/components/ui/Tabs.tsx:120` — `{tab.content()}` is called for every mounted panel
- **Partially mitigated**: YAML content is now memoized via `useMemo` (P1 fix), but other tab content functions still re-execute

## Proposed Solutions

### Option A: Memoize Tab Content with Components (Recommended)
Change `TabDef.content` from `() => ComponentChildren` to a Preact component. Use `<tab.Content />` instead of `tab.content()` so Preact can skip re-renders via VDOM diffing.
- **Pros**: Idiomatic Preact, natural memoization
- **Cons**: Requires refactoring tab definitions
- **Effort**: Small
- **Risk**: Low

### Option B: Add `shouldUpdate` Guard to Tabs
Track a version/hash per tab and skip re-rendering when content hasn't changed.
- **Pros**: No API change to TabDef
- **Cons**: Complex, fragile
- **Effort**: Medium
- **Risk**: Medium

## Acceptance Criteria
- [ ] Hidden tab panels do not re-execute their content function on unrelated signal changes
- [ ] Tab switching remains instant
- [ ] No visual regressions
