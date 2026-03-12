# CLAUDE.md — KubeCenter: Kubernetes Management Platform

## Project Vision

KubeCenter is a web-based Kubernetes management platform that delivers vCenter-level functionality for Kubernetes clusters. It provides GUI-driven wizards for all cluster operations (deployments, CSI, CNI, networking, storage), integrated Prometheus/Grafana observability, RBAC-aware multi-tenancy, and full YAML escape hatches for power users. It is deployed via Helm chart inside the managed cluster, with architecture designed from day one to support multi-cluster management in a future phase.

---

## Technology Stack

| Layer | Technology | Version |
|---|---|---|
| Backend API | Go | 1.25.x |
| Kubernetes Client | client-go, controller-runtime | Latest stable matching target k8s version |
| HTTP Router | chi (go-chi/chi/v5) | v5.x |
| WebSocket | gorilla/websocket | v1.5.x |
| Frontend Runtime | Deno | 2.x (latest stable) |
| Frontend Framework | Fresh | 2.x (Deno-native, islands architecture) |
| Language | TypeScript | Strict mode, ESM only |
| CSS | Tailwind CSS | v4.x |
| YAML Editor | Monaco Editor | Latest (via ESM CDN import) |
| Charts (custom) | Apache ECharts or Chart.js | Latest |
| Monitoring | Prometheus + Grafana | kube-prometheus-stack compatible |
| Alerting | Prometheus Alertmanager + SMTP | Via Go SMTP client |
| Auth | OIDC / LDAP / Local (Argon2id) | Go packages: coreos/go-oidc, go-ldap |
| Deployment | Helm | v3.x chart |
| Container | Distroless / Alpine-based multi-stage | Scratch for Go, Deno slim for frontend |

---

## Project Structure

