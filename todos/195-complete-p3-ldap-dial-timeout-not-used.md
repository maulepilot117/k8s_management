---
status: pending
priority: p3
issue_id: "195"
tags: [code-review, performance, step-12]
dependencies: []
---

# 195: LDAP dial timeout constant defined but not used

## Problem Statement
The ldapDialTimeout constant (5s) exists in the LDAP provider code but ldap.DialURL does not use it. If the LDAP server is unreachable, the connection hangs for the system TCP timeout (2+ minutes), causing login requests to block.

## Findings
- `ldapDialTimeout` (5 seconds) is defined as a constant but never passed to the dial call
- `ldap.DialURL` uses the default system TCP timeout when no dialer is provided
- Unreachable LDAP servers cause login attempts to hang for 2+ minutes
- This creates a poor user experience and can exhaust HTTP server goroutines under load

## Technical Details
**Affected files:**
- `backend/internal/auth/ldap.go` (dial call in Authenticate or similar method)

**Effort:** Small

## Acceptance Criteria
- [ ] ldap.DialURL uses net.Dialer with Timeout set to ldapDialTimeout via DialWithDialer option
- [ ] LDAP connection attempts fail fast (within 5s) when server is unreachable
- [ ] Timeout error is wrapped with user-friendly message
