package ai

type EventType string

const (
	ErrorEventType                  EventType = "error"
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
