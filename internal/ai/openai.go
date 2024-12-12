package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pixaverse-studios/websocket-server/internal/config"
	"github.com/pixaverse-studios/websocket-server/pkg/audio"

	"github.com/gorilla/websocket"
)

const (
	// WebSocket configuration
	writeWait = 10 * time.Second
)

// ChatGPTClient manages the WebSocket connection to the ChatGPT server
type OpenAIClient struct {
	conn    *websocket.Conn
	logger  *slog.Logger
	headers http.Header

	mu        sync.Mutex
	done      chan struct{}
	closeOnce sync.Once

	responseStream chan audio.Audio
	// eventsStream lets the client know when some important events happen in the model, like when the model has detected the start of speech, end of speech, completed the response etc. The client can use these to events to curate the behaviour of the system.
	eventsStream chan EventType
	config       config.AzureConfig
	aiconfig     config.AIConfig
}

func NewOpenAIClient(azureConfig config.AzureConfig, aiConfig config.AIConfig) *OpenAIClient {
	return &OpenAIClient{
		logger:         slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		done:           make(chan struct{}),
		headers:        http.Header{},
		responseStream: make(chan audio.Audio),
		eventsStream:   make(chan EventType),
		config:         azureConfig,
		aiconfig:       aiConfig,
	}
}

// ctx is used to cancel
func (c *OpenAIClient) Initialize(ctx context.Context) error {
	c.headers.Set("api-key", c.config.OpenAIKey)
	err := c.connect()
	if err != nil {
		return fmt.Errorf("Could not connect to OpenAI server: %v", err)
	}
	err = c.initializeSession()
	if err != nil {
		return fmt.Errorf("Could not initialize OpenAI session: %v", err)
	}
	go c.watchServerEvents(ctx)
	return nil

}

func (c *OpenAIClient) connect() error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, resp, err := dialer.Dial(c.config.ServiceURL, c.headers)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("websocket connection failed with status %d: %v", resp.StatusCode, err)
		}
		return fmt.Errorf("websocket connection failed: %v", err)
	}

	c.conn = conn
	c.logger.Info("Connected to server", "url", c.config.ServiceURL)
	return nil
}

func (c *OpenAIClient) loadSystemPrompt() string {
	if c.aiconfig.SystemPromptFilePath != "" {
		byt, err := os.ReadFile(c.aiconfig.SystemPromptFilePath)
		if err != nil {
			return ""
		}
		return string(byt)
	}
	return ""
}

func (c *OpenAIClient) initializeSession() error {
	sessionEvent := map[string]interface{}{
		"type": "session.update",
		"session": map[string]interface{}{
			"modalities":         []string{"audio", "text"},
			"input_audio_format": "pcm16",
			"instructions":       c.loadSystemPrompt(),
			// turn should be detected automatically
			"turn_detection": map[string]interface{}{
				"type":                "server_vad",
				"threshold":           0.5,
				"prefix_padding_ms":   300,
				"silence_duration_ms": 500,
			},
		},
	}
	fmt.Println("Initializing session...")
	return c.writeJSON(sessionEvent)
}

func (c *OpenAIClient) writeJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.conn.WriteJSON(v)
}

func (c *OpenAIClient) processEvent(eventType EventType, msg []byte) error {
	switch eventType {
	case ErrorEventType:
		var errorEvent ErrorEvent
		if err := json.Unmarshal(msg, &errorEvent); err != nil {
			return fmt.Errorf("failed to parse error event: %v", err)
		}
		c.logger.Error("Received error event from OpenAI",
			"type", errorEvent.Error.Type,
			"code", errorEvent.Error.Code,
			"message", errorEvent.Error.Message)
		return fmt.Errorf("server error: %s", errorEvent.Error.Message)

	case ResponseAudioDoneEventType:
		// send the remaining bytes
		c.eventsStream <- ResponseAudioDoneEventType
		return nil
	case ResponseAudioDeltaEventType:
		fmt.Println("Received audio delta")
		var data string
		var resp map[string]interface{}
		if err := json.Unmarshal(msg, &resp); err != nil {
			return fmt.Errorf("failed to parse delta event: %v", err)
		}
		data = resp["delta"].(string)
		pcm16Data, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return fmt.Errorf("Could not decode base64 audio")
		}

		a := audio.FromPCM16(pcm16Data, 24000, 1)
		c.responseStream <- a
		return nil

	default:
		var resp map[string]interface{}
		if err := json.Unmarshal(msg, &resp); err != nil {
			return fmt.Errorf("failed to parse delta event: %v", err)
		}
		c.logger.Info("Unhandled event", "event json", resp)
		return nil
	}
}

func (c *OpenAIClient) watchServerEvents(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.done:
			return fmt.Errorf("client closed")
		default:
			_, msg, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return nil
				}

				c.logger.Error("failed to read message from openai server", "error", err)
				continue
			}

			var baseEvent EventBase
			if err := json.Unmarshal(msg, &baseEvent); err != nil {
				c.logger.Error("failed to parse base event from openai server", "error", err)
				continue
			}
			if err := c.processEvent(baseEvent.Type, msg); err != nil {
				c.logger.Error("failed to process OpenAI event", "type", baseEvent.Type, "error", err)
			}

		}
	}

}

func (c *OpenAIClient) GetEventsStream() <-chan EventType {
	return c.eventsStream
}

func (c *OpenAIClient) GetResponseStream() <-chan audio.Audio {
	return c.responseStream
}

func (c *OpenAIClient) SendAudio(a audio.Audio) error {
	// OpenAI requires 16 bit pcm, 1 channel audio, 24khz samplerate
	if a.GetChannels() != 1 && a.GetChannels() == 2 {
		a.StereoToMono()
	}

	if a.GetSampleRate() != 24000 {
		a.Resample(24000)
	}

	return c.AppendToAudioBuffer(base64.StdEncoding.EncodeToString(a.AsPCM16()))

}

func (c *OpenAIClient) AppendToAudioBuffer(audio string) error {
	event := map[string]interface{}{
		"type":  InputAudioBufferAppendEventType,
		"audio": audio,
	}
	return c.writeJSON(event)
}
func (c *OpenAIClient) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		if c.conn != nil {
			c.conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			c.conn.Close()
		}
	})
}