```
kubecenter/
├── CLAUDE.md                          # This file — project context for Claude Code
├── README.md                          # User-facing documentation
├── Makefile                           # Build, test, lint, Docker targets
├── docker-compose.dev.yml             # Local dev environment
│
├── backend/                           # Go 1.25 backend
│   ├── go.mod
│   ├── go.sum
│   ├── cmd/
│   │   └── kubecenter/
│   │       └── main.go                # Entrypoint — starts HTTP server, k8s informers, WS hub
│   ├── internal/
│   │   ├── server/
│   │   │   ├── server.go              # HTTP server setup, middleware chain, route registration
│   │   │   ├── routes.go              # All route definitions (delegates to handlers)
│   │   │   └── middleware/
│   │   │       ├── auth.go            # JWT validation, RBAC enforcement middleware
│   │   │       ├── audit.go           # Audit logging middleware
│   │   │       ├── ratelimit.go       # Rate limiting
│   │   │       └── cors.go            # CORS configuration
│   │   │
│   │   ├── auth/
│   │   │   ├── provider.go            # Auth provider interface
│   │   │   ├── local.go               # Local account provider (Argon2id)
│   │   │   ├── oidc.go                # OIDC provider (coreos/go-oidc/v3)
│   │   │   ├── ldap.go                # LDAP provider (go-ldap/ldap/v3)
│   │   │   ├── jwt.go                 # JWT token creation, validation, refresh
│   │   │   ├── rbac.go                # Map GUI users → k8s RBAC via impersonation
│   │   │   └── session.go             # Session management, token store
│   │   │
│   │   ├── k8s/
│   │   │   ├── client.go              # Kubernetes client factory, kubeconfig loading
│   │   │   ├── informers.go           # Shared informer factory, cache warming
│   │   │   ├── impersonation.go       # User impersonation for RBAC-scoped k8s calls
│   │   │   ├── resources/
│   │   │   │   ├── deployments.go     # CRUD + scale + rollback for Deployments
│   │   │   │   ├── statefulsets.go    # StatefulSet operations
│   │   │   │   ├── daemonsets.go      # DaemonSet operations
│   │   │   │   ├── pods.go            # Pod list, logs, exec, delete
│   │   │   │   ├── services.go        # Service CRUD (ClusterIP, NodePort, LoadBalancer)
│   │   │   │   ├── ingresses.go       # Ingress/IngressClass management
│   │   │   │   ├── configmaps.go      # ConfigMap CRUD
│   │   │   │   ├── secrets.go         # Secret CRUD (values masked in API responses)
│   │   │   │   ├── namespaces.go      # Namespace lifecycle
│   │   │   │   ├── nodes.go           # Node list, cordon, drain, labels
│   │   │   │   ├── pvcs.go            # PVC management, storage class binding
│   │   │   │   ├── jobs.go            # Job and CronJob operations
│   │   │   │   ├── networkpolicies.go # NetworkPolicy CRUD
│   │   │   │   ├── rbac.go            # Roles, ClusterRoles, Bindings viewer
│   │   │   │   ├── crds.go            # CRD discovery and generic CR management
│   │   │   │   └── generic.go         # Generic resource handler for unstructured objects
│   │   │   ├── storage/
│   │   │   │   ├── csi.go             # CSI driver discovery, StorageClass management
│   │   │   │   ├── csi_wizard.go      # Structured CSI configuration logic
│   │   │   │   └── snapshot.go        # VolumeSnapshot operations
│   │   │   └── networking/
│   │   │       ├── cni.go             # CNI detection (Cilium, Calico, Flannel)
│   │   │       ├── cilium.go          # Cilium-specific: CiliumNetworkPolicy, Hubble, ClusterMesh
│   │   │       └── cni_wizard.go      # CNI configuration wizard backend logic
│   │   │
│   │   ├── monitoring/
│   │   │   ├── discovery.go           # Auto-discover existing Prometheus/Grafana in cluster
│   │   │   ├── prometheus.go          # Prometheus client — PromQL queries, range queries
│   │   │   ├── grafana.go             # Grafana API client — dashboard provisioning, org/user setup
│   │   │   ├── metrics.go             # Pre-built metric query definitions for each resource type
│   │   │   ├── dashboards/            # Grafana dashboard JSON provisioning templates
│   │   │   │   ├── cluster_overview.json
│   │   │   │   ├── node_detail.json
│   │   │   │   ├── pod_detail.json
│   │   │   │   ├── deployment_detail.json
│   │   │   │   ├── pvc_storage.json
│   │   │   │   ├── service_networking.json
│   │   │   │   └── cilium_networking.json
│   │   │   └── alerts/
│   │   │       ├── rules.go           # Default PrometheusRule definitions
│   │   │       └── defaults.yaml      # Baseline alert rules YAML
│   │   │
│   │   ├── alerting/
│   │   │   ├── manager.go             # Alert pipeline — receives from Alertmanager webhook
│   │   │   ├── smtp.go                # SMTP email sender (Go net/smtp + TLS)
│   │   │   ├── templates/             # Go html/template email templates
│   │   │   │   ├── alert.html
│   │   │   │   └── digest.html
│   │   │   └── store.go              # Alert history persistence (SQLite or in-cluster ConfigMap)
│   │   │
│   │   ├── yaml/
│   │   │   ├── parser.go             # Multi-doc YAML parsing and validation
│   │   │   ├── validator.go           # Schema validation against OpenAPI specs from API server
│   │   │   ├── applier.go            # Server-side apply logic for arbitrary YAML
│   │   │   └── differ.go             # YAML diff engine for showing changes before apply
│   │   │
│   │   ├── websocket/
│   │   │   ├── hub.go                # Connection hub — fan-out resource events to subscribers
│   │   │   ├── client.go             # Per-connection client, auth, subscriptions
│   │   │   └── events.go             # Event types: resource updates, log streams, alerts
│   │   │
│   │   ├── audit/
│   │   │   ├── logger.go             # Structured audit log (who did what, when, to which resource)
│   │   │   └── store.go              # Audit log persistence
│   │   │
│   │   └── config/
│   │       ├── config.go             # App configuration struct, env + file loading
│   │       └── defaults.go           # Sensible defaults
│   │
│   ├── pkg/                          # Public packages (potentially shared with CLI later)
│   │   ├── api/
│   │   │   └── types.go              # Shared API request/response types
│   │   └── version/
│   │       └── version.go            # Build version info (ldflags)
│   │
│   └── Dockerfile                    # Multi-stage: Go build → distroless/static
│
├── frontend/                         # Deno 2.x + Fresh 2.x frontend
│   ├── deno.json                     # Deno config: compilerOptions, imports, tasks
│   ├── deno.lock
│   ├── main.ts                       # Fresh app entrypoint
│   ├── fresh.config.ts               # Fresh configuration
│   ├── tailwind.config.ts            # Tailwind v4 configuration
│   ├── static/
│   │   ├── favicon.ico
│   │   ├── logo.svg
│   │   └── styles/
│   │       └── global.css            # Tailwind directives + custom properties
│   │
│   ├── routes/                       # Fresh file-based routing
│   │   ├── _app.tsx                  # Root layout — sidebar nav, top bar, auth context
│   │   ├── _layout.tsx               # Authenticated layout wrapper
│   │   ├── index.tsx                 # Dashboard home — cluster overview
│   │   ├── login.tsx                 # Login page (local + SSO options)
│   │   │
│   │   ├── cluster/
│   │   │   ├── index.tsx             # Cluster overview — nodes, health, capacity
│   │   │   ├── nodes/
│   │   │   │   ├── index.tsx         # Node list with status, resources
│   │   │   │   └── [name].tsx        # Node detail — metrics, pods, labels, taints
│   │   │   └── events.tsx            # Cluster events stream
│   │   │
│   │   ├── workloads/
│   │   │   ├── index.tsx             # Workloads overview — all deployment types
│   │   │   ├── deployments/
│   │   │   │   ├── index.tsx         # Deployment list
│   │   │   │   ├── [ns]/[name].tsx   # Deployment detail + performance tab
│   │   │   │   └── create.tsx        # CREATE WIZARD: repo, image, replicas, resources, env, volumes
│   │   │   ├── statefulsets/
│   │   │   │   ├── index.tsx
│   │   │   │   ├── [ns]/[name].tsx
│   │   │   │   └── create.tsx
│   │   │   ├── daemonsets/
│   │   │   │   ├── index.tsx
│   │   │   │   ├── [ns]/[name].tsx
│   │   │   │   └── create.tsx
│   │   │   ├── jobs/
│   │   │   │   ├── index.tsx
│   │   │   │   └── [ns]/[name].tsx
│   │   │   └── pods/
│   │   │       ├── index.tsx         # Pod list, filterable by namespace/label
│   │   │       └── [ns]/[name].tsx   # Pod detail — logs, exec terminal, events, metrics
│   │   │
│   │   ├── networking/
│   │   │   ├── index.tsx             # Networking overview
│   │   │   ├── services/
│   │   │   │   ├── index.tsx
│   │   │   │   ├── [ns]/[name].tsx
│   │   │   │   └── create.tsx        # Service creation wizard
│   │   │   ├── ingresses/
│   │   │   │   ├── index.tsx
│   │   │   │   ├── [ns]/[name].tsx
│   │   │   │   └── create.tsx
│   │   │   ├── policies/
│   │   │   │   ├── index.tsx         # NetworkPolicy + CiliumNetworkPolicy list
│   │   │   │   └── create.tsx        # Policy wizard — visual rule builder
│   │   │   └── cni/
│   │   │       ├── index.tsx         # CNI status, detected plugin
│   │   │       └── configure.tsx     # CNI CONFIGURATION WIZARD (Cilium, Calico, etc.)
│   │   │
│   │   ├── storage/
│   │   │   ├── index.tsx             # Storage overview — PVCs, PVs, StorageClasses
│   │   │   ├── pvcs/
│   │   │   │   ├── index.tsx
│   │   │   │   ├── [ns]/[name].tsx
│   │   │   │   └── create.tsx
│   │   │   ├── classes/
│   │   │   │   ├── index.tsx         # StorageClass list
│   │   │   │   └── create.tsx        # StorageClass wizard
│   │   │   └── csi/
│   │   │       ├── index.tsx         # CSI driver status, health
│   │   │       └── configure.tsx     # CSI CONFIGURATION WIZARD
│   │   │
│   │   ├── config/
│   │   │   ├── configmaps/
│   │   │   │   ├── index.tsx
│   │   │   │   ├── [ns]/[name].tsx
│   │   │   │   └── create.tsx
│   │   │   ├── secrets/
│   │   │   │   ├── index.tsx         # Secret list (values hidden)
│   │   │   │   ├── [ns]/[name].tsx   # Secret detail (reveal on click with audit)
│   │   │   │   └── create.tsx
│   │   │   └── rbac/
│   │   │       ├── index.tsx         # Roles, ClusterRoles, Bindings viewer
│   │   │       └── create.tsx
│   │   │
│   │   ├── monitoring/
│   │   │   ├── index.tsx             # PERFORMANCE TAB — cluster-wide Grafana dashboards
│   │   │   ├── dashboards.tsx        # Grafana dashboard browser/embed
│   │   │   ├── alerts/
│   │   │   │   ├── index.tsx         # Active alerts, alert history
│   │   │   │   ├── rules.tsx         # Alert rule management
│   │   │   │   └── settings.tsx      # SMTP config, notification routing
│   │   │   └── prometheus.tsx        # Direct PromQL query interface (power users)
│   │   │
│   │   ├── yaml/
│   │   │   ├── apply.tsx             # YAML apply page — upload or paste, validate, diff, apply
│   │   │   └── editor.tsx            # Full Monaco YAML editor with k8s schema completion
│   │   │
│   │   ├── settings/
│   │   │   ├── index.tsx             # App settings overview
│   │   │   ├── auth.tsx              # Auth provider config (OIDC, LDAP, local users)
│   │   │   ├── monitoring.tsx        # Prometheus/Grafana connection settings
│   │   │   └── about.tsx             # Version, cluster info, license
│   │   │
│   │   └── api/                      # Fresh API routes (BFF pattern — proxy to Go backend)
│   │       └── [...path].ts          # Catch-all API proxy to Go backend
│   │
│   ├── islands/                      # Fresh interactive islands (hydrated on client)
│   │   ├── Sidebar.tsx               # Navigation sidebar — resource tree like vCenter
│   │   ├── TopBar.tsx                # Namespace selector, user menu, cluster indicator
│   │   ├── ResourceTable.tsx         # Generic sortable/filterable table for any k8s resource
│   │   ├── ResourceDetail.tsx        # Generic resource detail with tabbed view
│   │   ├── PerformancePanel.tsx      # Grafana embed + custom ECharts metrics panel
│   │   ├── YamlEditor.tsx            # Monaco editor island for YAML editing
│   │   ├── YamlDiffViewer.tsx        # Side-by-side diff view before applying changes
│   │   ├── TerminalEmbed.tsx         # xterm.js pod exec terminal
│   │   ├── LogViewer.tsx             # Real-time log streaming viewer
│   │   ├── WizardStepper.tsx         # Reusable multi-step wizard component
│   │   ├── DeploymentWizard.tsx      # Step-by-step deployment creation
│   │   ├── CsiWizard.tsx             # CSI driver configuration wizard
│   │   ├── CniWizard.tsx             # CNI configuration wizard (Cilium-aware)
│   │   ├── ServiceWizard.tsx         # Service creation wizard
│   │   ├── AlertBanner.tsx           # Real-time alert notification banner
│   │   ├── EventStream.tsx           # Live cluster event feed
│   │   └── ResourceTopology.tsx      # Visual resource relationship graph (D3-based)
│   │
│   ├── components/                   # Non-interactive shared components (SSR-safe)
│   │   ├── ui/                       # Base UI primitives
│   │   │   ├── Button.tsx
│   │   │   ├── Input.tsx
│   │   │   ├── Select.tsx
│   │   │   ├── Modal.tsx
│   │   │   ├── Tabs.tsx
│   │   │   ├── Badge.tsx
│   │   │   ├── Card.tsx
│   │   │   ├── Toast.tsx
│   │   │   ├── Tooltip.tsx
│   │   │   ├── DataTable.tsx
│   │   │   └── Pagination.tsx
│   │   ├── layout/
│   │   │   ├── PageHeader.tsx
│   │   │   ├── Section.tsx
│   │   │   └── EmptyState.tsx
│   │   └── k8s/                      # Kubernetes-specific display components
│   │       ├── StatusBadge.tsx        # Pod phase, deployment status indicators
│   │       ├── ResourceIcon.tsx       # Icons for each k8s resource type
│   │       ├── NamespaceSelector.tsx
│   │       ├── LabelSelector.tsx
│   │       └── ResourceLink.tsx       # Clickable link to any k8s resource
│   │
│   ├── lib/                          # Shared utilities
│   │   ├── api.ts                    # API client — typed fetch wrapper to backend
│   │   ├── ws.ts                     # WebSocket client — connect, subscribe, reconnect
│   │   ├── auth.ts                   # Auth context, token management, login/logout
│   │   ├── k8s-types.ts             # TypeScript types mirroring k8s API objects
│   │   ├── formatters.ts            # CPU, memory, age, status formatters
│   │   ├── yaml-utils.ts            # YAML parse/stringify helpers
│   │   └── constants.ts             # Route paths, resource kinds, default values
│   │
│   └── Dockerfile                    # Deno-based container image
│
├── helm/
│   └── kubecenter/
│       ├── Chart.yaml
│       ├── values.yaml               # Comprehensive defaults with documentation
│       ├── values.schema.json         # JSON schema for values validation
│       ├── templates/
│       │   ├── _helpers.tpl
│       │   ├── namespace.yaml
│       │   ├── deployment-backend.yaml
│       │   ├── deployment-frontend.yaml
│       │   ├── service-backend.yaml
│       │   ├── service-frontend.yaml
│       │   ├── ingress.yaml
│       │   ├── serviceaccount.yaml
│       │   ├── clusterrole.yaml       # Read/write permissions for managed resources
│       │   ├── clusterrolebinding.yaml
│       │   ├── configmap-app.yaml     # Application configuration
│       │   ├── secret-app.yaml        # Sensitive config (JWT secret, SMTP creds)
│       │   ├── networkpolicy.yaml     # Restrict traffic to/from KubeCenter pods
│       │   ├── poddisruptionbudget.yaml
│       │   │
│       │   ├── monitoring/            # Conditional: deploy monitoring stack
│       │   │   ├── prometheus-values.yaml   # Values override for kube-prometheus-stack subchart
│       │   │   ├── grafana-datasource.yaml
│       │   │   ├── grafana-dashboards-cm.yaml
│       │   │   └── alertmanager-config.yaml
│       │   │
│       │   └── tests/
│       │       └── test-connection.yaml
│       │
│       └── charts/                    # Subcharts (conditional)
│           └── .gitkeep               # kube-prometheus-stack added as dependency in Chart.yaml
│
├── docs/
│   ├── architecture.md               # Detailed architecture document
│   ├── api-reference.md              # REST API documentation
│   ├── development.md                # Developer setup guide
│   ├── deployment.md                 # Production deployment guide
│   └── security.md                   # Security model documentation
│
├── scripts/
│   ├── dev-setup.sh                  # Bootstrap local dev environment
│   ├── generate-certs.sh             # Self-signed TLS certs for dev
│   └── seed-demo-data.sh             # Populate cluster with demo workloads
│
└── .github/
    └── workflows/
        ├── ci.yml                    # Lint, test, build
        └── release.yml               # Build images, push to registry, Helm package
```

