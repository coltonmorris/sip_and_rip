package domain

import (
	"bytes"
	"context"
	"fmt"

	"sip_and_rip/adapters"
	"sip_and_rip/ports"

	"github.com/looplab/fsm"
)

var errCSeqRetry = fmt.Errorf("cseq retry")

// SipFsm - a finite state machine for handling sip's dialogs
type SipFsm struct {
	FSM            *fsm.FSM
	ctx            context.Context
	key            string // unique key for this fsm
	prevCSeqMethod string
	prevCSeq       int
	callIds        []string
	registerCallId string
	addr           string
}

// NewSipFsm - creates a new sip finite state machine for handling sip's dialogs
func NewSipFsm(ctx context.Context, key string, addr string) (*SipFsm, error) {
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	f := &SipFsm{
		ctx:  ctx,
		key:  key,
		addr: addr,
	}

	f.FSM = fsm.NewFSM(
		"init", // the first state
		fsm.Events{
			// TODO maybe we want a different FSM for sip and one for rtp, because sometimes we get register requests during a call
			// what happens if we fail a register request during a call?

			// TODO sometimes the state could be repeated if we get a retransmission, so we should start over. like if we get a cancel request from the client `recv_cancel` we are supposed to respond with a 200 ok, but the client might no get that and will send the `recv_cancel` again
			// TODO the callee (server) can interrupt the session at any time by sending a CANCEL
			{Name: "register_send_200",
				Src: []string{
					"call_terminated",
					"waiting_for_register",
					"init",
					"register_sent_200",
				}, Dst: "register_sent_200"},
			{Name: "register_failed", Src: []string{"register_sent_200"}, Dst: "waiting_for_register"},
			// sometimes register isn't used, so can come from the init state.
			{Name: "invite_send_100", Src: []string{"register_sent_200", "init", "call_terminated", "call_cancelled"}, Dst: "invite_sent_100"},
			{Name: "invite_send_180", Src: []string{"invite_sent_100"}, Dst: "invite_sent_180"},
			{Name: "invite_send_183", Src: []string{"invite_sent_180"}, Dst: "invite_sent_183"},
			// TODO change 183 if needed to 180
			{Name: "invite_send_200", Src: []string{"init", "invite_sent_183", "call_terminated", "call_cancelled", "register_sent_200"}, Dst: "invite_sent_200"},
			{Name: "invite_recv_ack", Src: []string{"invite_sent_200"}, Dst: "call_established"},
			{Name: "recv_cancel", Src: []string{"invite_sent_100", "invite_sent_180", "invite_sent_183", "invite_sent_200", "call_established", "call_terminated"}, Dst: "call_cancelled"},
			{Name: "send_bye", Src: []string{"call_established"}, Dst: "sent_bye"},
			{Name: "recv_bye", Src: []string{"call_established", "invite_sent_200", "call_cancelled"}, Dst: "call_terminated"},
			{Name: "recv_200", Src: []string{"sent_bye"}, Dst: "call_terminated"},
		},
		fsm.Callbacks{
			"enter_state": func(ctx context.Context, e *fsm.Event) { fmt.Printf("STATE CHANGE: %s -> %s\n", e.Src, e.Dst) },
		},
	)

	return f, nil
}

func (f *SipFsm) Close() error {
	return nil
}

func (f *SipFsm) BeforeHook(sipMsg *adapters.SipMsg) error {
	f.AddCallId(sipMsg.GetCallID())

	cseq, method := sipMsg.GetCSeq()

	if cseq == 0 || method == "" { // check that cseq is valid
		return fmt.Errorf("cseq is required")
	} else if f.prevCSeqMethod == "" || f.prevCSeq == 0 { // check for first time
		f.prevCSeqMethod = method
		f.prevCSeq = cseq
	} else if f.prevCSeq == cseq && f.prevCSeqMethod == method { // check for retries
		return errCSeqRetry
	}

	return nil
}

