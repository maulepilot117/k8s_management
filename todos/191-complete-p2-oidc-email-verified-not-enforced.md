---
status: pending
priority: p2
issue_id: "191"
tags: [code-review, security, step-12]
dependencies: []
---

# 191: OIDC email_verified claim not enforced

## Problem Statement

The OIDC provider parses the `email_verified` claim from the ID token but never checks its value. When the username claim is set to `email`, an attacker could register with a victim's email address at the identity provider, receive `email_verified=false`, and still authenticate as that user in KubeCenter.

## Findings

- The `oidcClaims` struct includes an `EmailVerified` field that is populated from the ID token.
- `HandleCallback` never inspects the `EmailVerified` value before proceeding with authentication.
- If `UsernameClaim` is configured as `"email"`, the unverified email is used as the user's identity.
- This enables account impersonation if the IdP allows registration with unverified email addresses.

## Technical Details

**Affected files:**
- `backend/internal/auth/oidc.go`

## Acceptance Criteria

- [ ] When `UsernameClaim` is `"email"`, reject authentication if `email_verified` is `false` or absent
- [ ] Return a clear error message explaining that the email address is not verified at the identity provider
- [ ] Log the rejected authentication attempt at WARN level with the unverified email
- [ ] Add unit test covering the rejection of unverified email claims
