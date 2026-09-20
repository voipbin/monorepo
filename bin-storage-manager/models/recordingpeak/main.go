package recordingpeak

// RecordingFilePeak holds the computed waveform peaks and duration for a
// single recording audio file, keyed elsewhere by filename.
type RecordingFilePeak struct {
	// Peaks is the absolute-magnitude (0.0 to 1.0) downsampled waveform peak
	// values for the whole file. Empty when peak computation failed.
	Peaks []float64 `json:"peaks"`
	// Duration is the audio duration in seconds, computed from the WAV header.
	// 0 when the header could not be parsed.
	Duration float64 `json:"duration"`
}
