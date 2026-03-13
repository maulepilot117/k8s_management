package yaml

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kubecenter/kubecenter/internal/audit"
	"github.com/kubecenter/kubecenter/internal/auth"
	"github.com/kubecenter/kubecenter/internal/k8s"
	"github.com/kubecenter/kubecenter/pkg/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Handler provides HTTP handlers for YAML operations.
type Handler struct {
	K8sClient   *k8s.ClientFactory
	AuditLogger audit.Logger
	Logger      *slog.Logger
	ClusterID   string
}

// HandleValidate validates YAML against the cluster's schema using dry-run apply.
// POST /api/v1/yaml/validate
func (h *Handler) HandleValidate(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}

	data, err := readYAMLBody(w, r)
	if err != nil {
		return
	}

	docs, err := ParseMultiDoc(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid YAML: "+err.Error(), "")
		return
	}

	dynClient, err := h.K8sClient.DynamicClientForUser(user.KubernetesUsername, user.KubernetesGroups)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create kubernetes client", err.Error())
		return
	}
	mapper := h.K8sClient.RESTMapper()

	type validationError struct {
		Field   string `json:"field,omitempty"`
		Message string `json:"message"`
	}
	type docResult struct {
		Index     int               `json:"index"`
		Kind      string            `json:"kind"`
		Name      string            `json:"name"`
		Namespace string            `json:"namespace,omitempty"`
		Valid     bool              `json:"valid"`
		Errors    []validationError `json:"errors,omitempty"`
	}
	type validateResponse struct {
		Documents []docResult `json:"documents"`
		Valid     bool        `json:"valid"`
	}

	resp := validateResponse{
		Documents: make([]docResult, 0, len(docs)),
		Valid:     true,
	}

	for i, obj := range docs {
		dr := docResult{
			Index:     i,
			Kind:      obj.GetKind(),
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
			Valid:     true,
		}

		// Dry-run apply validates against the cluster's schema
		diffResp := DiffDocuments(r.Context(), dynClient, mapper, docs[i:i+1], h.Logger)
		if len(diffResp.Documents) > 0 && diffResp.Documents[0].Error != "" {
			dr.Valid = false
			dr.Errors = []validationError{{
				Message: diffResp.Documents[0].Error,
			}}
			resp.Valid = false
		}

		resp.Documents = append(resp.Documents, dr)
	}

	writeData(w, resp)
}

// HandleApply applies YAML via server-side apply.
// POST /api/v1/yaml/apply?force=true
func (h *Handler) HandleApply(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}

	data, err := readYAMLBody(w, r)
	if err != nil {
		return
	}

	docs, err := ParseMultiDoc(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid YAML: "+err.Error(), "")
		return
	}

	force := r.URL.Query().Get("force") == "true"

	dynClient, err := h.K8sClient.DynamicClientForUser(user.KubernetesUsername, user.KubernetesGroups)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create kubernetes client", err.Error())
		return
	}
	mapper := h.K8sClient.RESTMapper()

	resp := ApplyDocuments(r.Context(), dynClient, mapper, docs, force, h.Logger)

	// Audit log each document apply
	for _, result := range resp.Results {
		auditResult := audit.ResultSuccess
		if result.Action == "failed" {
			auditResult = audit.ResultFailure
		}
		h.AuditLogger.Log(r.Context(), audit.Entry{
			Timestamp:         time.Now(),
			ClusterID:         h.ClusterID,
			User:              user.Username,
			SourceIP:          r.RemoteAddr,
			Action:            audit.ActionApply,
			ResourceKind:      result.Kind,
			ResourceNamespace: result.Namespace,
			ResourceName:      result.Name,
			Result:            auditResult,
			Detail:            result.Action,
		})
	}

	writeData(w, resp)
}

