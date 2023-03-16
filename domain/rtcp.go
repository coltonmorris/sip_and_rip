package domain

import (
	"fmt"
	"net"
	"time"

	"github.com/beevik/ntp"
	"github.com/pion/rtcp"
)

func buildRtcpPacket(ssrc uint32, timestampOffset uint32, packetsSinceLastRtcp uint32, payloadOctectCount uint32) ([]byte, error) {
	t, err := ntp.Time("0.beevik-ntp.pool.ntp.org")
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	ntpTime := uint64(t.Unix())

	sr := &rtcp.SenderReport{
		SSRC:    ssrc,
		NTPTime: ntpTime,
		// TODO unclear if this is the right way to calculate the RTP timestamp
		RTPTime:           timestampOffset + uint32(ntpTime),
		PacketCount:       packetsSinceLastRtcp,
		OctetCount:        payloadOctectCount,
		Reports:           []rtcp.ReceptionReport{},
		ProfileExtensions: []byte{},
	}

	return sr.Marshal()
}

// should send an RTCP report to be 5% of the session bandwidth, so every 20~ packets
func startRtcp(remoteAddr *net.UDPAddr) error {
	// TODO grab this info from the rtp
	ssrc := uint32(0)
	timestampOffset := uint32(0)
	packetsSinceLastRtcp := uint32(0)
	payloadOctectCount := uint32(0)

	// Set up UDP listener for incoming SIP messages
	fmt.Println("listening for RTCP on:", remoteAddr.Port)
	localAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("0.0.0.0:%d", remoteAddr.Port))
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// TODO remove
	time.Sleep(5 * time.Second)

	go func() {
		buf := make([]byte, 10*1024*1024)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				fmt.Println("RTCP addr: ", addr, "Error reading RTCP packet:", err)
				time.Sleep(2 * time.Second)
				continue
			}

			fmt.Println("Received RTCP packet from", addr.String(), "of length", n, "bytes")
			fmt.Println("RTCP packet:", buf[:n])
		}
	}()

	data, err := buildRtcpPacket(ssrc, timestampOffset, packetsSinceLastRtcp, payloadOctectCount)
	if err != nil {
		fmt.Println("error building rtcp packet: ", err.Error())
		return err
	}

	fmt.Println("sending RTCP packet to", remoteAddr.String(), "of length", len(data), "bytes")

	// Create a UDP connection to the client
	rtcpClientConn, err := net.DialUDP("udp", &net.UDPAddr{Port: remoteAddr.Port}, remoteAddr)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer rtcpClientConn.Close()

	// TODO remove
	time.Sleep(5 * time.Second)

	fmt.Println("gonna try and send rtcp now")

	if _, err := rtcpClientConn.WriteToUDP(data, remoteAddr); err != nil {
		fmt.Printf("RTCP: Error writing UDP packet to %s: %v\n:", remoteAddr.String(), err)
	}

	return nil
}

func oldStartRtcp(addr string, cname string, ssrc uint32, seq uint16, timestampOffset uint32, packetsSinceLastRtcp uint32, payloadOctectCount uint32) error {
	remoteAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		fmt.Println("error resolving remote addr: ", err.Error())
		// TODO probably need to update fsm and either retry over a certain interval or send a BYE request (include callId and the tag parameter in the To header field). you would get back a 200 ok from the client.
		return err
	}

	data, err := buildRtcpPacket(ssrc, timestampOffset, packetsSinceLastRtcp, payloadOctectCount)
	if err != nil {
		fmt.Println("error building rtcp packet: ", err.Error())
		return err
	}

	// Create a UDP connection to the client
	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer conn.Close()

	_, err = conn.WriteToUDP(data, remoteAddr)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
