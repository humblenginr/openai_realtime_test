package wake

// AudioStats holds statistics about audio frames
type AudioStats struct {
	rms       float64
	peakLevel int16
	silence   bool
}
