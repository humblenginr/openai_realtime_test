// chat/chat.go
package chat

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
)

type ChatGPTClient struct {
	Conn                 *websocket.Conn
	ResponseAudioDeltaCh chan string
	ResponseAudioDoneCh  chan bool
}

func NewAzureClient(respAudioDeltaCh chan string, respAudioDoneCh chan bool) (*ChatGPTClient, error) {
	url := "wss://pixa-realtime.openai.azure.com/openai/realtime?api-version=2024-10-01-preview&deployment=gpt-4o-realtime-preview"
	header := http.Header{}
	header.Set("api-key", os.Getenv("AZURE_API_KEY"))
	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ChatGPT WebSocket server: %v", err)
	}

	fmt.Println("Connected to Azure OpenAI server.")

	// Send session update command
	sessionEvent := map[string]interface{}{
		"type": "session.update",
		"session": map[string]interface{}{
			"modalities":         []string{"audio", "text"},
			"input_audio_format": "pcm16",
		},
	}

	if err := conn.WriteJSON(sessionEvent); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send session event: %v", err)
	}

	return &ChatGPTClient{Conn: conn, ResponseAudioDeltaCh: respAudioDeltaCh, ResponseAudioDoneCh: respAudioDoneCh}, nil
}

func NewChatGPTClient(respAudioDeltaCh chan string, respAudioDoneCh chan bool) (*ChatGPTClient, error) {
	url := "wss://api.openai.com/v1/realtime?model=gpt-4o-realtime-preview-2024-10-01"
	//url := "wss://pixa-realtime.openai.azure.com/openai/realtime?api-version=2024-08-01-preview&deployment=gpt-4o-realtime-preview&api-key=e9bbb248e632416f85c5de0b2e446ea1"
	headers := http.Header{
		"Authorization": {"Bearer " + os.Getenv("OPENAI_API_KEY")},
	}

	// Connect to the ChatGPT WebSocket server
	conn, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ChatGPT WebSocket server: %v", err)
	}

	fmt.Println("Connected to ChatGPT server.")
	// Send initial session update event
	sessionEvent := map[string]interface{}{
		"type": "session.update",
		"session": map[string]interface{}{
			"modalities": []string{"audio", "text"},
		},
	}
	if err := conn.WriteJSON(sessionEvent); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send session event: %v", err)
	}

	return &ChatGPTClient{Conn: conn, ResponseAudioDeltaCh: respAudioDeltaCh, ResponseAudioDoneCh: respAudioDoneCh}, nil
}

// watch for the events emitted by the ChatGPT Realtime API server
// API Reference: https://platform.openai.com/docs/api-reference/realtime-server-events/
func (c *ChatGPTClient) WatchServerEvents() error {
	for {
		// Listen for ChatGPT response
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			fmt.Printf("error reading response from ChatGPT: %v\n", err)
		}
		var parsedMsg EventBase
		if err := json.Unmarshal(msg, &parsedMsg); err != nil {
			return fmt.Errorf("failed to parse message: %v", err)
		}
		switch parsedMsg.Type {
		case ErrorEventType:
			// parse error message
			var errorE ErrorEvent
			if err := json.Unmarshal(msg, &errorE); err != nil {
				return fmt.Errorf("failed to parse message: %v", err)
			}
			// handle error
			fmt.Println(errorE.Error.Message)
		case SessionCreatedEventType:
			fmt.Println("Received a session created message")
		case "response.audio.delta":
			fmt.Println("Received a response audio")
			var resp map[string]interface{}
			if err := json.Unmarshal(msg, &resp); err != nil {
				return fmt.Errorf("failed to parse message: %v", err)
			}
			delta := resp["delta"].(string)
			c.ResponseAudioDeltaCh <- delta
		case "response.audio.done":
			c.ResponseAudioDoneCh <- true
		default:
			fmt.Println(string(msg))
		}

	}
}

func (c *ChatGPTClient) SendCreateResponseEvent() error {
	createResponseEvent := map[string]interface{}{
		"type": ResponseCreateEventType,
	}
	// Send the conversation event
	if err := c.Conn.WriteJSON(createResponseEvent); err != nil {
		return fmt.Errorf("failed to send conversation event: %v", err)
	}
	return nil
}

func (c *ChatGPTClient) AppendToAudioBuffer(audio string) error {
	e := map[string]interface{}{"type": "input_audio_buffer.append", "audio": audio}

	if err := c.Conn.WriteJSON(e); err != nil {
		return fmt.Errorf("failed to send input audio append event: %v", err)
	}

	return nil
}

func (c *ChatGPTClient) CommitAudioBuffer() error {
	e := map[string]interface{}{"type": "input_audio_buffer.commit"}

	if err := c.Conn.WriteJSON(e); err != nil {
		return fmt.Errorf("failed to send input audio buffer commit event: %v", err)
	}
	return nil
}

func (c *ChatGPTClient) SendConversationItemCreateEvent(audio string) error {
	conversationEvent := ConversationEvent{}
	conversationEvent.Type = ConverstationItemCreateEventType
	item := MessageItem{}
	item.Type = "message"
	item.Role = "user"
	content := ContentItem{}
	content.Type = "input_audio"
	content.Audio = audio
	item.Content = make([]ContentItem, 0)
	item.Content = append(item.Content, content)
	conversationEvent.Item = item

	// Send the conversation event
	if err := c.Conn.WriteJSON(conversationEvent); err != nil {
		return fmt.Errorf("failed to send conversation event: %v", err)
	}

	return nil
}