---

## Architecture Principles

### 1. Backend (Go) Design Rules

- **All Kubernetes API calls go through user impersonation.** Never use the service account's own permissions for user-initiated actions. The backend impersonates the authenticated user's k8s identity so that Kubernetes RBAC is enforced server-side. The service account needs `impersonate` permissions only.
- **Informers for read, direct API calls for write.** Use `SharedInformerFactory` with label/field selectors to maintain an in-memory cache of cluster state. All list/get operations read from the informer cache. All create/update/delete operations go through the API server directly, with impersonation.
- **Server-side apply for all YAML operations.** Use `PATCH` with `application/apply-patch+yaml` content type. Never use `kubectl apply` under the hood.
- **WebSocket hub pattern for real-time updates.** A central hub goroutine receives events from informers and fans them out to connected WebSocket clients. Clients subscribe to specific resource types and namespaces. Authenticate WebSocket connections with the same JWT used for REST.
- **Structured logging with slog.** Use Go 1.25's `log/slog` package with JSON output. Include request ID, user identity, resource kind, namespace, and name in all log entries.
- **Error handling: never expose internal errors to users.** Wrap k8s API errors into user-friendly messages. Return appropriate HTTP status codes. Log full error details server-side.
- **Configuration via environment variables with YAML file fallback.** Use a single config struct loaded at startup. Env vars override YAML file values. All secrets come from env vars or k8s Secrets, never config files.

