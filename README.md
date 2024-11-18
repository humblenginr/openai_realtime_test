# Pixa-Audio-Relay-Server: WebSocket Server for Real-Time Chat

This project demonstrates a WebSocket server written in Go to support real-time chat functionality. It provides a foundation for scalable and interactive web applications requiring live updates and messaging.

## Setup Instructions

### Prerequisites

- Go 1.20 or later
- Internet access for testing client connections

### Steps to Run the Server

1. Clone the repository:

   ```bash
   git clone https://github.com/humblenginr/pixa-audio-relay-server.git
   cd pixa-audio-relay-server
   ```

2. Build and run the server:

    ```bash
    AZURE_API_KEY=YOUR_KEY_HERE go run main.go
    ```

3. The server starts on port 80. You can test WebSocket functionality with a WebSocket client (e.g., browser-based tools or libraries like websocat).
