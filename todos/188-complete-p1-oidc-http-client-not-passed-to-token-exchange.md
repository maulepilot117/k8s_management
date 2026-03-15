---
status: pending
priority: p1
issue_id: "188"
tags: [code-review, security, performance, step-12]
dependencies: []
---

# 188: OIDC HTTP client not passed to token exchange — custom TLS broken

## Problem Statement

The custom HTTP client built by `buildOIDCHTTPClient` (which respects `TLSInsecure` and `CACertPath` settings) is used for OIDC discovery but is not passed to the OAuth2 token exchange. The token exchange falls back to `http.DefaultClient`, ignoring custom TLS configuration. This breaks OIDC authentication for any deployment using private CAs or self-signed certificates.

## Findings

- `buildOIDCHTTPClient` in `backend/internal/auth/oidc.go` creates a custom `*http.Client` with a configured TLS transport for OIDC discovery.
- The custom client is used in a context for the `go-oidc` provider discovery call but is not stored on the `OIDCProvider` struct.
- In `HandleCallback`, `oauth2Config.Exchange(r.Context(), code)` is called with the request context, which does not carry the custom HTTP client.
- The `golang.org/x/oauth2` library checks the context for an `*http.Client` value (via `oauth2.HTTPClient` key). Without it, `http.DefaultClient` is used.
- Result: token exchange requests to the OIDC provider's token endpoint use default TLS settings, failing with certificate errors when the provider uses a private CA.

## Proposed Solutions

### Option A: Store httpClient on OIDCProvider and inject via context

Store the custom `*http.Client` on the `OIDCProvider` struct during initialization. In `HandleCallback`, wrap the request context with the custom client using `context.WithValue(r.Context(), oauth2.HTTPClient, p.httpClient)` before calling `oauth2Config.Exchange`.

## Technical Details

**Affected files:**
- `backend/internal/auth/oidc.go`
  - Add `httpClient *http.Client` field to `OIDCProvider` struct
  - Store the client from `buildOIDCHTTPClient` during `NewOIDCProvider`
  - In `HandleCallback`, inject client into context: `ctx := oidc.ClientContext(r.Context(), p.httpClient)`
  - Pass `ctx` to `oauth2Config.Exchange` and `p.verifier.Verify`

**Effort:** Small

## Acceptance Criteria

- [ ] Custom TLS settings (CACertPath, TLSInsecure) apply to both OIDC discovery and token exchange
- [ ] OIDC authentication works with providers using private CA certificates
- [ ] The same HTTP client is used consistently for all OIDC HTTP requests
- [ ] Unit test verifies custom HTTP client is passed to token exchange context
