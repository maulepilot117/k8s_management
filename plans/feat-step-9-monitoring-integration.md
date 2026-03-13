# feat(monitoring): Step 9 — Prometheus & Grafana Monitoring Integration

## Overview

Add Prometheus and Grafana integration to KubeCenter so users can view resource-level performance metrics (CPU, memory, network, disk) directly in the resource detail pages, and access a monitoring overview with dashboard browsing and raw PromQL query interface. The backend auto-discovers existing monitoring stacks in the cluster, proxies PromQL queries, reverse-proxies Grafana for iframe embedding, and optionally deploys kube-prometheus-stack as a Helm subchart.

---

## Problem Statement / Motivation

Resource detail pages currently show a "Coming in Step 9" placeholder in the Metrics tab. Operators cannot correlate Kubernetes resource state with performance data without switching to a separate Grafana/Prometheus UI. KubeCenter should surface monitoring data inline — when you're looking at a pod that's crashlooping, you should see its CPU/memory graphs right there.

---

## Architecture Decisions

These decisions must be recorded before implementation begins.

### D1. Grafana Service Account Token Lifecycle

**Decision:** The Grafana API token is provided via configuration (`KUBECENTER_MONITORING_GRAFANA_APITOKEN` env var or `monitoring.grafana.apiToken` in values.yaml). The operator creates the service account in Grafana manually or via a Helm post-install Job. KubeCenter does NOT auto-create Grafana service accounts — this avoids needing Grafana admin credentials.

**Rationale:** Auto-creating service accounts requires the Grafana admin password, which is a security risk to store. Explicit token configuration is simpler and follows the "bring your own credentials" pattern used for JWT secrets.

**Fallback:** If no token is configured, Grafana features are degraded: no dashboard provisioning, no iframe embedding. Prometheus PromQL proxy still works independently.

### D2. Discovery Namespace Strategy

**Decision:** Discovery searches cluster-wide for services matching well-known names and label selectors. Configurable namespace hints via `KUBECENTER_MONITORING_NAMESPACE` (default: searches all namespaces). When multiple Prometheus/Grafana instances are found, select the first match in priority order:
1. Service in the configured namespace hint
2. Service with label `app.kubernetes.io/part-of=kube-prometheus-stack`
3. First match alphabetically by namespace/name

**Well-known service names (Prometheus):** `prometheus-kube-prometheus-prometheus`, `prometheus-operated`, `prometheus-server`, `prometheus`
**Well-known service names (Grafana):** `prometheus-grafana`, `kube-prometheus-stack-grafana`, `grafana`
**Label selectors:** `app.kubernetes.io/name=prometheus`, `app.kubernetes.io/name=grafana`

### D3. Grafana Proxy Path Allowlist

**Decision:** The Grafana reverse proxy at `/api/v1/monitoring/grafana/proxy/*` restricts paths to:
- `/d/` — Dashboard view
- `/d-solo/` — Solo panel (iframe)
- `/api/dashboards/` — Dashboard CRUD (for provisioning)
- `/api/folders/` — Folder CRUD (for provisioning)
- `/api/search` — Dashboard search
- `/public/` — Static assets (CSS/JS/fonts)

All other paths (especially `/api/admin/`, `/api/users/`, `/api/org/`) return 403. Path traversal (`..`, `%2e`) is blocked.

### D4. Monitoring RBAC Policy

**Decision:** All authenticated KubeCenter users can access monitoring endpoints (view metrics, browse dashboards). The raw PromQL query page (`/monitoring/prometheus`) is available to all authenticated users. PromQL does not write data and Prometheus metrics are not namespaced in the same way as Kubernetes resources — enforcing namespace-scoped PromQL would require query rewriting, which is out of scope for MVP.

**Future:** Step 12 (OIDC/LDAP) could add role-based restrictions on the PromQL proxy.

### D5. Dashboard Provisioning Policy

**Decision:** `overwrite: true` — dashboards are re-provisioned on every backend startup. Manual customizations to KubeCenter-provisioned dashboards (identified by `kubecenter-` UID prefix) will be overwritten. This ensures dashboards stay in sync with the KubeCenter version.

### D6. Grafana Client Library

