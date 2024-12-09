package websocket

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pixaverse-studios/websocket-server/internal/config"
)

// Client represents a WebSocket client connection
type Client struct {
	conn   *websocket.Conn
	logger *slog.Logger
	mu     sync.Mutex
	config *config.Config
}

// NewClient creates a new WebSocket client
func NewClient(conn *websocket.Conn, logger *slog.Logger, cfg *config.Config) *Client {
	return &Client{
		conn:   conn,
		logger: logger,
		config: cfg,
	}
}

// Close closes the WebSocket connection and cleans up resources
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	writeWait, _ := time.ParseDuration(c.config.Websocket.WriteWait)

	if c.conn != nil {
		c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(writeWait),
		)
		c.conn.Close()
	}
}

// StartPingTicker starts sending periodic pings to the client
func (c *Client) StartPingTicker(ctx context.Context) {
	pingInterval, err := time.ParseDuration(c.config.Websocket.PingInterval)
	if err != nil {
		c.logger.Error("Invalid ping interval", "error", err)
		return
	}

	pongWait, err := time.ParseDuration(c.config.Websocket.PongWait)
	if err != nil {
		c.logger.Error("Invalid pong wait", "error", err)
		return
	}

	ticker := time.NewTicker(pingInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.mu.Lock()
				writeWait, _ := time.ParseDuration(c.config.Websocket.WriteWait)
				err := c.conn.WriteControl(
					websocket.PingMessage,
					[]byte{},
					time.Now().Add(writeWait),
				)
				if err != nil {
					c.logger.Error("Failed to write ping", "error", err)
					c.mu.Unlock()
					return
				}

				// Set deadline for pong response
				err = c.conn.SetReadDeadline(time.Now().Add(pongWait))
				c.mu.Unlock()

				if err != nil {
					c.logger.Error("Failed to set read deadline", "error", err)
					return
				}
			}
		}
	}()

	// Set up pong handler
	c.conn.SetPongHandler(func(string) error {
		c.mu.Lock()
		defer c.mu.Unlock()
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
}
