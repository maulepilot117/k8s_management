---
status: pending
priority: p2
issue_id: "190"
tags: [code-review, security, step-12]
dependencies: []
---

# 190: LDAP plaintext connections allowed without warning

## Problem Statement

The LDAP provider's `connect()` function allows `ldap://` URLs without StartTLS, meaning user credentials (bind DN password and user passwords) are sent in cleartext over the network. There is no validation or warning when this insecure configuration is used.

## Findings

- `ldap.go` `connect()` dials `ldap://` URLs without enforcing or recommending StartTLS.
- When `StartTLS` is set to `false` and the URL scheme is `ldap://`, credentials are transmitted in plaintext.
- No log warning is emitted to alert administrators about the insecure configuration.
- This could lead to credential interception on untrusted networks.

## Technical Details

**Affected files:**
- `backend/internal/auth/ldap.go`

## Acceptance Criteria

- [ ] Log a WARN-level message when `ldap://` is used without StartTLS, indicating credentials will be sent in cleartext
- [ ] Consider adding a config option (e.g., `allowInsecure: true`) that must be explicitly set to permit plaintext LDAP connections
- [ ] By default, reject `ldap://` without StartTLS unless `allowInsecure` is set
- [ ] Document the security implications in config comments and values.yaml
