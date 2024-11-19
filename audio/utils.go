package audio

import (
	"encoding/base64"
	"encoding/binary"
)

// ResampleAudio resamples audio data from one sample rate to another using linear interpolation
// inputData: the input audio samples as []float32
// inputSampleRate: the original sampling rate in Hz
// targetSampleRate: the desired output sampling rate in Hz
// Returns: resampled audio data as []float32
func ResampleAudio(inputData []float32, inputSampleRate, targetSampleRate float64) []float32 {
	ratio := targetSampleRate / inputSampleRate
	outputLength := int(float64(len(inputData)) * ratio)
	output := make([]float32, outputLength)

	for i := 0; i < outputLength; i++ {
		position := float64(i) / ratio
		index := int(position)
		decimal := position - float64(index)

		// Get sample a (current sample)
		var a float32 = 0
		if index < len(inputData) {
			a = inputData[index]
		}

		// Get sample b (next sample)
		var b float32 = 0
		if index+1 < len(inputData) {
			b = inputData[index+1]
		} else if len(inputData) > 0 {
			b = inputData[len(inputData)-1]
		}

		// Linear interpolation
		output[i] = a + (b-a)*float32(decimal)
	}

	return output
}

func DecodeBase64(base64String string) ([]byte, error) {
	// Decode base64 string to bytes
	bytes, err := base64.StdEncoding.DecodeString(base64String)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// DecodePCM16FromBase64 decodes a base64 string into PCM16 samples
// Returns the decoded int16 samples and any error encountered
func DecodePCM16FromBase64(base64String string) ([]int16, error) {
	bytes, err := DecodeBase64(base64String)
	if err != nil {
		return nil, err
	}
	// Convert bytes to int16 samples
	samples := make([]int16, len(bytes)/2)
	for i := 0; i < len(bytes); i += 2 {
		samples[i/2] = int16(binary.LittleEndian.Uint16(bytes[i : i+2]))
	}

	return samples, nil
}

// Float32To16BitPCM converts float32 audio samples to PCM16 format
// Returns the converted bytes
func Float32To16BitPCM(float32Array []float32) []byte {
	buffer := make([]byte, len(float32Array)*2)

	for i := 0; i < len(float32Array); i++ {
		// Clamp value between -1 and 1
		sample := float32Array[i]
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}

		// Convert to int16
		var value int16
		if sample < 0 {
			value = int16(sample * 0x8000)
		} else {
			value = int16(sample * 0x7fff)
		}

		// Write to buffer in little-endian
		binary.LittleEndian.PutUint16(buffer[i*2:], uint16(value))
	}

	return buffer
}

// Base64EncodeAudio converts float32 audio samples to base64-encoded PCM16 data
// The function processes the data in chunks to handle large arrays efficiently
func Base64EncodeAudio(float32Array []float32) string {
	// Convert to PCM16 first
	pcmData := Float32To16BitPCM(float32Array)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(pcmData)
}

// function to convert PCM16 back to float32
func PCM16ToFloat32(pcmData []int16) []float32 {
	float32Array := make([]float32, len(pcmData))
	for i, sample := range pcmData {
		if sample < 0 {
			float32Array[i] = float32(sample) / 0x8000
		} else {
			float32Array[i] = float32(sample) / 0x7fff
		}
	}
	return float32Array
}
