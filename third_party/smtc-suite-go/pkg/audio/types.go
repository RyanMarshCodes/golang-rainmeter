package audio

// SampleFormat identifies the sample representation in an audio stream.
type SampleFormat int

const (
	SampleFormatUnknown SampleFormat = iota
	SampleFormatFloat32
	SampleFormatInt16
	SampleFormatInt24
	SampleFormatInt32
)

// String returns a short display name for the sample format.
func (f SampleFormat) String() string {
	switch f {
	case SampleFormatFloat32:
		return "Float32"
	case SampleFormatInt16:
		return "Int16"
	case SampleFormatInt24:
		return "Int24"
	case SampleFormatInt32:
		return "Int32"
	default:
		return "Unknown"
	}
}

// Format describes the PCM layout produced by an audio stream.
type Format struct {
	SampleRate    int
	Channels      int
	BitsPerSample int
	SampleFormat  SampleFormat
	BlockAlign    int
}

// BytesPerFrame returns the number of bytes in one interleaved audio frame.
func (f Format) BytesPerFrame() int {
	return f.BlockAlign
}

// IsKnown reports whether the format contains a recognized sample layout.
func (f Format) IsKnown() bool {
	return f.SampleFormat != SampleFormatUnknown && f.SampleRate > 0 && f.Channels > 0 && f.BlockAlign > 0
}
