---
status: pending
priority: p1
issue_id: "127"
tags: [code-review, backend, security, stability]
dependencies: []
---

# resource.MustParse Panic on User Input in ToDeployment()

## Problem Statement
`ToDeployment()` in `backend/internal/wizard/deployment.go` calls `resource.MustParse()` which panics on invalid input. While `Validate()` is called first in the current handler flow, `ToDeployment()` is an exported method that can be called without prior validation. A malformed value reaching `MustParse` would crash the entire server process. The `chimw.Recoverer` middleware catches panics but still aborts with a 500 and stack trace.

## Findings
- `resource.MustParse()` called at lines 236-245 for CPU/memory request/limit quantities
- `Validate()` correctly uses `resource.ParseQuantity` (non-panicking) but guards are not structural
- 4 of 6 review agents flagged this as the top finding
- Violates project rule: "never expose internal errors to users"

## Proposed Solutions

### Option A: Change ToDeployment() to return error (Recommended)
- Change signature to `(*appsv1.Deployment, error)`
- Replace `resource.MustParse` with `resource.ParseQuantity`
- Propagate error in handler
- **Pros:** Defense-in-depth, safe for any future caller
- **Cons:** Small API change
- **Effort:** Small (30 min)
- **Risk:** Low

### Option B: Add documented precondition
- Keep `MustParse`, add `// Precondition: Validate() must be called first` comment
- **Pros:** No API change
- **Cons:** Relies on caller discipline, still crashes on violation
- **Effort:** Trivial
- **Risk:** Medium — time bomb

## Recommended Action
Option A — change return signature to `(*appsv1.Deployment, error)`

## Technical Details
- **Affected files:** `backend/internal/wizard/deployment.go`, `backend/internal/wizard/handler.go`
- **Lines:** deployment.go:236-245

## Acceptance Criteria
- [ ] `ToDeployment()` returns `(*appsv1.Deployment, error)`
- [ ] All `resource.MustParse` replaced with `resource.ParseQuantity`
- [ ] Handler propagates error with appropriate HTTP response
- [ ] Tests updated for new return signature
- [ ] No panics possible from user-controlled input

## Work Log
- 2026-03-13: Created from PR #14 code review (6-agent analysis)

## Resources
- PR: #14
- Agents: security-sentinel, performance-oracle, architecture-strategist, pattern-recognition-specialist
