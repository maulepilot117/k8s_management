package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kubecenter/kubecenter/internal/audit"
	"github.com/kubecenter/kubecenter/pkg/api"
)

// handleAuditLogs returns paginated, filterable audit log entries.
func (s *Server) handleAuditLogs(w http.ResponseWriter, r *http.Request) {
	if s.AuditStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, api.Response{
			Error: &api.APIError{Code: 503, Message: "audit log persistence is not enabled"},
		})
		return
	}

	q := r.URL.Query()
	params := audit.QueryParams{
		User:         q.Get("user"),
		Action:       q.Get("action"),
		ResourceKind: q.Get("kind"),
		Namespace:    q.Get("namespace"),
	}

	if since := q.Get("since"); since != "" {
		if t, err := time.Parse(time.RFC3339, since); err == nil {
			params.Since = t
		}
	}
	if until := q.Get("until"); until != "" {
		if t, err := time.Parse(time.RFC3339, until); err == nil {
			params.Until = t
		}
	}
	if page := q.Get("page"); page != "" {
		params.Page, _ = strconv.Atoi(page)
	}
	if pageSize := q.Get("pageSize"); pageSize != "" {
		params.PageSize, _ = strconv.Atoi(pageSize)
	}

	entries, total, err := s.AuditStore.Query(r.Context(), params)
	if err != nil {
		s.Logger.Error("failed to query audit logs", "error", err)
		writeJSON(w, http.StatusInternalServerError, api.Response{
			Error: &api.APIError{Code: 500, Message: "failed to query audit logs"},
		})
		return
	}

	writeJSON(w, http.StatusOK, api.Response{
		Data: entries,
		Metadata: &api.Metadata{
			Total:    total,
			Page:     params.Page,
			PageSize: params.PageSize,
		},
	})
}
