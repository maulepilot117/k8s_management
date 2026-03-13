---
status: pending
priority: p1
issue_id: "129"
tags: [code-review, frontend, correctness]
dependencies: []
---

# Hardcoded Route Category Ternary in WizardReviewStep

## Problem Statement
`WizardReviewStep.tsx` line 110-112 has a hardcoded ternary that maps `resourceKind` to route category:
```typescript
resourceKind === "deployments" ? "workloads" : "networking"
```
This silently breaks when a third wizard type is added (e.g., ConfigMap under "configuration"). The "View Resource" link would point to the wrong section.

## Findings
- The caller (DeploymentWizard/ServiceWizard) already knows the correct route section
- 3 review agents flagged this as a latent bug

## Proposed Solutions

### Option A: Replace resourceKind with detailBasePath prop (Recommended)
- Change prop from `resourceKind: string` to `detailBasePath: string` (e.g., `/workloads/deployments`)
- Build detail link as `${detailBasePath}/${namespace}/${name}`
- **Pros:** Correct for any wizard type, no mapping logic needed
- **Cons:** Changes component interface
- **Effort:** Small (15 min)
- **Risk:** None

### Option B: Add a mapping constant
- Create `WIZARD_ROUTE_SECTIONS` map
- **Pros:** Keeps current prop
- **Cons:** Still needs updating per wizard, indirection
- **Effort:** Small
- **Risk:** Low

## Recommended Action
Option A — pass the full base path from the caller

## Technical Details
- **Affected files:** `frontend/components/wizard/WizardReviewStep.tsx`, `frontend/islands/DeploymentWizard.tsx`, `frontend/islands/ServiceWizard.tsx`

## Acceptance Criteria
- [ ] `resourceKind` prop replaced with `detailBasePath`
- [ ] Both wizards pass correct base path
- [ ] "View Resource" link works correctly for both wizard types

## Work Log
- 2026-03-13: Created from PR #14 code review

## Resources
- PR: #14
- Agents: code-simplicity-reviewer, pattern-recognition-specialist, architecture-strategist
