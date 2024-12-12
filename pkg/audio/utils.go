package audio

import (
	//"bytes"
	"encoding/binary"
	"fmt"
	"math"
	//"github.com/viert/go-lame"
)

// Commenting this out because this requires the go-lame library which requires some complicated steps for installing the C dependencies.
/*func float32ToMP3(data []float32, sampleRate, channels int) ([]byte, error) {
	pcmData := Float32ToPcm16(data)
	buf := new(bytes.Buffer)
	encoder := lame.NewEncoder(buf)
	defer encoder.Close()

	if err := encoder.SetInSamplerate(sampleRate); err != nil {
		return nil, fmt.Errorf("failed to set input sample rate: %v", err)
	}
	if err := encoder.SetQuality(7); err != nil {
		return nil, fmt.Errorf("failed to set quality: %v", err)
	}
	if err := encoder.SetBrate(16); err != nil {
		return nil, fmt.Errorf("failed to set bitrate: %v", err)
	}
	if channels == 1 {
		if err := encoder.SetMode(lame.MpegMono); err != nil {
			return nil, fmt.Errorf("failed to set mono mode: %v", err)
		}
	}
	if err := encoder.SetLowPassFrequency(4000); err != nil {
		return nil, fmt.Errorf("failed to set lowpass filter: %v", err)
	}

	_, err := encoder.Write(pcmData)
	if err != nil {
		return nil, fmt.Errorf("error encoding: %v", err)
	}

	encoder.Close()
	return buf.Bytes(), nil
}*/

func ResampleAudio(inputData []float32, inputSampleRate, targetSampleRate float64) []float32 {
	ratio := targetSampleRate / inputSampleRate
	outputLength := int(float64(len(inputData)) * ratio)
	output := make([]float32, outputLength)

	// For downsampling, apply a proper low-pass filter
	if targetSampleRate < inputSampleRate {
		// Nyquist frequency is half the target sample rate
		cutoffFreq := targetSampleRate / 2
		windowSize := int(inputSampleRate / cutoffFreq * 2) // Filter window size

		// Apply Sinc filter (better than moving average)
		filtered := make([]float32, len(inputData))
		copy(filtered, inputData)

		for i := windowSize; i < len(inputData)-windowSize; i++ {
			sum := float32(0)
			weightSum := float32(0)

			for j := -windowSize / 2; j < windowSize/2; j++ {
				// Sinc function for low-pass filter
				x := float64(j) * math.Pi * cutoffFreq / inputSampleRate
				weight := float32(1)
				if x != 0 {
					weight = float32(math.Sin(x) / x)
				}
				// Apply Hanning window to reduce ringing
				weight *= float32(0.5 * (1 + math.Cos(2*math.Pi*float64(j)/float64(windowSize))))

				sum += inputData[i+j] * weight
				weightSum += weight
			}
			filtered[i] = sum / weightSum
		}
		inputData = filtered
	}

	// Perform resampling with linear interpolation
	for i := 0; i < outputLength; i++ {
		position := float64(i) / ratio
		index := int(position)
		decimal := position - float64(index)

		// Boundary check
		if index >= len(inputData)-1 {
			output[i] = inputData[len(inputData)-1]
			continue
		}

		// Linear interpolation
		a := inputData[index]
		b := inputData[index+1]
		output[i] = a + (b-a)*float32(decimal)
	}

	return output
}

func Float32ToPcm16(float32Array []float32) []byte {
	buffer := make([]byte, len(float32Array)*2)
	for i, sample := range float32Array {
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		var value int16
		if sample < 0 {
			value = int16(sample * 0x8000)
		} else {
			value = int16(sample * 0x7fff)
		}
		binary.LittleEndian.PutUint16(buffer[i*2:], uint16(value))
	}
	return buffer
}

// should be little endian
func Pcm16toFloat32(data []byte) []float32 {
	if len(data)%2 != 0 {
		panic("Input data length must be even for 16-bit PCM")
	}

	floatData := make([]float32, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		// Combine two bytes into a signed 16-bit integer (little-endian)
		sample := int16(data[i]) | int16(data[i+1])<<8

		// Normalize to the range [-1.0, 1.0]
		// We use 32768.0 consistently for both positive and negative values
		floatData[i/2] = float32(sample) / 32768.0
	}
	return floatData
}

func Int16ToFloat32(data []int16) []float32 {
	float32Data := make([]float32, len(data))
	for i, sample := range data {
		float32Data[i] = float32(sample) / 32768.0 // Normalize to [-1, 1]
	}
	return float32Data
}

func Float32ToInt16(data []float32) []int16 {
	output := make([]int16, len(data))
	for i, sample := range data {
		// Clamp the sample to the range [-1.0, 1.0]
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}

		// Scale to int16 range using consistent scaling factor
		scaledSample := sample * 32768.0

		// Convert to int16, handling the edge cases
		if scaledSample >= 32767.0 {
			output[i] = 32767
		} else if scaledSample <= -32768.0 {
			output[i] = -32768
		} else {
			output[i] = int16(scaledSample)
		}
	}
	return output
}

func Pcm16ToInt16Slice(data []byte) ([]int16, error) {
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("byte slice length is not a multiple of 2")
	}
	int16Slice := make([]int16, len(data)/2)
	for i := 0; i < len(int16Slice); i++ {
		int16Slice[i] = int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
	}
	return int16Slice, nil
}

func Int16ToPCM(data []int16) []byte {
	resultBytes := make([]byte, len(data)*2)
	for i, sample := range data {
		binary.LittleEndian.PutUint16(resultBytes[i*2:], uint16(sample))
	}
	return resultBytes
}

// DefaultSilenceThreshold represents -50dB in linear scale
const DefaultSilenceThreshold float32 = 0.01

// IsSilent checks if the audio data is below the given amplitude threshold
func IsSilent(data []float32, threshold float32) bool {
	for _, sample := range data {
		if abs(sample) > threshold {
			return false
		}
	}
	return true
}

// Helper function to get absolute value of float32
func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
