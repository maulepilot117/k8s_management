---
status: pending
priority: p3
issue_id: "101"
tags: [code-review, quality, step-6]
dependencies: []
---

# Overview Components Use Unchecked Type Assertions

## Problem Statement
All 18 overview components cast `K8sResource` to specific types (e.g., `resource as Deployment`) without runtime validation. If the API returns unexpected data or the types drift, components will silently render incorrect or missing data rather than showing an error.

## Findings
- **Agent**: pattern-recognition-specialist, architecture-strategist (PR #6 review)
- **Location**: All files in `frontend/components/k8s/detail/` (e.g., `DeploymentOverview.tsx:7`, `PodOverview.tsx:30`)
- Pattern is consistent across all 18 components — if fixed, should be fixed uniformly

## Proposed Solutions

### Option A: Optional Chaining Defensively (Recommended)
Add optional chaining (`?.`) on deeply nested accesses (e.g., `status?.readyReplicas`). Most components already do this for some fields but not all.
- **Effort**: Small
- **Risk**: Low

### Option B: Runtime Type Guard
Add a `isDeployment(r)` style type guard that checks `apiVersion` and `kind` fields.
- **Effort**: Medium (18 guards needed)
- **Risk**: Low

## Acceptance Criteria
- [ ] Components handle missing/unexpected fields gracefully
- [ ] No runtime crashes from undefined property access
