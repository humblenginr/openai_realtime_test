// websocket/handler.go
package websocket

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"pixa-demo/audio"
	"pixa-demo/chat"

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
	// Create chat client
	chatClient, err := chat.NewAzureClient(ctx, chat.WithLogger(h.logger))
	if err != nil {
		return fmt.Errorf("failed to create chat client: %w", err)
	}
	defer chatClient.Close()

	// Create error channel for goroutines
	errChan := make(chan error, 2)

	// Start chat event monitoring
	go func() {
		if err := chatClient.WatchServerEvents(ctx, client.conn); err != nil {
			errChan <- fmt.Errorf("chat server event error: %w", err)
		}
	}()

	// Start message handling
	go func() {
		if err := h.readPump(ctx, client, chatClient); err != nil {
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
func (h *Handler) readPump(ctx context.Context, client *Client, chatClient *chat.ChatGPTClient) error {
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
			if err := h.handleMessage(message, chatClient, typ); err != nil {
				h.logger.Error("Message handling error", "error", err)
				continue
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (h *Handler) handleMessage(message []byte, chatClient *chat.ChatGPTClient, msgType int) error {
	// the hardware device will send binary PCM data
	// 1 means the message type is TextMessage
	// 2 means the message type is BinaryMessage
	if msgType == 2 {
		return h.handleAudioAppend(message, chatClient)
	}
	return fmt.Errorf("Message type: %d is not handled", msgType)
}

func processAudio(data []byte) (string, error) {
	a, err := audio.FromPCM16(data, 16000, 2)
	if err != nil {
		return "", fmt.Errorf("Invalid pcm16 data: %v", err)
	}
	a.StereoToMono()
	a.Resample(24000)

	// Step 6: Base64 encode
	result := base64.StdEncoding.EncodeToString(a.AsPCM16())
	return result, nil
}

func (h *Handler) handleAudioAppend(data []byte, chatClient *chat.ChatGPTClient) error {

	go func() {
		processed, err := processAudio(data)
		if err != nil {
			h.logger.Error("Failed to process audio data", "error", err)
			return
		}
		h.logger.Info("Successfully processed audio data",
			"inputLength", len(data),
			"outputLength", len(processed))

		err = chatClient.AppendToAudioBuffer(processed)
		if err != nil {
			h.logger.Error("Failed to append audio to input buffer", "error", err)
			return
		}
		h.logger.Info("Successfully appended audio data to input buffer")
	}()
	return nil
}
