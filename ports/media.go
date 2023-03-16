package ports

// MediaReader - reads a media stream from a source
type MediaReader interface {
	NextRtpFrame() ([]byte, error)
}

// MediaOptions - the type of media that will be sent
type MediaOptions struct {
	// an arbitrary number, but a practical one. balances packet size, network delay, and codec efficiency. commonly used for ulaw/g711.
	// a smaller packetization time can result in lower latency and more granular control over the encoding process, but can also increase packet overhead and decrease codec efficieny. A larger number reduces packet overhead and increases codec efficiency, but can result in higher latency and reduced control over the encoding process.
	// typically 20ms for ulaw/g711
	PacketizationTimeMs int
	// the number of analog samples taken per second to convert to digital form. So 8000 analog samples are taken per second to convert to digital form. Higher the sampling rate, better is the quality
	// a typical g711 sample rate. 8000hz.
	SampleRateHz int
	// Lets us know how many bytes are in a sample size
	// g711 typically uses 16bit PCM samples, so 2 bytes.
	SampleFormatPcmBytes int
	// mono channels are size 1, stereo channels are size 2, and maybe possibly more for idk why?
	ChannelSize int
}

// GetFramesPerSecond - the number of frames per second
func (m *MediaOptions) GetFramesPerSecond() int {
	if m == nil {
		m = &MediaOptions{}
	}

	// default to 20ms
	if m.PacketizationTimeMs == 0 {
		m.PacketizationTimeMs = 20
	}

	// 1000ms is 1 second
	return 1000 / m.PacketizationTimeMs
}

// GetSampleSize - the number of samples per frame
func (m *MediaOptions) GetSampleSize() int {
	if m == nil {
		m = &MediaOptions{}
	}

	// default to 8000hz
	if m.SampleRateHz == 0 {
		m.SampleRateHz = 8000
	}

	return m.SampleRateHz / m.GetFramesPerSecond()
}

// GetBufferSize - the size of the buffer needed to hold a frame
func (m *MediaOptions) GetBufferSize() int {
	if m == nil {
		m = &MediaOptions{}
	}

	// default to 1 byte per sample
	if m.SampleFormatPcmBytes == 0 {
		m.SampleFormatPcmBytes = 1
	}

	// default to mono
	if m.ChannelSize == 0 {
		m.ChannelSize = 1
	}

	return m.GetSampleSize() * m.SampleFormatPcmBytes * m.ChannelSize
}