func (f *SipFsm) RecvCancel(sipMsg *adapters.SipMsg, send ports.SendResponseCallback) error {
	// send fsm a recv_cancel
	err := f.FSM.Event(f.ctx, "recv_cancel")
	if err != nil {
		fmt.Println("FSM: error recieving CANCEL: ", err.Error())
		return err
	}

	response, err := sipMsg.NewResponse(200)
	if err != nil {
		fmt.Println("FSM: error sending 200 OK to CANCEL: ", err.Error())
		return err
	}

	var b bytes.Buffer
	response.Append(&b)

	if err := send(b.Bytes()); err != nil {
		if fsmErr := f.FSM.Event(f.ctx, "register_failed"); fsmErr != nil {
			err = fmt.Errorf("%w; FSM: error in RecvCancel %v: ", err, fsmErr)
		}

		return err
	}

	fmt.Println("sent 200 OK in response to CANCEL")

	return nil
}

func (f *SipFsm) RegisterSendOk(sipMsg *adapters.SipMsg, send ports.SendResponseCallback) error {
	// check for states that should loop back (not change to "invite_sent_200")
	if f.FSM.Current() != "call_established" && f.FSM.Current() != "invite_sent_200" && f.FSM.Current() != "register_sent_200" {
		// first update fsm to make sure our state is valid
		err := f.FSM.Event(f.ctx, "register_send_200")
		if err != nil {
			fmt.Printf("FSM: curr state: %s, error sending REGISTER 200 OK: %v\n", f.FSM.Current(), err.Error())
			return err
		}
	}

	if err := f.BeforeHook(sipMsg); err == errCSeqRetry {
		// TODO might need to care about retries in the future, cant see a reason to why for now
		fmt.Println("duplicate cseq header found for fsm: ", f.String())
	} else if err != nil {
		return err
	}

	f.registerCallId = sipMsg.GetCallID()

	res, err := sipMsg.NewResponse(200)
	if err != nil {
		fmt.Println("error getting response: ", err.Error())
		return err
	}

	var buf bytes.Buffer
	res.Append(&buf)

	if err := send(buf.Bytes()); err != nil {
		if fsmErr := f.FSM.Event(f.ctx, "register_failed"); fsmErr != nil {
			err = fmt.Errorf("%w; FSM: error in RegisterSendOk %v: ", err, fsmErr)
		}

		return err
	}

	fmt.Println("REGISTER response: ", res)

	return nil
}

func (f *SipFsm) SendTrying(sipMsg *adapters.SipMsg, send ports.SendResponseCallback) error {
	// first update fsm to make sure our state is valid
	err := f.FSM.Event(f.ctx, "invite_send_100")
	if err != nil {
		// TODO if it fails we should set the fsm state
		fmt.Println("FSM: error sending trying: ", err.Error())
		return err
	}

	if err := f.BeforeHook(sipMsg); err == errCSeqRetry {
		// TODO might need to care about cseq retries in the future, cant see a reason to why for now
		fmt.Println("duplicate cseq header found for fsm: ", f.String())
	} else if err != nil {
		return err
	}

	// TODO pretty sure this is adding an sdp to the response
	res, err := sipMsg.NewResponse(100)
	if err != nil {
		fmt.Println("error getting response: ", err.Error())
		return err
	}

	var b bytes.Buffer
	res.Append(&b)

	if err = send(b.Bytes()); err != nil {
		// TODO if it fails we should set the fsm state
		fmt.Println("FSM: error sending trying: ", err.Error())
		return err
	}

	fmt.Println("sent trying response: ", b.String())

	return f.SendRinging(sipMsg, send)
}

func (f *SipFsm) SendRinging(sipMsg *adapters.SipMsg, send ports.SendResponseCallback) error {
	err := f.FSM.Event(f.ctx, "invite_send_180")
	if err != nil {
		fmt.Println("FSM: error sending ringing: ", err.Error())
		return err
	}

	// TODO pretty sure this is adding an sdp to the response
	response, err := sipMsg.NewResponse(180)
	if err != nil {
		fmt.Println("error getting response: ", err.Error())
		return err
	}

	var b bytes.Buffer
	response.Append(&b)

	if err = send(b.Bytes()); err != nil {
		fmt.Println("FSM: error sending ringing: ", err.Error())
		return err
	}

	fmt.Println("sent ringing response: ", b.String())

	return f.SendSessionProgress(sipMsg, send)
}

