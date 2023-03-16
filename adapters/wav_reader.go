package adapters

import (
	"io"
	"os"

	"sip_and_rip/ports"
)

// WavReader - a .wav file reader
type WavReader struct {
	opts *ports.MediaOptions
	file *os.File
}

// NewWavReader - creates a new wav file reader
func NewWavReader(filename string, opts *ports.MediaOptions) (*WavReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	return &WavReader{
		opts: opts,
		file: file,
	}, nil
}

// NextRtpFrame - reads the next rtp frame from the .wav file and returns it in a buffer
func (w *WavReader) NextRtpFrame() ([]byte, error) {
	// Create a new buffer to read the frame into
	buf := make([]byte, w.opts.GetBufferSize())

	// Read one audio frame from the file
	n, err := w.file.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if n == 0 {
		return nil, io.EOF
	}

	return buf[:n], nil
}

// Close -
func (w *WavReader) Close() error {
	return w.file.Close()
}
