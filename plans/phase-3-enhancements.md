# Phase 3: Enhancements

6 incremental improvements, each a separate PR. Ordered by value and dependency.

## 1. Pod Exec with xterm.js

WebSocket-based terminal in the browser for pod exec.

**Backend** (`backend/internal/k8s/resources/pods.go`):
- `HandlePodExec` — upgrade to WebSocket, SPDY exec via `remotecommand.NewSPDYExecutor`
- RBAC: check `create` on `pods/exec` subresource
- Audit log exec sessions

**Frontend** (`frontend/islands/PodExec.tsx`):
- xterm.js terminal component (npm: `xterm`, `xterm-addon-fit`)
- Container selector dropdown (same as LogViewer)
- Add "Exec" tab to pod ResourceDetail

**Route**: `WS /api/v1/resources/pods/{namespace}/{name}/exec?container=X`

**Files**: `pods.go`, `routes.go`, `PodExec.tsx`, `ResourceDetail.tsx`

---

## 2. User Management UI

Admin page to list, delete users and change passwords.

**Backend** (`backend/internal/store/users.go`):
- Add `List(ctx) ([]UserRecord, error)` — returns all users (no passwords)
- Add `Delete(ctx, id) error`
- Add `UpdatePassword(ctx, id, newPHC) error`
- Update `auth.UserStore` interface + `MemoryUserStore`

**Backend** (`backend/internal/server/handle_users.go`):
- `GET /api/v1/users` — list users (admin only)
- `DELETE /api/v1/users/{id}` — delete user (admin only, cannot delete self)
- `PUT /api/v1/users/{id}/password` — change password (admin or self)

**Frontend** (`frontend/islands/UserManager.tsx`):
- Table: username, roles, k8s username, created date
- Delete button with confirmation
- Change password modal
- Route: `/settings/users`

**Files**: `store/users.go`, `auth/local.go`, `handle_users.go`, `routes.go`, `UserManager.tsx`, `constants.ts`

---

## 3. Dynamic Informer for CiliumNetworkPolicies

Enable WebSocket live updates for the Cilium policy list page.

**Backend** (`backend/internal/k8s/informers.go`):
- Add `dynamicinformer.NewDynamicSharedInformerFactory` for `cilium.io/v2/CiliumNetworkPolicy`
- Feed events into WebSocket hub via `HandleEvent`

**Backend** (`backend/internal/websocket/events.go`):
- Add `"ciliumnetworkpolicies"` to `allowedKinds`

**Backend** (`backend/internal/k8s/resources/cilium.go`):
- Optional: switch list/get to read from informer cache instead of dynamic client

**Files**: `informers.go`, `events.go`, `cilium.go`

---

## 4. WebSocket Streaming for Hubble Flows

Real-time flow streaming instead of HTTP refresh.

**Backend** (`backend/internal/networking/hubble_client.go`):
- Add `StreamFlows(ctx, namespace, verdict)` — uses `Follow: true` on GetFlows
- Feed flow events into WebSocket hub as `kind: "flows"`

**Backend** (`backend/internal/websocket/events.go`):
- Add `"flows"` to `allowedKinds` and `alwaysAllowKinds` (RBAC checked at subscribe time)

**Frontend** (`frontend/islands/FlowViewer.tsx`):
- Add WebSocket subscription option alongside HTTP refresh
- Live badge + row animation for new flows

**Files**: `hubble_client.go`, `events.go`, `handler.go`, `FlowViewer.tsx`

---

## 5. AlertBanner WebSocket Migration

Replace 30s polling with WebSocket subscription.

**Frontend** (`frontend/islands/AlertBanner.tsx`):
- Replace `setInterval` fetch with WebSocket subscribe to `kind: "alerts"`
- On WS message, update alert state
- Fallback to polling if WS unavailable

**Files**: `AlertBanner.tsx` (frontend only)

---

## 6. CSP Nonce Injection

Replace `unsafe-inline` with nonce-based CSP when Fresh supports it.

**Frontend** (`frontend/routes/_middleware.ts`):
- Generate per-request nonce
- Inject into CSP header: `script-src 'nonce-xxx'`
- Pass nonce to `_app.tsx` for inline scripts

**Blocked by**: Fresh 2.x nonce support. Monitor `@fresh/core` releases.

**Files**: `_middleware.ts`, `_app.tsx`

---

## Implementation Order

| # | Feature | Effort | Dependencies |
|---|---------|--------|--------------|
| 1 | Pod Exec | Medium | None |
| 2 | User Management UI | Medium | None |
| 3 | Dynamic CiliumNetworkPolicy Informer | Small | None |
| 4 | WebSocket Hubble Flows | Medium | #3 pattern |
| 5 | AlertBanner WebSocket | Small | None |
| 6 | CSP Nonce | Small | Fresh 2.x support (blocked) |

Items 1-5 can be done now. Item 6 is blocked.
