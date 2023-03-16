package adapters

import (
	"fmt"
	"net"

	"sip_and_rip/ports"
)

// UDPServer - a UDP server that listens for SIP messages
type UDPServer struct {
	addr *net.UDPAddr
	conn *net.UDPConn

	api ports.Api
}

// NewUDPServer - creates a new UDP server that listens on the given address
func NewUDPServer(addr string, api ports.Api) (*UDPServer, error) {
	serverAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	return &UDPServer{
		addr: serverAddr,
		api:  api,
	}, nil
}

// Serve - starts the UDP server
func (s *UDPServer) Serve() error {
	conn, err := net.ListenUDP("udp", s.addr)
	if err != nil {
		return err
	}

	s.conn = conn

	buf := make([]byte, 10*1024*1024) // 10mb max message size
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading UDP packet:", err)
			continue
		}

		if err := s.api.HandleSipMessage(remoteAddr, buf[:n], func(b []byte) error {
			if _, err := conn.WriteToUDP(b, remoteAddr); err != nil {
				fmt.Printf("Error writing UDP packet to %s: %v\n:", remoteAddr.String(), err)
			}
			return nil
		}); err != nil {
			fmt.Println("Error handling SIP message:", err)
			// TODO we can have the domain return formalized error types and map those to http codes
		}
	}

	return nil
}

// Close - closes the UDP server
func (s *UDPServer) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}

	return nil
}
