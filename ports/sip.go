package ports

import (
	"bytes"
	"net"

	"github.com/jart/gosip/sip"
)

// SipMessage represents a sip message
type SipMessage interface {
	// create a sip response based on the sip message
	NewResponse(statusCode int) (SipMessage, error)
	// determine if the sip message has all the proper fields
	Validate() error
	String() string
	Copy() SipMessage
	// turns a sip message back into a packet
	Append(*bytes.Buffer)
	// gets the sip message's cseq number and method
	GetCSeq() (int, string)
	// some sip messages contain an urn, which is a unique identifier for the user
	GetUrn() (string, error)
	GetMethod() string
	// the destination URI
	GetRequest() *sip.URI
	// identifies the call within its dialog
	GetCallID() string
	// where to send response packets to (or nil)
	GetContact() *sip.Addr
	// seconds registration should expire
	GetExpires() int
	// the media options chosen from the sdp
	GetMediaOptions() *MediaOptions
	// the address to send rtp media to. found in the sdp message
	GetRtpAddress() (*net.UDPAddr, error)
	// the address to send rtcp info to. found in the sdp message
	GetRtcpAddress() (*net.UDPAddr, error)
	// Get the ssrc info which includes the cname if available
	GetSsrc() (uint32, string, error)
	// determines if the sip message is a register message with 0 expiration, indicating its meant to unregister a client
	IsUnregister() bool
}

const (
	// MethodInvite - Indicates a client is being invited to participate in a call session.
	MethodInvite = "INVITE"
	// MethodAck - Confirms that the client has received a final response to an INVITE request.
	MethodAck = "ACK"
	// MethodBye - Terminates a call and can be sent by either the caller or the callee.
	MethodBye = "BYE"
	// MethodCancel - Cancels any pending request.
	MethodCancel = "CANCEL"
	// MethodOptions - Queries the capabilities of servers.
	MethodOptions = "OPTIONS"
	// MethodRegister - Registers the address listed in the To header field with a SIP server.
	MethodRegister = "REGISTER"
	// MethodPrack - Provisional acknowledgement.
	MethodPrack = "PRACK"
	// MethodSubscribe - Subscribes for an Event of Notification from the Notifier.
	MethodSubscribe = "SUBSCRIBE"
	// MethodNotify - Notify the subscriber of a new Event.
	MethodNotify = "NOTIFY"
	// MethodPublish - Publishes an event to the Server.
	MethodPublish = "PUBLISH"
	// MethodInfo - Sends mid-session information that does not modify the session state.
	MethodInfo = "INFO"
	// MethodRefer - Asks recipient to issue SIP request (call transfer.)
	MethodRefer = "REFER"
	// MethodMessage - Transports instant messages using SIP.
	MethodMessage = "MESSAGE"
	// MethodUpdate - Modifies the state of a session without changing the state of the dialog.
	MethodUpdate = "UPDATE"
)
