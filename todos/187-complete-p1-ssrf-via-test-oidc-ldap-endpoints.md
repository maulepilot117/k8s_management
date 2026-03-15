---
status: pending
priority: p1
issue_id: "187"
tags: [code-review, security, step-12]
dependencies: []
---

# 187: SSRF via test OIDC/LDAP endpoints — no admin authorization

## Problem Statement

The settings endpoints for testing OIDC and LDAP connectivity are accessible to all authenticated users, not just administrators. These endpoints make outbound HTTP/LDAP connections to user-supplied URLs, enabling any authenticated user to perform Server-Side Request Forgery (SSRF) to scan internal networks and access internal services.

## Findings

- `GET/POST /api/v1/settings/auth/*` routes are protected only by the standard auth middleware (any valid JWT).
- `handleTestOIDC` makes HTTP requests to a user-supplied `issuerURL`, allowing probing of internal HTTP services.
- `handleTestLDAP` connects to a user-supplied LDAP URL, allowing probing of internal LDAP/TCP services.
- An attacker with any valid user account can:
  - Scan internal IP ranges and ports
  - Access cloud metadata endpoints (e.g., `http://169.254.169.254/`)
  - Probe internal services not exposed externally
  - Potentially exfiltrate data from internal HTTP endpoints via error messages

## Proposed Solutions

### Option A: Add admin role check middleware to settings routes

Add an admin authorization middleware to all `/settings/auth/*` routes in `routes.go`. Only users with the admin role should be able to access auth provider configuration and test endpoints.

## Technical Details

**Affected files:**
- `backend/internal/server/routes.go` (add admin-only middleware to `/settings/auth/*` route group)
- `backend/internal/server/handle_settings.go` (optionally add inline role check as defense-in-depth)

**Effort:** Small

## Acceptance Criteria

- [ ] `/api/v1/settings/auth/*` endpoints return 403 for non-admin users
- [ ] Only admin users can trigger OIDC and LDAP test connections
- [ ] Non-admin users cannot supply arbitrary URLs for outbound connections
- [ ] Unit test verifies 403 response for non-admin user on settings endpoints
