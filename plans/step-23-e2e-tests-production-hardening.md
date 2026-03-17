# Step 23: Security-Critical Test Coverage + Readyz Fix

## Overview

Final Phase 2 step. Add unit tests for the two most security-critical untested components (RBAC cache, WebSocket hub) and fix the readyz probe to include PostgreSQL health. Manual smoke testing against the homelab remains the E2E strategy.

## Revised Scope (per reviewer feedback)

- **Cut**: Playwright E2E suite (manual smoke test is sufficient for solo homelab)
- **Cut**: AlertBanner WebSocket migration (polling works, scope creep)
- **Cut**: Hubble client tests as a phase (simple mappers, tested by use)
- **Cut**: Cilium E2E spec (conditional tests breed flakiness)
- **Keep**: RBAC cache unit tests (security-critical caching + concurrency)
- **Keep**: WebSocket hub unit tests (concurrency, RBAC revalidation)
- **Keep**: PostgreSQL readyz check (one-line fix, correct probe behavior)

## Implementation

### 1. RBAC Cache Tests (`backend/internal/auth/rbac_test.go`)

The `RBACChecker` has mutex-guarded cache with TTL and eviction. Test:
- Cache hit returns stored result without API call
- Cache miss calls SelfSubjectRulesReview
- Expired entry triggers fresh check
- Concurrent access is safe (`-race` flag)

### 2. WebSocket Hub Tests (`backend/internal/websocket/hub_test.go`)

The hub is the most complex concurrency code in the project. Test:
- Client registration increments count, unregistration decrements
- Event broadcast reaches matching subscriptions only
- MaxClients limit rejects new connections
- Context cancellation stops the hub's Run loop

### 3. PostgreSQL Readyz Check (`backend/internal/server/handle_health.go`)

Add `db.Ping()` to the readiness probe. Currently only checks informer sync — if PostgreSQL is down but informers are synced, readyz incorrectly reports healthy.

## Acceptance Criteria

- [ ] RBAC cache has 4+ unit tests covering hit, miss, expiry, concurrency
- [ ] WebSocket hub has 4+ unit tests covering registration, broadcast, max clients, shutdown
- [ ] `/readyz` returns unhealthy when PostgreSQL is unreachable
- [ ] All existing 227 tests still pass
- [ ] `go test -race ./...` passes

## Files Changed

| File | Change |
|------|--------|
| `auth/rbac_test.go` | New — RBAC cache tests |
| `websocket/hub_test.go` | New — WebSocket hub tests |
| `server/handle_health.go` | Add PostgreSQL ping to readyz |
| `server/server.go` | Add DB field for health check |
