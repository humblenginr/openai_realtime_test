package wake

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	porcupine "github.com/Picovoice/porcupine/binding/go/v3"
)

// Take in a input stream of audio, and output the stream that can be sent to the client
func Porc(ctx context.Context, access_key string, input_stream chan []int16) (chan bool, error) {
	outputCh := make(chan bool)
	if access_key == "" {
		return outputCh, errors.New("Access key is required")
	}

	kp, err := filepath.Abs("./assets/porcupine_keyword_model/Hey-Kon_en_mac_v3_0_0.ppn")
	if err != nil {
		return nil, err
	}

	p := porcupine.Porcupine{
		AccessKey: access_key,
		//BuiltInKeywords: []porcupine.BuiltInKeyword{porcupine.COMPUTER},
		KeywordPaths:  []string{kp},
		Sensitivities: []float32{0.7}, // Default sensitivity
	}

	err = p.Init()
	if err != nil {
		return outputCh, fmt.Errorf("failed to initialize Porcupine: %v", err)
	}

	fmt.Printf("Porcupine initialized. Required frame length: %d. Required Sample Rate: %d\n", porcupine.FrameLength, porcupine.SampleRate)

	audioBuffer := make([]int16, 0, porcupine.FrameLength*8)
	lastDetectionTime := time.Now()

	go func() {
		defer p.Delete()
		frame := make([]int16, porcupine.FrameLength)

		for {
			select {
			case <-ctx.Done():
				return
			case samples := <-input_stream:
				// Add samples to buffer
				audioBuffer = append(audioBuffer, samples...)

				// Process complete frames
				for len(audioBuffer) >= porcupine.FrameLength {
					// Extract frame
					copy(frame, audioBuffer[:porcupine.FrameLength])

					// Process frame
					keywordIndex, err := p.Process(frame)
					if err != nil {
						fmt.Printf("Process error: %v\n", err)
					} else if keywordIndex >= 0 && time.Since(lastDetectionTime) > time.Second {
						fmt.Printf("Wake word detected!\n")
						outputCh <- true
						lastDetectionTime = time.Now()
					}

					// Remove processed frame
					audioBuffer = audioBuffer[porcupine.FrameLength:]
				}
			}
		}
	}()

	return outputCh, nil
}
