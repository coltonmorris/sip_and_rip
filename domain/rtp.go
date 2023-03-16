package domain

import (
	"fmt"
	"io"
	"time"

	"sip_and_rip/adapters"
)

func sendWav(sipMsg *adapters.SipMsg) error {
	rtpAddr, err := sipMsg.GetRtpAddress()
	if err != nil {
		fmt.Println("sendWav: error getting rtp addr: ", err)
		return err
	}

	ssrc, _, err := sipMsg.GetSsrc()
	if err != nil {
		fmt.Println("sendWav: error getting ssrc: ", err)
		return err
	}

	mediaOpts := sipMsg.GetMediaOptions()

	ulawRtpClient, err := adapters.NewRtpClient(rtpAddr, ssrc, mediaOpts)
	if err != nil {
		fmt.Println("sendWav: error creating rtp client:", err)
		return err
	}

	ulawReader, err := adapters.NewWavReader("ulaw-test.wav", mediaOpts)
	if err != nil {
		fmt.Println("sendWav: error creating wav reader")
		return err
	}
	defer ulawReader.Close()

	fmt.Println("sendWav: sending wav file to: ", rtpAddr.String())
	for {
		frame, err := ulawReader.NextRtpFrame()
		if err != nil && err != io.EOF {
			return err
		} else if err == io.EOF {
			break
		}

		// send the buffer to the sip client using rtp
		if _, err := ulawRtpClient.Write(frame); err != nil {
			fmt.Printf("failed to send RTP packet: %v\n", err)
			return err
		}

		// now we need to sleep for the packetization time
		time.Sleep(time.Duration(mediaOpts.PacketizationTimeMs) * time.Millisecond)
	}

	return nil
}