**Decision:** Use raw `net/http` calls for Grafana API interactions. The `grafana-api-golang-client` library has maintenance gaps and we only need 3 endpoints (create folder, upsert dashboard, search dashboards). Raw HTTP is simpler and avoids an external dependency.

### D7. Prometheus Client Library

**Decision:** Use `github.com/prometheus/client_golang/api/prometheus/v1` for typed PromQL queries. This handles response parsing, error mapping, and timeout injection. Do NOT add `github.com/prometheus/prometheus` (the full server package) just for the PromQL parser — use response-based validation instead.

### D8. iframe Rendering Strategy

**Decision:** The Metrics tab content is mounted on first activation and stays mounted (not unmounted on tab switch). Use Grafana's `?kiosk=1` parameter for clean rendering. The iframe `src` includes `&refresh=30s` for auto-refresh. If the tab container is hidden via CSS, use `display: none` (not `visibility: hidden`) and trigger an iframe `postMessage` resize on tab re-activation.

### D9. Manual Re-discovery

**Decision:** Add `POST /api/v1/monitoring/rediscover` endpoint that triggers an immediate re-check and returns the new status. The monitoring status page shows a "Re-scan" button. This avoids the 5-minute wait during initial setup.

---

## MonitoringConfig Struct

Add to `backend/internal/config/config.go`:

```go
// config.go
type MonitoringConfig struct {
    Namespace     string `koanf:"namespace"`      // Namespace hint for discovery (default: "")
    PrometheusURL string `koanf:"prometheusurl"`  // Override auto-discovery
    GrafanaURL    string `koanf:"grafanaurl"`      // Override auto-discovery
    GrafanaToken  string `koanf:"grafanatoken"`    // Grafana service account token
}
```

**Environment variables:**
```
KUBECENTER_MONITORING_NAMESPACE        # Namespace hint for discovery
KUBECENTER_MONITORING_PROMETHEUSURL    # Override Prometheus URL (skip discovery)
KUBECENTER_MONITORING_GRAFANAURL       # Override Grafana URL (skip discovery)
KUBECENTER_MONITORING_GRAFANATOKEN     # Grafana service account token
```

---

## API Endpoints

### New Endpoints

```
GET    /api/v1/monitoring/status              # Discovery results
POST   /api/v1/monitoring/rediscover          # Force re-discovery
GET    /api/v1/monitoring/query               # PromQL instant query proxy
GET    /api/v1/monitoring/query_range         # PromQL range query proxy
GET    /api/v1/monitoring/dashboards          # List provisioned dashboards
ALL    /api/v1/monitoring/grafana/proxy/*     # Grafana reverse proxy
```

### Response Schemas

**GET /api/v1/monitoring/status:**
```json
{
  "data": {
    "prometheus": {
      "available": true,
      "url": "http://prometheus-kube-prometheus-prometheus.monitoring:9090",
      "detectionMethod": "service-label",
      "lastChecked": "2026-03-13T10:30:00Z"
    },
    "grafana": {
      "available": true,
      "url": "http://prometheus-grafana.monitoring:80",
      "detectionMethod": "service-name",
      "proxied": true,
      "lastChecked": "2026-03-13T10:30:00Z"
    },
    "dashboards": {
      "provisioned": true,
      "count": 4,
      "error": null
    },
    "hasOperator": true
  }
}
```

**GET /api/v1/monitoring/query:**
Query params: `query` (required), `time` (optional, RFC3339/unix), `timeout` (optional, max 30s)
Response: Prometheus native JSON format (passthrough), wrapped in KubeCenter envelope:
```json
{
  "data": {
    "resultType": "vector",
    "result": [{"metric": {"__name__": "up"}, "value": [1710322200, "1"]}]
  }
}
```

**GET /api/v1/monitoring/query_range:**
Query params: `query` (required), `start` (required), `end` (required), `step` (required), `timeout` (optional)
Response: Same shape with `resultType: "matrix"`.

**GET /api/v1/monitoring/dashboards:**
```json
{
  "data": [
    {
      "uid": "kubecenter-cluster-overview",
      "title": "Cluster Overview",
      "tags": ["kubecenter"],
      "url": "/d/kubecenter-cluster-overview/cluster-overview"
    }
  ]
}
```

---

