package server

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	ws "github.com/kubecenter/kubecenter/internal/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     nil, // set in handleWSResources based on config
}

// handleWSResources handles WebSocket upgrade for the resource event stream.
func (s *Server) handleWSResources(w http.ResponseWriter, r *http.Request) {
	if s.Hub == nil {
		http.Error(w, "websocket not available", http.StatusServiceUnavailable)
		return
	}

	// Validate Origin header against allowed origins
	origin := r.Header.Get("Origin")
	if origin != "" && !s.isAllowedOrigin(origin) {
		s.Logger.Warn("websocket origin rejected", "origin", origin)
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}

	// Upgrade with permissive CheckOrigin since we validated above
	up := upgrader
	up.CheckOrigin = func(r *http.Request) bool { return true }

	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		s.Logger.Error("websocket upgrade failed", "error", err)
		return
	}

	client := ws.NewClient(s.Hub, conn, s.TokenManager, s.Logger)

	s.Hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}

// isAllowedOrigin checks if the origin is in the allowed origins list.
func (s *Server) isAllowedOrigin(origin string) bool {
	if s.Config.Dev {
		return true // allow all origins in dev mode
	}
	for _, allowed := range s.Config.CORS.AllowedOrigins {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}