// TODO we will need to fix the SDP body for this to actually get a ringing tone
func (f *SipFsm) SendSessionProgress(sipMsg *adapters.SipMsg, send ports.SendResponseCallback) error {
	err := f.FSM.Event(f.ctx, "invite_send_183")
	if err != nil {
		fmt.Println("FSM: error sending session progress: ", err.Error())
		return err
	}

	response, err := sipMsg.NewResponse(183)
	if err != nil {
		fmt.Println("error getting response: ", err.Error())
		return err
	}

	var b bytes.Buffer
	response.Append(&b)

	if err = send(b.Bytes()); err != nil {
		fmt.Println("FSM: error sending session progress: ", err.Error())
		return err
	}

	fmt.Println("sent session progress response: ", b.String())

	return f.SendOk(sipMsg, send)
}

func (f *SipFsm) SendOk(sipMsg *adapters.SipMsg, send ports.SendResponseCallback) error {
	err := f.FSM.Event(f.ctx, "invite_send_200")
	if err != nil {
		// TODO should make sure we send an error response to the client
		fmt.Println("FSM: error sending ok: ", err.Error())
		return err
	}

	fmt.Println("before hook")
	if err := f.BeforeHook(sipMsg); err == errCSeqRetry {
		// TODO might need to care about retries in the future, cant see a reason to why for now
		fmt.Println("duplicate cseq header found for fsm: ", f.String())
	} else if err != nil {
		return err
	}

	response, err := sipMsg.NewResponse(200)
	if err != nil {
		fmt.Println("error getting response: ", err.Error())
		return err
	}

	var b bytes.Buffer
	response.Append(&b)

	if err = send(b.Bytes()); err != nil {
		fmt.Println("FSM: error sending ringing: ", err.Error())
		return err
	}

	fmt.Println("sent ok response: ", b.String())

	go func() {
		if err := sendWav(sipMsg); err != nil {
			fmt.Println("error sending wav: ", err.Error())
			// TODO need to set the fsm state to call ended or something similar
			return
		}

		fmt.Println("done sending audio")
	}()

	return nil
}

func (f *SipFsm) RecvAck() error {
	// TODO  weird things happening from sip client when it does not like sdp response, so it immediately sends a BYE request, but also sends an ACK ? Yes, after testing, it is sending the ACK unrelated to the BYE, so probably a race condition on the clients side
	// TODO should check the current state, if we recieve an ACK from a BYE or from a 200 response
	err := f.FSM.Event(f.ctx, "invite_recv_ack")
	if err != nil {
		fmt.Println("FSM: error recieving ack: ", err.Error())
		return err
	}

	return nil
}

func (f *SipFsm) RecvBye(sipMsg *adapters.SipMsg, send ports.SendResponseCallback) error {
	err := f.FSM.Event(f.ctx, "recv_bye")
	if err != nil {
		fmt.Println("FSM: error recieving bye. Sending 481: ", err.Error())
		response, err := sipMsg.NewResponse(481)
		if err != nil {
			fmt.Println("error getting response: ", err.Error())
			return err
		}

		var b bytes.Buffer
		response.Append(&b)

		send(b.Bytes()) // we dont care about the error here, we are already in a bad state

		return err
	}

	response, err := sipMsg.NewResponse(200)
	if err != nil {
		fmt.Println("error getting response: ", err.Error())
		return err
	}

	var b bytes.Buffer
	response.Append(&b)

	if err := send(b.Bytes()); err != nil {
		return err
	}

	fmt.Println("sent 200 OK in response to BYE")

	return nil
}

func (f *SipFsm) AddCallId(callId string) error {
	if callId == "" {
		return fmt.Errorf("FSM: call id is empty")
	}

	for _, id := range f.callIds {
		if id == callId {
			return fmt.Errorf("FSM: call id already exists")
		}
	}

	f.callIds = append(f.callIds, callId)

	return nil
}

func (f *SipFsm) String() string {
	callIds := ""
	for _, id := range f.callIds {
		callIds += id + " "
	}

	return fmt.Sprintf("FSM state: %s. Key: %s. Addr: %s. CallIds: %s", f.FSM.Current(), f.key, f.addr, callIds)
}

func (f *SipFsm) RemoveCallId(callId string) error {
	if callId == "" {
		return fmt.Errorf("FSM: call id is empty")
	}

	for i, id := range f.callIds {
		if id == callId {
			f.callIds = append(f.callIds[:i], f.callIds[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("FSM: call id not found")
}
