package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kubecenter/kubecenter/internal/store"
	"github.com/kubecenter/kubecenter/pkg/api"
)

// handleListClusters returns all registered clusters.
func (s *Server) handleListClusters(w http.ResponseWriter, r *http.Request) {
	if s.ClusterStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, api.Response{
			Error: &api.APIError{Code: 503, Message: "cluster management requires a database"},
		})
		return
	}

	clusters, err := s.ClusterStore.List(r.Context())
	if err != nil {
		s.Logger.Error("failed to list clusters", "error", err)
		writeJSON(w, http.StatusInternalServerError, api.Response{
			Error: &api.APIError{Code: 500, Message: "failed to list clusters"},
		})
		return
	}

	writeJSON(w, http.StatusOK, api.Response{Data: clusters})
}

// handleGetCluster returns a single cluster by ID (credentials excluded).
func (s *Server) handleGetCluster(w http.ResponseWriter, r *http.Request) {
	if s.ClusterStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, api.Response{
			Error: &api.APIError{Code: 503, Message: "cluster management requires a database"},
		})
		return
	}

	id := chi.URLParam(r, "clusterID")
	cluster, err := s.ClusterStore.Get(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, api.Response{
			Error: &api.APIError{Code: 404, Message: "cluster not found"},
		})
		return
	}

	// Strip credentials from response
	cluster.CAData = nil
	cluster.AuthData = nil

	writeJSON(w, http.StatusOK, api.Response{Data: cluster})
}

// handleCreateCluster registers a new cluster.
func (s *Server) handleCreateCluster(w http.ResponseWriter, r *http.Request) {
	if s.ClusterStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, api.Response{
			Error: &api.APIError{Code: 503, Message: "cluster management requires a database"},
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB max (kubeconfig can be large)

	var req struct {
		Name         string `json:"name"`
		DisplayName  string `json:"displayName"`
		APIServerURL string `json:"apiServerUrl"`
		CACert       string `json:"caCert"`
		Token        string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, api.Response{
			Error: &api.APIError{Code: 400, Message: "invalid request body"},
		})
		return
	}

	if req.Name == "" || req.APIServerURL == "" || req.Token == "" {
		writeJSON(w, http.StatusBadRequest, api.Response{
			Error: &api.APIError{Code: 400, Message: "name, apiServerUrl, and token are required"},
		})
		return
	}

	id, _ := generateClusterID()

	record := store.ClusterRecord{
		ID:           id,
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		APIServerURL: req.APIServerURL,
		CAData:       []byte(req.CACert),
		AuthType:     "token",
		AuthData:     []byte(req.Token),
	}

	if err := s.ClusterStore.Create(r.Context(), record); err != nil {
		s.Logger.Error("failed to create cluster", "error", err)
		writeJSON(w, http.StatusInternalServerError, api.Response{
			Error: &api.APIError{Code: 500, Message: "failed to register cluster"},
		})
		return
	}

	// Return the cluster (without credentials)
	record.CAData = nil
	record.AuthData = nil
	writeJSON(w, http.StatusCreated, api.Response{Data: record})
}

// handleDeleteCluster removes a registered cluster.
func (s *Server) handleDeleteCluster(w http.ResponseWriter, r *http.Request) {
	if s.ClusterStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, api.Response{
			Error: &api.APIError{Code: 503, Message: "cluster management requires a database"},
		})
		return
	}

	id := chi.URLParam(r, "clusterID")
	if err := s.ClusterStore.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusBadRequest, api.Response{
			Error: &api.APIError{Code: 400, Message: err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, api.Response{Data: map[string]string{"status": "deleted"}})
}

func generateClusterID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
