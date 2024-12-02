package audio

import (
	"fmt"
)

type AudioFormat int

const (
	PCM16BIT AudioFormat = iota
	MP3
	WAV
)

type Audio struct {
	float32Data []float32
	sampleRate  int
	channels    int
}

//func FromMP3(){}
//func FromWAV(){}

func (a *Audio) AsPCM16() []byte {
	return float32ToPcm16(a.float32Data)
}

func (a *Audio) AsMP3() ([]byte, error) {
	mp3Data, err := float32ToMP3(a.float32Data, a.sampleRate, a.channels)
	if err != nil {
		return nil, err
	}
	return mp3Data, nil
}

func FromPCM16(data []byte, sampleRate int, channels int) (Audio, error) {
	return Audio{
		float32Data: pcm16toFloat32(data),
		sampleRate:  sampleRate,
		channels:    channels,
	}, nil
}

func (a *Audio) Resample(targetSampleRate int) error {
	a.float32Data = resample(a.float32Data, float64(a.sampleRate), float64(targetSampleRate))
	a.sampleRate = targetSampleRate
	return nil
}

func (a *Audio) Downmix(targetChannels int) error {
	if a.channels == targetChannels {
		return nil // Already at the target
	}
	if targetChannels != 1 {
		return fmt.Errorf("only mono downmixing is supported")
	}
	a.float32Data = downmixToMono(a.float32Data, a.channels)
	a.channels = targetChannels
	return nil
}
