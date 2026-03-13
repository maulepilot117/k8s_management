# Step 8: Resource Creation Wizards

## Overview

Add Deployment and Service creation wizards with a reusable multi-step wizard shell. Users fill form fields across steps, preview generated YAML in Monaco, optionally edit it, and apply via the existing SSA pipeline. Form-to-YAML is one-way only (no bidirectional sync).

## Problem Statement / Motivation

Users currently create resources only via raw YAML (`/tools/yaml-apply`) or the existing typed CRUD endpoints (which have no frontend UI for creation). A guided wizard reduces errors, enforces best practices (resource limits, probes), and makes KubeCenter accessible to users who don't know YAML.

## Proposed Solution

### Architecture

**Data flow** (per CLAUDE.md):
1. User fills wizard form (frontend signals)
2. Frontend POSTs JSON to `POST /api/v1/wizards/{kind}/preview`
3. Backend constructs typed k8s object via client-go structs, serializes to YAML via `sigs.k8s.io/yaml`
4. Frontend shows YAML in Monaco editor (editable)
5. User clicks Apply → frontend POSTs YAML to existing `POST /api/v1/yaml/apply`
6. Backend validates and applies via SSA (existing pipeline, field manager `"kubecenter"`)

### Key Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Wizard as single island | Yes | State shared across steps; Fresh 2.x prohibits function props across island boundary |
| Step components | Non-island children in `components/wizard/` | Can receive callbacks as props since they're imported directly |
| YAML regeneration on re-entering Review | Always regenerate | Matches one-way form→YAML design; manual YAML edits are ephemeral |
| YamlEditor extraction | Extract core to `components/ui/MonacoEditor.tsx`, keep thin island wrapper | Avoids nested-island problem; YamlApplyPage and ResourceDetail still work via wrapper |
| Post-apply behavior | Show success banner with "View Resource" link | Don't auto-redirect; user may want to create another |
| Namespace pre-fill | Use `selectedNamespace` signal if not "all" | Matches TopBar context |
| Labels pre-fill | Auto-generate `app: <name>` | Kubernetes convention; updates reactively as name changes |
| ExternalName service type | Defer to Phase 2 | Conditional step hiding adds complexity; rarely used |
| Probe types | HTTP GET and TCP socket only | Exec and gRPC available via YAML editing on Review step |
| ConfigMap/Secret ref UI | Freeform text inputs (name + key) | Dropdown with API call deferred to future enhancement |
| Existence check before apply | None (SSA handles it) | Apply result shows "configured" if resource existed |
| Early RBAC check | None in Phase 1 | Apply step surfaces 403; simpler |
| Cancel button | Yes, with beforeunload guard | Navigates to resource list page |
| Env var / port limits | 50 env vars, 20 ports (client-side) | Prevents runaway arrays |

### Backend API Contract

#### `POST /api/v1/wizards/deployment/preview`

Request:
```go
// backend/internal/wizard/deployment.go
type DeploymentInput struct {
    Name      string            `json:"name"`
    Namespace string            `json:"namespace"`
    Image     string            `json:"image"`
    Replicas  int32             `json:"replicas"`
    Labels    map[string]string `json:"labels,omitempty"`
    Ports     []PortInput       `json:"ports,omitempty"`
    EnvVars   []EnvVarInput     `json:"envVars,omitempty"`
    Resources *ResourcesInput   `json:"resources,omitempty"`
    Probes    *ProbesInput      `json:"probes,omitempty"`
    Strategy  *StrategyInput    `json:"strategy,omitempty"`
}

type PortInput struct {
    Name          string `json:"name,omitempty"`
    ContainerPort int32  `json:"containerPort"`
    Protocol      string `json:"protocol,omitempty"` // "TCP" (default), "UDP"
}

type EnvVarInput struct {
    Name         string `json:"name"`
    Value        string `json:"value,omitempty"`
    ConfigMapRef string `json:"configMapRef,omitempty"`
    SecretRef    string `json:"secretRef,omitempty"`
    Key          string `json:"key,omitempty"`
}

type ResourcesInput struct {
    RequestCPU    string `json:"requestCpu,omitempty"`    // e.g., "100m"
    RequestMemory string `json:"requestMemory,omitempty"` // e.g., "128Mi"
    LimitCPU      string `json:"limitCpu,omitempty"`
    LimitMemory   string `json:"limitMemory,omitempty"`
}

type ProbesInput struct {
    Liveness  *ProbeInput `json:"liveness,omitempty"`
    Readiness *ProbeInput `json:"readiness,omitempty"`
}

type ProbeInput struct {
    Type                string `json:"type"` // "http" or "tcp"
    Path                string `json:"path,omitempty"` // HTTP only
    Port                int32  `json:"port"`
    InitialDelaySeconds int32  `json:"initialDelaySeconds,omitempty"`
    PeriodSeconds       int32  `json:"periodSeconds,omitempty"`
}

type StrategyInput struct {
    Type           string `json:"type,omitempty"` // "RollingUpdate" (default) or "Recreate"
    MaxSurge       string `json:"maxSurge,omitempty"`
    MaxUnavailable string `json:"maxUnavailable,omitempty"`
}
```

