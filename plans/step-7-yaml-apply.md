# Step 7: YAML Apply — Monaco Editor, Validation, Diff, Server-Side Apply

## Overview

Add a full YAML editing and apply workflow to KubeCenter. Users can edit existing resources via the YAML tab in resource detail views (upgrading from read-only CodeBlock to Monaco editor), apply arbitrary multi-document YAML from a standalone editor page, validate YAML against the cluster's schema, preview diffs before applying, and export resources as clean reapply-ready YAML.

## Problem Statement

Users currently have read-only YAML views in resource detail pages (Step 6). Power users need to:
- Edit resource YAML directly and apply changes (the "escape hatch" for anything not covered by wizards)
- Apply multi-document YAML bundles (e.g., paste from Helm template output, copy from docs)
- Validate YAML before applying to catch errors early
- See what will change (diff) before committing
- Export clean YAML for backup, migration, or GitOps workflows

---

## Design Decisions

### D1. Secret Export Safety
**Decision:** The export endpoint returns HTTP 422 for Secrets with the message "Secrets cannot be exported via YAML to prevent accidental data loss." The YAML tab in resource detail for Secrets remains read-only (no edit mode). Users manage secrets through the dedicated Secrets UI with audit-logged reveal.
**Why:** Exporting masked values (`****`) and re-applying would silently destroy real secret data. This is a catastrophic data-loss risk.

### D2. SSA Force-Conflicts
**Decision:** Expose a "Force field ownership" checkbox in the apply confirmation dialog, unchecked by default. When checked, include a visible warning: "This overrides Helm/GitOps field ownership and may break automated deployments." The force flag is passed as `force: true` in PatchOptions.
**Why:** Most resources in production clusters are Helm-managed. Without force support, the YAML editor is unusable for Helm-managed resources. The warning prevents accidental misuse.

### D3. CRD Support Scope
**Decision:** Support CRDs in Step 7. Use the dynamic client + REST mapper (discovery-based) to resolve arbitrary `apiVersion/kind` to the correct REST endpoint. RBAC checking uses `SelfSubjectAccessReview` with the discovered resource/group, falling back to letting the impersonating client enforce k8s RBAC if discovery fails.
**Why:** Real clusters have CRDs (cert-manager, Cilium, etc.). Rejecting them makes the YAML editor feel incomplete. The dynamic client approach handles this naturally.

### D4. Diff Response Format
**Decision:** Return two clean YAML strings per document (`{ current: "...", proposed: "..." }`). Frontend renders using Monaco's built-in `createDiffEditor`. For new resources (no current state), `current` is an empty string.
**Why:** Monaco's diff editor provides syntax highlighting, line-level navigation, and inline/side-by-side toggle out of the box. Returning raw YAML strings is simpler than computing unified diffs server-side.

### D5. Monaco Loading Strategy
**Decision:** Self-host Monaco assets in `frontend/static/monaco/` (copied from npm package during build). This avoids CDN dependency (works in air-gapped clusters) and simplifies CSP.
**Why:** Air-gapped clusters are a primary deployment target. CDN loading adds a runtime dependency and requires CSP changes. Self-hosting with Vite bundling is cleaner.

### D6. YAML Body Size Limit
**Decision:** YAML endpoints use 2MB limit (`2 << 20`) with a max of 100 documents per request. Enforced via `http.MaxBytesReader` in the YAML handler, separate from the existing 1MB `maxRequestBodySize` for other handlers.
**Why:** Multi-document YAML bundles (e.g., Helm template output) can exceed 1MB. 100-document limit prevents abuse.

### D7. Multi-Doc RBAC Failures
**Decision:** Best-effort, consistent with D4 from the original plan. Each document's RBAC is checked individually. RBAC failures are reported per-document without stopping remaining documents.
**Why:** Consistent with the existing "best-effort, document-by-document" decision. Pre-validating all RBAC before any apply adds complexity with minimal benefit.

