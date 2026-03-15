---
status: pending
priority: p3
issue_id: "199"
tags: [code-review, quality, step-12]
dependencies: []
---

# 199: Mixed JSON/redirect error responses in OIDC callback handler

## Problem Statement
The OIDC callback handler inconsistently returns JSON error responses for some error paths and redirect responses for others. Since the callback is browser-facing, JSON errors are displayed as raw text instead of a user-friendly login page.

## Findings
- Some error paths (e.g., missing state or code parameters) return writeJSON with 400 status
- Other error paths redirect to `/login?error=...` which renders a proper error message
- Users hitting JSON error paths see raw `{"error":{"code":400,...}}` in the browser
- The callback endpoint is always accessed via browser redirect from the identity provider, never via API client

## Technical Details
**Affected files:**
- `backend/internal/server/handle_auth.go` (OIDC callback handler)

**Effort:** Small

## Acceptance Criteria
- [ ] All error paths in the OIDC callback handler use redirects to `/login?error=<message>`
- [ ] No raw JSON responses are returned from the callback endpoint
- [ ] Error messages in redirect query params are URL-encoded and user-friendly
- [ ] Detailed error information is logged server-side for debugging
