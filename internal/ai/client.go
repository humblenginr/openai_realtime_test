package ai

import (
	"context"
	"github.com/pixaverse-studios/websocket-server/pkg/audio"
)

// This package provides an interface to interact with the Speech-to-Speech LLM.

type AIClient interface {
	// Initialize configures the LLM and initalizes the communication channel with the LLM
	Initialize(context.Context) error
	// GetResponseStream returns a channel through which the LLM responses are streamed
	GetResponseStream() <-chan audio.Audio
	// SendAudio is used to send audio packets to the LLM
	SendAudio(audio.Audio) error
	// Close closes the connection with the LLM
	Close()
}
