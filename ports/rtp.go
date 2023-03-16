package ports

// RtpClient - represents an rtp client
type RtpClient interface {
	Close() error
	Write(payload []byte) (int, error)
	// TODO not sure this is how we want to do this. can just use the mediaOptions to get the buffer size
	// returns a buffer of the proper size for an rtp packet
	GetBuffer() ([]byte, error)
}
