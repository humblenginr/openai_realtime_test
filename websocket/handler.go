package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"pixa-demo/chat"

	"github.com/gorilla/websocket"
)

// Upgrader for WebSocket connection, allowing connections from any origin
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// HandleConnections handles WebSocket connections and message routing
func HandleConnections(w http.ResponseWriter, r *http.Request, handleRecordedInput func(*chat.ChatGPTClient, string) error) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		return
	}
	defer conn.Close()
	wc := WebsocketClient{c: conn}
	respAudioDeltaCh := make(chan string)
	respAudioDoneCh := make(chan bool)
	go wc.handleResponseAudio(respAudioDeltaCh, respAudioDoneCh)

	// Establish a ChatGPT client for this session
	chatClient, err := chat.NewAzureClient(respAudioDeltaCh, respAudioDoneCh)
	if err != nil {
		log.Printf("Failed to establish ChatGPT connection: %v", err)
		return
	}
	defer chatClient.Conn.Close()

	go chatClient.WatchServerEvents()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Invalid message format: %v", err)
			continue
		}

		if msg.Type == "audio" {
			dataBytes, _ := msg.Data.(string)
			err := handleRecordedInput(chatClient, dataBytes)
			if err != nil {
				log.Printf("Error handling recorded input: %v", err)
				continue
			}
		} else {
			log.Printf("Unhandled message type: %s", msg.Type)
		}
	}
	fmt.Println("Client disconnected")
}
