package domain

import (
	"context"
	"fmt"
	"sync"

	"sip_and_rip/adapters"
)

var errMissingKey = fmt.Errorf("key is empty")
var errMissingContact = fmt.Errorf("contact header is empty")
var errFsmNotFound = fmt.Errorf("fsm not found")

// FsmCache - a cache that stores fsm's by their unique key
type FsmCache struct {
	sync.RWMutex
	m map[string]*SipFsm
}

// NewFsmCache - creates a new cache that stores fsm's by their key
func NewFsmCache() *FsmCache {
	m := make(map[string]*SipFsm)

	return &FsmCache{
		m: m,
	}
}

// TODO use ports not adapters when defining the input
// NewSipFsm - creates a new sipFsm and stores it in the cache
func (f *FsmCache) NewSipFsm(ctx context.Context, sipMsg *adapters.SipMsg, addr string) (*SipFsm, error) {
	key, err := f.GetKey(sipMsg)
	if err != nil {
		return nil, err
	}

	fsm, err := NewSipFsm(ctx, key, addr)
	if err != nil {
		return nil, err
	}

	f.m[key] = fsm

	return fsm, nil
}

func (f *FsmCache) Delete(sipMsg *adapters.SipMsg) (string, error) {
	key, err := f.GetKey(sipMsg)
	if err != nil {
		return key, err
	}

	return key, f.DeleteWithKey(key)
}

func (f *FsmCache) DeleteWithKey(key string) error {
	f.Lock()
	defer f.Unlock()

	if _, ok := f.m[key]; !ok {
		return fmt.Errorf("cant delete fsm, key %s not found", key)
	}

	delete(f.m, key)

	return nil
}

func (f *FsmCache) DeleteWithUrn(urn string) error {
	f.Lock()
	defer f.Unlock()

	if _, ok := f.m[urn]; !ok {
		return fmt.Errorf("cant delete using urn, key %s not found", urn)
	}

	delete(f.m, urn)

	return nil
}

func (f *FsmCache) Get(sipMsg *adapters.SipMsg) (*SipFsm, error) {
	key, err := f.GetKey(sipMsg)
	if err == errMissingKey || err == errMissingContact {
		// lets check by the callId instead of the key
		for _, v := range f.m {
			if v.registerCallId == sipMsg.GetCallID() {
				return v, nil
			}

			for _, callId := range v.callIds {
				if callId == sipMsg.GetCallID() {
					return v, nil
				}
			}
		}
	} else if err != nil {
		return nil, err
	}

	fsm, err := f.GetWithKey(key)
	if err != nil {
		return nil, err
	}

	return fsm, nil
}

func (f *FsmCache) GetWithKey(key string) (*SipFsm, error) {
	fsm, ok := f.m[key]
	if !ok || fsm == nil {
		return nil, errFsmNotFound
	}

	return fsm, nil
}

func (f *FsmCache) Set(sipMsg *adapters.SipMsg, value *SipFsm) error {
	key, err := f.GetKey(sipMsg)
	if err != nil {
		return err
	}

	return f.SetWithKey(key, value)
}

func (f *FsmCache) SetWithKey(key string, value *SipFsm) error {
	f.Lock()
	defer f.Unlock()
	f.m[key] = value

	return nil
}

func (*FsmCache) GetKey(sipMsg *adapters.SipMsg) (string, error) {
	if sipMsg.GetContact() == nil {
		return "", errMissingContact
	}

	key := sipMsg.GetContact().String()

	urn, err := sipMsg.GetUrn()
	if err != nil {
		return "", err
	}

	if urn != "" {
		key = urn
	}

	return key, nil
}

func (f *FsmCache) Len() int {
	f.RLock()
	defer f.RUnlock()

	return len(f.m)
}

func (f *FsmCache) CloseFsm(sipMsg *adapters.SipMsg) error {
	fsm, err := f.Get(sipMsg)
	if err != nil {
		return err
	}

	if err := fsm.Close(); err != nil {
		fmt.Println("Error closing fsm: ", err)
		return err
	}

	// delete the key in the fsm map
	if err := f.DeleteWithKey(fsm.key); err != nil {
		fmt.Printf("Error deleting FSM for key %s from %s: %v\n", fsm.key, fsm.addr, err)
		return err
	}

	fmt.Printf("fsm new length: %d; deleted fsm for %s\n", f.Len(), fsm.key)

	return nil
}

func (f *FsmCache) String() string {
	out := "--------- FSM CACHE CONTENTS:---------------\n"

	if len(f.m) == 0 {
		out += "empty\n"
	}

	for _, fsm := range f.m {
		out += fmt.Sprintf("%s\n", fsm.String())
	}
	out += "--------------------"

	return out
}
