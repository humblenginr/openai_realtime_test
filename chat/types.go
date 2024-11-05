package chat

type EventType string

const (
	SessionUpdateEventType            = "session.update"
	SessionCreatedEventType           = "session.created"
	ErrorEventType                    = "error"
	ResponseCreateEventType           = "response.create"
	ResponseCreatedEventType          = "response.created"
	ConverstationItemCreateEventType  = "conversation.item.create"
	ConverstationItemCreatedEventType = "conversation.item.created"
	InputAudioBufferAppendEventType   = "input_audio_buffer.append"
)

type EventBase struct {
	EventID *string   `json:"event_id"`
	Type    EventType `json:"type"`
}

// ErrorEvent represents the structure for the error event
type ErrorEvent struct {
	EventBase
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents the details of the error
type ErrorDetail struct {
	Type    string  `json:"type"`
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Param   *string `json:"param"` // Param can be null, so it's a pointer
	EventID string  `json:"event_id"`
}

// SessionUpdateEvent represents the structure for the session.update event
type SessionUpdateEvent struct {
	EventID string      `json:"event_id"`
	Type    string      `json:"type"`
	Session SessionData `json:"session"`
}

// SessionCreatedEvent represents the structure for the session.created event
type SessionCreatedEvent struct {
	EventBase
	Session SessionData `json:"session"`
}

// SessionData represents session-specific data
type SessionData struct {
	Modalities              []string                `json:"modalities"`
	Instructions            string                  `json:"instructions"`
	Voice                   string                  `json:"voice"`
	InputAudioFormat        string                  `json:"input_audio_format"`
	OutputAudioFormat       string                  `json:"output_audio_format"`
	InputAudioTranscription InputAudioTranscription `json:"input_audio_transcription"`
	TurnDetection           TurnDetection           `json:"turn_detection"`
	Tools                   []Tool                  `json:"tools"`
	ToolChoice              string                  `json:"tool_choice"`
	Temperature             float64                 `json:"temperature"`
	MaxResponseOutputTokens string                  `json:"max_response_output_tokens"`
}

// InputAudioTranscription represents the transcription model details
type InputAudioTranscription struct {
	Model string `json:"model"`
}

// TurnDetection represents the turn detection configuration
type TurnDetection struct {
	Type              string  `json:"type"`
	Threshold         float64 `json:"threshold"`
	PrefixPaddingMs   int     `json:"prefix_padding_ms"`
	SilenceDurationMs int     `json:"silence_duration_ms"`
}

// Tool represents a tool that can be used in the session
type Tool struct {
	Type        string     `json:"type"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  ToolParams `json:"parameters"`
}

// ToolParams represents the parameters for a tool
type ToolParams struct {
	Type       string           `json:"type"`
	Properties map[string]Param `json:"properties"`
	Required   []string         `json:"required"`
}

// Param represents a parameter with its type
type Param struct {
	Type string `json:"type"`
}

type ResponseEvent struct {
	EventBase
	Response Response `json:"response"`
}

type Response struct {
	Modalities        []string `json:"modalities"`
	Instructions      string   `json:"instructions"`
	Voice             string   `json:"voice"`
	OutputAudioFormat string   `json:"output_audio_format"`
	Tools             []Tool   `json:"tools"`
	ToolChoice        string   `json:"tool_choice"`
	Temperature       float64  `json:"temperature"`
	MaxOutputTokens   int      `json:"max_output_tokens"`
}

type ConversationEvent struct {
	EventBase
	PreviousItemID *string     `json:"previous_item_id"` // Use *string to allow null values
	Item           MessageItem `json:"item"`
}

type MessageItem struct {
	ID      *string       `json:"id"`
	Type    string        `json:"type"`
	Role    string        `json:"role"`
	Content []ContentItem `json:"content"`
}

type ContentItem struct {
	Type  string `json:"type"`
	Audio string `json:"audio"`
}