Response (standard envelope): `{"data": {"yaml": "apiVersion: apps/v1\nkind: Deployment\n..."}}`

Validation error (422):
```json
{
  "error": {
    "code": 422,
    "message": "validation failed",
    "fields": [
      {"field": "name", "message": "must be a valid DNS label"},
      {"field": "ports[0].containerPort", "message": "must be 1-65535"}
    ]
  }
}
```

#### `POST /api/v1/wizards/service/preview`

Request:
```go
// backend/internal/wizard/service.go
type ServiceInput struct {
    Name      string            `json:"name"`
    Namespace string            `json:"namespace"`
    Type      string            `json:"type"` // "ClusterIP", "NodePort", "LoadBalancer"
    Labels    map[string]string `json:"labels,omitempty"`
    Selector  map[string]string `json:"selector"`
    Ports     []ServicePortInput `json:"ports"`
}

type ServicePortInput struct {
    Name       string `json:"name,omitempty"`
    Port       int32  `json:"port"`
    TargetPort int32  `json:"targetPort"`
    Protocol   string `json:"protocol,omitempty"` // "TCP" (default), "UDP"
    NodePort   int32  `json:"nodePort,omitempty"` // only for NodePort/LoadBalancer
}
```

Response: same envelope as deployment preview.

### Implementation Phases

#### Phase 1: Backend — Wizard Package (backend changes)

**New files:**
- `backend/internal/wizard/deployment.go` — `DeploymentInput`, `ToDeployment()`, `Validate()`
- `backend/internal/wizard/service.go` — `ServiceInput`, `ToService()`, `Validate()`
- `backend/internal/wizard/handler.go` — HTTP handlers for preview endpoints
- `backend/internal/wizard/wizard_test.go` — unit tests for conversion + validation

**Modified files:**
- `backend/internal/server/server.go` — add `WizardHandler` to `Server` and `Deps`
- `backend/internal/server/routes.go` — register wizard routes under authenticated group
- `backend/go.mod` — add `sigs.k8s.io/yaml` direct dependency (already transitive)

**Handler pattern** (matches existing `yamlpkg.Handler`):
```go
type Handler struct {
    Logger *slog.Logger
}

func (h *Handler) HandleDeploymentPreview(w http.ResponseWriter, r *http.Request) {
    var input DeploymentInput
    if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&input); err != nil {
        httputil.WriteError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    if errs := input.Validate(); len(errs) > 0 {
        writeValidationErrors(w, errs)
        return
    }
    dep := input.ToDeployment()
    yamlBytes, err := sigsyaml.Marshal(dep)
    if err != nil {
        httputil.WriteError(w, http.StatusInternalServerError, "failed to generate YAML")
        return
    }
    httputil.WriteData(w, http.StatusOK, map[string]string{"yaml": string(yamlBytes)})
}
```

**Route registration:**
```go
// In routes.go, inside the authenticated group:
if s.WizardHandler != nil {
    r.Route("/wizards", func(wr chi.Router) {
        wr.Use(yamlRL) // share YAML rate limiter (30 req/min)
        wr.Post("/deployment/preview", s.WizardHandler.HandleDeploymentPreview)
        wr.Post("/service/preview", s.WizardHandler.HandleServicePreview)
    })
}
```

