package audio

import (
	//"bytes"
	"encoding/binary"
	"fmt"
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

func Resample(inputData []float32, inputSampleRate, targetSampleRate float64) []float32 {
	ratio := targetSampleRate / inputSampleRate
	outputLength := int(float64(len(inputData)) * ratio)
	output := make([]float32, outputLength)

	for i := 0; i < outputLength; i++ {
		position := float64(i) / ratio
		index := int(position)
		decimal := position - float64(index)

		var a float32
		if index < len(inputData) {
			a = inputData[index]
		}

		var b float32
		if index+1 < len(inputData) {
			b = inputData[index+1]
		} else if len(inputData) > 0 {
			b = inputData[len(inputData)-1]
		}

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

func Pcm16toFloat32(data []byte) []float32 {
	if len(data)%2 != 0 {
		panic("Input data length must be even for 16-bit PCM")
	}

	floatData := make([]float32, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		// Combine two bytes into a signed 16-bit integer
		sample := int16(data[i]) | int16(data[i+1])<<8

		// Normalize to the range [-1.0, 1.0]
		if sample < 0 {
			floatData[i/2] = float32(sample) / 0x8000
		} else {
			floatData[i/2] = float32(sample) / 0x7FFF
		}
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
	// Create a slice of int16 with the same length as the input
	output := make([]int16, len(data))

	for i, sample := range data {
		// Clamp the sample to the range [-1.0, 1.0]
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}

		// Scale and convert to int16
		if sample < 0 {
			output[i] = int16(sample * 32768) // 0x8000
		} else {
			output[i] = int16(sample * 32767) // 0x7FFF
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
