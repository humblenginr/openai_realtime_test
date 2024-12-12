package audio

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
	bitDepth    int
}

//func FromMP3(){}
//func FromWAV(){}

func (a *Audio) GetChannels() int {
	return a.channels
}

func (a *Audio) GetSampleRate() int {
	return a.sampleRate
}

// Number of samples in the frame
func (a *Audio) FrameLength() int {
	if a.bitDepth == 16 {
		return len(a.AsInt16())
	}
	return 0
}

func (a *Audio) AsBytes() []byte {
	return Int16ToPCM(Float32ToInt16(a.float32Data))
}

func (a *Audio) AsInt16() []int16 {
	return Float32ToInt16(a.float32Data)
}

func (a *Audio) AsFloat32() []float32 {
	return a.float32Data
}
func (a *Audio) AsPCM16() []byte {
	return Int16ToPCM(Float32ToInt16(a.float32Data))
}

func (a *Audio) AsMP3() ([]byte, error) {
	// Commenting this out because this requires the go-lame library which requires some complicated steps for installing the C dependencies.
	// mp3Data, err := float32ToMP3(a.float32Data, a.sampleRate, a.channels)
	// if err != nil {
	// 	return nil, err
	// }
	// return mp3Data, nil
	return nil, nil
}

func FromPCM16(data []byte, sampleRate int, channels int) Audio {
	return Audio{
		float32Data: Pcm16toFloat32(data),
		sampleRate:  sampleRate,
		channels:    channels,
		bitDepth:    16,
	}
}

func (a *Audio) Resample(targetSampleRate int) {
	a.float32Data = ResampleAudio(a.float32Data, float64(a.sampleRate), float64(targetSampleRate))
	a.sampleRate = targetSampleRate
}

// Convert stereo to mono if input is 2 channels
// Assuming interleaved stereo samples: [left1, right1, left2, right2, ...]
func (a *Audio) StereoToMono() {
	audioSlice := Float32ToInt16(a.float32Data)

	monoSlice := make([]int16, len(audioSlice)/2)
	for i := 0; i < len(monoSlice); i++ {
		// Average the left and right channels
		left := float64(audioSlice[i*2])
		right := float64(audioSlice[i*2+1])
		monoSample := int16((left + right) / 2)
		monoSlice[i] = monoSample
	}

	a.float32Data = Int16ToFloat32(monoSlice)
	a.channels = 1
}

// IsSilent checks if the audio is silent using the default threshold
func (a *Audio) IsSilent() bool {
	return IsSilent(a.float32Data, DefaultSilenceThreshold)
}

// IsSilentWithThreshold checks if the audio is silent using a custom threshold
func (a *Audio) IsSilentWithThreshold(threshold float32) bool {
	return IsSilent(a.float32Data, threshold)
}