**Validation rules:**
- Name: RFC 1123 DNS label (`/^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/`)
- Namespace: non-empty
- Image: non-empty
- Replicas: 0–1000
- Ports: containerPort 1–65535, unique, protocol TCP or UDP
- Resource quantities: parsed via `resource.ParseQuantity` (not `MustParse`)
- Env var names: non-empty, valid k8s env var name
- Probe port: 1–65535, HTTP path starts with `/`
- Service ports: port 1–65535, targetPort 1–65535, nodePort 30000–32767 or 0
- Service selector: non-empty (at least one key-value pair)

**Critical implementation notes:**
- Always set `TypeMeta` (apiVersion, Kind) on constructed objects — `sigs.k8s.io/yaml` only includes them if populated
- Use `resource.ParseQuantity` (not `MustParse`) — return validation error on invalid input
- Use `intstr.FromInt32()` for Service targetPort
- Auto-generate `app: <name>` label if not provided
- Field manager is `"kubecenter"` (matches existing YAML apply pipeline)

#### Phase 2: Frontend — MonacoEditor Extraction (prerequisite refactor)

Extract Monaco editor core from `islands/YamlEditor.tsx` to `components/ui/MonacoEditor.tsx`:

**New file:**
- `frontend/components/ui/MonacoEditor.tsx` — the actual Monaco logic (CDN loading, init, value sync, markers, fallback textarea)

**Modified files:**
- `frontend/islands/YamlEditor.tsx` — becomes thin wrapper that imports and re-exports `MonacoEditor`

This is a refactor with no behavior change. `YamlApplyPage.tsx` and `ResourceDetail.tsx` continue importing from `islands/YamlEditor.tsx` unchanged.

#### Phase 3: Frontend — WizardStepper + UI Components

**New files:**
- `frontend/components/wizard/WizardStepper.tsx` — step indicator bar (horizontal numbered steps with completion state)
- `frontend/components/ui/Select.tsx` — dropdown select component (for namespace, service type, protocol, pull policy)

**WizardStepper props:**
```tsx
interface WizardStepperProps {
  steps: Array<{ title: string; description?: string }>;
  currentStep: number;
  onStepClick?: (step: number) => void; // only allow clicking completed steps
}
```

#### Phase 4: Frontend — Deployment Wizard

**New files:**
- `frontend/islands/DeploymentWizard.tsx` — main wizard island (single island, 4 internal steps)
- `frontend/components/wizard/DeploymentBasicsStep.tsx` — Step 1: name, namespace, image, replicas, labels
- `frontend/components/wizard/DeploymentNetworkStep.tsx` — Step 2: ports, env vars
- `frontend/components/wizard/DeploymentResourcesStep.tsx` — Step 3: CPU/memory, probes
- `frontend/components/wizard/WizardReviewStep.tsx` — Step 4: Monaco YAML preview + Apply (reusable)
- `frontend/routes/workloads/deployments/new.tsx` — route page

**Deployment wizard state (single signal):**
```tsx
interface DeploymentFormState {
  // Step 1
  name: string;
  namespace: string;
  image: string;
  replicas: number;
  labels: Array<{ key: string; value: string }>;
  // Step 2
  ports: Array<{ name: string; containerPort: number; protocol: string }>;
  envVars: Array<{ name: string; type: "literal" | "configmap" | "secret"; value: string; ref: string; key: string }>;
  // Step 3
  cpuRequest: string;
  memoryRequest: string;
  cpuLimit: string;
  memoryLimit: string;
  livenessProbe: ProbeState | null;
  readinessProbe: ProbeState | null;
  strategy: { type: string; maxSurge: string; maxUnavailable: string };
}
```

**Step 1 — Basics:**
- Name: `Input` with RFC 1123 validation, error inline
- Namespace: `Select` dropdown populated from `GET /api/v1/resources/namespaces`, pre-filled from `selectedNamespace`
- Image: `Input` with placeholder `nginx:1.25`
- Replicas: number `Input`, default 1, min 0, max 1000
- Labels: dynamic key-value list, auto-populated with `app: <name>`

**Step 2 — Networking:**
- Container ports: dynamic list (name, containerPort, protocol dropdown)
- Env vars: dynamic list with type selector (Literal → value input, ConfigMap → ref + key, Secret → ref + key)