## Resource-to-Dashboard Mapping

| Resource Kind | Dashboard UID | Template Variables |
|---|---|---|
| pods | `kubecenter-pod-detail` | `var-namespace`, `var-pod` |
| deployments | `kubecenter-deployment-detail` | `var-namespace`, `var-deployment` |
| statefulsets | `kubecenter-statefulset-detail` | `var-namespace`, `var-statefulset` |
| daemonsets | `kubecenter-daemonset-detail` | `var-namespace`, `var-daemonset` |
| nodes | `kubecenter-node-detail` | `var-node` |
| pvcs (persistentvolumeclaims) | `kubecenter-pvc-detail` | `var-namespace`, `var-pvc` |
| — (cluster-wide) | `kubecenter-cluster-overview` | (none) |

**Resource types without metrics dashboards:** services, ingresses, configmaps, secrets, namespaces, jobs, cronjobs, networkpolicies, roles, clusterroles, rolebindings, clusterrolebindings. For these, the Metrics tab shows: "No metrics dashboard available for this resource type."

---

## PromQL Query Templates

Defined in `backend/internal/monitoring/metrics.go`:

```go
// metrics.go

var QueryTemplates = map[string]QueryTemplate{
    // Pod metrics
    "pod_cpu_usage": {
        Name:      "pod_cpu_usage",
        Query:     `sum(rate(container_cpu_usage_seconds_total{container!="",pod="$pod",namespace="$namespace"}[5m])) by (container)`,
        Variables: []string{"namespace", "pod"},
    },
    "pod_memory_usage": {
        Name:      "pod_memory_usage",
        Query:     `sum(container_memory_working_set_bytes{container!="",pod="$pod",namespace="$namespace"}) by (container)`,
        Variables: []string{"namespace", "pod"},
    },
    "pod_network_rx": {
        Name:      "pod_network_rx",
        Query:     `sum(rate(container_network_receive_bytes_total{pod="$pod",namespace="$namespace"}[5m]))`,
        Variables: []string{"namespace", "pod"},
    },
    "pod_network_tx": {
        Name:      "pod_network_tx",
        Query:     `sum(rate(container_network_transmit_bytes_total{pod="$pod",namespace="$namespace"}[5m]))`,
        Variables: []string{"namespace", "pod"},
    },

    // Node metrics
    "node_cpu_utilization": {
        Name:      "node_cpu_utilization",
        Query:     `100 - (avg by (instance) (rate(node_cpu_seconds_total{mode="idle",instance=~"$node.*"}[5m])) * 100)`,
        Variables: []string{"node"},
    },
    "node_memory_utilization": {
        Name:      "node_memory_utilization",
        Query:     `100 * (1 - node_memory_MemAvailable_bytes{instance=~"$node.*"} / node_memory_MemTotal_bytes{instance=~"$node.*"})`,
        Variables: []string{"node"},
    },
    "node_disk_utilization": {
        Name:      "node_disk_utilization",
        Query:     `100 - (node_filesystem_avail_bytes{instance=~"$node.*",mountpoint="/",fstype!="rootfs"} / node_filesystem_size_bytes{instance=~"$node.*",mountpoint="/",fstype!="rootfs"} * 100)`,
        Variables: []string{"node"},
    },
    "node_pod_count": {
        Name:      "node_pod_count",
        Query:     `count(kube_pod_info{node="$node"})`,
        Variables: []string{"node"},
    },

    // Deployment metrics
    "deployment_replica_health": {
        Name:      "deployment_replica_health",
        Query:     `kube_deployment_status_replicas_available{namespace="$namespace",deployment="$deployment"} / kube_deployment_spec_replicas{namespace="$namespace",deployment="$deployment"}`,
        Variables: []string{"namespace", "deployment"},
    },

    // PVC metrics
    "pvc_usage_percent": {
        Name:      "pvc_usage_percent",
        Query:     `kubelet_volume_stats_used_bytes{namespace="$namespace",persistentvolumeclaim="$pvc"} / kubelet_volume_stats_capacity_bytes{namespace="$namespace",persistentvolumeclaim="$pvc"} * 100`,
        Variables: []string{"namespace", "pvc"},
    },
}
```

Variable values are validated against `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` before substitution to prevent PromQL injection.