### D8. Navigation Entry Point
**Decision:** Add "Apply YAML" to the sidebar under a new "Tools" section. Route: `/yaml/apply`. Also accessible via a "Apply YAML" button in the top bar.
**Why:** YAML apply is a cross-cutting tool, not tied to a specific resource type.

### D9. Dirty State Navigation Guard
**Decision:** Show browser `beforeunload` confirmation when editor has unsaved changes. Also intercept sidebar navigation clicks with a custom "Discard changes?" dialog.
**Why:** Losing a long YAML edit is frustrating and hard to recover from.

### D10. Post-Apply Behavior
**Decision:** After successful apply in the YAML tab, show a success toast. Do NOT auto-refresh editor content. The WS "updated externally" banner appears but does not discard the user's buffer. User can manually click "Reload from cluster" to pull fresh state.
**Why:** The user may want to make further edits immediately after applying.

---

## Technical Approach

### Architecture

```
                     ┌──────────────────────────────┐
                     │    Frontend (Fresh 2.x)       │
                     │                               │
                     │  YamlEditor.tsx (Monaco)       │
                     │  YamlDiffViewer.tsx (Monaco)   │
                     │  ResourceDetail.tsx (upgraded) │
                     │  YamlApplyResults.tsx          │
                     │                               │
                     │  routes/yaml/apply.tsx         │
                     └──────────┬───────────────────┘
                                │ BFF Proxy
                     ┌──────────▼───────────────────┐
                     │    Backend (Go)               │
                     │                               │
                     │  internal/yaml/               │
                     │    handler.go  (HTTP handlers) │
                     │    parser.go   (multi-doc)    │
                     │    applier.go  (SSA)          │
                     │    differ.go   (dry-run diff) │
                     │    export.go   (clean export) │
                     │    security.go (bomb detect)  │
                     │                               │
                     │  internal/k8s/client.go       │
                     │    + DynamicClientForUser()   │
                     │    + RESTMapper()             │
                     └──────────┬───────────────────┘
                                │ Impersonated SSA
                     ┌──────────▼───────────────────┐
                     │  Kubernetes API Server        │
                     │  (PATCH apply-patch+yaml)     │
                     └──────────────────────────────┘
```

### Implementation Phases

#### Phase 1: Backend Foundation (backend/internal/yaml/ + k8s client changes)

**Files to create:**

```
backend/internal/yaml/
├── handler.go     # HTTP handlers for validate, apply, diff, export
├── parser.go      # Multi-doc YAML parsing with security checks
├── applier.go     # Server-side apply via dynamic client
├── differ.go      # Dry-run apply + diff generation
├── export.go      # Clean YAML export (strip server fields)
├── security.go    # YAML bomb detection, unsafe tag rejection
└── handler_test.go # Tests
```

**Files to modify:**

```
backend/internal/k8s/client.go       # Add DynamicClientForUser(), RESTMapper()
backend/internal/server/routes.go     # Register YAML endpoints
backend/internal/server/server.go     # Wire yaml handler into deps
```

**Implementation details:**

`backend/internal/k8s/client.go` — Add dynamic client support:
```go
// DynamicClientForUser returns an impersonating dynamic.Interface
func (f *ClientFactory) DynamicClientForUser(username string, groups []string) (dynamic.Interface, error)

// RESTMapper returns a cached discovery-based REST mapper for GVK→GVR resolution
func (f *ClientFactory) RESTMapper() meta.RESTMapper
```

Uses `k8s.io/client-go/dynamic`, `k8s.io/client-go/discovery/cached/memory`, `k8s.io/client-go/restmapper`.

`backend/internal/yaml/parser.go` — Multi-document YAML parsing:
- Use `k8s.io/apimachinery/pkg/util/yaml.NewYAMLOrJSONDecoder` (streaming decoder, already a transitive dep)
- Parse each document into `unstructured.Unstructured`
- Skip empty documents
- Validate required fields: `apiVersion`, `kind`, `metadata.name`
- Enforce max document count (100)
- Call security checks before parsing

