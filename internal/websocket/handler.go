// websocket/handler.go
package websocket

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/pixaverse-studios/websocket-server/internal/ai"
	"github.com/pixaverse-studios/websocket-server/internal/config"
	"github.com/pixaverse-studios/websocket-server/internal/utils"
	"github.com/pixaverse-studios/websocket-server/pkg/audio"

	"github.com/gorilla/websocket"
)

// Handler manages WebSocket connections and message routing
type Handler struct {
	upgrader websocket.Upgrader
	logger   *slog.Logger
	config   *config.Config
}

// NewHandler creates a new WebSocket handler with the provided options
func NewHandler(cfg *config.Config) *Handler {
	pingInterval, _ := time.ParseDuration(cfg.Websocket.PingInterval)

	h := &Handler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // In production, implement proper origin checking
			},
			HandshakeTimeout: pingInterval,
			WriteBufferPool:  nil, // Use default pool
		},
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		config: cfg,
	}

	return h
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

	client := NewClient(conn, h.logger, h.config)
	defer client.Close()

	if err := h.handleClient(ctx, client); err != nil {
		h.logger.Error("Client handling error", "error", err)
	}
}

// handleClient manages the client connection and message routing
func (h *Handler) handleClient(ctx context.Context, client *Client) error {
	aiClient := ai.NewOpenAIClient(client.config.Azure)
	ab := utils.NewBufferSizeController(4096)

	// Listen to the buffer controller output channel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case audio := <-ab.GetOutputChannel():
				client.conn.WriteMessage(websocket.BinaryMessage, audio)
			}
		}
	}()

	// Listen for critical events from the AI model
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case e := <-aiClient.GetEventsStream():
				if e == ai.ResponseAudioDoneEventType {
					ab.Flush()
				}
			}
		}
	}()

	// Start handling AI responses
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case a := <-aiClient.GetResponseStream():
				if a.GetSampleRate() != h.config.Audio.SampleRate {
					a.Resample(h.config.Audio.SampleRate)
				}
				err := ab.Write(a.AsPCM16())
				if err != nil {
					h.logger.Error("Cannot write to BufferSizeController buffer", "error", err)
				}
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
	pongWait, _ := time.ParseDuration(h.config.Websocket.PongWait)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

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

			if typ == websocket.BinaryMessage {
				a := audio.FromPCM16(message, h.config.Audio.SampleRate, h.config.Audio.Channels)
				err := chatClient.SendAudio(a)
				if err != nil {
					h.logger.Error("Could not send audio to AI Client", "error", err)
				}

			}
		}
	}
}
