package wake

import (
	"container/ring"
	"context"
	"fmt"
	"log"
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

	go func() {
		defer p.Delete()
		lastNonSilentTime := time.Now()

		/*
			We need this ring, since the AI client requires continuity in frames to understand the context
			In the below figure, if porcupine detects the wake word in [t0], then we would just be sending t0, t1, and t2. In my testing, I have seen
			the AI struggles to identify the right context. Therefore, we buffer 10 frames and send them to give context to the AI client.


			Time:     t-5   t-4   t-3   t-2   t-1   t0    t1    t2
			Speech:   ...   ...   "He   y     Ko    n     how   are"
			Buffer:   [t-5] [t-4] [t-3] [t-2] [t-1]
			Detection:                               ^
			Sent:     [t-5] [t-4] [t-3] [t-2] [t-1] [t0]  [t1]  [t2]
		*/
		residualRing := ring.New(10)
		for {
			select {
			case <-ctx.Done():
				return
			case frame := <-inputCh:
				if frame.FrameLength() != porcupine.FrameLength {
					log.Fatalf("Invalid frame length, actual: %d, expected: %d", frame.FrameLength(), 512)
				}

				// Always process for wake word detection
				keywordIndex, err := p.Process(frame.AsInt16())
				if err != nil {
					fmt.Printf("Process error: %v\n", err)
					continue
				}

				// Check for wake word
				if keywordIndex >= 0 {
					fmt.Printf("Wake word detected!\n")
					if a.wakeWordDetected == false {
						residualRing.Do(func(frame any) {
							outputChan <- frame.(audio.Audio)
						})
					}
					a.wakeWordDetected = true
					lastNonSilentTime = time.Now()
				}

				// Update the ring
				residualRing.Value = frame
				residualRing = residualRing.Next()

				// If wake word is detected, start forwarding frames
				if a.wakeWordDetected {
					// Check for long silence to reset wake word detection
					silenceThreshold := audio.DefaultSilenceThreshold
					if a.config.SilenceThreshold != 0 {
						silenceThreshold = a.config.SilenceThreshold
					}
					if frame.IsSilentWithThreshold(silenceThreshold) {
						if time.Since(lastNonSilentTime) > 30*time.Second {
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
