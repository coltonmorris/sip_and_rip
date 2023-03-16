package ports

import (
	"net"
)

// SendResponseCallback - is a callback function to use to send a resonse
type SendResponseCallback func(b []byte) error

// Api - is the interface for the sip/rtp servers api
type Api interface {
	HandleSipMessage(remoteAddr *net.UDPAddr, msg []byte, sendFunc SendResponseCallback) error
}
