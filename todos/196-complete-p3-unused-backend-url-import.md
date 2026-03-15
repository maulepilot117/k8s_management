---
status: pending
priority: p3
issue_id: "196"
tags: [code-review, quality, step-12]
dependencies: []
---

# 196: Unused BACKEND_URL import in AuthProviderButtons.tsx

## Problem Statement
BACKEND_URL is imported from constants in AuthProviderButtons.tsx but never used. This will cause a deno lint failure.

## Findings
- The import statement pulls in BACKEND_URL from `lib/constants.ts`
- No code in the component references BACKEND_URL
- `deno lint` flags unused imports as errors by default

## Technical Details
**Affected files:**
- `frontend/islands/AuthProviderButtons.tsx`

**Effort:** Small

## Acceptance Criteria
- [ ] Unused BACKEND_URL import is removed from AuthProviderButtons.tsx
- [ ] `deno lint` passes without warnings for this file
