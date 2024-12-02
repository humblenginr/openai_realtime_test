// websocket/handler.go
package websocket

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"pixa-demo/ai"
	"pixa-demo/audio"

	"github.com/gorilla/websocket"
)

const ()

// Handler manages WebSocket connections and message routing
type Handler struct {
	upgrader websocket.Upgrader
	logger   *slog.Logger
}

// Message defines the structure of WebSocket messages
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// NewHandler creates a new WebSocket handler with the provided options
func NewHandler(opts ...Option) *Handler {
	h := &Handler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
	}

	// Apply options
	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Option allows for customizing the Handler
type Option func(*Handler)

// WithLogger sets a custom logger
func WithLogger(logger *slog.Logger) Option {
	return func(h *Handler) {
		h.logger = logger
	}
}

// WithUpgrader sets a custom WebSocket upgrader
func WithUpgrader(upgrader websocket.Upgrader) Option {
	return func(h *Handler) {
		h.upgrader = upgrader
	}
}

// ServeHTTP handles WebSocket connections
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection", "error", err)
		return
	}

	client := &Client{
		conn:   conn,
		logger: h.logger,
	}
	defer client.Close()

	if err := h.handleClient(ctx, client); err != nil {
		h.logger.Error("Client handling error", "error", err)
	}
}

// Client represents a WebSocket client connection
type Client struct {
	conn   *websocket.Conn
	logger *slog.Logger
	mu     sync.Mutex
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.conn.Close()
	}
}

// handleClient manages the client connection and message routing
func (h *Handler) handleClient(ctx context.Context, client *Client) error {
	aiClient := ai.NewOpenAIClient(ai.AzureURL)
	// Start handling AI responses
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case a := <-aiClient.GetResponseStream():
				if a.GetSampleRate() != 16000 {
					a.Resample(16000)
				}
				client.conn.WriteMessage(2, a.AsPCM16())
			}

		}
	}()

	err := aiClient.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("Could not initialize AI Client: %v", err)
	}

	// Create error channel for goroutines
	errChan := make(chan error, 2)

	// Start handling messages from the client
	go func() {
		if err := h.readPump(ctx, client, aiClient); err != nil {
			errChan <- fmt.Errorf("client message handling error: %w", err)
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// readPump handles incoming messages from the WebSocket client
func (h *Handler) readPump(ctx context.Context, client *Client, chatClient ai.AIClient) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			typ, message, err := client.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					h.logger.Error("WebSocket read error", "error", err)
				}
				return err
			}

			if typ == 2 {
				a := audio.FromPCM16(message, 16000, 2)
				err := chatClient.SendAudio(a)
				if err != nil {
					h.logger.Error("Could not send audio to AI Client", "error", err)
				}

			}
		}
	}
}
