package adapters

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	"sip_and_rip/ports"

	"github.com/jart/gosip/dialog"
	"github.com/jart/gosip/sdp"
	"github.com/jart/gosip/sip"
)

// SipMsg - a wrapper around gosip's sip messages
type SipMsg struct {
	msg *sip.Msg
}

// ParseSipMsg - parses a sip message from a byte array
func ParseSipMsg(b []byte) (*SipMsg, error) {
	m, err := sip.ParseMsg(b)
	if err != nil {
		return nil, err
	}

	sipMsg := &SipMsg{
		msg: m,
	}

	if err = sipMsg.Validate(); err != nil {
		return nil, err
	}

	return sipMsg, nil
}

// NewSipMsg - creates a new sip message given a jart/gosip message
func NewSipMsg(m *sip.Msg) (*SipMsg, error) {
	sipMsg := &SipMsg{
		msg: m,
	}

	if err := sipMsg.Validate(); err != nil {
		return nil, err
	}

	return sipMsg, nil
}

// String - returns a string representation of the sip message
func (s *SipMsg) String() string {
	return s.msg.String()
}

// Copy - creates a copy of the sip message
func (s *SipMsg) Copy() *SipMsg {
	return &SipMsg{
		msg: s.msg.Copy(),
	}
}

// Append - turns a sip message back into a packet
func (s *SipMsg) Append(buf *bytes.Buffer) {
	s.msg.Append(buf)
}

// GetMethod - returns the method of the sip message
func (s *SipMsg) GetMethod() string {
	return s.msg.Method
}

// GetRequest - the distination URI
func (s *SipMsg) GetRequest() *sip.URI {
	return s.msg.Request
}

// GetCallID - identifies the call for the duration of the dialog
func (s *SipMsg) GetCallID() string {
	return s.msg.CallID
}

// GetContact - where to send response packets to
func (s *SipMsg) GetContact() *sip.Addr {
	return s.msg.Contact
}

// NewResponse - create a sip response based on the sip message
func (s *SipMsg) NewResponse(code int) (*SipMsg, error) {
	var sipMsg *sip.Msg
	var err error

	switch s.msg.Method {
	case sip.MethodInvite:
		sipMsg, err = s.newInviteResponse(code)
	case sip.MethodRegister:
		sipMsg, err = s.newRegisterResponse(code)
	case sip.MethodAck:
		return nil, fmt.Errorf("cannot create a response for an ACK")
	case sip.MethodBye:
		sipMsg = s.newByeResponse(code)
	case sip.MethodCancel:
		sipMsg = s.newCancelResponse(code)
	default:
		return nil, fmt.Errorf("unsupported method: %s", s.msg.Method)
	}

	if err != nil {
		return nil, err
	}

	msg, err := NewSipMsg(sipMsg)
	if err != nil {
		return nil, fmt.Errorf("error creating sip message %v", err)
	}

	return msg, nil
}

func (s *SipMsg) newByeResponse(code int) *sip.Msg {
	response := dialog.NewResponse(s.msg, code)

	response.Allow = ""
	if response.To == nil {
		response.To = &sip.Addr{
			Uri: s.GetRequest(),
		}
	}

	return response
}

func (s *SipMsg) newCancelResponse(code int) *sip.Msg {
	response := dialog.NewResponse(s.msg, code)

	response.Allow = ""
	if response.To == nil {
		response.To = &sip.Addr{
			Uri: s.GetRequest(),
		}
	}

	return response
}

func (s *SipMsg) newInviteResponse(code int) (*sip.Msg, error) {
	rtpAddr, err := s.GetRtpAddress()
	if err != nil {
		return nil, err
	}

	rtcpAddr, err := s.GetRtcpAddress()
	if err != nil {
		return nil, err
	}

	sdpMsg, err := sdp.Parse(string(s.msg.Payload.Data()))
	if err != nil {
		return nil, fmt.Errorf("error parsing SDP message %v", err)
	}

	sdpRes := sdp.New(rtpAddr, sdp.ULAWCodec)
	sdpRes.SendOnly = true
	sdpRes.RecvOnly = false

	// TODO we need a better way to determine if we should set the ssrc. the method auto generates a random one if it doesnt exist in the request
	// sdpRes.Attrs = append(sdpRes.Attrs, [2]string{"ssrc", fmt.Sprintf("%d cname:%s", si.ssrc, si.cname)})

	if rtcpAddr != nil {
		sdpRes.Attrs = append(sdpRes.Attrs, [2]string{"rtcp", fmt.Sprintf("%d IN IP4 %s", rtcpAddr.Port, rtcpAddr.IP)})
	}

	sdpRes.Origin.ID = sdpMsg.Origin.ID
	sdpRes.Origin.Version = sdpMsg.Origin.ID
	sdpRes.Session = sdpMsg.Session
	sdpRes.Ptime = s.GetMediaOptions().PacketizationTimeMs

	response := dialog.NewResponse(s.msg, code)

	// TODO shoud be our server instead?
	// response.Contact = si.sipMsg.Contact
	response.Contact = &sip.Addr{
		Uri: s.msg.Request,
	}
	response.Allow = ""
	response.Payload = sdpRes

	return response, nil
}