// HandleDiff performs a dry-run apply and returns current vs proposed YAML.
// POST /api/v1/yaml/diff
func (h *Handler) HandleDiff(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}

	data, err := readYAMLBody(w, r)
	if err != nil {
		return
	}

	docs, err := ParseMultiDoc(data)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid YAML: "+err.Error(), "")
		return
	}

	dynClient, err := h.K8sClient.DynamicClientForUser(user.KubernetesUsername, user.KubernetesGroups)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create kubernetes client", err.Error())
		return
	}
	mapper := h.K8sClient.RESTMapper()

	resp := DiffDocuments(r.Context(), dynClient, mapper, docs, h.Logger)
	writeData(w, resp)
}

// HandleExport exports a resource as clean, reapply-ready YAML.
// GET /api/v1/yaml/export/{kind}/{namespace}/{name}
func (h *Handler) HandleExport(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}

	kind := chi.URLParam(r, "kind")
	namespace := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	if kind == "" || name == "" {
		writeError(w, http.StatusBadRequest, "kind and name are required", "")
		return
	}

	// Block secret export to prevent data loss (D1)
	if strings.EqualFold(kind, "secrets") || strings.EqualFold(kind, "secret") {
		writeError(w, http.StatusUnprocessableEntity,
			"Secrets cannot be exported via YAML to prevent accidental data loss. Use the Secrets management interface with audit-logged reveal.",
			"")
		return
	}

	// Use "_" for cluster-scoped resources (no namespace)
	if namespace == "_" {
		namespace = ""
	}

	dynClient, err := h.K8sClient.DynamicClientForUser(user.KubernetesUsername, user.KubernetesGroups)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create kubernetes client", err.Error())
		return
	}

	gvr, err := resolveGVR(h.K8sClient, kind)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown resource kind: %s", kind), err.Error())
		return
	}

	var obj *unstructured.Unstructured
	if namespace != "" {
		obj, err = dynClient.Resource(gvr).Namespace(namespace).Get(r.Context(), name, metav1.GetOptions{})
	} else {
		obj, err = dynClient.Resource(gvr).Get(r.Context(), name, metav1.GetOptions{})
	}
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("%s '%s' not found", kind, name), err.Error())
		return
	}

	yamlBytes, err := ExportToYAML(obj)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to export YAML", err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.yaml", kind, name))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(yamlBytes)
}

// --- Helpers ---

// readYAMLBody reads and validates the raw YAML body from the request.
func readYAMLBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySize)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			writeError(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("request body exceeds maximum size of %d bytes", MaxBodySize), "")
		} else {
			writeError(w, http.StatusBadRequest, "failed to read request body", err.Error())
		}
		return nil, err
	}

	if err := CheckSecurity(data); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return nil, err
	}

	return data, nil
}

// requireUser extracts the authenticated user from context or writes a 401.
func requireUser(w http.ResponseWriter, r *http.Request) (*auth.User, bool) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required", "")
		return nil, false
	}
	return user, true
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, message, detail string) {
	if detail != "" && status >= 500 {
		slog.Error("internal error detail", "status", status, "message", message, "detail", detail)
		detail = ""
	}
	writeJSON(w, status, api.Response{
		Error: &api.APIError{
			Code:    status,
			Message: message,
			Detail:  detail,
		},
	})
}

// writeData writes a data response with the standard envelope.
func writeData(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, api.Response{Data: data})
}

// resolveGVR resolves a plural resource kind name to a GroupVersionResource
// using the API server's discovery API.
func resolveGVR(clientFactory *k8s.ClientFactory, kind string) (schema.GroupVersionResource, error) {
	kind = strings.ToLower(kind)

	disc := clientFactory.DiscoveryClient()
	_, apiResourceLists, err := disc.ServerGroupsAndResources()
	if err != nil {
		// ServerGroupsAndResources may return partial results with an error
		// for unavailable API groups. Only fail if no results were returned.
		if apiResourceLists == nil {
			return schema.GroupVersionResource{}, fmt.Errorf("discovering API resources: %w", err)
		}
	}

	for _, list := range apiResourceLists {
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}
		for _, r := range list.APIResources {
			if strings.EqualFold(r.Name, kind) {
				return schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: r.Name,
				}, nil
			}
		}
	}

	return schema.GroupVersionResource{}, fmt.Errorf("resource %q not found in API server", kind)
}
