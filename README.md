# Pixa WebSocket Server

A production-grade WebSocket server that facilitates real-time communication between Pixa hardware clients. This server acts as the central communication hub for Pixa devices, enabling seamless feature interactions and data relay between connected clients.

## System Architecture

The server is built with the following key components:

- **WebSocket Handler**: Manages client connections and message routing
- **Audio Processing**: Handles audio data transformation and relay
- **AI Integration**: Processes and augments client communication with AI capabilities

## Requirements

- Go 1.23.2 or later
- Docker 24.0.0 or later (for containerized deployment)
- Minimum 1GB RAM, recommended 2GB for production use
- Network access for WebSocket connections (port 80)

## Environment Variables

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `AZURE_API_KEY` | Yes | Azure API key for audio processing | `azure_key_xxx` |
| `LOG_LEVEL` | No | Logging level (debug, info, warn, error) | `info` |
| `MAX_CONNECTIONS` | No | Maximum concurrent client connections | `1000` |
| `PING_INTERVAL` | No | WebSocket ping interval in seconds | `30` |
| `AI_MODEL_VERSION` | No | Version of AI model to use | `v1.0` |

## Configuration

The server can be configured using a `config.yaml` file in the root directory:

```yaml
server:
  port: 80
  read_timeout: 60s
  write_timeout: 60s
  max_message_size: 1024

websocket:
  ping_interval: 30s
  pong_wait: 60s
  write_wait: 10s
  max_message_queue: 256

audio:
  sample_rate: 44100
  channels: 2
  bit_depth: 16
```

## Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/humblenginr/pixa-audio-relay-server.git
   cd pixa-audio-relay-server
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Set up environment variables:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. Run tests:
   ```bash
   go test ./...
   ```

5. Start the development server:
   ```bash
   go run cmd/server/main.go
   ```

## Production Deployment

### Docker Deployment

1. Build the container:
   ```bash
   docker build -t pixa-websocket-server:latest .
   ```

2. Run in production:
   ```bash
   docker run -d \
     --name pixa-server \
     -p 80:80 \
     -e AZURE_API_KEY=your_key_here \
     pixa-websocket-server:latest
   ```

### Kubernetes Deployment

Kubernetes manifests are available in the `deploy/k8s` directory. Deploy using:

```bash
kubectl apply -f deploy/k8s/
```

## Client Protocol

Clients connect via WebSocket to `ws://server:80/`. The protocol supports sending binary message of audio data in 16-Bit PCM format for now. 

## Project Structure

```
.
├── cmd/                # Application entrypoints
│   └── server/        # Server implementation
├── internal/          # Private application code
│   ├── ai/           # AI processing logic
│   ├── utils/        # Internal utilities
│   └── websocket/    # WebSocket handling
├── pkg/
│   └── audio/        # Public audio processing package
└── deploy/           # Deployment configurations
    ├── docker/       # Docker compositions
    └── k8s/          # Kubernetes manifests
```

## Security

For security concerns, please email tech@pixa.com instead of using the issue tracker.

## License

This project is proprietary software owned by Pixaverse Studios. All rights reserved.

## Support

For support:
- Technical issues: Create an issue in the repository
- Enterprise support: Contact tech@pixa.com