### 2. Frontend (Deno/Fresh) Design Rules

- **Islands architecture strictly enforced.** Only components that require client-side interactivity (forms, editors, WebSocket consumers, charts) are islands. Everything else is server-rendered HTML. This minimizes JavaScript sent to the client.
- **API client is a typed wrapper.** All backend calls go through `lib/api.ts` which handles auth token injection, error parsing, and response typing. Never use raw `fetch` in components.
- **Wizard components follow a consistent pattern.** Every wizard uses `WizardStepper.tsx` as its shell. Steps are defined as an array of `{ title, component, validate }` objects. The wizard handles navigation, validation, and final submission. On the final step, the wizard shows a YAML preview of what will be applied, with an option to edit the YAML before applying.
- **Dual-mode for every configuration.** Every resource creation/edit page offers both a wizard/form mode and a raw YAML mode. A toggle switches between them. Changes in one mode are reflected in the other in real-time (form→YAML serialization and YAML→form parsing).
- **Grafana embeds use `<iframe>` with auth proxy.** The backend proxies Grafana with proper auth headers. The frontend embeds Grafana panels via iframe pointed at the backend proxy endpoint. This avoids exposing Grafana directly and handles auth seamlessly.
- **Tailwind CSS utility-only.** No custom CSS files except for the global Tailwind directives and CSS custom properties for theming (dark mode support). Use Tailwind's `@apply` sparingly and only in the global stylesheet for base element styles.
- **Consistent color semantics.** Use CSS custom properties for status colors: `--color-success` (green, healthy/running), `--color-warning` (amber, pending/degraded), `--color-danger` (red, failed/error), `--color-info` (blue, informational). Map k8s resource states to these consistently everywhere.

