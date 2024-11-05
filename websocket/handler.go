package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"pixa-demo/chat"

	"github.com/gorilla/websocket"
)

// Message defines the structure of incoming messages
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Upgrader for WebSocket connection, allowing connections from any origin
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// HandleConnections handles WebSocket connections and message routing
func HandleConnections(w http.ResponseWriter, r *http.Request, handleRecordedInput func(*chat.ChatGPTClient, string) error) {
	// Upgrade the HTTP connection to a WebSocket connection
	clientConnection, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer clientConnection.Close()

	// Establish a ChatGPT client for this session
	chatClient, err := chat.NewAzureClient()
	if err != nil {
		log.Printf("Failed to establish ChatGPT connection: %v", err)
		return
	}
	defer chatClient.Conn.Close()
	go chatClient.WatchServerEvents(clientConnection)
	for {
		_, message, err := clientConnection.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Invalid message format: %v", err)
			continue
		}

		if msg.Type == "input_audio_buffer.append" {
			data, _ := msg.Data.(string)
			// TODO: we probably would have to do the processing here
			err := chatClient.AppendToAudioBuffer(data)
			if err != nil {
				fmt.Printf("failed to send append to audio buffer event: %v\n", err)
			}

		} else if msg.Type == "input_audio_buffer.clear" {
			err := chatClient.ClearAudioBuffer()
			if err != nil {
				fmt.Printf("failed to send append to audio buffer event: %v\n", err)
			}
		} else {
			log.Printf("Unhandled message type: %s", msg.Type)
		}
	}
	fmt.Println("Client disconnected")
}
