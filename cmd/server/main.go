package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/pixaverse-studios/websocket-server/internal/websocket"
)

// setting up the server
func main() {
	handler := websocket.NewHandler()
	http.HandleFunc("/", handler.ServeHTTP)
	fmt.Println("WebSocket server starting on :80")
	if err := http.ListenAndServe(":80", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
