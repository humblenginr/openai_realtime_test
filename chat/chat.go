package chat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"pixa-demo/audio"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// URLs for different environments
	AzureURL = "wss://pixa-realtime.openai.azure.com/openai/realtime?api-version=2024-10-01-preview&deployment=gpt-4o-realtime-preview"

	// WebSocket configuration
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1 * 1024 * 1024 // 1Mib

	OutputAudioChunkSize = 4096
)

// ClientOption allows for customizing the ChatGPTClient
type ClientOption func(*ChatGPTClient)

// WithLogger sets a custom logger for the client
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *ChatGPTClient) {
		c.logger = logger
	}
}

// ChatGPTClient manages the WebSocket connection to the ChatGPT server
type ChatGPTClient struct {
	conn      *websocket.Conn
	url       string
	headers   http.Header
	logger    *slog.Logger
	mu        sync.Mutex
	done      chan struct{}
	closeOnce sync.Once
}

// WithCustomHeaders adds additional headers to the WebSocket connection
func WithCustomHeaders(headers http.Header) ClientOption {
	return func(c *ChatGPTClient) {
		// Create a new header if none exists
		if c.headers == nil {
			c.headers = http.Header{}
		}
		// Merge the custom headers with existing headers
		for key, values := range headers {
			for _, value := range values {
				c.headers.Add(key, value)
			}
		}
	}
}

// NewAzureClient creates a new ChatGPT client using Azure credentials
func NewAzureClient(ctx context.Context, opts ...ClientOption) (*ChatGPTClient, error) {
	apiKey := os.Getenv("AZURE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("AZURE_API_KEY environment variable is not set")
	}

	// Initialize with default headers
	client := &ChatGPTClient{
		url:     AzureURL,
		headers: http.Header{},
		logger:  slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		done:    make(chan struct{}),
	}

	// Apply options first
	for _, opt := range opts {
		opt(client)
	}

	// Ensure required headers are set (and can't be overridden)
	client.headers.Set("api-key", apiKey)

	// Initialize WebSocket connection
	if err := client.connect(ctx); err != nil {
		return nil, err
	}

	// Start ping-pong handler
	go client.pingHandler()

	return client, nil
}

// connect establishes the WebSocket connection
func (c *ChatGPTClient) connect(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	debugHeaders := make(http.Header)
	for k, v := range c.headers {
		if k != "api-key" && k != "Authorization" {
			debugHeaders[k] = v
		}
	}
	c.logger.Debug("Connecting with headers", "headers", debugHeaders)

	conn, resp, err := dialer.DialContext(ctx, c.url, c.headers)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("websocket connection failed with status %d: %v", resp.StatusCode, err)
		}
		return fmt.Errorf("websocket connection failed: %v", err)
	}

	c.conn = conn
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	c.logger.Info("Connected to server", "url", c.url)

	// Initialize session
	if err := c.initializeSession(); err != nil {
		c.Close()
		return fmt.Errorf("failed to initialize session: %v", err)
	}

	return nil
}

func (c *ChatGPTClient) initializeSession() error {
	sessionEvent := map[string]interface{}{
		"type": "session.update",
		"session": map[string]interface{}{
			"modalities":         []string{"audio", "text"},
			"input_audio_format": "pcm16",
			// turn should be detected automatically
			"turn_detection": map[string]interface{}{
				"type":                "server_vad",
				"threshold":           0.5,
				"prefix_padding_ms":   300,
				"silence_duration_ms": 500,
			},
		},
	}
	return c.writeJSON(sessionEvent)
}

