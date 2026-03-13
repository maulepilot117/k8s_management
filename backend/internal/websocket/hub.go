package websocket

import (
	"context"
	"log/slog"
)

// AccessChecker verifies RBAC permissions for WebSocket subscriptions.
type AccessChecker interface {
	CanAccess(ctx context.Context, username string, groups []string, verb, resource, namespace string) (bool, error)
}

// subChange represents a subscription addition or removal.
type subChange struct {
	client *Client
	key    subKey
	add    bool // true = subscribe, false = unsubscribe
}

// Hub maintains active clients and fans out informer events
// to clients based on their subscriptions. All map mutations
// happen in the single Run goroutine via channels.
type Hub struct {
	clients       map[*Client]bool
	subscriptions map[subKey]map[*Client]bool

	register   chan *Client
	unregister chan *Client
	events     chan ResourceEvent
	addSub     chan subChange

	accessChecker AccessChecker
	logger        *slog.Logger
}

// NewHub creates a new Hub.
func NewHub(logger *slog.Logger, accessChecker AccessChecker) *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		subscriptions: make(map[subKey]map[*Client]bool),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		events:        make(chan ResourceEvent, 1024),
		addSub:        make(chan subChange, 256),
		accessChecker: accessChecker,
		logger:        logger,
	}
}

// Events returns the channel for feeding informer events into the hub.
func (h *Hub) Events() chan<- ResourceEvent {
	return h.events
}

// HandleEvent is an EventCallback-compatible function that feeds events into the hub.
// It performs a non-blocking send to the events channel.
func (h *Hub) HandleEvent(eventType, kind, namespace, name string, obj any) {
	event := ResourceEvent{
		EventType: eventType,
		Kind:      kind,
		Namespace: namespace,
		Name:      name,
		Object:    obj,
	}
	select {
	case h.events <- event:
	default:
		h.logger.Warn("event channel full, dropping event",
			"kind", kind, "name", name, "eventType", eventType)
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Run is the main hub goroutine. Call as `go hub.Run(ctx)`.
func (h *Hub) Run(ctx context.Context) {
	h.logger.Info("websocket hub started")
	defer h.logger.Info("websocket hub stopped")

	for {
		select {
		case <-ctx.Done():
			h.shutdown()
			return

		case client := <-h.register:
			h.clients[client] = true
			h.logger.Info("client connected",
				"user", client.username(),
				"clients", len(h.clients),
			)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				h.removeClient(client)
				h.logger.Info("client disconnected",
					"user", client.username(),
					"clients", len(h.clients),
				)
			}

		case change := <-h.addSub:
			if change.add {
				if h.subscriptions[change.key] == nil {
					h.subscriptions[change.key] = make(map[*Client]bool)
				}
				h.subscriptions[change.key][change.client] = true
			} else {
				if subs, ok := h.subscriptions[change.key]; ok {
					delete(subs, change.client)
					if len(subs) == 0 {
						delete(h.subscriptions, change.key)
					}
				}
			}

		case event := <-h.events:
			h.broadcastEvent(event)
		}
	}
}

// broadcastEvent sends an event to all clients subscribed to the event's kind+namespace.
func (h *Hub) broadcastEvent(event ResourceEvent) {
	// Collect target clients: exact namespace match + all-namespace subscribers
	targets := make(map[*Client]bool)

	exactKey := subKey{Kind: event.Kind, Namespace: event.Namespace}
	for client := range h.subscriptions[exactKey] {
		targets[client] = true
	}

	allNsKey := subKey{Kind: event.Kind, Namespace: ""}
	for client := range h.subscriptions[allNsKey] {
		targets[client] = true
	}

	for client := range targets {
		id := client.subIDForKey(exactKey)
		if id == "" {
			id = client.subIDForKey(allNsKey)
		}

		msg := OutgoingMessage{
			Type:      MsgTypeEvent,
			ID:        id,
			EventType: event.EventType,
			Object:    event.Object,
		}
		data, err := MarshalOutgoing(msg)
		if err != nil {
			h.logger.Error("failed to marshal event", "error", err, "kind", event.Kind)
			continue
		}

		// Non-blocking send — drop slow clients
		select {
		case client.send <- data:
		default:
			h.removeClient(client)
			h.logger.Warn("dropped slow client", "user", client.username())
		}
	}
}

// removeClient removes a client from all subscriptions and closes its send channel.
func (h *Hub) removeClient(client *Client) {
	for key, subs := range h.subscriptions {
		delete(subs, client)
		if len(subs) == 0 {
			delete(h.subscriptions, key)
		}
	}
	delete(h.clients, client)
	close(client.send)
}

// shutdown gracefully closes all client connections.
func (h *Hub) shutdown() {
	for client := range h.clients {
		h.removeClient(client)
	}
}