---

## Implementation Phases

### Phase 1: Backend Foundation

**Files to create:**

```
backend/internal/monitoring/
├── handler.go           # Handler struct, HTTP handler methods
├── discovery.go         # Prometheus/Grafana auto-discovery logic
├── prometheus.go        # Prometheus API client wrapper
├── grafana.go           # Grafana reverse proxy + API client
├── metrics.go           # Named PromQL query templates
├── monitoring_test.go   # Unit tests
└── dashboards/          # Embedded JSON dashboard definitions
    ├── embed.go         # go:embed directive
    ├── cluster_overview.json
    ├── pod_detail.json
    ├── deployment_detail.json
    ├── node_detail.json
    ├── statefulset_detail.json
    ├── daemonset_detail.json
    └── pvc_detail.json
```

**Files to modify:**

```
backend/internal/config/config.go     # Add MonitoringConfig struct
backend/internal/config/defaults.go   # Add monitoring defaults
backend/internal/server/server.go     # Add MonitoringHandler to Server + Deps
backend/internal/server/routes.go     # Add registerMonitoringRoutes()
backend/cmd/kubecenter/main.go        # Wire monitoring handler, start discovery goroutine
backend/go.mod                        # Add prometheus/client_golang
```

**Implementation details:**

`backend/internal/monitoring/handler.go`:
```go
// handler.go
type Handler struct {
    Discoverer *Discoverer
    Logger     *slog.Logger
}

// HandleStatus — GET /api/v1/monitoring/status
// HandleRediscover — POST /api/v1/monitoring/rediscover
// HandleQuery — GET /api/v1/monitoring/query (proxy to Prometheus)
// HandleQueryRange — GET /api/v1/monitoring/query_range
// HandleDashboards — GET /api/v1/monitoring/dashboards
// GrafanaProxy — ALL /api/v1/monitoring/grafana/proxy/* (reverse proxy)
```

`backend/internal/monitoring/discovery.go`:
```go
// discovery.go
type Discoverer struct {
    k8sClient   *k8s.ClientFactory
    config      config.MonitoringConfig
    logger      *slog.Logger

    mu          sync.RWMutex
    status      *MonitoringStatus
    promClient  *PrometheusClient  // nil if not discovered
    grafProxy   http.Handler       // nil if not discovered
    grafClient  *GrafanaClient     // nil if no token configured
}

type MonitoringStatus struct {
    Prometheus  ComponentStatus `json:"prometheus"`
    Grafana     ComponentStatus `json:"grafana"`
    Dashboards  DashboardStatus `json:"dashboards"`
    HasOperator bool            `json:"hasOperator"`
}

type ComponentStatus struct {
    Available       bool   `json:"available"`
    URL             string `json:"url,omitempty"`
    DetectionMethod string `json:"detectionMethod,omitempty"`
    LastChecked     string `json:"lastChecked"`
}

// Discover runs the discovery sequence (CRD check, service scan, label selector)
// RunDiscoveryLoop starts a goroutine that re-discovers every 5 minutes
// Status returns the cached discovery results
```

`backend/internal/monitoring/grafana.go`:
- `NewGrafanaProxy()` — creates `httputil.ReverseProxy` with `.Rewrite` (Go 1.26 pattern)
  - Strips `/api/v1/monitoring/grafana/proxy` prefix
  - Injects `Authorization: Bearer <token>` header
  - Path allowlist validation (see D3)
  - `ModifyResponse`: strips `X-Frame-Options` and `Content-Security-Policy` from Grafana responses
  - `ErrorHandler`: returns 502 with structured error
- `GrafanaClient` — raw HTTP client for dashboard provisioning
  - `CreateFolder(ctx, uid, title)` — POST /api/folders
  - `UpsertDashboard(ctx, dashboard, folderUID)` — POST /api/dashboards/db
  - `SearchDashboards(ctx, tag)` — GET /api/search?tag=kubecenter
- `ProvisionDashboards(ctx)` — loads embedded JSON, upserts each into "KubeCenter" folder

