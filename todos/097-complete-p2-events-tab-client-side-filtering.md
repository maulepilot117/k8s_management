---
status: pending
priority: p2
issue_id: "097"
tags: [code-review, performance, step-6]
dependencies: []
---

# Events Tab O(n) Client-Side Filtering

## Problem Statement
The Events tab in ResourceDetail fetches ALL events in a namespace via `/v1/resources/events/{namespace}`, then filters client-side by `involvedObject.kind` and `involvedObject.name`. In namespaces with thousands of events, this transfers excessive data and runs a linear scan in the browser.

## Findings
- **Agent**: performance-oracle (PR #6 review)
- **Location**: `frontend/islands/ResourceDetail.tsx:176-188`
- The backend already has field selector support on informers; a server-side `fieldSelector=involvedObject.name=X` query would be far more efficient
- Not a blocker for Step 6 since event counts are typically manageable, but will degrade at scale

## Proposed Solutions

### Option A: Server-Side Field Selector (Recommended)
Add `fieldSelector` query param support to the events endpoint. The Kubernetes API natively supports `involvedObject.name` and `involvedObject.kind` field selectors.
- **Pros**: Minimal data transfer, no client filtering needed
- **Cons**: Requires backend changes to events handler
- **Effort**: Small
- **Risk**: Low

### Option B: Dedicated Events-For-Resource Endpoint
Add `GET /v1/resources/:kind/:namespace/:name/events` that returns pre-filtered events.
- **Pros**: Clean API, single call
- **Cons**: New endpoint per resource type
- **Effort**: Medium
- **Risk**: Low

## Acceptance Criteria
- [ ] Events tab loads only events related to the viewed resource
- [ ] No client-side filtering of unrelated events
- [ ] Works for both namespaced and cluster-scoped resources
