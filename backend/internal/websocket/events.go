package websocket

import "encoding/json"

// ResourceEvent is emitted by informer event handlers and consumed by the Hub.
type ResourceEvent struct {
	EventType string `json:"eventType"` // ADDED, MODIFIED, DELETED
	Kind      string `json:"kind"`      // deployments, pods, etc.
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Object    any    `json:"object"` // full k8s object (same shape as REST response)
}

// subKey identifies a subscription topic (resource kind + namespace).
type subKey struct {
	Kind      string
	Namespace string // empty = all namespaces
}

// Message types for the WebSocket wire protocol.
const (
	MsgTypeAuth        = "auth"
	MsgTypeAuthOK      = "auth_ok"
	MsgTypeSubscribe   = "subscribe"
	MsgTypeUnsubscribe = "unsubscribe"
	MsgTypeSubscribed  = "subscribed"
	MsgTypeEvent       = "event"
	MsgTypeError       = "error"
)

// IncomingMessage is the envelope for client-to-server messages.
type IncomingMessage struct {
	Type      string `json:"type"`
	Token     string `json:"token,omitempty"`     // auth
	ID        string `json:"id,omitempty"`        // subscribe/unsubscribe
	Kind      string `json:"kind,omitempty"`      // subscribe
	Namespace string `json:"namespace,omitempty"` // subscribe
}

// OutgoingMessage is the envelope for server-to-client messages.
type OutgoingMessage struct {
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	EventType string `json:"eventType,omitempty"` // ADDED/MODIFIED/DELETED
	Code      int    `json:"code,omitempty"`
	Message   string `json:"message,omitempty"`
	Object    any    `json:"object,omitempty"`
}

// MarshalOutgoing serializes an OutgoingMessage to JSON bytes.
func MarshalOutgoing(msg OutgoingMessage) ([]byte, error) {
	return json.Marshal(msg)
}
