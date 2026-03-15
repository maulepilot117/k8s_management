---
status: pending
priority: p3
issue_id: "198"
tags: [code-review, quality, step-12]
dependencies: []
---

# 198: Duplicate ProviderInfo TypeScript type defined in two islands

## Problem Statement
The same ProviderInfo interface is defined in both AuthProviderButtons.tsx and AuthSettings.tsx. This violates DRY and risks the definitions drifting apart.

## Findings
- ProviderInfo type/interface is duplicated across two island components
- Both definitions describe the same shape (auth provider metadata from the backend)
- Shared types should live in `frontend/lib/k8s-types.ts` per project conventions
- If the backend response shape changes, both copies must be updated independently

## Technical Details
**Affected files:**
- `frontend/islands/AuthProviderButtons.tsx` (duplicate definition)
- `frontend/islands/AuthSettings.tsx` (duplicate definition)
- `frontend/lib/k8s-types.ts` (target location for shared type)

**Effort:** Small

## Acceptance Criteria
- [ ] ProviderInfo type is defined once in `frontend/lib/k8s-types.ts`
- [ ] Both islands import ProviderInfo from the shared location
- [ ] No duplicate type definitions remain
