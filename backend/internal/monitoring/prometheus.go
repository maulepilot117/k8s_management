package monitoring

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// PrometheusClient wraps the Prometheus v1 API for KubeCenter queries.
type PrometheusClient struct {
	api v1.API
}

// NewPrometheusClient creates a typed API client for the given Prometheus address.
func NewPrometheusClient(address string) (*PrometheusClient, error) {
	client, err := api.NewClient(api.Config{Address: address})
	if err != nil {
		return nil, fmt.Errorf("creating prometheus client: %w", err)
	}
	return &PrometheusClient{api: v1.NewAPI(client)}, nil
}

// Query runs a PromQL instant query.
func (p *PrometheusClient) Query(ctx context.Context, query string, ts time.Time) (model.Value, v1.Warnings, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return p.api.Query(ctx, query, ts)
}

// QueryRange runs a PromQL range query.
func (p *PrometheusClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (model.Value, v1.Warnings, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return p.api.QueryRange(ctx, query, v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
}

// QueryTemplate is a named PromQL query with variable substitution.
type QueryTemplate struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Query       string   `json:"query"`
	Variables   []string `json:"variables"`
}

// Render substitutes validated variable values into the template.
func (t QueryTemplate) Render(vars map[string]string) (string, error) {
	q := t.Query
	for _, v := range t.Variables {
		val, ok := vars[v]
		if !ok {
			return "", fmt.Errorf("missing required variable: %s", v)
		}
		if !isValidK8sName(val) {
			return "", fmt.Errorf("invalid value for %s: %q", v, val)
		}
		q = strings.ReplaceAll(q, "$"+v, val)
	}
	return q, nil
}

// isValidK8sName validates a value is a safe Kubernetes name (prevents PromQL injection).
func isValidK8sName(s string) bool {
	if s == "" || len(s) > 253 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '.') {
			return false
		}
	}
	return true
}
