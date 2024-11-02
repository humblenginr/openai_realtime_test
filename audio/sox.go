package audio

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

// PlayBase64PCM16Audio decodes a base64-encoded PCM16 audio and plays it.
// Returns an error if decoding or playback fails.
func PlayBase64PCM16Audio(base64Audio string) error {
	// Decode the base64 string to raw PCM data
	audioData, err := base64.StdEncoding.DecodeString(base64Audio)
	if err != nil {
		return fmt.Errorf("error decoding base64: %v", err)
	}

	// Convert PCM16 data to a WAV format in memory
	wavData, err := PCMToWAV(audioData, 44100, 2)
	if err != nil {
		return fmt.Errorf("error converting PCM to WAV: %v", err)
	}

	// Load the WAV data into a beep.StreamSeekCloser
	streamer, format, err := wav.Decode(bytes.NewReader(wavData))
	if err != nil {
		return fmt.Errorf("error decoding WAV: %v", err)
	}
	defer streamer.Close()

	// Initialize speaker with the sample rate from the WAV format
	if err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)); err != nil {
		return fmt.Errorf("error initializing speaker: %v", err)
	}

	// Play the audio
	done := make(chan bool)
	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		done <- true
	})))

	// Wait for playback to finish
	<-done
	fmt.Println("Playback finished")
	return nil
}

// PCMToWAV converts PCM data to a WAV file in memory with specified sample rate and channels
func PCMToWAV(pcmData []byte, sampleRate, channels int) ([]byte, error) {
	var buf bytes.Buffer

	// WAV header
	buf.Write([]byte("RIFF"))
	buf.Write([]byte{0, 0, 0, 0}) // Placeholder for file size
	buf.Write([]byte("WAVE"))
	buf.Write([]byte("fmt "))                                                                                                                    // Format chunk
	buf.Write([]byte{16, 0, 0, 0})                                                                                                               // Size of the format chunk (16 bytes)
	buf.Write([]byte{1, 0})                                                                                                                      // Audio format (1 = PCM)
	buf.Write([]byte{byte(channels), 0})                                                                                                         // Number of channels
	buf.Write([]byte{byte(sampleRate & 0xFF), byte((sampleRate >> 8) & 0xFF), byte((sampleRate >> 16) & 0xFF), byte((sampleRate >> 24) & 0xFF)}) // Sample rate
	byteRate := sampleRate * channels * 2
	buf.Write([]byte{byte(byteRate & 0xFF), byte((byteRate >> 8) & 0xFF), byte((byteRate >> 16) & 0xFF), byte((byteRate >> 24) & 0xFF)}) // Byte rate
	blockAlign := channels * 2
	buf.Write([]byte{byte(blockAlign), 0})                                                                                                               // Block align
	buf.Write([]byte{16, 0})                                                                                                                             // Bits per sample (16 bits)
	buf.Write([]byte("data"))                                                                                                                            // Data chunk
	buf.Write([]byte{byte(len(pcmData) & 0xFF), byte((len(pcmData) >> 8) & 0xFF), byte((len(pcmData) >> 16) & 0xFF), byte((len(pcmData) >> 24) & 0xFF)}) // Data size
	buf.Write(pcmData)

	// Update file size in RIFF header
	data := buf.Bytes()
	fileSize := len(data) - 8
	data[4] = byte(fileSize & 0xFF)
	data[5] = byte((fileSize >> 8) & 0xFF)
	data[6] = byte((fileSize >> 16) & 0xFF)
	data[7] = byte((fileSize >> 24) & 0xFF)

	return data, nil
}