`backend/internal/monitoring/prometheus.go`:
- Uses `github.com/prometheus/client_golang/api/prometheus/v1`
- `NewPrometheusClient(address)` — creates typed API client
- `Query(ctx, query, time)` — instant query with 10s timeout
- `QueryRange(ctx, query, start, end, step)` — range query with 30s timeout
- Results serialized as JSON for the HTTP response

`backend/internal/server/routes.go` — add inside authenticated group:
```go
func (s *Server) registerMonitoringRoutes(ar chi.Router) {
    ar.Route("/monitoring", func(mr chi.Router) {
        mr.Get("/status", s.MonitoringHandler.HandleStatus)
        mr.Post("/rediscover", s.MonitoringHandler.HandleRediscover)
        mr.Get("/query", s.MonitoringHandler.HandleQuery)
        mr.Get("/query_range", s.MonitoringHandler.HandleQueryRange)
        mr.Get("/dashboards", s.MonitoringHandler.HandleDashboards)
        mr.HandleFunc("/grafana/proxy/*", s.MonitoringHandler.GrafanaProxy)
    })
}
```

**go.mod addition:**
```
github.com/prometheus/client_golang v1.23.2
```

**Acceptance criteria (Phase 1):**
- [ ] `GET /api/v1/monitoring/status` returns discovery results
- [ ] Discovery finds Prometheus/Grafana in the homelab `monitoring` namespace
- [ ] `GET /api/v1/monitoring/query?query=up` returns data from Prometheus
- [ ] `GET /api/v1/monitoring/query_range` returns range data
- [ ] Grafana proxy serves dashboard pages with auth injection
- [ ] Grafana proxy blocks disallowed paths (returns 403 for `/api/admin/`)
- [ ] Dashboard provisioning creates dashboards in Grafana on startup
- [ ] `POST /api/v1/monitoring/rediscover` triggers immediate re-check
- [ ] `GET /api/v1/monitoring/dashboards` lists provisioned dashboards
- [ ] All endpoints are rate limited (reuse YAML rate limiter)
- [ ] Discovery goroutine shuts down cleanly on context cancellation
- [ ] 15+ unit tests covering discovery, proxy path validation, query template rendering

### Phase 2: Frontend Integration

**Files to create:**

```
frontend/islands/PerformancePanel.tsx     # Grafana iframe embed + fallback states
frontend/islands/MonitoringStatus.tsx     # Monitoring overview island
frontend/islands/PromQLQuery.tsx          # Raw PromQL query interface
frontend/routes/monitoring/index.tsx      # Monitoring overview page
frontend/routes/monitoring/dashboards.tsx # Dashboard list page
frontend/routes/monitoring/prometheus.tsx # PromQL query page
```

**Files to modify:**

```
frontend/islands/ResourceDetail.tsx    # Replace metrics placeholder with PerformancePanel
frontend/lib/constants.ts              # Add Monitoring section to NAV_SECTIONS
frontend/routes/_middleware.ts         # Add frame-src 'self' to CSP
```

**Implementation details:**

`frontend/islands/PerformancePanel.tsx`:
```tsx
// Props: kind, name, namespace (from ResourceDetail)
// 1. Calls GET /api/v1/monitoring/status on mount
// 2. If monitoring unavailable → shows "not configured" card with link to /monitoring
// 3. If Grafana unavailable but Prometheus available → shows "Grafana not configured" with link to /monitoring/prometheus
// 4. If resource kind has no dashboard → shows "No metrics for this resource type"
// 5. If all good → renders iframe:
//    src="/api/v1/monitoring/grafana/proxy/d-solo/{dashboardUID}/{slug}?orgId=1&kiosk=1&refresh=30s&var-namespace={ns}&var-{kind}={name}"
// 6. Shows loading overlay until iframe's onLoad fires
```

Dashboard UID mapping (hardcoded, matches provisioned dashboards):
```ts
const DASHBOARD_MAP: Record<string, { uid: string; varName: string }> = {
  pods: { uid: "kubecenter-pod-detail", varName: "pod" },
  deployments: { uid: "kubecenter-deployment-detail", varName: "deployment" },
  statefulsets: { uid: "kubecenter-statefulset-detail", varName: "statefulset" },
  daemonsets: { uid: "kubecenter-daemonset-detail", varName: "daemonset" },
  nodes: { uid: "kubecenter-node-detail", varName: "node" },
  persistentvolumeclaims: { uid: "kubecenter-pvc-detail", varName: "pvc" },
};
```