**Step 3 — Resources & Health (all optional):**
- CPU/Memory request/limit inputs with placeholder hints (100m, 128Mi, 500m, 512Mi)
- Liveness probe: toggle on/off → type selector (HTTP/TCP) → fields (path, port, initialDelay, period)
- Readiness probe: same pattern
- Update strategy: radio (RollingUpdate/Recreate), maxSurge/maxUnavailable inputs when RollingUpdate

**Step 4 — Review:**
- On enter: POST form state to `/api/v1/wizards/deployment/preview`, show loading spinner
- Show YAML in `MonacoEditor` component (readOnly=false)
- Apply button → POST YAML to `/api/v1/yaml/apply`
- Show results (created/configured/failed) with link to resource detail page
- Cancel and Back buttons

**Entry point:** "Create Deployment" button on `/workloads/deployments` resource list page (add to `ResourceTable` as a prop-driven action button).

#### Phase 5: Frontend — Service Wizard

**New files:**
- `frontend/islands/ServiceWizard.tsx` — main wizard island (3 internal steps)
- `frontend/components/wizard/ServiceBasicsStep.tsx` — Step 1: name, namespace, type
- `frontend/components/wizard/ServicePortsStep.tsx` — Step 2: ports, selector
- `frontend/routes/networking/services/new.tsx` — route page

**Service wizard state:**
```tsx
interface ServiceFormState {
  name: string;
  namespace: string;
  type: "ClusterIP" | "NodePort" | "LoadBalancer";
  labels: Array<{ key: string; value: string }>;
  selector: Array<{ key: string; value: string }>;
  ports: Array<{ name: string; port: number; targetPort: number; protocol: string; nodePort: number }>;
}
```

**Type-conditional fields in Step 2:**
- NodePort field: only visible when type is NodePort or LoadBalancer
- Selector: always visible (required for ClusterIP/NodePort/LoadBalancer)

**Reuses:** `WizardReviewStep.tsx` from Phase 4 (same Monaco preview + Apply pattern).

#### Phase 6: Integration + Polish

- Add "Create" buttons to ResourceTable for deployments and services
- `beforeunload` guard on both wizards (dirty state detection)
- Keyboard: Enter on last field in step triggers Next; Tab order across form fields
- Loading states on Next (validation), on Review (preview fetch), on Apply (SSA)
- Error handling: inline field errors, step-level error banner, apply failure banner

## Technical Considerations

### Architecture
- New `internal/wizard/` package keeps wizard-specific logic separate from generic resource handlers
- Preview endpoints are stateless — no server-side wizard session
- Apply reuses existing SSA pipeline (no new k8s client code)

### Performance
- Namespace list for dropdown: cached by informer (no extra API call)
- Preview endpoint: lightweight (JSON parse → struct construction → YAML marshal, ~1ms)
- Monaco CDN load: one-time per session (cached by browser)

### Security
- Preview endpoints behind auth + CSRF middleware (same as all authenticated endpoints)
- Wizard rate limiting shares YAML bucket (30 req/min) — more than sufficient
- No YAML injection risk: `sigs.k8s.io/yaml` marshals typed Go structs, not raw strings
- User input sanitized by typed struct construction (e.g., `\n` in image name serialized correctly)
- Apply uses impersonation — RBAC enforced by k8s API server

## Acceptance Criteria

### Backend
- [ ] `POST /api/v1/wizards/deployment/preview` accepts DeploymentInput JSON, returns valid YAML
- [ ] `POST /api/v1/wizards/service/preview` accepts ServiceInput JSON, returns valid YAML
- [ ] Validation returns 422 with per-field errors for invalid input
- [ ] Generated YAML includes TypeMeta (apiVersion, kind)
- [ ] Generated YAML applies successfully via `POST /api/v1/yaml/apply`
- [ ] Unit tests for DeploymentInput.Validate(), ToDeployment(), ToYAML()
- [ ] Unit tests for ServiceInput.Validate(), ToService(), ToYAML()
- [ ] Handler tests for preview endpoints (valid input, validation errors, edge cases)