func (s *SipMsg) newRegisterResponse(code int) (*sip.Msg, error) {
	response := dialog.NewResponse(s.msg, code)

	response.Contact = s.GetContact()
	response.Expires = s.GetExpires()

	response.Allow = ""

	return response, nil
}

// GetUrn - some sip messages contain an URN, which is a unique identifier for the user
func (s *SipMsg) GetUrn() (string, error) {
	if s.msg.Contact == nil {
		return "", fmt.Errorf("No contact header found")
	}
	if s.msg.Contact.Param == nil {
		return "", nil
	}

	// this is a linked list, so we need to iterate over it
	p := s.msg.Contact.Param
	for {
		if p == nil {
			// no urn was found
			return "", nil
		}

		if p.Name == "" {
			return "", fmt.Errorf("encountered a param with an empty name")
		}

		if p.Value == "" {
			return "", fmt.Errorf("encountered a param with an empty value")
		}

		// this is what its called in Linphone clients
		if p.Name == "+sip.instance" {
			v := strings.Split(p.Value, ":")
			if len(v) != 3 {
				return "", fmt.Errorf("error parsing sip.instance urn")
			}

			return strings.TrimSuffix(v[2], ">"), nil
		}
		p = p.Next
	}
}

// GetExpires - returns the expires header from a sip message
func (s *SipMsg) GetExpires() int {
	return s.msg.Expires
}

// Validate - validates the sip message
func (s *SipMsg) Validate() error {
	// we dont worry about validating responses because they are made by us
	if s.isResponse() {
		fmt.Println("skipping validation for response")
		return nil
	}

	switch s.msg.Method {
	case sip.MethodInvite:
		return s.validateInvite()
	case sip.MethodRegister:
		return s.validateRegister()
	case sip.MethodAck:
		return nil
	case sip.MethodBye:
		return nil
	case sip.MethodCancel:
		return nil
	default:
		return fmt.Errorf("unsupported method: %s", s.msg.Method)
	}
	return nil
}

func (s *SipMsg) validateRegister() error {
	// the tag parameter is used to uniquely identify a specific dialog between two endpoints. The initial SIP request includes the unique tag in the `From` header, and the response should echo this back in the `To` header. This allows both endpoints to identify and keep track of the specific dialog.
	tag := s.msg.From.Param.Get("tag")
	if tag.Value == "" {
		return fmt.Errorf("tag is empty in the `from` attribute")
	}

	// some clients dont fill out the to header field, so we do it for them
	if s.msg.To == nil {
		s.msg.To = s.msg.From.Copy()
	}

	// TODO the linphone Contact header has extra values (doesnt seem to affect much):
	//	Contact: <sip:test@127.0.0.1:65199;transport=udp>;+sip.instance="<urn:uuid:f7777fd8-042b-0016-90f1-d92f69bcdd12>";+org.linphone.specs=lime
	//
	//      transport=udp indicates the transport protocol to use
	//      +sip.instance="<urn:uuid:{the_uuid}>"; indicates the unique identifier for the device useful for multiple devices behind the same NAT
	//	+org.linphone.specs=lime: This is a custom parameter that is specific to the Linphone SIP client. It is used to signal that the client supports the Linphone Instant Messaging and Presence Extension (LIME), which provides secure messaging and presence functionality.

	if s.msg.CSeq == 0 || s.msg.CSeqMethod == "" { // check that cseq is valid
		return fmt.Errorf("invalid cseq")
	}

	// callId is used to uniquely identify a specific SIP transaction, it can change between transactions (a client connection will have multiple callIds over its lifetime)
	if s.msg.CallID == "" { // check that call id is valid
		return fmt.Errorf("invalid call id")
	}

	return nil
}

