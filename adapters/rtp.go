package adapters

import (
	"net"

	"github.com/pion/rtp"

	"sip_and_rip/ports"
)

// TODO this is hardcoded to PCMU (ulaw). You can go see what the different payload types are in `pion/rtp`
var UlawPayloadType = uint8(0)

// RtpClient - represents an rtp client
type RtpClient struct {
	rtpAddr *net.UDPAddr
	conn    *net.UDPConn
	// the sequence number is used to identify the order of packets
	seq uint16
	// used for ordering rtp packets
	timestamp uint32
	// helps identify the source of the RTP stream
	ssrc uint32
	opts *ports.MediaOptions
}

// NewRtpClient - creates a new rtp client. The `ssrc` is found in the sdp request. The `opts` lets us know what kind of media we have agreed to send (negotiated through sdp). The `rtpAddr` is also found in the sdp request.
func NewRtpClient(rtpAddr *net.UDPAddr, ssrc uint32, opts *ports.MediaOptions) (*RtpClient, error) {
	// use a random timestamp offset to avoid collisions
	timestampOffset := RandUint32()
	seq := RandUint16()
	if ssrc == 0 {
		ssrc = RandUint32()
	}

	// Create a UDP connection to the client
	conn, err := net.DialUDP("udp", nil, rtpAddr)
	if err != nil {
		return nil, err
	}

	return &RtpClient{
		rtpAddr:   rtpAddr,
		seq:       seq,
		timestamp: timestampOffset,
		ssrc:      ssrc,
		opts:      opts,
		conn:      conn,
	}, nil
}

// Close - closes the rtp client
func (r *RtpClient) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}

	return nil
}

// Write - writes the rtp payload to the rtp client
func (r *RtpClient) Write(rtpPayload []byte) (int, error) {
	packet := r.newPacket(rtpPayload)

	// Serialize the RTP packet into a byte slice
	data, err := packet.Marshal()
	if err != nil {
		return 0, err
	}

	// Send the RTP packet over UDP
	n, err := r.conn.Write(data)
	if err != nil {
		return n, err
	}

	r.seq++
	r.timestamp += uint32(r.opts.GetSampleSize())

	return n, nil
}

// GetBuffer - returns a buffer of the proper size for an rtp packet
func (r *RtpClient) GetBuffer() []byte {
	return make([]byte, r.opts.GetBufferSize())
}

func (r *RtpClient) newPacket(payload []byte) *rtp.Packet {
	return &rtp.Packet{
		Header: rtp.Header{
			Version:        2,
			SequenceNumber: r.seq,
			PayloadType:    UlawPayloadType,
			Timestamp:      r.timestamp,
			SSRC:           r.ssrc,
		},
		Payload: payload,
	}
}
