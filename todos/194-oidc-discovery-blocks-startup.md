---
status: pending
priority: p3
issue_id: "194"
tags: [code-review, performance, step-12]
dependencies: []
---

# 194: OIDC discovery blocks startup — should be lazy

## Problem Statement
NewOIDCProvider performs OIDC discovery synchronously during main() startup. A slow or unreachable provider blocks server start for up to 15s per provider, causing readiness probe failures during rolling updates in Kubernetes.

## Findings
- OIDC discovery (fetching .well-known/openid-configuration) happens in the constructor
- If the identity provider is temporarily unreachable, the entire KubeCenter backend fails to start
- In k8s rolling update scenarios, the new pod fails readiness checks and the rollout stalls
- Multiple configured OIDC providers compound the delay (15s timeout x N providers)

## Technical Details
**Affected files:**
- `backend/internal/auth/oidc.go` (NewOIDCProvider constructor)
- `backend/cmd/kubecenter/main.go` (provider initialization during startup)

**Effort:** Medium

## Acceptance Criteria
- [ ] OIDC discovery is deferred to first login redirect using sync.Once
- [ ] Server starts immediately regardless of OIDC provider reachability
- [ ] Discovery errors are returned at login time, not startup time
- [ ] Readiness probe passes without waiting for OIDC discovery
- [ ] Background retry with backoff if initial lazy discovery fails