func (s *SipMsg) validateInvite() error {
	sdpMsg, err := sdp.Parse(string(s.msg.Payload.Data()))
	if err != nil {
		return fmt.Errorf("error parsing SDP message %v", err)
	}

	tag := s.msg.From.Param.Get("tag")
	if tag.Value == "" {
		return fmt.Errorf("tag is empty in the `from` attribute")
	}

	if s.msg.To == nil {
		s.msg.To = &sip.Addr{
			Uri: s.msg.Request,
		}
	}

	// make sure we generate a tag now
	s.msg.To.Tag()

	// TODO the Session-Expires header is set by some sip clients to negotiate how long the session will be

	// only do audio
	if sdpMsg.Audio == nil {
		return fmt.Errorf("no audio in sdp")
	}

	// make sure the client supports our codec (g.711/u-law/PCMU)
	isSupportedCodec := false
	for _, c := range sdpMsg.Audio.Codecs {
		if c.Name == "PCMU" {
			isSupportedCodec = true
		}
	}
	if !isSupportedCodec {
		// TODO we need to send a 488 Not Acceptable Here response or something similar instead of continue
		return fmt.Errorf("client does not support our codec")
	}

	if s.msg.CSeq == 0 || s.msg.CSeqMethod == "" { // check that cseq is valid
		return fmt.Errorf("invalid cseq")
	}

	// callId is used to uniquely identify a specific SIP transaction, it can change between transactions (a client connection will have multiple callIds over its lifetime)
	if s.msg.CallID == "" { // check that call id is valid
		return fmt.Errorf("invalid call id")
	}

	return nil
}

// GetRtpAddress - returns the rtp address from the sdp message
func (s *SipMsg) GetRtpAddress() (*net.UDPAddr, error) {
	sdpMsg, err := sdp.Parse(string(s.msg.Payload.Data()))
	if err != nil {
		return nil, fmt.Errorf("error parsing SDP message %v", err)
	}

	addr := fmt.Sprintf("%s:%d", sdpMsg.Addr, sdpMsg.Audio.Port)

	// TODO maybe just do this once and save the info to our object
	rtpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("error resolving the rtpAddr %s: %v", addr, err)
	}

	return rtpAddr, nil
}

// GetRtcpAddress - returns the rtcp address from the sdp message
func (s *SipMsg) GetRtcpAddress() (*net.UDPAddr, error) {
	sdpMsg, err := sdp.Parse(string(s.msg.Payload.Data()))
	if err != nil {
		return nil, fmt.Errorf("error parsing SDP message %v", err)
	}

	_, _, rtcpAddr, err := getAttributesFromSdp(sdpMsg)
	if err != nil {
		return nil, fmt.Errorf("error getting attributes from sdp: %v", err)
	}

	return rtcpAddr, nil
}

// GetSsrc - returns the ssrc from the sdp message, includes the cname if available. used to identify the stream
func (s *SipMsg) GetSsrc() (uint32, string, error) {
	sdpMsg, err := sdp.Parse(string(s.msg.Payload.Data()))
	if err != nil {
		return 0, "", fmt.Errorf("error parsing SDP message %v", err)
	}

	ssrc, cname, _, err := getAttributesFromSdp(sdpMsg)
	if err != nil {
		return 0, "", fmt.Errorf("error getting attributes from sdp: %v", err)
	}

	return ssrc, cname, nil
}

// GetMediaOptions - returns the media options from the sdp message
func (*SipMsg) GetMediaOptions() *ports.MediaOptions {
	// TODO we should grab this dynamically when we implement more than just ulaw
	return &ports.MediaOptions{
		PacketizationTimeMs:  20,
		SampleRateHz:         8000,
		SampleFormatPcmBytes: 1,
		ChannelSize:          1,
	}
}

// IsUnregister - returns true if the message is a sip unregister request
func (s *SipMsg) IsUnregister() bool {
	if s.msg.Method != ports.MethodRegister {
		return false
	}

	if s.msg.Expires <= 0 {
		return true
	}

	return false
}

// GetCSeq - returns the cseq number and method from the sip message
func (s *SipMsg) GetCSeq() (int, string) {
	return s.msg.CSeq, s.msg.CSeqMethod
}

func (s *SipMsg) isResponse() bool {
	return s.msg.IsResponse()
}
