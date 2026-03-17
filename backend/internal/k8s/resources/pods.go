package resources

import (
	"bufio"
	"io"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/kubecenter/kubecenter/internal/audit"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/remotecommand"
)

const kindPod = "pods"

var validContainerName = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,252}$`)

func (h *Handler) HandleListPods(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	params := parseListParams(r)

	sel, ok := parseSelectorOrReject(w, params.LabelSelector)
	if !ok {
		return
	}

	var all []*corev1.Pod
	var err error
	if params.Namespace != "" {
		if !h.checkAccess(w, r, user, "list", kindPod, params.Namespace) {
			return
		}
		all, err = h.Informers.Pods().Pods(params.Namespace).List(sel)
	} else {
		if !h.checkAccess(w, r, user, "list", kindPod, "") {
			return
		}
		all, err = h.Informers.Pods().List(sel)
	}
	if err != nil {
		mapK8sError(w, err, "list", "Pod", params.Namespace, "")
		return
	}
	items, cont := paginate(all, params.Limit, params.Continue)
	writeList(w, items, len(all), cont)
}

func (h *Handler) HandleGetPod(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")
	if !h.checkAccess(w, r, user, "get", kindPod, ns) {
		return
	}
	obj, err := h.Informers.Pods().Pods(ns).Get(name)
	if err != nil {
		mapK8sError(w, err, "get", "Pod", ns, name)
		return
	}
	writeData(w, obj)
}

// HandlePodLogs returns the last N lines of a pod's container logs.
// GET /api/v1/resources/pods/{namespace}/{name}/logs?container=X&tailLines=500&previous=false&timestamps=true
func (h *Handler) HandlePodLogs(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	// RBAC: check get on pods/log subresource
	if !h.checkAccess(w, r, user, "get", "pods/log", ns) {
		return
	}

	q := r.URL.Query()
	container := q.Get("container")

	// F7: Validate container name
	if container != "" && !validContainerName.MatchString(container) {
		writeError(w, http.StatusBadRequest, "invalid container name", "")
		return
	}

	tailLines := int64(500)
	if tl := q.Get("tailLines"); tl != "" {
		if v, err := strconv.ParseInt(tl, 10, 64); err == nil && v > 0 {
			tailLines = v
		}
	}
	if tailLines > 10000 {
		tailLines = 10000
	}

	previous := q.Get("previous") == "true"
	timestamps := q.Get("timestamps") != "false" // default true
	limitBytes := int64(5 * 1024 * 1024)          // 5 MB max response

	opts := &corev1.PodLogOptions{
		Container:  container,
		TailLines:  &tailLines,
		Previous:   previous,
		Timestamps: timestamps,
		LimitBytes: &limitBytes,
	}

	cs, err := h.impersonatingClient(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create client", "")
		return
	}

	stream, err := cs.CoreV1().Pods(ns).GetLogs(name, opts).Stream(r.Context())
	if err != nil {
		mapK8sError(w, err, "get", "Pod logs", ns, name)
		return
	}
	defer stream.Close()

	var lines []string
	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		// F6: Check context cancellation periodically (every 100 lines)
		if len(lines)%100 == 0 {
			select {
			case <-r.Context().Done():
				return
			default:
			}
		}
		lines = append(lines, scanner.Text())
	}

	truncated := false
	if err := scanner.Err(); err != nil {
		truncated = true
	}

	// F5: Audit log the log access
	h.auditWrite(r, user, audit.ActionReadLogs, "Pod", ns, name, audit.ResultSuccess)

	writeData(w, map[string]any{
		"lines":     lines,
		"container": container,
		"pod":       name,
		"namespace": ns,
		"count":     len(lines),
		"truncated": truncated,
	})
}

// HandlePodExec upgrades to WebSocket and opens an exec session to a pod container.
// WS /api/v1/ws/exec/{namespace}/{name}/{container}
func (h *Handler) HandlePodExec(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")
	container := chi.URLParam(r, "container")

	if !h.checkAccess(w, r, user, "create", "pods/exec", ns) {
		return
	}

	if container != "" && !validContainerName.MatchString(container) {
		writeError(w, http.StatusBadRequest, "invalid container name", "")
		return
	}

	// Upgrade to WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.Logger.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	// Create SPDY executor
	cs, err := h.impersonatingClient(user)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"failed to create client"}`))
		return
	}

	execReq := cs.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(name).
		Namespace(ns).
		SubResource("exec").
		Param("container", container).
		Param("stdin", "true").
		Param("stdout", "true").
		Param("stderr", "true").
		Param("tty", "true").
		Param("command", "/bin/sh")

	cfg := h.K8sClient.BaseConfig()
	cfg.Impersonate.UserName = user.KubernetesUsername
	cfg.Impersonate.Groups = user.KubernetesGroups

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", execReq.URL())
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"failed to create exec session"}`))
		h.Logger.Error("SPDY executor creation failed", "error", err)
		return
	}

	h.auditWrite(r, user, audit.ActionCreate, "Pod/exec", ns, name, audit.ResultSuccess)

	// Bridge WebSocket ↔ SPDY streams
	wsStream := newWSStream(conn)
	err = exec.StreamWithContext(r.Context(), remotecommand.StreamOptions{
		Stdin:  wsStream,
		Stdout: wsStream,
		Stderr: wsStream,
		Tty:    true,
	})
	if err != nil {
		h.Logger.Debug("exec session ended", "error", err, "pod", name)
	}
}

// wsStream bridges a gorilla WebSocket connection to io.Reader/io.Writer
// for use with remotecommand SPDY streams.
type wsStream struct {
	conn   *websocket.Conn
	buf    []byte
	closed bool
}

func newWSStream(conn *websocket.Conn) *wsStream {
	return &wsStream{conn: conn}
}

func (s *wsStream) Read(p []byte) (int, error) {
	if s.closed {
		return 0, io.EOF
	}
	if len(s.buf) > 0 {
		n := copy(p, s.buf)
		s.buf = s.buf[n:]
		return n, nil
	}
	_, msg, err := s.conn.ReadMessage()
	if err != nil {
		s.closed = true
		return 0, io.EOF
	}
	n := copy(p, msg)
	if n < len(msg) {
		s.buf = msg[n:]
	}
	return n, nil
}

func (s *wsStream) Write(p []byte) (int, error) {
	if s.closed {
		return 0, io.EOF
	}
	err := s.conn.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		s.closed = true
		return 0, err
	}
	return len(p), nil
}

func (h *Handler) HandleDeletePod(w http.ResponseWriter, r *http.Request) {
	user, ok := requireUser(w, r)
	if !ok {
		return
	}
	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")
	if !h.checkAccess(w, r, user, "delete", kindPod, ns) {
		return
	}
	cs, err := h.impersonatingClient(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create client", err.Error())
		return
	}
	if err := cs.CoreV1().Pods(ns).Delete(r.Context(), name, metav1.DeleteOptions{}); err != nil {
		h.auditWrite(r, user, audit.ActionDelete, "Pod", ns, name, audit.ResultFailure)
		mapK8sError(w, err, "delete", "Pod", ns, name)
		return
	}
	h.auditWrite(r, user, audit.ActionDelete, "Pod", ns, name, audit.ResultSuccess)
	w.WriteHeader(http.StatusNoContent)
}