### 3. Security Rules

- **TLS everywhere.** The backend serves HTTPS. In-cluster, use cert-manager to provision TLS certificates. The Helm chart includes cert-manager Certificate resources.
- **JWT tokens are short-lived (15 min) with refresh tokens (7 day).** Refresh tokens are stored server-side (not in localStorage). Access tokens are sent as `Authorization: Bearer` headers. Refresh via a dedicated `/api/auth/refresh` endpoint.
- **Secrets are never returned in full.** The secrets API endpoint returns metadata and masked values (`****`). A separate `reveal` endpoint returns the actual value, requires explicit user action, and is audit-logged.
- **Content Security Policy headers.** Strict CSP that allows only same-origin scripts, the Monaco CDN, and Grafana iframe sources.
- **Network Policies deployed by default.** The Helm chart includes NetworkPolicy resources that restrict ingress/egress to only what KubeCenter needs.
- **Pod Security Standards: restricted.** KubeCenter pods run as non-root, read-only root filesystem, no privilege escalation, drop all capabilities.
- **Audit logging for all write operations.** Every create, update, delete, and secret reveal is logged with: timestamp, user identity, source IP, resource type, resource name, namespace, action, and result.

### 4. Monitoring Integration Rules

- **Auto-discovery on startup.** The backend probes the cluster for existing Prometheus (by ServiceMonitor CRDs and well-known service names) and Grafana instances. Results are cached and re-checked periodically.
- **If bringing your own Prometheus/Grafana:** the backend configures itself as a Prometheus client pointing at the discovered endpoint. For Grafana, it provisions dashboards via the Grafana HTTP API using a service account token.
- **If deploying fresh:** the Helm chart includes `kube-prometheus-stack` as a conditional subchart dependency (`monitoring.enabled: true` in values.yaml). Prometheus, Grafana, kube-state-metrics, and node-exporter are deployed with pre-configured scrape targets and dashboards.
- **Pre-built PromQL queries for every resource type.** The `internal/monitoring/metrics.go` file contains named query templates for: pod CPU/memory, deployment replica health, PVC usage, service latency (if available), node resource utilization, Cilium network flow metrics.
- **Grafana dashboards are provisioned as ConfigMaps.** JSON dashboard definitions are baked into the Helm chart and loaded via Grafana's sidecar provisioner. They are parameterized with template variables for namespace, pod, node, etc.

---

## API Design

### REST Endpoints (Go Backend)

All endpoints are prefixed with `/api/v1`.

