---
status: pending
priority: p1
issue_id: "185"
tags: [code-review, security, step-12]
dependencies: []
---

# 185: Kubernetes group name mismatch: "kubecenter:users" vs "k8scenter:users"

## Problem Statement

Local auth provider and OIDC/LDAP auth providers assign different default Kubernetes group names to users. This causes RBAC policies targeting one group name to not apply to users from other providers, creating a security gap where users may have more or fewer permissions than intended.

## Findings

- `backend/internal/auth/local.go` line 155 assigns group `"kubecenter:users"` to local users.
- `backend/internal/auth/oidc.go` line 271 assigns group `"k8scenter:users"` to OIDC users.
- `backend/internal/auth/ldap.go` line 287 assigns group `"k8scenter:users"` to LDAP users.
- Any ClusterRoleBinding or RoleBinding targeting `"k8scenter:users"` will not apply to local users, and vice versa.
- This is a silent failure — no error is raised, users simply get unexpected permissions.

## Proposed Solutions

### Option A: Unify to "k8scenter:users" everywhere

Change `local.go` line 155 from `"kubecenter:users"` to `"k8scenter:users"` to match the OIDC and LDAP providers. This aligns with the current project name (k8sCenter).

## Technical Details

**Affected files:**
- `backend/internal/auth/local.go` (line 155 — change `"kubecenter:users"` to `"k8scenter:users"`)
- `backend/internal/auth/oidc.go` (line 271 — already correct)
- `backend/internal/auth/ldap.go` (line 287 — already correct)

**Effort:** Small

## Acceptance Criteria

- [ ] All three auth providers (local, OIDC, LDAP) assign the same default Kubernetes group name
- [ ] Group name is `"k8scenter:users"` consistently
- [ ] Existing RBAC policies referencing `"k8scenter:users"` apply to all user types
- [ ] Unit tests updated to verify the correct group name for local users