`frontend/islands/ResourceDetail.tsx` — replace lines 515-528 (metrics placeholder):
```tsx
{
  id: "metrics",
  label: "Metrics",
  content: () => (
    <PerformancePanel kind={kind} name={name} namespace={namespace} />
  ),
}
```

`frontend/lib/constants.ts` — add after "Config" section:
```ts
{
  title: "Monitoring",
  icon: "chart",
  items: [
    { label: "Overview", href: "/monitoring" },
    { label: "Dashboards", href: "/monitoring/dashboards" },
    { label: "Prometheus", href: "/monitoring/prometheus" },
  ],
}
```

`frontend/routes/_middleware.ts` — add `frame-src 'self'` to CSP:
```
frame-src 'self';
```

`frontend/islands/PromQLQuery.tsx`:
- Text input for PromQL expression
- Time range picker (last 1h, 6h, 24h, 7d, custom)
- "Run Query" button
- Results displayed as a table (instant query) or simple time series (range query)
- Uses the existing `apiGet` helper to call `/api/v1/monitoring/query` or `/query_range`

**Acceptance criteria (Phase 2):**
- [ ] Metrics tab on pod detail shows CPU/memory Grafana panels via iframe
- [ ] Metrics tab on node detail shows node utilization panels
- [ ] Metrics tab for unsupported resource types shows informational message
- [ ] Monitoring unavailable → Metrics tab shows "not configured" with link
- [ ] /monitoring page shows discovery status (Prometheus/Grafana URLs, detection method)
- [ ] /monitoring/dashboards lists provisioned dashboards with links
- [ ] /monitoring/prometheus allows running raw PromQL queries
- [ ] Sidebar shows Monitoring section with 3 entries
- [ ] CSP allows same-origin iframes
- [ ] Monitoring status page has "Re-scan" button

### Phase 3: Helm Chart & Dashboards

**Files to create:**

```
helm/kubecenter/templates/monitoring/
├── grafana-dashboards-cm.yaml    # ConfigMap with dashboard JSON (when monitoring.deploy=true)
└── grafana-config-cm.yaml        # Grafana datasource + settings ConfigMap
```

**Files to modify:**

```
helm/kubecenter/Chart.yaml        # Add kube-prometheus-stack conditional dependency
helm/kubecenter/values.yaml       # Add monitoring section
helm/kubecenter/templates/clusterrole.yaml  # Add discovery permissions
```

**Implementation details:**

`helm/kubecenter/Chart.yaml`:
```yaml
dependencies:
  - name: kube-prometheus-stack
    version: "~82.10"
    repository: https://prometheus-community.github.io/helm-charts
    condition: monitoring.deploy
    alias: monitoring-stack
```

`helm/kubecenter/values.yaml` — add monitoring section:
```yaml
monitoring:
  deploy: false
  namespace: ""              # Namespace hint for discovery (empty = search all)
  prometheus:
    url: ""                  # Override auto-discovery
  grafana:
    url: ""                  # Override auto-discovery
    apiToken: ""             # Service account token

# Subchart overrides (only when monitoring.deploy=true)
monitoring-stack:
  alertmanager:
    enabled: false           # KubeCenter handles alerting in Step 11
  grafana:
    enabled: true
    sidecar:
      dashboards:
        enabled: true
        label: grafana_dashboard
    grafana.ini:
      security:
        allow_embedding: true
      auth.proxy:
        enabled: true
        header_name: X-WEBAUTH-USER
  kubeStateMetrics:
    enabled: true
  nodeExporter:
    enabled: true
```

`helm/kubecenter/templates/clusterrole.yaml` — add:
```yaml
- apiGroups: ["monitoring.coreos.com"]
  resources: ["servicemonitors", "prometheuses"]
  verbs: ["list", "get"]
```

**Dashboard JSON files** — Grafana dashboard definitions embedded in Go via `go:embed`. Each dashboard contains:
- Template variables matching the resource-to-dashboard mapping table
- Panels for the PromQL queries defined in `metrics.go`
- `schemaVersion: 41`, `uid: "kubecenter-*"` prefix
- Time range default: last 1 hour