### Frontend
- [ ] MonacoEditor extracted to component (YamlEditor island still works as wrapper)
- [ ] WizardStepper shows step progress (completed/current/upcoming)
- [ ] Deployment wizard: 4 steps with all form fields, validation on Next
- [ ] Service wizard: 3 steps with all form fields, validation on Next
- [ ] Review step shows backend-generated YAML in Monaco (editable)
- [ ] Apply button sends YAML to existing apply endpoint, shows results
- [ ] Success shows "View Resource" link to detail page
- [ ] Cancel button with dirty state guard
- [ ] Namespace pre-filled from TopBar selector
- [ ] Labels auto-populated with `app: <name>`
- [ ] Dynamic lists (ports, env vars, labels) with add/remove
- [ ] Route pages at `/workloads/deployments/new` and `/networking/services/new`
- [ ] "Create" buttons on deployment and service list pages

### Testing
- [ ] Backend: `go test ./internal/wizard/... -race -count=1` passes
- [ ] Frontend: `deno lint && deno fmt --check` passes
- [ ] Smoke test: create a deployment via wizard against homelab, verify pod runs
- [ ] Smoke test: create a service via wizard against homelab, verify endpoint exists

## Dependencies & Risks

**Dependencies:**
- Monaco editor extraction (Phase 2) must complete before wizard islands (Phase 4-5)
- `sigs.k8s.io/yaml` already in go.mod transitively; may need direct import

**Risks:**
- Fresh 2.x route resolution: `routes/workloads/deployments/new.tsx` must not conflict with `[namespace]/[name].tsx`. Fresh resolves static before dynamic at the same depth — verified safe.
- Monaco CDN load race with preview fetch on Review step: the existing `YamlEditor` handles `setValue` after init via `useEffect` — safe.
- SSA silently updates existing resources (no "already exists" warning): accepted trade-off for Phase 1.

## Files to Create

```
backend/internal/wizard/
  deployment.go         # DeploymentInput, ToDeployment(), Validate()
  service.go            # ServiceInput, ToService(), Validate()
  handler.go            # HandleDeploymentPreview, HandleServicePreview
  wizard_test.go        # Unit tests

frontend/components/
  ui/MonacoEditor.tsx   # Extracted Monaco core
  ui/Select.tsx         # Dropdown select component
  wizard/WizardStepper.tsx          # Step indicator + navigation
  wizard/DeploymentBasicsStep.tsx   # Step 1
  wizard/DeploymentNetworkStep.tsx  # Step 2
  wizard/DeploymentResourcesStep.tsx # Step 3
  wizard/ServiceBasicsStep.tsx      # Step 1
  wizard/ServicePortsStep.tsx       # Step 2
  wizard/WizardReviewStep.tsx       # Shared Review step

frontend/islands/
  DeploymentWizard.tsx  # Deployment wizard island
  ServiceWizard.tsx     # Service wizard island

frontend/routes/
  workloads/deployments/new.tsx   # Route page
  networking/services/new.tsx     # Route page
```

## Files to Modify

```
backend/internal/server/server.go   # Add WizardHandler to Server/Deps
backend/internal/server/routes.go   # Register /wizards routes
frontend/islands/YamlEditor.tsx     # Thin wrapper around MonacoEditor component
frontend/islands/ResourceTable.tsx  # Add optional "Create" action button
```

## References

- Existing YAML apply pipeline: `backend/internal/yaml/handler.go`
- Existing deployment CRUD: `backend/internal/k8s/resources/deployments.go`
- Existing service CRUD: `backend/internal/k8s/resources/services.go`
- YamlEditor island: `frontend/islands/YamlEditor.tsx`
- YamlApplyPage island: `frontend/islands/YamlApplyPage.tsx`
- ResourceDetail island: `frontend/islands/ResourceDetail.tsx`
- API client: `frontend/lib/api.ts` (`apiPost` for JSON, `apiPostRaw` for YAML)
- Namespace signal: `frontend/lib/namespace.ts`
- UI components: `frontend/components/ui/` (Button, Input, Card, Tabs, etc.)
- Route pattern: `frontend/routes/workloads/deployments.tsx`
- CLAUDE.md Step 8 specification
- Related PRs: #1-#7 (Steps 1-7), #8-#10 (smoke test fixes)
