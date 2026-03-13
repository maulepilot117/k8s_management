package resources

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	corev1 "k8s.io/api/core/v1"
)

const kindEvent = "events"

// HandleListEvents handles GET /api/v1/resources/events[/:namespace]
func (h *Handler) HandleListEvents(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	params := parseListParams(r)

	sel, ok := parseSelectorOrReject(w, params.LabelSelector)
	if !ok {
		return
	}

	var all []*corev1.Event
	var err error
	if params.Namespace != "" {
		if !h.checkAccess(w, r, user, "list", kindEvent, params.Namespace) {
			return
		}
		all, err = h.Informers.Events().Events(params.Namespace).List(sel)
	} else {
		if !h.checkAccess(w, r, user, "list", kindEvent, "") {
			return
		}
		all, err = h.Informers.Events().List(sel)
	}
	if err != nil {
		mapK8sError(w, err, "list", "Event", params.Namespace, "")
		return
	}

	// Optional server-side filtering by involvedObject fields.
	if objKind := r.URL.Query().Get("involvedObjectKind"); objKind != "" {
		filtered := all[:0]
		for _, e := range all {
			if e.InvolvedObject.Kind == objKind {
				filtered = append(filtered, e)
			}
		}
		all = filtered
	}
	if objName := r.URL.Query().Get("involvedObjectName"); objName != "" {
		filtered := all[:0]
		for _, e := range all {
			if e.InvolvedObject.Name == objName {
				filtered = append(filtered, e)
			}
		}
		all = filtered
	}

	items, cont := paginate(all, params.Limit, params.Continue)
	writeList(w, items, len(all), cont)
}

// HandleGetEvent handles GET /api/v1/resources/events/:namespace/:name
func (h *Handler) HandleGetEvent(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")
	if !h.checkAccess(w, r, user, "get", kindEvent, ns) {
		return
	}
	obj, err := h.Informers.Events().Events(ns).Get(name)
	if err != nil {
		mapK8sError(w, err, "get", "Event", ns, name)
		return
	}
	writeData(w, obj)
}