`backend/internal/yaml/security.go` — YAML bomb prevention:
- Reject YAML anchors/aliases (`&anchor`, `*alias`) — k8s manifests never need them
- Reject unsafe YAML tags (`!!python/`, `!!ruby/`, etc.)
- Post-parse expansion ratio check (JSON output > 100x YAML input = bomb)
- Size limit enforcement (2MB)

`backend/internal/yaml/applier.go` — Server-side apply:
```go
type ApplyResult struct {
    Index     int    `json:"index"`
    Kind      string `json:"kind"`
    Name      string `json:"name"`
    Namespace string `json:"namespace,omitempty"`
    Action    string `json:"action"` // "created", "configured", "unchanged", "failed"
    Error     string `json:"error,omitempty"`
}

func (h *Handler) Apply(ctx context.Context, user auth.User, docs []*unstructured.Unstructured, force bool) []ApplyResult
```

- Resolve GVK→GVR via REST mapper
- Create impersonating dynamic client for the user
- SSA PATCH with `types.ApplyPatchType`, field manager `"kubecenter"`
- Determine action by comparing pre/post `resourceVersion` (unchanged if same)
- Audit log each document apply

`backend/internal/yaml/differ.go` — Diff generation:
- Dry-run apply (`DryRun: []string{metav1.DryRunAll}`) to get proposed state
- GET current state from cluster
- Strip server-managed fields from both (using `export.go` logic)
- Serialize both to YAML strings
- Return `{current: string, proposed: string}` per document

`backend/internal/yaml/export.go` — Clean export:
- Strip: `metadata.uid`, `resourceVersion`, `generation`, `creationTimestamp`, `deletionTimestamp`, `deletionGracePeriodSeconds`, `selfLink`, `managedFields`, `ownerReferences`
- Strip: `status` block entirely
- Strip annotations: `kubectl.kubernetes.io/last-applied-configuration`, `deployment.kubernetes.io/revision`
- Remove empty `annotations` map after stripping
- Refuse to export Secrets (return 422)
- Use `sigs.k8s.io/yaml` for deterministic output (sorted keys)

**Endpoints:**

```
POST /api/v1/yaml/validate              # Validate YAML, return per-document errors
POST /api/v1/yaml/apply                 # Apply YAML via SSA, return per-document results
POST /api/v1/yaml/diff                  # Dry-run apply, return current vs proposed YAML
GET  /api/v1/yaml/export/:kind/:ns/:name  # Export clean YAML (GET, not POST)
```

Request/response formats:

```
# POST /api/v1/yaml/validate
# Request: Content-Type: text/yaml (raw YAML body)
# Response:
{
  "data": {
    "documents": [
      {
        "index": 0,
        "kind": "Deployment",
        "name": "my-app",
        "namespace": "default",
        "valid": true
      },
      {
        "index": 1,
        "kind": "Service",
        "name": "my-svc",
        "namespace": "default",
        "valid": false,
        "errors": [
          { "field": "spec.ports[0].targetPort", "message": "required field" }
        ]
      }
    ],
    "valid": false
  }
}

# POST /api/v1/yaml/apply
# Request: Content-Type: text/yaml (raw YAML body)
# Query params: ?force=true (optional, for SSA conflict override)
# Response:
{
  "data": {
    "results": [
      {
        "index": 0,
        "kind": "Deployment",
        "name": "my-app",
        "namespace": "default",
        "action": "configured"
      }
    ],
    "applied": 1,
    "failed": 0
  }
}

# POST /api/v1/yaml/diff
# Request: Content-Type: text/yaml (raw YAML body)
# Response:
{
  "data": {
    "documents": [
      {
        "index": 0,
        "kind": "Deployment",
        "name": "my-app",
        "namespace": "default",
        "isNew": false,
        "current": "apiVersion: apps/v1\nkind: Deployment\n...",
        "proposed": "apiVersion: apps/v1\nkind: Deployment\n..."
      }
    ]
  }
}

# GET /api/v1/yaml/export/:kind/:ns/:name
# Response: Content-Type: text/yaml (raw YAML body)
# For cluster-scoped: GET /api/v1/yaml/export/:kind/_/:name
```

