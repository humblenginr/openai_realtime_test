// websocket/handler.go
package websocket

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/pixaverse-studios/websocket-server/internal/ai"
	"github.com/pixaverse-studios/websocket-server/internal/utils"
	"github.com/pixaverse-studios/websocket-server/pkg/audio"

	"github.com/gorilla/websocket"
)

// Handler manages WebSocket connections and message routing
type Handler struct {
	upgrader websocket.Upgrader
	logger   *slog.Logger
}

// NewHandler creates a new WebSocket handler with the provided options
func NewHandler() *Handler {
	h := &Handler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
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

	client := &Client{
		conn:   conn,
		logger: h.logger,
	}
	defer client.Close()

	if err := h.handleClient(ctx, client); err != nil {
		h.logger.Error("Client handling error", "error", err)
	}
}

// handleClient manages the client connection and message routing
func (h *Handler) handleClient(ctx context.Context, client *Client) error {
	aiClient := ai.NewOpenAIClient(ai.AzureURL)
	ab := utils.NewBufferSizeController(4096)

	// Listen to the buffer controller output channel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case audio := <-ab.GetOutputChannel():
				client.conn.WriteMessage(2, audio)

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
				if a.GetSampleRate() != 16000 {
					a.Resample(16000)
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
