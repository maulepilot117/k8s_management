---
status: complete
priority: p2
issue_id: "031"
tags: [code-review, performance, security]
dependencies: []
---

# Rollback Lists All ReplicaSets in Namespace Without Label Selector

## Problem Statement
`HandleRollbackDeployment` lists ALL ReplicaSets in the namespace with no label selector (`deployments.go:236`). In large namespaces with many deployments (e.g., 500 deployments x 10 revisions = 5,000 ReplicaSets), this fetches 10-25MB of data per rollback request. It also exposes ReplicaSets belonging to other deployments, which is a minor information leak and makes revision matching fragile.

## Findings
- `deployments.go:236` — `ReplicaSets(ns).List(ctx, metav1.ListOptions{})` with empty options
- The deployment's own label selector is available but unused for filtering
- Each unnecessary ReplicaSet object is ~2-5KB, scaling linearly with namespace size

Flagged by: Security Sentinel (Finding 10), Performance Oracle (Finding 3).

## Proposed Solutions
### Option A: Filter by deployment's label selector
Fetch the deployment from cache, extract its `Spec.Selector`, and use it as `LabelSelector` in the ReplicaSet list call: `LabelSelector: metav1.FormatLabelSelector(dep.Spec.Selector)`.
- **Pros:** Correct scoping, massive performance improvement in large namespaces, server-side filtering
- **Cons:** Requires fetching deployment first (already available in handler context)
- **Effort:** Small (~10 lines)
- **Risk:** Low

### Option B: Filter client-side after fetch
Keep the broad list but filter ReplicaSets by matching owner references to the target deployment.
- **Pros:** Simpler logic, no selector formatting needed
- **Cons:** Still fetches all RS from API server, wastes bandwidth and memory
- **Effort:** Small (~5 lines)
- **Risk:** Low but suboptimal

## Recommended Action


## Technical Details
- **Affected files:** `backend/internal/k8s/resources/deployments.go`
- **Components:** Deployment rollback, ReplicaSet listing, label selectors

## Acceptance Criteria
- [ ] Rollback only fetches ReplicaSets matching the target deployment's label selector
- [ ] Test with multiple deployments in same namespace verifies only relevant RS are returned
- [ ] Rollback performance is not degraded in namespaces with many deployments

## Work Log
| Date | Action | Notes |
|------|--------|-------|
| 2026-03-12 | Created | From PR #3 code review |

## Resources
- PR: #3
