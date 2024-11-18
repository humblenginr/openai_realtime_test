// websocket/handler.go
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"pixa-demo/audio"
	"pixa-demo/chat"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Maximum message size allowed from peer
	maxMessageSize = 1024 * 1024 // 1MB
)

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
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
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
	// Configure connection
	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

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
			_, message, err := client.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					h.logger.Error("WebSocket read error", "error", err)
				}
				return err
			}

			if err := h.handleMessage(ctx, message, chatClient); err != nil {
				h.logger.Error("Message handling error", "error", err)
				continue
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (h *Handler) handleMessage(ctx context.Context, message []byte, chatClient *chat.ChatGPTClient) error {
	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		return fmt.Errorf("invalid message format: %w", err)
	}

	switch msg.Type {
	case "input_audio_buffer.append":
		return h.handleAudioAppend(msg.Data, chatClient)
	case "input_audio_buffer.clear":
		return chatClient.ClearAudioBuffer()
	default:
		h.logger.Warn("Unhandled message type", "type", msg.Type)
		return nil
	}
}

// handleAudioAppend processes and sends audio data to the chat client
func (h *Handler) handleAudioAppend(data interface{}, chatClient *chat.ChatGPTClient) error {
	audioData, ok := data.(string)
	if !ok {
		return fmt.Errorf("invalid audio data format")
	}

	go func() {
		processed, err := processAudio(audioData)
		if err != nil {
			h.logger.Error("Failed to process audio data", "error", err)
			return
		}
		err = chatClient.AppendToAudioBuffer(processed)
		if err != nil {
			h.logger.Error("Failed to append audio to input buffer", "error", err)
			return
		}
	}()
	return nil

}

// processAudio handles audio format conversion
func processAudio(data string) (string, error) {
	pcm16, err := audio.DecodePCM16FromBase64(data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	float32Data := audio.PCM16ToFloat32(pcm16)
	resampledData := audio.ResampleAudio(float32Data, 16000, 24000)

	result := audio.Base64EncodeAudio(resampledData)
	return result, nil
}
