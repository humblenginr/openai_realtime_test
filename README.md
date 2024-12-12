# Pixa WebSocket Server

A production-grade WebSocket server that facilitates real-time communication between Pixa hardware clients. This server acts as the central communication hub for Pixa devices, enabling seamless feature interactions and data relay between connected clients.

## System Architecture

The server is built with the following key components:

- **WebSocket Handler**: Manages client connections and message routing with robust connection health monitoring
- **Audio Processing**: Handles audio data transformation and relay
- **AI Integration**: Processes and augments client communication with AI capabilities

## Requirements

- Go 1.23 or later
- Docker 24.0.0 or later (for containerized deployment)
- Minimum 1GB RAM, recommended 2GB for production use
- Azure OpenAI API access

## Configuration

The server supports multiple configuration methods in the following order of precedence:
1. Command-line flags
2. Environment variables
3. Configuration file
4. Default values

## Build Requirements

### CGO Requirements
This application uses Porcupine wake word detection, which requires CGO. When building, ensure CGO is enabled:

```bash
# For local builds
CGO_ENABLED=1 go build

# For cross-compilation
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build
```

> **Note**: CGO must be enabled (`CGO_ENABLED=1`) because the Porcupine wake word detection system uses native C bindings. Make sure you have a C compiler (like gcc) installed on your system.

### Dependencies
- Go 1.21 or higher
- GCC or another C compiler
- Porcupine wake word detection library

### Environment Variables

All configuration options can be set via environment variables with the prefix `PIXA_`. For example:
- `PIXA_SERVER_PORT=8080`
- `PIXA_WEBSOCKET_PING_INTERVAL=30s`
- `PIXA_AUDIO_SAMPLE_RATE=16000`

Required Azure OpenAI environment variables:
- `AZURE_OPENAI_KEY`: Your Azure OpenAI API key
- `AZURE_OPENAI_URL`: Your Azure OpenAI service WebSocket URL

### Configuration File

The server looks for `config.yaml` in the following locations:
- Current directory
- `./config/` directory
- `/etc/pixa/` directory

Example `config.yaml`:
```yaml
server:
  port: 8080

websocket:
  ping_interval: 30s
  pong_wait: 60s
  write_wait: 10s
  max_message_queue: 256

audio:
  sample_rate: 16000
  channels: 2
  format: "pcm_16"  # Supported formats: pcm_16, wav, mp3

azure:
  service_url: "your-azure-openai-websocket-url"  # Can also be set via AZURE_OPENAI_URL
  # Note: API key should be set via environment variable AZURE_OPENAI_KEY
```

## Development Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/pixaverse-studios/websocket-server.git
   cd websocket-server
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Create a configuration file:
   ```bash
   cp config.yaml.example config.yaml
   # Edit config.yaml with your settings
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
     -p 8080:8080 \
     -v /path/to/config.yaml:/etc/pixa/config.yaml \
     pixa-websocket-server:latest
   ```

### Kubernetes Deployment

Kubernetes manifests are available in the `deploy/k8s` directory. Deploy using:

```bash
kubectl apply -f deploy/k8s/
```

## Client Protocol

Clients connect via WebSocket to `ws://server:8080/`. The protocol supports sending binary message of audio data in 16-Bit PCM format for now. 

## Project Structure

```
.
├── cmd/                # Application entrypoints
│   └── server/        # Server implementation
├── internal/          # Private application code
│   ├── ai/           # AI processing logic
│   ├── config/       # Configuration management
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