```
# Authentication
POST   /api/v1/auth/login            # Local login (username + password)
POST   /api/v1/auth/oidc/callback    # OIDC callback
POST   /api/v1/auth/refresh           # Refresh access token
POST   /api/v1/auth/logout            # Invalidate session
GET    /api/v1/auth/providers         # List configured auth providers
GET    /api/v1/auth/me                # Current user info + k8s RBAC summary

# Generic Kubernetes Resources (pattern repeats for each resource type)
GET    /api/v1/resources/:kind                    # List across all namespaces
GET    /api/v1/resources/:kind/:namespace          # List in namespace
GET    /api/v1/resources/:kind/:namespace/:name    # Get specific resource
POST   /api/v1/resources/:kind/:namespace          # Create resource (JSON or YAML body)
PUT    /api/v1/resources/:kind/:namespace/:name    # Update resource
DELETE /api/v1/resources/:kind/:namespace/:name    # Delete resource
PATCH  /api/v1/resources/:kind/:namespace/:name    # Patch resource (strategic merge)

# Specialized Resource Endpoints
POST   /api/v1/deployments/:ns/:name/scale        # Scale deployment
POST   /api/v1/deployments/:ns/:name/rollback      # Rollback to revision
POST   /api/v1/deployments/:ns/:name/restart        # Rolling restart
GET    /api/v1/pods/:ns/:name/logs                  # Stream pod logs (SSE)
POST   /api/v1/pods/:ns/:name/exec                  # WebSocket pod exec
GET    /api/v1/nodes/:name/drain                    # Drain node (long-running)
POST   /api/v1/nodes/:name/cordon                   # Cordon/uncordon

# YAML Operations
POST   /api/v1/yaml/validate          # Validate YAML against cluster's OpenAPI schema
POST   /api/v1/yaml/apply             # Server-side apply (supports multi-doc)
POST   /api/v1/yaml/diff              # Dry-run apply and return diff against current state
POST   /api/v1/yaml/export/:kind/:ns/:name   # Export resource as clean YAML

# Storage (CSI)
GET    /api/v1/storage/drivers         # List CSI drivers and their capabilities
GET    /api/v1/storage/classes         # List StorageClasses with CSI driver info
POST   /api/v1/storage/classes         # Create StorageClass via wizard payload
GET    /api/v1/storage/snapshots       # List VolumeSnapshots

# Networking (CNI)
GET    /api/v1/networking/cni          # Detected CNI plugin and version
GET    /api/v1/networking/cni/config   # Current CNI configuration (Cilium CiliumConfig, etc.)
PUT    /api/v1/networking/cni/config   # Update CNI configuration via wizard payload
GET    /api/v1/networking/cilium/status  # Cilium agent status, Hubble status

# Monitoring
GET    /api/v1/monitoring/status       # Prometheus + Grafana connection status
GET    /api/v1/monitoring/query        # Proxy PromQL instant query
GET    /api/v1/monitoring/query_range  # Proxy PromQL range query
GET    /api/v1/monitoring/dashboards   # List available Grafana dashboards
GET    /api/v1/monitoring/grafana/proxy/*  # Reverse proxy to Grafana for iframe embedding

# Alerting
GET    /api/v1/alerts                  # Current active alerts
GET    /api/v1/alerts/history          # Alert history
GET    /api/v1/alerts/rules            # Configured alert rules
POST   /api/v1/alerts/rules            # Create/update alert rule
DELETE /api/v1/alerts/rules/:id        # Delete alert rule
PUT    /api/v1/alerts/settings         # SMTP configuration, notification routing
POST   /api/v1/alerts/test             # Send test email

# Cluster
GET    /api/v1/cluster/info            # Cluster version, node count, resource summary
GET    /api/v1/cluster/events          # Cluster events (paginated)
GET    /api/v1/cluster/namespaces      # Namespace list (for selector dropdowns)
GET    /api/v1/cluster/api-resources   # Available API resources (for dynamic resource discovery)

# Audit
GET    /api/v1/audit/logs              # Audit log entries (paginated, filterable)

# Settings
GET    /api/v1/settings                # Current application settings
PUT    /api/v1/settings                # Update application settings
GET    /api/v1/settings/auth           # Auth provider configuration
PUT    /api/v1/settings/auth           # Update auth provider configuration
```

### WebSocket Endpoints

```
WS /api/v1/ws/resources    # Subscribe to resource events (watch)
                            # Client sends: { "subscribe": { "kind": "pods", "namespace": "default" } }
                            # Server sends: { "type": "ADDED|MODIFIED|DELETED", "object": {...} }

WS /api/v1/ws/logs/:ns/:pod/:container   # Real-time log stream

WS /api/v1/ws/exec/:ns/:pod/:container   # Pod exec terminal (stdin/stdout/stderr/resize)

WS /api/v1/ws/alerts       # Real-time alert notifications
```

---

## Key Implementation Details

### Kubernetes Client Initialization (backend/internal/k8s/client.go)

```go
// Use in-cluster config since we deploy via Helm inside the cluster.
// The service account is configured with impersonation permissions.
// For every user-initiated request, create an impersonating client:
//
//   config, _ := rest.InClusterConfig()
//   config.Impersonate = rest.ImpersonationConfig{
//       UserName: authenticatedUser.KubernetesUsername,
//       Groups:   authenticatedUser.KubernetesGroups,
//   }
//   clientset, _ := kubernetes.NewForConfig(config)
//
// The informer factory uses the SERVICE ACCOUNT's own permissions (broad read access)
// but all write operations use the impersonating client.
```

