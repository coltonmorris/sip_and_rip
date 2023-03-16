package ports

import (
	"github.com/jart/gosip/sdp"
)

type SdpMessage interface {
	// gets the uri where the server should send media to as indicated in the sdp message
	getMediaAddress() string

	// determines if the provided codec is supported as indicated in the sdp message
	isCodecSupported(sdp.Codec) bool

	// create a sdp response based on the sdp message
	newSdpResponse(acceptedCodecs ...sdp.Codec) SdpMessage

	// gets the uri where the server should send rtcp info to as indicated in the sdp message
	getRtcpAddress() string

	// gets the ssrc (helps identify the source of the rtp stream) as indicated in the sdp message
	getSsrc() uint32
	// gets the cname (used to associate a ssrc with a particual user or device) as indicated in the sdp message
	getCname() string

	// determines if the sdp message is a request (sent from a client) or a response (sent from our server)
	isRequest() bool
}
