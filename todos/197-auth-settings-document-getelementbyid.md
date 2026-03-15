---
status: pending
priority: p3
issue_id: "197"
tags: [code-review, quality, step-12]
dependencies: []
---

# 197: AuthSettings.tsx uses document.getElementById instead of signals

## Problem Statement
AuthSettings.tsx reads form input values via document.getElementById, breaking the Preact reactive model. This is inconsistent with the pattern used in LoginForm.tsx which uses signals for input state.

## Findings
- Form inputs are read imperatively via DOM queries instead of declarative signal bindings
- This bypasses Preact's virtual DOM diffing and reactive update model
- LoginForm.tsx already demonstrates the correct pattern using useSignal
- Imperative DOM access in islands creates fragile code that breaks with SSR/hydration edge cases

## Technical Details
**Affected files:**
- `frontend/islands/AuthSettings.tsx`

**Effort:** Small

## Acceptance Criteria
- [ ] All document.getElementById calls replaced with useSignal-based state management
- [ ] Input elements use value/onInput bindings tied to signals
- [ ] Pattern is consistent with LoginForm.tsx approach