### Wizard → YAML Pipeline

Every wizard follows this data flow:
1. User fills in wizard steps (frontend form state)
2. Frontend serializes form state into a structured JSON payload
3. Backend receives JSON, constructs a Kubernetes object programmatically using client-go typed structs
4. Backend serializes the object to YAML
5. Backend returns the YAML to the frontend for preview
6. User reviews YAML in Monaco editor (can edit)
7. User clicks "Apply" — backend validates and applies via server-side apply
8. If the user toggled to YAML mode at any point, the form state is repopulated by parsing the YAML back into the structured payload

### Monitoring Bootstrap Sequence (on Helm install)

```
1. Helm install starts
2. If values.monitoring.deploy = true:
   a. kube-prometheus-stack subchart deploys Prometheus, Grafana, kube-state-metrics, node-exporter
   b. ConfigMaps with KubeCenter Grafana dashboards are deployed
   c. Alertmanager is configured with webhook receiver pointing to KubeCenter backend
3. If values.monitoring.deploy = false (bring your own):
   a. KubeCenter backend starts and runs discovery:
      - Checks for Prometheus via ServiceMonitor CRD existence + well-known service names
      - Checks for Grafana via well-known service names + Grafana CRD
   b. If found, backend configures itself to use discovered endpoints
   c. Backend provisions dashboards into existing Grafana via API
   d. Backend configures webhook receiver in existing Alertmanager via API
4. Backend exposes /api/v1/monitoring/status for frontend to check readiness
```

---

## Build System

### Makefile Targets

```makefile
# Development
make dev              # Start backend + frontend in dev mode with hot reload
make dev-backend      # Backend only with air (hot reload)
make dev-frontend     # Frontend only with Deno --watch

# Build
make build            # Build both backend and frontend
make build-backend    # go build -o bin/kubecenter ./cmd/kubecenter
make build-frontend   # deno task build (Fresh static export)

# Docker
make docker-build     # Build both container images
make docker-push      # Push to registry

# Testing
make test             # Run all tests
make test-backend     # go test ./... -race -cover
make test-frontend    # deno test
make test-e2e         # End-to-end tests against a kind cluster
make lint             # golangci-lint + deno lint + deno fmt --check

# Helm
make helm-lint        # helm lint
make helm-template    # helm template (dry-run render)
make helm-package     # helm package
```

### Go Module (backend/go.mod)

```
module github.com/kubecenter/kubecenter

go 1.25

require (
    k8s.io/client-go            v0.32.x
    k8s.io/apimachinery          v0.32.x
    k8s.io/api                   v0.32.x
    sigs.k8s.io/controller-runtime v0.20.x
    github.com/go-chi/chi/v5     v5.x
    github.com/gorilla/websocket  v1.5.x
    github.com/coreos/go-oidc/v3  v3.x
    github.com/go-ldap/ldap/v3    v3.x
    github.com/golang-jwt/jwt/v5  v5.x
    golang.org/x/crypto           latest   // Argon2id
    github.com/prometheus/client_golang  latest
    github.com/grafana/grafana-api-golang-client latest
    gopkg.in/yaml.v3               v3.x
    github.com/santhosh-tekuri/jsonschema/v6 latest
)
```

### Deno Config (frontend/deno.json)

```json
{
  "compilerOptions": {
    "jsx": "react-jsx",
    "jsxImportSource": "preact",
    "strict": true
  },
  "nodeModulesDir": "auto",
  "imports": {
    "preact": "https://esm.sh/preact@10.x",
    "preact/": "https://esm.sh/preact@10.x/",
    "@preact/signals": "https://esm.sh/@preact/signals@2.x",
    "$fresh/": "https://deno.land/x/fresh@2.x/",
    "echarts": "https://esm.sh/echarts@5.x",
    "monaco-editor": "https://esm.sh/monaco-editor@latest"
  },
  "tasks": {
    "dev": "deno run -A --watch main.ts",
    "build": "deno run -A main.ts build",
    "preview": "deno run -A main.ts preview",
    "lint": "deno lint",
    "fmt": "deno fmt",
    "test": "deno test -A"
  }
}
```

---

## Development Setup

### Prerequisites
- Go 1.25+
- Deno 2.x+
- Docker + Docker Compose
- kind (Kubernetes in Docker) for local testing
- Helm 3.x
- kubectl

### Local Development Flow
```bash
# 1. Create local kind cluster with ingress
kind create cluster --config scripts/kind-config.yaml

# 2. Deploy monitoring stack (optional, for full feature testing)
helm install monitoring prometheus-community/kube-prometheus-stack -n monitoring --create-namespace

# 3. Start backend in dev mode (connects to kind cluster via kubeconfig)
cd backend && air  # or: go run ./cmd/kubecenter --dev

# 4. Start frontend in dev mode (proxies API to backend)
cd frontend && deno task dev

# 5. Access at http://localhost:8000 (Fresh dev server)
#    Backend API at http://localhost:8080/api/v1
```