**Go dependencies to add:** None — all needed packages are already transitive deps:
- `k8s.io/client-go/dynamic` (via client-go)
- `k8s.io/client-go/discovery/cached/memory` (via client-go)
- `k8s.io/client-go/restmapper` (via client-go)
- `k8s.io/apimachinery/pkg/util/yaml` (via apimachinery)
- `sigs.k8s.io/yaml` (via client-go)

#### Phase 2: Frontend — Monaco Editor Island

**Files to create:**

```
frontend/islands/YamlEditor.tsx        # Monaco editor wrapper (editable YAML)
frontend/islands/YamlDiffViewer.tsx     # Monaco diff editor wrapper
frontend/islands/YamlApplyResults.tsx   # Per-document apply result table
frontend/routes/yaml/apply.tsx         # Standalone YAML apply page
```

**Files to modify:**

```
frontend/islands/ResourceDetail.tsx    # Upgrade YAML tab: CodeBlock → Monaco, add Edit/Apply/Diff/Export buttons
frontend/lib/api.ts                    # Add apiPostRaw() for text/yaml body
frontend/lib/constants.ts              # Add "Tools" nav section with "Apply YAML" entry
frontend/deno.json                     # Add monaco-editor dependency (for types only)
frontend/routes/_middleware.ts          # CSP update if needed (likely not if self-hosting)
```

**Implementation details:**

`frontend/islands/YamlEditor.tsx`:
- Load Monaco dynamically via `import()` inside `useEffect` with `IS_BROWSER` guard
- Self-hosted from `/static/monaco/` or loaded from `esm.sh` CDN with CSP update
- YAML language mode, dark theme (`vs-dark`), no minimap, auto-layout
- Props: `value`, `onChange`, `readOnly`, `height`, `markers` (validation errors)
- Expose imperative methods via ref: `getValue()`, `setValue()`, `setMarkers()`
- Handle Monaco disposal on unmount
- Show loading skeleton while Monaco initializes

`frontend/islands/YamlDiffViewer.tsx`:
- Uses `monaco.editor.createDiffEditor`
- Props: `current: string`, `proposed: string`, `height`
- Side-by-side by default, toggle for inline diff
- Read-only (both sides)

`frontend/islands/YamlApplyResults.tsx`:
- Table showing per-document apply results
- Columns: #, Kind, Name, Namespace, Action, Error
- Color-coded: green (created/configured), gray (unchanged), red (failed)
- Failed rows are expandable to show error details

`frontend/routes/yaml/apply.tsx`:
- Full-page YAML editor with toolbar
- Toolbar: Upload File | Validate | Diff | Apply | Format
- Upload: `<input type="file" accept=".yaml,.yml,.json">`, reads via FileReader, normalizes CRLF→LF, strips BOM
- Drag-and-drop onto editor area replaces content (with confirmation if dirty)
- Results panel below editor (collapsible)
- Diff panel slides in as overlay or below editor

`frontend/islands/ResourceDetail.tsx` — YAML tab upgrade:
- Replace `CodeBlock` with `YamlEditor` (conditionally — keep CodeBlock as fallback if Monaco fails to load)
- Add action buttons: Edit (toggles readOnly), Apply, Diff, Export, Reload
- Edit mode shows "Apply" and "Discard" buttons
- Export button calls `GET /api/v1/yaml/export/...` and downloads as file
- For Secrets: Edit button is disabled with tooltip "Secrets cannot be edited via YAML"
- Dirty state tracked via signal, gates navigation with `beforeunload`

`frontend/lib/api.ts` — Add raw body support:
```typescript
export async function apiPostRaw(path: string, body: string, contentType = "text/yaml"): Promise<...>
```

#### Phase 3: Integration and Polish

**Tasks:**
- Wire YAML tab edit → apply flow with success/error feedback
- Wire standalone editor → validate → diff → apply flow
- Add "Edit in YAML Editor" action on resource detail that opens standalone editor with pre-loaded content (via `sessionStorage`)
- Add "Apply YAML" to sidebar navigation under "Tools" section
- Add keyboard shortcuts: Cmd/Ctrl+S = Apply, Cmd/Ctrl+Shift+D = Diff
- File upload with drag-and-drop support
- Error recovery: on partial apply failure, highlight failed documents
- Ensure WS updates don't clobber in-progress edits