**Acceptance criteria (Phase 3):**
- [ ] `helm dependency update` pulls kube-prometheus-stack
- [ ] `helm template` with `monitoring.deploy=false` does NOT include monitoring-stack resources
- [ ] `helm template` with `monitoring.deploy=true` includes full kube-prometheus-stack
- [ ] Dashboard ConfigMaps have `grafana_dashboard: "1"` label
- [ ] `helm lint` passes
- [ ] ClusterRole includes monitoring.coreos.com API group

---

## Grafana Dashboard Content

Each dashboard should contain these panels:

**kubecenter-pod-detail:**
- CPU usage per container (line chart, 5m rate)
- Memory working set per container (line chart)
- Network I/O (rx/tx, line chart)
- CPU throttling (stat panel)
- Container restart count (stat panel)

**kubecenter-node-detail:**
- CPU utilization % (gauge + line chart)
- Memory utilization % (gauge + line chart)
- Disk utilization % (gauge)
- Pod count (stat panel)
- Network I/O (line chart)

**kubecenter-deployment-detail:**
- Replica availability ratio (gauge)
- Unavailable replicas (stat panel)
- Pod CPU/memory aggregated across replicas (line chart)

**kubecenter-cluster-overview:**
- Total nodes / ready nodes (stat panels)
- Total pods / running pods (stat panels)
- Cluster CPU utilization (gauge)
- Cluster memory utilization (gauge)
- Top 5 CPU-consuming pods (bar chart)
- Top 5 memory-consuming pods (bar chart)

**kubecenter-pvc-detail:**
- Volume usage % (gauge)
- Used bytes (stat panel)
- Capacity bytes (stat panel)

---

## Security Considerations

1. **Grafana proxy SSRF prevention**: Fixed target URL from discovery (never from user input). Path allowlist (see D3). Traversal blocking (`..`, `%2e`, `%2E`).

2. **PromQL proxy**: No write operations possible via PromQL. Rate limited (30 req/min, matching YAML endpoints). Maximum query length: 4096 bytes. Timeout capped at 30s server-side regardless of client request.

3. **Grafana proxy response sanitization**: Strip `Set-Cookie`, `X-Frame-Options`, and `Content-Security-Policy` headers from proxied Grafana responses.

4. **Service account token**: Stored as env var or k8s Secret, never logged, never returned in API responses. The `/api/v1/monitoring/status` endpoint returns the Grafana URL but never the token.

5. **CSP**: `frame-src 'self'` only — Grafana is proxied through the same origin. `frame-ancestors 'none'` remains (KubeCenter itself cannot be embedded).

---

## Graceful Degradation Matrix

| State | Metrics Tab | /monitoring | /monitoring/prometheus |
|---|---|---|---|
| Nothing available | "Monitoring not configured" + setup link | Status: unavailable | "Prometheus not available" |
| Prometheus only | "Grafana not configured" + link to /monitoring/prometheus | Status: partial | Fully functional |
| Grafana only (rare) | "Prometheus not available" | Status: partial | "Prometheus not available" |
| Both available | Grafana iframe | Status: available | Fully functional |
| Prometheus goes down | Grafana shows "No data" in panels | Status updates on next check | Returns 503 |

---

## Files Summary

### Create (Backend — 12 files)

| File | Purpose |
|---|---|
| `backend/internal/monitoring/handler.go` | HTTP handlers for all monitoring endpoints |
| `backend/internal/monitoring/discovery.go` | Auto-discovery logic + periodic re-check goroutine |
| `backend/internal/monitoring/prometheus.go` | Prometheus API client wrapper |
| `backend/internal/monitoring/grafana.go` | Grafana reverse proxy + API client |
| `backend/internal/monitoring/metrics.go` | Named PromQL query templates |
| `backend/internal/monitoring/monitoring_test.go` | Unit tests |
| `backend/internal/monitoring/dashboards/embed.go` | go:embed for dashboard JSON files |
| `backend/internal/monitoring/dashboards/cluster_overview.json` | Cluster overview dashboard |
| `backend/internal/monitoring/dashboards/pod_detail.json` | Pod metrics dashboard |
| `backend/internal/monitoring/dashboards/deployment_detail.json` | Deployment metrics dashboard |
| `backend/internal/monitoring/dashboards/node_detail.json` | Node metrics dashboard |
| `backend/internal/monitoring/dashboards/pvc_detail.json` | PVC metrics dashboard |

