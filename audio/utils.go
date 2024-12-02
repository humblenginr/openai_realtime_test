package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/viert/go-lame"
)

func float32ToMP3(data []float32, sampleRate, channels int) ([]byte, error) {
	pcmData := float32ToPcm16(data)
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
}

func resample(inputData []float32, inputSampleRate, targetSampleRate float64) []float32 {
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

func downmixToMono(inputData []float32, channels int) []float32 {
	if channels <= 1 {
		return inputData
	}
	output := make([]float32, len(inputData)/channels)
	for i := 0; i < len(output); i++ {
		sum := float32(0)
		for j := 0; j < channels; j++ {
			sum += inputData[i*channels+j]
		}
		output[i] = sum / float32(channels)
	}
	return output
}

func float32ToPcm16(float32Array []float32) []byte {
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

func pcm16toFloat32(data []byte) []float32 {
	floatData := make([]float32, len(data))
	for i, sample := range data {
		if sample < 0 {
			floatData[i] = float32(sample) / 0x8000
		} else {
			floatData[i] = float32(sample) / 0x7fff
		}
	}
	return floatData
}

func toInt16(data []byte) []int16 {
	samples := make([]int16, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		samples[i/2] = int16(binary.LittleEndian.Uint16(data[i : i+2]))
	}
	return samples
}