---

## Acceptance Criteria

### Functional Requirements
- [ ] Monaco editor loads with YAML syntax highlighting in resource detail YAML tab
- [ ] YAML tab edit mode: user can edit YAML and apply changes via SSA
- [ ] Secrets YAML tab remains read-only (edit disabled with explanation)
- [ ] Standalone YAML editor page at `/yaml/apply` with file upload support
- [ ] File upload accepts `.yaml`, `.yml`, `.json` files up to 2MB
- [ ] Validate button shows per-document, per-field validation errors inline
- [ ] Diff button shows side-by-side comparison (current vs proposed) using Monaco diff editor
- [ ] Apply button applies YAML via SSA and shows per-document results
- [ ] Multi-document YAML applies each document best-effort and reports per-document status
- [ ] SSA conflict errors show a "Force field ownership" option with warning
- [ ] RBAC denied on apply shows clear per-document error message
- [ ] Export button on resource detail exports clean YAML (no managedFields, resourceVersion, status, etc.)
- [ ] Export returns 422 for Secrets with explanation
- [ ] CRDs are supported (dynamic client resolves arbitrary apiVersion/kind)
- [ ] "Apply YAML" entry in sidebar navigation under "Tools" section

### Non-Functional Requirements
- [ ] Monaco loads in under 3 seconds on typical connection
- [ ] Fallback to basic textarea if Monaco fails to load (CDN unreachable, etc.)
- [ ] YAML bomb prevention: anchors/aliases rejected, expansion ratio checked
- [ ] Unsafe YAML tags (`!!python/`, etc.) rejected
- [ ] Body size limit: 2MB for YAML endpoints, max 100 documents
- [ ] All apply operations are audit-logged (per-document)
- [ ] Dirty state navigation guard prevents accidental loss of edits

### Quality Gates
- [ ] Backend unit tests for parser, applier, differ, export, security checks
- [ ] Frontend tests for YAML editor state management
- [ ] Smoke tested against homelab k3s cluster:
  - Edit a Deployment via YAML tab and verify change applied
  - Apply multi-document YAML from standalone editor
  - Validate YAML with intentional errors and verify inline feedback
  - Preview diff for an existing resource
  - Export a Deployment as clean YAML and verify it can be re-applied
  - Verify Secret export is blocked
  - Verify YAML bomb is rejected

---

## Risk Analysis & Mitigation

| Risk | Impact | Mitigation |
|---|---|---|
| Monaco fails to load in air-gapped clusters | Editor unusable | Self-host Monaco assets; fallback to basic `<textarea>` with syntax highlighting via CodeBlock |
| SSA conflicts with Helm-managed resources | Confusing errors for most users | Force-conflicts checkbox with clear warning; explain which field manager owns conflicting fields |
| YAML bomb DoS | Backend memory/CPU exhaustion | Pre-parse anchor rejection, expansion ratio check, 2MB limit, 100-doc limit |
| Secret data loss via export→re-apply | Catastrophic data loss | Block Secret export entirely (422); disable YAML edit for Secrets |
| Large Monaco bundle slows page load | Poor UX on first load | Lazy-load Monaco only when YAML tab/page is accessed; show loading skeleton |
| Concurrent edits via WS updates clobbering editor | User loses edits | Don't auto-refresh editor content; show "updated externally" banner with manual reload option |

---

## Dependencies & Prerequisites

- **Step 6 complete** (resource detail views with YAML tab) — merged
- **Dynamic client** added to `k8s/client.go` (new work in this step)
- **No new Go dependencies** — all needed packages are transitive deps of client-go
- **Monaco editor** — self-hosted or CDN-loaded (new frontend dependency)

---

## Files Summary

