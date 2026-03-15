---
status: pending
priority: p2
issue_id: "189"
tags: [code-review, security, step-12]
dependencies: []
---

# 189: BFF token exchange endpoint lacks CSRF protection

## Problem Statement

The frontend OIDC token exchange endpoint reads an httpOnly cookie and returns a JWT in the response body without requiring a CSRF protection header. Since the cookie is SameSite=Lax, it will be sent on cross-origin top-level navigations, creating a 60-second CSRF window after the OIDC callback completes.

## Findings

- `frontend/routes/api/auth/oidc-token-exchange.ts` reads the httpOnly `oidc_access_token` cookie and returns a JWT access token in the JSON response body.
- No `X-Requested-With` header check is performed on the POST request.
- The cookie is set with `SameSite=Lax`, which means it is sent on cross-origin navigations (e.g., a link or form POST from an attacker site).
- The 60-second TTL of the cookie creates a window during which CSRF is possible after a user completes the OIDC callback flow.

## Technical Details

**Affected files:**
- `frontend/routes/api/auth/oidc-token-exchange.ts`

## Acceptance Criteria

- [ ] POST `/api/auth/oidc-token-exchange` requires `X-Requested-With: XMLHttpRequest` header
- [ ] Requests without the header return 403 Forbidden
- [ ] Frontend OIDC callback code sends the header when calling the token exchange endpoint
- [ ] Unit test covers rejection of requests missing the CSRF header
