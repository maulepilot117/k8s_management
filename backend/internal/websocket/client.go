package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kubecenter/kubecenter/internal/auth"
)

const (
	writeWait        = 10 * time.Second
	pongWait         = 60 * time.Second
	pingPeriod       = (pongWait * 9) / 10
	maxMessageSize   = 64 * 1024
	authTimeout      = 5 * time.Second
	maxSubscriptions = 20
	sendBufferSize   = 256
)

// TokenValidator validates JWT tokens and returns user info.
type TokenValidator interface {
	ValidateAccessToken(tokenString string) (*auth.TokenClaims, error)
}

// Client represents a single WebSocket connection.
type Client struct {
	hub            *Hub
	conn           *websocket.Conn
	send           chan []byte
	user           *auth.User
	subs           map[subKey]string // subKey → subscription ID
	tokenValidator TokenValidator
	logger         *slog.Logger
}

// NewClient creates a new Client for a WebSocket connection.
func NewClient(hub *Hub, conn *websocket.Conn, tv TokenValidator, logger *slog.Logger) *Client {
	return &Client{
		hub:            hub,
		conn:           conn,
		send:           make(chan []byte, sendBufferSize),
		subs:           make(map[subKey]string),
		tokenValidator: tv,
		logger:         logger,
	}
}

func (c *Client) username() string {
	if c.user != nil {
		return c.user.Username
	}
	return "unauthenticated"
}

// subIDForKey returns the subscription ID for a given subKey, or empty string.
func (c *Client) subIDForKey(key subKey) string {
	return c.subs[key]
}

// ReadPump reads messages from the WebSocket connection. Runs in its own goroutine.
func (c *Client) ReadPump() {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("panic in readPump", "recover", r, "user", c.username())
		}
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(authTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	if !c.waitForAuth() {
		return
	}

	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
			) {
				c.logger.Warn("websocket read error", "error", err, "user", c.username())
			}
			return
		}

		var msg IncomingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.sendError("", 400, "invalid message format")
			continue
		}

		switch msg.Type {
		case MsgTypeSubscribe:
			c.handleSubscribe(msg)
		case MsgTypeUnsubscribe:
			c.handleUnsubscribe(msg)
		default:
			c.sendError("", 400, fmt.Sprintf("unknown message type: %s", msg.Type))
		}
	}
}

// WritePump writes messages from the send channel to the WebSocket. Runs in its own goroutine.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("panic in writePump", "recover", r, "user", c.username())
		}
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// waitForAuth reads the first message and validates the JWT token.
func (c *Client) waitForAuth() bool {
	_, message, err := c.conn.ReadMessage()
	if err != nil {
		c.logger.Warn("websocket auth read failed", "error", err)
		return false
	}

	var msg IncomingMessage
	if err := json.Unmarshal(message, &msg); err != nil || msg.Type != MsgTypeAuth || msg.Token == "" {
		c.sendCloseError(4001, "first message must be auth with token")
		return false
	}

	claims, err := c.tokenValidator.ValidateAccessToken(msg.Token)
	if err != nil {
		c.sendCloseError(4001, "invalid token")
		return false
	}

	c.user = auth.UserFromClaims(claims)

	data, _ := MarshalOutgoing(OutgoingMessage{Type: MsgTypeAuthOK})
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return false
	}

	c.logger.Info("websocket client authenticated", "user", c.user.Username)
	return true
}

func (c *Client) handleSubscribe(msg IncomingMessage) {
	if msg.ID == "" || msg.Kind == "" {
		c.sendError(msg.ID, 400, "subscribe requires id and kind")
		return
	}

	if len(c.subs) >= maxSubscriptions {
		c.sendError(msg.ID, 429, "too many subscriptions")
		return
	}

	allowed, err := c.hub.accessChecker.CanAccess(
		context.Background(),
		c.user.KubernetesUsername,
		c.user.KubernetesGroups,
		"list",
		msg.Kind,
		msg.Namespace,
	)
	if err != nil {
		c.logger.Error("RBAC check failed",
			"error", err, "user", c.user.Username,
			"kind", msg.Kind, "namespace", msg.Namespace)
		c.sendError(msg.ID, 500, "RBAC check failed")
		return
	}
	if !allowed {
		c.sendError(msg.ID, 403, fmt.Sprintf("cannot list %s in namespace %q", msg.Kind, msg.Namespace))
		return
	}

	key := subKey{Kind: msg.Kind, Namespace: msg.Namespace}
	c.subs[key] = msg.ID
	c.hub.addSub <- subChange{client: c, key: key, add: true}

	data, _ := MarshalOutgoing(OutgoingMessage{Type: MsgTypeSubscribed, ID: msg.ID})
	select {
	case c.send <- data:
	default:
	}

	c.logger.Debug("subscription added",
		"user", c.user.Username, "id", msg.ID,
		"kind", msg.Kind, "namespace", msg.Namespace)
}

func (c *Client) handleUnsubscribe(msg IncomingMessage) {
	if msg.ID == "" {
		c.sendError(msg.ID, 400, "unsubscribe requires id")
		return
	}

	for key, id := range c.subs {
		if id == msg.ID {
			delete(c.subs, key)
			c.hub.addSub <- subChange{client: c, key: key, add: false}
			break
		}
	}
}

func (c *Client) sendError(id string, code int, message string) {
	data, _ := MarshalOutgoing(OutgoingMessage{
		Type:    MsgTypeError,
		ID:      id,
		Code:    code,
		Message: message,
	})
	select {
	case c.send <- data:
	default:
	}
}

func (c *Client) sendCloseError(code int, reason string) {
	c.conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason))
}
