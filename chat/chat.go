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
	Conn *websocket.Conn
}

func NewAzureClient() (*ChatGPTClient, error) {
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
			// turn should be detected automatically
			"turn_detection": map[string]interface{}{
				"type":                "server_vad",
				"threshold":           0.5,
				"prefix_padding_ms":   300,
				"silence_duration_ms": 500,
			},
		},
	}

	if err := conn.WriteJSON(sessionEvent); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send session event: %v", err)
	}

	return &ChatGPTClient{Conn: conn}, nil
}

func NewChatGPTClient() (*ChatGPTClient, error) {
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
			"turn_detection": map[string]interface{}{
				"silence_duration_ms": 750,
				"threshold":           0.3,
			},
		},
	}
	if err := conn.WriteJSON(sessionEvent); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send session event: %v", err)
	}

	return &ChatGPTClient{Conn: conn}, nil
}

// Message defines the structure of messages to be sent to the client
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// watch for the events emitted by the ChatGPT Realtime API server
// API Reference: https://platform.openai.com/docs/api-reference/realtime-server-events/
// eventsRelayCh is used to relay events to the client.
func (c *ChatGPTClient) WatchServerEvents(clientWs *websocket.Conn) error {
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
		case "response.audio.delta":
			var resp map[string]interface{}
			if err := json.Unmarshal(msg, &resp); err != nil {
				fmt.Printf("failed to parse message: %v\n", err)
			}
			delta := resp["delta"].(string)
			msg := Message{Type: string(parsedMsg.Type), Data: delta}
			if clientWs == nil {
				fmt.Println("Websocket connection is not present")

			}
			err = clientWs.WriteJSON(msg)
			if err != nil {
				fmt.Println(err)
			}
		case "response.audio.done":
			msg := Message{Type: string(parsedMsg.Type), Data: ""}
			if clientWs == nil {
				return fmt.Errorf("Websocket connection is not present")

			}
			err = clientWs.WriteJSON(msg)
			if err != nil {
				fmt.Println(err)
			}
		case "response.audio_transcript.delta":
			var resp map[string]interface{}
			if err := json.Unmarshal(msg, &resp); err != nil {
				fmt.Printf("failed to parse message: %v\n", err)
			}
			delta := resp["delta"].(string)
			msg := Message{Type: string(parsedMsg.Type), Data: delta}
			if clientWs == nil {
				fmt.Println("Websocket connection is not present")

			}
			err = clientWs.WriteJSON(msg)
			if err != nil {
				fmt.Println(err)
			}
		case "response.audio_transcript.done":
			msg := Message{Type: string(parsedMsg.Type), Data: ""}
			if clientWs == nil {
				return fmt.Errorf("Websocket connection is not present")

			}
			err = clientWs.WriteJSON(msg)
			if err != nil {
				fmt.Println(err)
			}
		case "input_audio_buffer.speech_started":
			msg := Message{Type: string(parsedMsg.Type), Data: ""}
			if clientWs == nil {
				return fmt.Errorf("Websocket connection is not present")

			}
			err = clientWs.WriteJSON(msg)
			if err != nil {
				fmt.Println(err)
			}
		case "input_audio_buffer.speech_stopped":
			msg := Message{Type: string(parsedMsg.Type), Data: ""}
			if clientWs == nil {
				return fmt.Errorf("Websocket connection is not present")

			}
			err = clientWs.WriteJSON(msg)
			if err != nil {
				fmt.Println(err)
			}
		case "input_audio_buffer.cleared":
			msg := Message{Type: string(parsedMsg.Type), Data: ""}
			if clientWs == nil {
				return fmt.Errorf("Websocket connection is not present")

			}
			err = clientWs.WriteJSON(msg)
			if err != nil {
				fmt.Println(err)
			}
		default:
			fmt.Println(string(msg))
		}

	}
}

// We will not receive any confirmation message from the server
func (c *ChatGPTClient) AppendToAudioBuffer(audio string) error {
	e := map[string]interface{}{"type": "input_audio_buffer.append", "audio": audio}

	if err := c.Conn.WriteJSON(e); err != nil {
		return fmt.Errorf("failed to send input audio append event: %v", err)
	}

	return nil
}

func (c *ChatGPTClient) ClearAudioBuffer() error {
	e := map[string]interface{}{"type": "input_audio_buffer.clear"}

	if err := c.Conn.WriteJSON(e); err != nil {
		return fmt.Errorf("failed to send input audio append event: %v", err)
	}

	return nil
}