### Create (Frontend — 6 files)

| File | Purpose |
|---|---|
| `frontend/islands/PerformancePanel.tsx` | Grafana iframe embed with fallback states |
| `frontend/islands/MonitoringStatus.tsx` | Monitoring overview island |
| `frontend/islands/PromQLQuery.tsx` | Raw PromQL query interface |
| `frontend/routes/monitoring/index.tsx` | Monitoring overview page |
| `frontend/routes/monitoring/dashboards.tsx` | Dashboard list page |
| `frontend/routes/monitoring/prometheus.tsx` | PromQL query page |

### Create (Helm — 2 files)

| File | Purpose |
|---|---|
| `helm/kubecenter/templates/monitoring/grafana-dashboards-cm.yaml` | Dashboard ConfigMap |
| `helm/kubecenter/templates/monitoring/grafana-config-cm.yaml` | Grafana datasource ConfigMap |

### Modify (Backend — 6 files)

| File | Change |
|---|---|
| `backend/internal/config/config.go` | Add `MonitoringConfig` struct |
| `backend/internal/config/defaults.go` | Add monitoring defaults |
| `backend/internal/server/server.go` | Add `MonitoringHandler` to Server + Deps |
| `backend/internal/server/routes.go` | Add `registerMonitoringRoutes()` |
| `backend/cmd/kubecenter/main.go` | Wire monitoring handler, start discovery goroutine |
| `backend/go.mod` | Add `prometheus/client_golang` |

### Modify (Frontend — 3 files)

| File | Change |
|---|---|
| `frontend/islands/ResourceDetail.tsx` | Replace metrics placeholder (lines 515-528) with PerformancePanel |
| `frontend/lib/constants.ts` | Add Monitoring section to NAV_SECTIONS |
| `frontend/routes/_middleware.ts` | Add `frame-src 'self'` to CSP |

### Modify (Helm — 3 files)

| File | Change |
|---|---|
| `helm/kubecenter/Chart.yaml` | Add kube-prometheus-stack subchart dependency |
| `helm/kubecenter/values.yaml` | Add monitoring section |
| `helm/kubecenter/templates/clusterrole.yaml` | Add monitoring.coreos.com permissions |

---

## Dependencies & Prerequisites

- Homelab k3s cluster has kube-prometheus-stack deployed in `monitoring` namespace (for smoke testing)
- Grafana service account token must be created in homelab Grafana before smoke testing
- `helm repo add prometheus-community https://prometheus-community.github.io/helm-charts` must be run before `helm dependency update`

## Testing Strategy

- **Unit tests**: Discovery logic (mock k8s clientset), proxy path validation, query template rendering, config parsing
- **Integration tests**: Prometheus client against a mock HTTP server, Grafana client against a mock HTTP server
- **Smoke tests**: Full discovery against homelab k3s, PromQL query against real Prometheus, Grafana iframe render against real Grafana
- **Helm tests**: `helm lint`, `helm template` with `monitoring.deploy=true` and `false`

## References

- Existing handler pattern: `backend/internal/yaml/handler.go`
- Existing route registration: `backend/internal/server/routes.go`
- Existing BFF proxy (SSRF protection pattern): `frontend/routes/api/[...path].ts`
- Metrics placeholder to replace: `frontend/islands/ResourceDetail.tsx:515-528`
- Sidebar nav structure: `frontend/lib/constants.ts:68-157`
- CSP headers: `frontend/routes/_middleware.ts:19`
- Step 9 in master plan: `plans/feat-kubecenter-phase1-mvp.md:914-978`
- PR #1-#7 for established patterns
- [Prometheus HTTP API](https://prometheus.io/docs/prometheus/latest/querying/api/)
- [Grafana Dashboard API](https://grafana.com/docs/grafana/latest/developer-resources/api-reference/http-api/dashboard/)
- [Go httputil.ReverseProxy.Rewrite](https://pkg.go.dev/net/http/httputil#ReverseProxy)
- [kube-prometheus-stack chart](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack)
