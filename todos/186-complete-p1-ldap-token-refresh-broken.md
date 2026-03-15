---
status: pending
priority: p1
issue_id: "186"
tags: [code-review, architecture, step-12]
dependencies: []
---

# 186: LDAP token refresh is broken — sessions expire after 15 minutes

## Problem Statement

LDAP users cannot refresh their JWT access tokens. After the 15-minute access token expires, they are forced to re-authenticate with username and password. This makes the platform unusable for LDAP users in any sustained session.

## Findings

- `LDAPProvider` does not implement `UserLookup`, so there is no way to resolve an LDAP user from a stored session during token refresh.
- In `backend/internal/server/response.go` line 66, the `CachedUser` (used for token refresh) is only populated when `user.Provider == "oidc"`.
- LDAP users are excluded from the caching path, so when their access token expires and the frontend calls `/api/v1/auth/refresh`, the server cannot resolve the user and the refresh fails.
- Local users are unaffected because they look up the user from the local store. OIDC users are unaffected because they are explicitly cached.

## Proposed Solutions

### Option A: Cache user info for all non-local providers

Change the condition in `response.go` line 66 from `user.Provider == "oidc"` to `user.Provider != "local"` (or remove the condition entirely and cache for all providers). This ensures LDAP users' identity information is available during token refresh without requiring an LDAP bind.

### Option B: Implement UserLookup on LDAPProvider

Add a `UserLookup` method to `LDAPProvider` that performs an LDAP search to resolve the user during refresh. This is more correct but adds latency and requires the LDAP server to be reachable during refresh.

## Technical Details

**Affected files:**
- `backend/internal/server/response.go` (line 66 — broaden the caching condition)
- `backend/internal/auth/ldap.go` (optionally add `UserLookup` interface implementation)

**Effort:** Small

## Acceptance Criteria

- [ ] LDAP users can refresh their access tokens without re-authenticating
- [ ] Sessions persist across the 15-minute access token expiry
- [ ] Token refresh returns a valid new access token for LDAP users
- [ ] Integration test verifies LDAP user token refresh flow
