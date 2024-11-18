package chat

type EventType string

const (
	SessionUpdateEventType           EventType = "session.update"
	SessionCreatedEventType          EventType = "session.created"
	ErrorEventType                   EventType = "error"
	ResponseCreateEventType          EventType = "response.create"
	ResponseCreatedEventType         EventType = "response.created"
	ConversationItemCreateEventType  EventType = "conversation.item.create"
	ConversationItemCreatedEventType EventType = "conversation.item.created"

	InputAudioBufferAppendEventType EventType = "input_audio_buffer.append"

	ResponseAudioDeltaEventType EventType = "response.audio.delta"
	ResponseAudioDoneEventType  EventType = "response.audio.done"

	AudioTranscriptDeltaEventType EventType = "response.audio_transcript.delta"
	AudioTranscriptDoneEventType  EventType = "response.audio_transcript.done"

	// this
	SpeechStartedEventType      EventType = "input_audio_buffer.speech_started"
	SpeechStoppedEventType      EventType = "input_audio_buffer.speech_stopped"
	AudioBufferClearedEventType EventType = "input_audio_buffer.cleared"
)

/*
	{
		"type": "input_audio_buffer.append",
		"data" : "base64encoded audio chunks"
	}
*/

// Message represents the structure of messages sent to the client
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// EventBase represents the base structure for all events
type EventBase struct {
	EventID *string   `json:"event_id,omitempty"`
	Type    EventType `json:"type"`
}

// ErrorEvent represents an error from the server
type ErrorEvent struct {
	EventBase
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains detailed error information
type ErrorDetail struct {
	Type    string  `json:"type"`
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Param   *string `json:"param,omitempty"`
	EventID string  `json:"event_id"`
}

// SessionUpdateEvent represents a session update
type SessionUpdateEvent struct {
	EventBase
	Session SessionData `json:"session"`
}

// SessionCreatedEvent represents a new session creation
type SessionCreatedEvent struct {
	EventBase
	Session SessionData `json:"session"`
}

// SessionData contains session configuration
type SessionData struct {
	Modalities              []string                `json:"modalities"`
	Instructions            string                  `json:"instructions,omitempty"`
	Voice                   string                  `json:"voice,omitempty"`
	InputAudioFormat        string                  `json:"input_audio_format,omitempty"`
	OutputAudioFormat       string                  `json:"output_audio_format,omitempty"`
	InputAudioTranscription InputAudioTranscription `json:"input_audio_transcription,omitempty"`
	TurnDetection           TurnDetection           `json:"turn_detection"`
	Tools                   []Tool                  `json:"tools,omitempty"`
	ToolChoice              string                  `json:"tool_choice,omitempty"`
	Temperature             float64                 `json:"temperature,omitempty"`
	MaxResponseOutputTokens string                  `json:"max_response_output_tokens,omitempty"`
}

// InputAudioTranscription represents audio transcription settings
type InputAudioTranscription struct {
	Model string `json:"model"`
}

// TurnDetection contains voice activity detection settings
type TurnDetection struct {
	Type              string  `json:"type"`
	Threshold         float64 `json:"threshold"`
	PrefixPaddingMs   int     `json:"prefix_padding_ms"`
	SilenceDurationMs int     `json:"silence_duration_ms"`
}

// Tool represents an available tool in the session
type Tool struct {
	Type        string     `json:"type"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  ToolParams `json:"parameters"`
}

// ToolParams defines the parameters for a tool
type ToolParams struct {
	Type       string           `json:"type"`
	Properties map[string]Param `json:"properties"`
	Required   []string         `json:"required"`
}

// Param represents a single parameter
type Param struct {
	Type string `json:"type"`
}

// ResponseEvent represents a response from the server
type ResponseEvent struct {
	EventBase
	Response Response `json:"response"`
}

// Response contains the response configuration
type Response struct {
	Modalities        []string `json:"modalities"`
	Instructions      string   `json:"instructions,omitempty"`
	Voice             string   `json:"voice,omitempty"`
	OutputAudioFormat string
}
