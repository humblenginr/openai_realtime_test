package wake

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	porcupine "github.com/Picovoice/porcupine/binding/go/v3"
	"github.com/pixaverse-studios/websocket-server/internal/config"
	"github.com/pixaverse-studios/websocket-server/pkg/audio"
)

type AudioPipeline struct {
	wakeWordDetected bool
	config           *config.WakeWordConfig
}

// NewAudioPipeline creates a new AudioPipeline with the given configuration
func NewAudioPipeline(cfg *config.WakeWordConfig) *AudioPipeline {
	return &AudioPipeline{
		wakeWordDetected: false,
		config:           cfg,
	}
}

/*
The AI client is very sensitive to the continuity of the frames. If we consider
the following scenario, after the wake word,
Input frames:      [1] [2] [3] [4] [5]
Output frames: 	   [1] [2]         [5]
This means that the AI client will receive non-continuos frames, and it will not recognise the speech.

Therefore, after wake word, we need to send all the frames to the AI Client to ensure continuity.

It's like reading a sentence without spaces - "thisishardertoread" vs "this is harder to read". The AI needs those "spaces" (silent frames) to properly understand the speech!
*/
func (a *AudioPipeline) Start(ctx context.Context, inputCh chan audio.Audio) (chan audio.Audio, error) {
	outputChan := make(chan audio.Audio, 10)

	// Initialize Pico using config
	kp, err := filepath.Abs(a.config.PorcupineConfig.KeywordModelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve keyword model path: %w", err)
	}

	p := porcupine.Porcupine{
		AccessKey:     a.config.PorcupineConfig.APIKey,
		KeywordPaths:  []string{kp},
		Sensitivities: []float32{a.config.PorcupineConfig.Sensitivity},
	}

	err = p.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Porcupine: %v", err)
	}

	fmt.Printf("Porcupine initialized. Required frame length: %d. Required Sample Rate: %d\n",
		porcupine.FrameLength, porcupine.SampleRate)

	go func() {
		defer p.Delete()
		defer close(outputChan)
		lastNonSilentTime := time.Now()

		for {
			select {
			case <-ctx.Done():
				return
			case frame := <-inputCh:
				// Always process for wake word detection
				keywordIndex, err := p.Process(frame.AsInt16())
				if err != nil {
					fmt.Printf("Process error: %v\n", err)
					continue
				}

				// Check for wake word
				if keywordIndex >= 0 {
					fmt.Printf("Wake word detected!\n")
					a.wakeWordDetected = true
					lastNonSilentTime = time.Now()
				}

				// If wake word is detected, start forwarding frames
				if a.wakeWordDetected {
					// Check for long silence to reset wake word detection
					silenceThreshold := audio.DefaultSilenceThreshold
					if a.config.SilenceThreshold != 0 {
						silenceThreshold = a.config.SilenceThreshold
					}
					if frame.IsSilentWithThreshold(silenceThreshold) {
						if time.Since(lastNonSilentTime) > 30*time.Second {
							fmt.Println("30 seconds of silence detected, resetting wake word detection")
							a.wakeWordDetected = false
							continue
						}
					} else {
						lastNonSilentTime = time.Now()
					}

					// Forward the frame to maintain continuity
					outputChan <- frame
				}
			}
		}
	}()

	return outputChan, nil
}
