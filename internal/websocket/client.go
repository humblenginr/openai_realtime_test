package websocket

import (
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
