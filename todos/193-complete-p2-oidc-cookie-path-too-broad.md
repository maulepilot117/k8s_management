---
status: pending
priority: p2
issue_id: "193"
tags: [code-review, security, step-12]
dependencies: []
---

# 193: OIDC access token cookie path too broad

## Problem Statement

The `oidc_access_token` cookie is set with `Path: "/"`, causing it to be sent with every request to the application for its 60-second lifetime. The cookie is only needed by the token exchange endpoint and should be scoped to minimize exposure.

## Findings

- The OIDC callback handler sets the `oidc_access_token` cookie with `Path: "/"`.
- This cookie is only consumed by the `/api/auth/oidc-token-exchange` endpoint.
- With `Path: "/"`, the cookie is unnecessarily included in all requests (static assets, API calls, WebSocket connections) for 60 seconds.
- Broader cookie scope increases the attack surface if any other endpoint has a vulnerability that could leak cookie values.

## Technical Details

**Affected files:**
- `backend/internal/server/handle_auth.go` (or wherever the OIDC callback sets the cookie)
- `frontend/routes/api/auth/oidc-token-exchange.ts`

## Acceptance Criteria

- [ ] Set the `oidc_access_token` cookie with `Path: "/api/auth/oidc-token-exchange"` (or the equivalent backend path)
- [ ] Verify the token exchange endpoint still receives the cookie correctly
- [ ] Verify the cookie is not sent with unrelated requests (e.g., static asset loads)
- [ ] Update any cookie deletion/cleanup code to use the matching path