// WatchServerEvents monitors server events and forwards them to the provided WebSocket client
func (c *ChatGPTClient) WatchServerEvents(ctx context.Context, clientWs *websocket.Conn) error {
	if clientWs == nil {
		return fmt.Errorf("client WebSocket connection is nil")
	}

	outChan := make(chan []byte)
	inChan := make(chan []byte)
	errChan := make(chan error)
	bsc := NewBufferSizeController(OutputAudioChunkSize, inChan, outChan, errChan)
	go func() {
		for {
			select {
			case err := <-errChan:
				c.logger.Error("error while splitting output data into chunks: ", "error", err)
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case out := <-outChan:
				clientWs.WriteMessage(2, out)
			case <-ctx.Done():
				return
			}
		}
	}()
	go bsc.Start()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.done:
			return fmt.Errorf("client closed")
		default:
			if err := c.handleServerEvent(clientWs, &bsc); err != nil {
				c.logger.Error("Error handling server event", "error", err)
				return err
			}
		}
	}
}

func (c *ChatGPTClient) handleServerEvent(clientWs *websocket.Conn, autoBuffer *BufferSizeController) error {

	_, msg, err := c.conn.ReadMessage()
	if err != nil {
		if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
			return nil
		}
		return fmt.Errorf("error reading message: %v", err)
	}

	var baseEvent EventBase
	if err := json.Unmarshal(msg, &baseEvent); err != nil {
		return fmt.Errorf("failed to parse base event: %v", err)
	}

	return c.processEvent(baseEvent.Type, msg, clientWs, autoBuffer)
}

func transformOutputAudio(data string) ([]byte, error) {
	pcm16Data, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("Could not decode base64 audio")
	}

	a, err := audio.FromPCM16(pcm16Data, 24000, 1)
	if err != nil {
		return nil, fmt.Errorf("invalid pcm16 data: %v", err)
	}
	a.Resample(16000)
	return a.AsPCM16(), nil
}

func (c *ChatGPTClient) processEvent(eventType EventType, msg []byte, clientWs *websocket.Conn, bsc *BufferSizeController) error {
	switch eventType {
	case ErrorEventType:
		var errorEvent ErrorEvent
		if err := json.Unmarshal(msg, &errorEvent); err != nil {
			return fmt.Errorf("failed to parse error event: %v", err)
		}
		c.logger.Error("Received error event",
			"type", errorEvent.Error.Type,
			"code", errorEvent.Error.Code,
			"message", errorEvent.Error.Message)
		return fmt.Errorf("server error: %s", errorEvent.Error.Message)

	case "response.audio.done":
		// send the remaining bytes
		bsc.Flush()
		return nil
	case "response.audio.delta":
		c.logger.Info("Received response.audio.delta")
		var data string
		var resp map[string]interface{}
		if err := json.Unmarshal(msg, &resp); err != nil {
			return fmt.Errorf("failed to parse delta event: %v", err)
		}
		data = resp["delta"].(string)
		go func() {
			audioBytes, err := transformOutputAudio(data)
			if err != nil {
				c.logger.Error("Could not transform output audio", "error", err)
			}
			bsc.inChan <- audioBytes
		}()
		return nil

	default:
		var resp map[string]interface{}
		if err := json.Unmarshal(msg, &resp); err != nil {
			return fmt.Errorf("failed to parse delta event: %v", err)
		}
		c.logger.Info("Unhandled event: ", "info", resp)
		return nil
	}
}

// AppendToAudioBuffer adds audio data to the buffer
func (c *ChatGPTClient) AppendToAudioBuffer(audio string) error {
	event := map[string]interface{}{
		"type":  InputAudioBufferAppendEventType,
		"audio": audio,
	}
	return c.writeJSON(event)
}

// ClearAudioBuffer clears the audio buffer
func (c *ChatGPTClient) ClearAudioBuffer() error {
	event := map[string]interface{}{
		"type": "input_audio_buffer.clear",
	}
	return c.writeJSON(event)
}

func (c *ChatGPTClient) writeJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteJSON(v)
}

func (c *ChatGPTClient) pingHandler() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Error("Failed to write ping message", "error", err)
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()
		case <-c.done:
			return
		}
	}
}

// Close gracefully closes the WebSocket connection
func (c *ChatGPTClient) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		if c.conn != nil {
			c.conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			c.conn.Close()
		}
	})
}
