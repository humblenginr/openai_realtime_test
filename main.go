package main

import (
	"fmt"
	"log"
	"net/http"
	"pixa-demo/chat"
	"pixa-demo/websocket"
)

func handleRecordedInput(client *chat.ChatGPTClient, input string) error {
	fmt.Printf("Received recording\n")
	err := client.SendConversationItemCreateEvent(input)
	if err != nil {
		fmt.Printf("failed to send add conversation item event: %v\n", err)
	}
	err = client.SendCreateResponseEvent()
	if err != nil {
		fmt.Printf("failed to send create response event event: %v\n", err)
	}

	return nil
}

func main() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.HandleConnections(w, r, handleRecordedInput)
	})

	fmt.Println("WebSocket server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
