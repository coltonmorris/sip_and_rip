package adapters

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/jart/gosip/sdp"
)

//
func getAttributesFromSdp(sdpMsg *sdp.SDP) (ssrc uint32, cname string, rtcpAddr *net.UDPAddr, err error) {
	for _, a := range sdpMsg.Attrs {
		switch a[0] {
		case "X-nat":
			// TODO will have to learn how to deal with NAT eventually
			// `a=X-nat:0` indicates that the sender of the SDP is not behind a NAT device
			if a[1] != "0" {
				return ssrc, cname, rtcpAddr, fmt.Errorf("client is behind a NAT device. Probably just need to make this not error and see what happens")
			}
		case "rtcp":
			uri := ""
			rtcpVals := strings.Split(a[1], " ")

			// TODO checking for linphone's rtcp which looks like this: `a=rtcp:97535`. This will probably be different if it wasnt on localhost
			if len(rtcpVals) == 1 {
				uri = fmt.Sprintf("127.0.0.1:%s", rtcpVals[0])
			} else if len(rtcpVals) == 4 {
				uri = fmt.Sprintf("%s:%s", rtcpVals[3], rtcpVals[0])
			} else {
				return ssrc, cname, rtcpAddr, fmt.Errorf("invalid rtcp attribute: %s", a[1])
			}

			rtcpAddr, err = net.ResolveUDPAddr("udp", uri)
			if err != nil {
				return ssrc, cname, rtcpAddr, fmt.Errorf("error resolving rtcp addr: %v", err)
			}
		case "ssrc":
			ssrcVals := strings.Split(a[1], " ")

			// grab the ssrc, it helps identify the source of the RTP stream
			ui64, pErr := strconv.ParseUint(ssrcVals[0], 10, 32)
			if pErr != nil {
				return ssrc, cname, rtcpAddr, fmt.Errorf("error parsing ssrc: %v", pErr)
			}
			ssrc = uint32(ui64)

			// now check for cname
			if len(ssrcVals) > 1 && len(ssrcVals) < 3 {
				if strings.Contains(ssrcVals[1], "cname") {
					// TODO cname is used identify the source of the RTP stream. it is used in the RTCP sender report (SR) or the receiver report (RR)
					cname = strings.Split(ssrcVals[1], ":")[1]
				} else {
					return ssrc, cname, rtcpAddr, fmt.Errorf("encountered unknown ssrc attribute: %s", ssrcVals[1])
				}
			} else {
				return ssrc, cname, rtcpAddr, fmt.Errorf("encountered unknown ssrc attribute: %s", ssrcVals[1])
			}
		case "rtcp-xr":
		// this attribute is used to convey extended RTP Control Protocol (RTCP) reports. An example from linphone:
		//	`a=rtcp-xr:rcvr-rtt=all:10000 stat-summary=loss,dup,jitt,TTL voip-metrics`
		//		rcvr-rtt=all:10000: This parameter enables round-trip time (RTT) measurement for all RTP receivers and sets the maximum interval between reports to 10,000 milliseconds. This information can be used to assess the quality of the media delivery and to detect any issues with the network.
		// 		stat-summary=loss,dup,jitt,TTL,voip-metrics: This parameter requests that several types of statistics be included in the RTCP report. The statistics requested are:
		//			loss: The fraction of lost RTP packets
		// 			dup: The fraction of duplicated RTP packets
		// 			jitt: The amount of jitter in the received RTP packets
		// 			TTL: The time-to-live (TTL) value of the received RTP packets
		// 			voip-metrics: Voice over IP (VoIP) quality metrics, such as the mean opinion score (MOS), delay, and packet loss rate.
		case "record":
		// indicates if the media session is being recorded. Can be "off", "on", "sendonly", or "recvonly"
		case "rtcp-fb":
		// used to specify the RTCP feedback messages that should be used for monitoring and controlling congestion in the session. An examle from linphone:
		//	a=rtcp-fb:* trr-int 1000
		//	a=rtcp-fb:* ccm tmmbr
		//		* trr-int 1000: (Transport-wide Receiver Report Interval) indicates the time interval for sending RTCP receiver reports (1000ms). wildcard indicates that this applies to all media types in the session.
		default:
			// TODO stop erroring on unknown attributes
			// erroring on unknown attributes to make sure we are handling all of them for now
			return ssrc, cname, rtcpAddr, fmt.Errorf("encountered unknown sdp attribute: %s", a[0])
		}
	}

	if ssrc == 0 {
		ssrc = RandUint32()
	}

	return ssrc, cname, rtcpAddr, err
}
