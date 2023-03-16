package domain

import (
	"bytes"
	"context"
	"fmt"
	"net"

	"sip_and_rip/adapters"
	"sip_and_rip/ports"
)

// Api - the api for this sip/rtp server
type Api struct {
	fsmCache *FsmCache
}

// NewApi - create a new api instance
func NewApi() *Api {
	return &Api{
		fsmCache: NewFsmCache(),
	}
}

// HandleSipMessage - a sip message has been received, advance its dialog
func (a *Api) HandleSipMessage(remoteAddr *net.UDPAddr, msg []byte, sendResponseCallback ports.SendResponseCallback) error {
	if a.isKeepAlive(msg) {
		return nil
	}

	sipMsg, err := adapters.ParseSipMsg(msg)
	if err != nil {
		fmt.Println("bad sip message: ", string(msg))

		return err
	}

	fmt.Printf("sip method %s, message length: %d; fsmCache length: %d\n", sipMsg.GetMethod(), len(msg), a.fsmCache.Len())

	fmt.Printf("sipMsg from addr %s: %v ", remoteAddr.String(), sipMsg)

	fsm, err := a.fsmCache.Get(sipMsg)
	if err == errFsmNotFound {
		fsm, err = a.fsmCache.NewSipFsm(context.Background(), sipMsg, remoteAddr.String())
		if err == errMissingContact || err == errMissingKey {
			// do nothing special, let each method handle this in their own ways
		} else if err != nil {
			fmt.Printf("Error creating FSM from %s: %v\n", remoteAddr.String(), err)
			fmt.Println("the offending sip message: ", sipMsg)
			return err
		}
	} else if err != nil {
		fmt.Printf("Error getting FSM from %s: %v", remoteAddr.String(), err)
		return err
	}

	switch sipMsg.GetMethod() {
	case ports.MethodBye:
		if fsm == nil {
			res, err := sipMsg.NewResponse(481)
			if err != nil {
				fmt.Println("Error creating 481 response: ", err)
				return err
			}

			var b bytes.Buffer
			res.Append(&b)

			fmt.Println("fsm not found for BYE request, sending 481 (Call/Transaction Does Not Exist) to client")

			if err := sendResponseCallback(b.Bytes()); err != nil {
				fmt.Println("error sending response: ", err)
			}

			return nil
		}

		if err := fsm.RecvBye(sipMsg, sendResponseCallback); err != nil {
			fmt.Printf("Error sending 200 OK in response to BYE %s: %v", remoteAddr.String(), err)
		}

		fmt.Println("sent response to BYE")

	case ports.MethodAck:
		// TODO i think ACK can sometimes contain updated sdp info for the call
		// TODO also might need to respond to the ACK
		if err := fsm.RecvAck(); err != nil {
			fmt.Printf("Error sending 200 OK in response to ACK %s: %v\n", remoteAddr.String(), err)
			return err
		}

		fmt.Println("recieved ACK", sipMsg)

	case ports.MethodCancel:
		// the fsm doesn't exist for this cancel request, so we should send a 481
		if fsm == nil {
			res, err := sipMsg.NewResponse(481)
			if err != nil {
				fmt.Println("Error creating 481 response: ", err)
				return err
			}

			fmt.Println("fsm not found for CANCEL request, sending 481 (Call/Transaction Does Not Exist) to client")
			var b bytes.Buffer
			res.Append(&b)

			if err := sendResponseCallback(b.Bytes()); err != nil {
				fmt.Println("error sending response: ", err)
			}

			return nil
		}

		if err := fsm.RecvCancel(sipMsg, sendResponseCallback); err != nil {
			fmt.Printf("Error sending 200 OK to CANCEL from %s: %v\n", remoteAddr.String(), err)
		}

	case ports.MethodInvite:
		if fsm == nil {
			fmt.Println("fsm not found for INVITE request")
			return fmt.Errorf("fsm not found for INVITE request")
		}

		// TODO will need to fix the 183 sdp body to get the 100rel flow working
		// check if "100rel" is in the `Supported` header. This indicates if the sip client supports the 100 -> 180 -> 200 flow.
		// if strings.Contains(sipMsg.Supported, "100rel") {
		// 	if err := fsm.SendTrying(sipMsg, sdpMsg, func(b bytes.Buffer) error {
		// 		if _, err := conn.WriteToUDP(b.Bytes(), remoteAddr); err != nil {
		// 			fmt.Printf("Error writing UDP packet to %s: %v\n:", remoteAddr.String(), err)
		// 		}
		// 		return nil
		// 	}); err != nil {
		// 		fmt.Printf("Error sending 100 Trying to %s: %v\n", remoteAddr.String(), err)
		// 	}
		//
		// 	continue
		// }

		if err := fsm.SendOk(sipMsg, sendResponseCallback); err != nil {
			fmt.Printf("Error sending 200 OK to %s: %v\n", remoteAddr.String(), err)
		}

	// We never need to send messages back to the sip client, we should just ack the message
	case ports.MethodRegister:
		// check for when client is unregistering
		if sipMsg.IsUnregister() {
			if err := a.fsmCache.CloseFsm(sipMsg); err != nil {
				fmt.Printf("Error closing FSM for key %s from %s: %v\n", fsm.key, remoteAddr.String(), err)
			}
			return nil
		}

		if err := fsm.RegisterSendOk(sipMsg, sendResponseCallback); err != nil {
			fmt.Printf("Error sending 200 OK for fsm %s from %s: %v\n", fsm.key, remoteAddr.String(), err)
			return err
		}
	default:
		fmt.Printf("received unknown message method type: %s\n", sipMsg.GetMethod())
	}

	fmt.Println(a.fsmCache.String())
	return nil
}

func (*Api) isKeepAlive(msg []byte) bool {
	// linphone's keep alive is 4 bytes
	// telephone's keep alive is 2 bytes
	if len(msg) <= 4 {
		return true
	}
	return false
}