---

## Key Conventions and Patterns

### Naming
- Go packages: lowercase, single word (`auth`, `k8s`, `monitoring`)
- Go files: snake_case (`csi_wizard.go`)
- TypeScript files: PascalCase for components (`DeploymentWizard.tsx`), camelCase for utilities (`api.ts`)
- API routes: kebab-case (`/api/v1/query-range`)
- CSS: Tailwind utility classes only. No custom class names.
- Helm values: camelCase (`monitoring.enabled`, `auth.oidc.issuerUrl`)

### Error Handling (Go)
```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to list deployments in namespace %s: %w", namespace, err)
}

// API handlers return structured errors
type APIError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Detail  string `json:"detail,omitempty"`
}
```

### API Response Format
```json
{
  "data": { ... },
  "metadata": {
    "total": 42,
    "page": 1,
    "pageSize": 20
  }
}
```

### Error Response Format
```json
{
  "error": {
    "code": 403,
    "message": "You do not have permission to delete pods in namespace production",
    "detail": "RBAC: user 'chris' lacks 'delete' permission on 'pods' in namespace 'production'"
  }
}
```

---

## Phase 1 Build Order (MVP)

Build in this order to have a working system at each step:

1. **Backend skeleton** — HTTP server, health check, config loading, in-cluster k8s client
2. **Auth system** — Local accounts with JWT, login/logout endpoints, auth middleware
3. **Resource listing** — Informer-backed list/get for pods, deployments, services, namespaces
4. **Frontend skeleton** — Fresh app with layout, sidebar, login page, dashboard home
5. **Resource browser** — Tables for pods, deployments, services with real-time WebSocket updates
6. **Resource detail** — Detail views with tabs (Overview, YAML, Events, Metrics placeholder)
7. **YAML apply** — Monaco editor, validation, diff, server-side apply
8. **Resource creation wizards** — Deployment wizard, Service wizard
9. **Monitoring integration** — Prometheus discovery/deploy, Grafana embed, performance tabs
10. **CSI/CNI wizards** — Storage and networking configuration
11. **Alerting** — Alertmanager webhook receiver, SMTP email notifications
12. **OIDC/LDAP auth** — SSO integration
13. **Helm chart** — Full production Helm chart with all configuration options
14. **Audit logging** — Comprehensive audit trail
15. **Polish** — Error handling, loading states, empty states, dark mode, keyboard shortcuts

---

## Multi-Cluster Preparation (Phase 2 Hooks)

Even in Phase 1, structure the code to support multi-cluster later:

- **Backend:** All k8s client operations accept a `clusterID` parameter (defaults to `"local"` in Phase 1). The client factory returns a client for the given cluster ID. In Phase 1, there is only one entry in the cluster registry.
- **Frontend:** The top bar includes a cluster selector component (disabled/hidden in Phase 1 with only one cluster). All API calls include a `X-Cluster-ID` header.
- **Database:** If any persistent state is added (audit logs, user preferences, alert history), include a `cluster_id` column from day one.
- **Helm:** The values.yaml includes a `clusters` array (with one entry in Phase 1) anticipating remote cluster kubeconfig registration.

---

## Testing Strategy

- **Backend unit tests:** Test each resource handler, auth provider, and monitoring client in isolation. Mock the k8s clientset using `k8s.io/client-go/kubernetes/fake`.
- **Backend integration tests:** Use `envtest` (from controller-runtime) to spin up a real API server for testing against actual k8s behavior.
- **Frontend tests:** Deno's built-in test runner for utility functions. Component tests with Preact Testing Library.
- **E2E tests:** Use a `kind` cluster with Playwright or Cypress driving the browser. Test the full wizard→apply→verify cycle.
- **Helm tests:** `helm lint`, `helm template` validation, and `helm test` hooks.

---

## Security Checklist (Enforce During Development)

- [ ] All API endpoints require authentication (except `/api/v1/auth/login`, `/api/v1/auth/oidc/callback`, `/health`, `/ready`)
- [ ] All user-initiated k8s operations use impersonation (never the service account's own permissions)
- [ ] Secret values are masked in all API responses and audit logs
- [ ] CSRF protection on all state-changing endpoints
- [ ] Rate limiting on auth endpoints (5 attempts/min per IP)
- [ ] Input validation on all API inputs (max lengths, allowed characters, k8s name regex)
- [ ] Container images run as non-root (UID 65534)
- [ ] No shell in production container images (distroless)
- [ ] Helm chart deploys NetworkPolicy restricting pod traffic
- [ ] TLS between all components (backend↔frontend, backend↔Prometheus, backend↔Grafana)
- [ ] JWT secrets are generated at install time and stored in k8s Secrets
- [ ] RBAC: ClusterRole has minimum required permissions with explicit resource lists (no wildcards)
- [ ] Audit log captures all write operations and secret accesses
- [ ] CSP headers prevent XSS via injected scripts
- [ ] WebSocket connections authenticated with same JWT as REST