### New files (backend)
| File | Purpose |
|---|---|
| `backend/internal/yaml/handler.go` | HTTP handlers for validate, apply, diff, export |
| `backend/internal/yaml/parser.go` | Multi-doc YAML parsing with streaming decoder |
| `backend/internal/yaml/applier.go` | Server-side apply via dynamic client |
| `backend/internal/yaml/differ.go` | Dry-run apply + diff generation |
| `backend/internal/yaml/export.go` | Clean YAML export (strip server fields) |
| `backend/internal/yaml/security.go` | YAML bomb detection, unsafe tag rejection |
| `backend/internal/yaml/handler_test.go` | Unit tests |

### New files (frontend)
| File | Purpose |
|---|---|
| `frontend/islands/YamlEditor.tsx` | Monaco editor wrapper |
| `frontend/islands/YamlDiffViewer.tsx` | Monaco diff editor wrapper |
| `frontend/islands/YamlApplyResults.tsx` | Per-document apply result table |
| `frontend/routes/yaml/apply.tsx` | Standalone YAML apply page |

### Modified files
| File | Change |
|---|---|
| `backend/internal/k8s/client.go` | Add `DynamicClientForUser()`, `RESTMapper()` |
| `backend/internal/server/routes.go` | Register YAML endpoints |
| `backend/internal/server/server.go` | Wire yaml handler |
| `frontend/islands/ResourceDetail.tsx` | Upgrade YAML tab to editable Monaco |
| `frontend/lib/api.ts` | Add `apiPostRaw()` for text/yaml bodies |
| `frontend/lib/constants.ts` | Add "Tools" nav section |
| `frontend/deno.json` | Monaco dependency (if needed) |

---

## Implementation Order

1. **Backend: `yaml/security.go` + `yaml/parser.go`** — YAML parsing with security checks (no k8s deps, easily testable)
2. **Backend: `k8s/client.go` changes** — Add dynamic client + REST mapper
3. **Backend: `yaml/export.go`** — Clean export (depends on dynamic client for GET)
4. **Backend: `yaml/applier.go`** — SSA apply (depends on dynamic client)
5. **Backend: `yaml/differ.go`** — Diff (depends on applier for dry-run)
6. **Backend: `yaml/handler.go` + `routes.go`** — Wire HTTP endpoints
7. **Backend: `yaml/handler_test.go`** — Unit tests
8. **Frontend: `YamlEditor.tsx`** — Monaco island (can develop standalone)
9. **Frontend: `ResourceDetail.tsx` upgrade** — YAML tab edit mode + Export
10. **Frontend: `YamlDiffViewer.tsx`** — Diff viewer
11. **Frontend: `YamlApplyResults.tsx`** — Apply results table
12. **Frontend: `routes/yaml/apply.tsx`** — Standalone editor page
13. **Frontend: navigation + constants** — Sidebar entry, API helpers
14. **Integration testing** — Full flow testing
15. **Smoke test against homelab** — Pre-merge validation

---

## References

### Internal
- Step 7 spec in `plans/feat-kubecenter-phase1-mvp.md:789-851`
- Decision D4 (multi-doc semantics): `plans/feat-kubecenter-phase1-mvp.md:82-84`
- Existing YAML tab: `frontend/islands/ResourceDetail.tsx:267-305`
- CodeBlock component: `frontend/components/ui/CodeBlock.tsx`
- Resource handler patterns: `backend/internal/k8s/resources/handler.go`
- Client factory: `backend/internal/k8s/client.go`
- Route registration: `backend/internal/server/routes.go`

### External
- [Kubernetes Server-Side Apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/)
- [client-go dynamic package](https://pkg.go.dev/k8s.io/client-go/dynamic)
- [k8s.io/apimachinery/pkg/util/yaml streaming decoder](https://pkg.go.dev/k8s.io/apimachinery/pkg/util/yaml)
- [sigs.k8s.io/yaml](https://pkg.go.dev/sigs.k8s.io/yaml)
- [Monaco Editor](https://microsoft.github.io/monaco-editor/)
- [monaco-yaml](https://github.com/remcohaszing/monaco-yaml)
