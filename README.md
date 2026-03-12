# KubeCenter

A web-based Kubernetes management platform that delivers vCenter-level functionality for Kubernetes clusters. Deploy inside your cluster via Helm and manage everything through a browser.

## Features

- **GUI-driven wizards** for deployments, services, storage (CSI), and networking (CNI)
- **Real-time cluster view** with WebSocket-powered live updates
- **Integrated monitoring** via Prometheus and Grafana (auto-discovered or deployed)
- **RBAC-aware multi-tenancy** with user impersonation (OIDC, LDAP, local accounts)
- **Full YAML escape hatch** with Monaco editor, validation, diff, and server-side apply
- **Pod management** including logs, exec terminal, and resource metrics
- **Alerting** via Alertmanager with email notifications
- **Audit logging** for all write operations and secret access
- **Multi-cluster ready** architecture (single-cluster in Phase 1)

## Architecture

```
Kubernetes Cluster
+--------------------------------------------------+
|  +------------+     +------------------+         |
|  |  Frontend   |---->|     Backend      |         |
|  |  Deno/Fresh |     |    Go 1.26       |         |
|  |  Port 8000  |     |    Port 8080     |         |
|  +------------+     +--------+---------+         |
|                              |                    |
|                  +-----------+-----------+        |
|                  |           |           |        |
|              +---+---+ +----+---+ +-----+---+   |
|              | K8s   | | Prom   | | Grafana  |   |
|              | API   | | etheus | |          |   |
|              +-------+ +--------+ +----------+   |
+--------------------------------------------------+
```

| Layer | Technology |
|---|---|
| Backend API | Go 1.26, chi router, client-go |
| Frontend | Deno 2.x, Fresh 2.x, Preact, Tailwind v4 |
| Monitoring | Prometheus + Grafana (kube-prometheus-stack) |
| Auth | JWT + OIDC / LDAP / local (Argon2id) |
| Deployment | Helm 3.x chart |
| Container | Distroless (Go), Deno slim (frontend) |

## Quick Start

### Prerequisites

- Go 1.26+
- Deno 2.x+
- Docker + Docker Compose
- [kind](https://kind.sigs.k8s.io/) for local testing
- Helm 3.x
- kubectl

### Local Development

```bash
# Create a local kind cluster
kind create cluster --name kubecenter

# Start the backend (connects via kubeconfig)
make dev-backend

# Backend API available at http://localhost:8080
# Health check: curl http://localhost:8080/healthz
# Cluster info (requires auth): curl http://localhost:8080/api/v1/cluster/info

# Set up the first admin account
curl -X POST http://localhost:8080/api/v1/setup/init \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"changeme","setupToken":"your-token"}'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Requested-With: XMLHttpRequest" \
  -d '{"username":"admin","password":"changeme"}'
```

### Deploy to Cluster

```bash
helm install kubecenter ./helm/kubecenter
```

## Build

```bash
make build-backend    # Build Go binary
make test-backend     # Run tests with race detection
make lint             # Run go vet
make docker-build     # Build container image
make helm-lint        # Lint Helm chart
```

## API

All endpoints are prefixed with `/api/v1`. Responses use a standard envelope:

```json
{
  "data": { ... },
  "metadata": { "total": 42, "continue": "token" },
  "error": null
}
```

Key endpoints:

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/healthz` | No | Liveness probe |
| GET | `/readyz` | No | Readiness probe |
| POST | `/api/v1/setup/init` | No | Create first admin account (one-time) |
| POST | `/api/v1/auth/login` | No | Login, returns JWT access token |
| POST | `/api/v1/auth/refresh` | No | Refresh access token (httpOnly cookie) |
| POST | `/api/v1/auth/logout` | No | Invalidate refresh token |
| GET | `/api/v1/auth/providers` | No | List configured auth providers |
| GET | `/api/v1/auth/me` | Yes | Current user info + RBAC summary |
| GET | `/api/v1/cluster/info` | Yes | Cluster version, node count, KubeCenter version |

Resource CRUD (15 types: deployments, statefulsets, daemonsets, pods, services, ingresses, configmaps, secrets, namespaces, nodes, pvcs, jobs, cronjobs, networkpolicies, RBAC):

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/v1/resources/:kind` | Yes | List across all namespaces |
| GET | `/api/v1/resources/:kind/:namespace` | Yes | List in namespace |
| GET | `/api/v1/resources/:kind/:namespace/:name` | Yes | Get specific resource |
| POST | `/api/v1/resources/:kind/:namespace` | Yes | Create resource |
| PUT | `/api/v1/resources/:kind/:namespace/:name` | Yes | Update resource |
| DELETE | `/api/v1/resources/:kind/:namespace/:name` | Yes | Delete resource |
| POST | `/api/v1/resources/nodes/:name/cordon` | Yes | Cordon node |
| POST | `/api/v1/resources/nodes/:name/drain` | Yes | Drain node (async, returns task ID) |
| GET | `/api/v1/tasks/:taskID` | Yes | Poll long-running task status |

See [CLAUDE.md](CLAUDE.md) for the complete API reference.

## Security

KubeCenter follows a strict security model:

- All user-initiated Kubernetes API calls use **user impersonation** so that cluster RBAC is enforced server-side
- The service account has **read-only** access for informer caches plus impersonation permissions
- Secrets are **never cached** in-process; they are fetched on-demand via the impersonated client
- JWT access tokens are held in memory only (not localStorage); refresh tokens use httpOnly cookies
- Containers run as **non-root** (UID 65534) with read-only root filesystem and all capabilities dropped
- All write operations and secret accesses are **audit logged**

See [SECURITY.md](SECURITY.md) for the full security policy and vulnerability reporting.

## Project Structure

```
kubecenter/
├── backend/              # Go 1.26 backend
│   ├── cmd/kubecenter/   # Entrypoint
│   ├── internal/
│   │   ├── server/       # HTTP server, routes, handlers
│   │   │   └── middleware/ # Auth, CSRF, rate limiting, CORS
│   │   ├── auth/         # JWT, local accounts, RBAC, sessions
│   │   ├── audit/        # Audit logging interface + slog impl
│   │   ├── k8s/          # Client factory, informers, resource handlers
│   │   │   └── resources/ # CRUD handlers for 15 k8s resource types
│   │   └── config/       # App configuration
│   └── pkg/              # Public packages (api types, version)
├── frontend/             # Deno 2.x + Fresh 2.x (Phase 1, Step 4+)
├── helm/kubecenter/      # Helm chart
├── todos/                # Tracked findings and improvements
├── .github/workflows/    # CI pipeline
└── plans/                # Implementation plans
```

## Contributing

1. Fork the repository
2. Create a feature branch from `main`
3. Follow the commit convention: `feat(scope): description`, `fix(scope): description`
4. Ensure `make lint` and `make test-backend` pass
5. Submit a pull request

## License

[Apache License 2.0](LICENSE)
