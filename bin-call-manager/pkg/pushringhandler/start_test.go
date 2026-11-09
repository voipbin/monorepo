package pushringhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"go.uber.org/mock/gomock"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	pmnotification "monorepo/bin-push-manager/models/notification"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/pushring"
)

func Test_Start(t *testing.T) {
	e := newEnv(t)

	res := e.start()
	if !res.Started || res.SessionID == uuid.Nil || !res.SIPRegistered {
		t.Fatalf("Wrong result: %+v", res)
	}

	s := e.session(res.SessionID)
	if s.CustomerID != tCustomerID || s.AgentID != tAgentID || s.ExtensionID != tExtensionID || s.ConfbridgeID != tConfbridge ||
		s.MasterCallID != tMasterCall || s.FlowID != tFlowID || s.State != pushring.StateRinging || !s.SIPRegistered ||
		len(s.Tickets) != 2 {
		t.Errorf("Wrong session: %+v", s)
	}
	if !s.ExpiresAt.Equal(e.now.Add(pushring.Window)) {
		t.Errorf("The window must start at the start. expect: %v, got: %v", e.now.Add(pushring.Window), s.ExpiresAt)
	}

	// the confbridge is marked, and the two expire requests are scheduled at 60 s and 90 s
	if len(e.cb.Flags) != 1 || e.cb.Flags[0] != confbridge.FlagPushRing {
		t.Errorf("The confbridge must be marked. flags: %v", e.cb.Flags)
	}
	if e.expires[60000] != 1 || e.expires[90000] != 1 || len(e.expires) != 2 {
		t.Errorf("Wrong expire requests: %v", e.expires)
	}

	// one incoming notification per device, each with its own ticket; the cache holds only the hash
	incoming := e.sendsOf(pmnotification.KindCallIncoming)
	if len(incoming) != 2 {
		t.Fatalf("Wrong notification count: %d", len(incoming))
	}
	ticketA, ticketB := e.ticketOf(tDeviceA), e.ticketOf(tDeviceB)
	if ticketA == ticketB || len(ticketA) != ticketLen || len(ticketB) != ticketLen {
		t.Errorf("Wrong tickets: %s %s", ticketA, ticketB)
	}
	if s.Tickets[hashTicket(ticketA)] != tDeviceA || s.Tickets[hashTicket(ticketB)] != tDeviceB {
		t.Errorf("The tickets must be stored by hash and bound to their device")
	}
	for _, n := range incoming {
		if n.data["call_id"] != tMasterCall.String() || n.data["caller_display"] != "+15550100" ||
			n.data["cancel_key"] != res.SessionID.String() || n.data["expires_at"] != e.now.Add(pushring.Window).UTC().Format(time.RFC3339) {
			t.Errorf("Wrong payload: %v", n.data)
		}
		if len(n.data) != 5 {
			t.Errorf("The payload carries only the five keys of the contract: %v", n.data)
		}
	}
	for _, k := range e.mr.Keys() {
		if k == "call:pushticket:"+ticketA || k == "call:pushticket:"+ticketB {
			t.Errorf("A key must never hold the plain ticket: %s", k)
		}
	}
}

func Test_Start_notStarted(t *testing.T) {
	tests := []struct {
		name  string
		setup func(e *env)
		dest  *commonaddress.Address
	}{
		{"no agent", func(e *env) { e.agentErr = errors.New("not found") }, nil},
		{"agent is nil", func(e *env) { e.agent = nil }, nil},
		{"agent of another customer", func(e *env) { e.agent.CustomerID = uuid.Must(uuid.NewV4()) }, nil},
		{"agent is deleted", func(e *env) {
			tm := time.Now()
			e.agent.TMDelete = &tm
		}, nil},
		{"agent without devices", func(e *env) { e.devices = nil }, nil},
		{"device list error", func(e *env) { e.deviceErr = errors.New("push manager down") }, nil},
		{"confbridge could not be marked", func(e *env) { e.flagErr = errors.New("db down") }, nil},
		{"destination is not an extension", func(e *env) {}, &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550199"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			tt.setup(e)

			dest := commonaddress.Address{Type: commonaddress.TypeExtension, Target: tExtensionID.String(), TargetName: "2000"}
			if tt.dest != nil {
				dest = *tt.dest
			}

			res, err := e.h.Start(context.Background(), tCustomerID, dest, tConfbridge, tMasterCall, tFlowID, commonaddress.Address{})
			if err != nil {
				t.Fatalf("A session that did not start is not an error: %v", err)
			}
			e.h.wg.Wait()

			if res.Started || res.SessionID != uuid.Nil {
				t.Errorf("Wrong result: %+v", res)
			}
			if n := len(e.sendsOf(pmnotification.KindCallIncoming)); n != 0 {
				t.Errorf("Nothing may be sent. count: %d", n)
			}
			if len(e.expires) != 0 {
				t.Errorf("No expire request may be scheduled: %v", e.expires)
			}
			// the connect continues as before: no hangup, nothing left open
			if len(e.hangupIDs()) != 0 {
				t.Errorf("Nothing may be hung up: %v", e.hangupIDs())
			}
		})
	}
}

// A FlagAdd failure removes the keys that were already created.
func Test_Start_flagFailureRemovesTheKeys(t *testing.T) {
	e := newEnv(t)
	e.flagErr = errors.New("db down")

	e.start()

	if _, err := e.cache.PushRingSessionIDGet(context.Background(), pushring.IndexConfbridge, tConfbridge); err != nil {
		t.Fatalf("the lookup key exists until the ttl: %v", err)
	}
	for _, k := range e.mr.Keys() {
		if len(k) > len("call:pushticket:") && k[:len("call:pushticket:")] == "call:pushticket:" {
			t.Errorf("the tickets must be deleted: %s", k)
		}
	}

	// the closed session refuses every join
	sid, _ := e.cache.PushRingSessionIDGet(context.Background(), pushring.IndexConfbridge, tConfbridge)
	if s := e.session(sid); s.State != pushring.StateClosed {
		t.Errorf("the session must be closed. state: %s", s.State)
	}
}

func Test_Start_deadline(t *testing.T) {
	e := newEnv(t)
	e.h.startTimeout = 50 * time.Millisecond
	e.agentBlock = true

	// the agent lookup does not answer inside the start deadline: no session, and the answer comes at the deadline.
	start := time.Now()
	res, err := e.h.Start(context.Background(), tCustomerID,
		commonaddress.Address{Type: commonaddress.TypeExtension, Target: tExtensionID.String(), TargetName: "2000"},
		tConfbridge, tMasterCall, tFlowID, commonaddress.Address{})
	if err != nil || res.Started {
		t.Fatalf("Wrong result: %+v, %v", res, err)
	}
	if time.Since(start) > time.Second {
		t.Errorf("The start must answer at its deadline. took: %v", time.Since(start))
	}
	if len(e.mr.Keys()) != 0 {
		t.Errorf("No session may be created: %v", e.mr.Keys())
	}
	if len(e.cb.Flags) != 0 {
		t.Errorf("The confbridge must not be marked: %v", e.cb.Flags)
	}
}

func Test_Start_sipRegistered(t *testing.T) {
	e := newEnv(t)
	e.contacts = nil

	res := e.start()
	if !res.Started || res.SIPRegistered {
		t.Errorf("Wrong result: %+v", res)
	}
	if s := e.session(res.SessionID); s.SIPRegistered {
		t.Errorf("The session must know that no contact is registered")
	}
}

func Test_getExtensionID(t *testing.T) {
	ctx := context.Background()

	byNumber := func(target, targetName string) *startRequest {
		return &startRequest{
			customerID:  tCustomerID,
			destination: commonaddress.Address{Type: commonaddress.TypeExtension, Target: target, TargetName: targetName},
		}
	}

	t.Run("a canonical id is used as it is, without a lookup", func(t *testing.T) {
		e := newEnv(t)
		got, err := e.h.getExtensionID(ctx, byNumber(tExtensionID.String(), ""))
		if err != nil || got != tExtensionID {
			t.Errorf("Wrong match. got: %v, %v", got, err)
		}
	})

	tests := []struct {
		name string

		target     string
		targetName string

		responseList []rmextension.Extension
		responseErr  error

		expectNumber string
		expectID     uuid.UUID
		expectErr    bool
	}{
		{
			name:         "the number is in the target name",
			target:       "not-an-id",
			targetName:   "2000",
			responseList: []rmextension.Extension{{Identity: commonidentity.Identity{ID: tExtensionID}}},
			expectNumber: "2000",
			expectID:     tExtensionID,
		},
		{
			name:         "the number is in the target",
			target:       "2001",
			responseList: []rmextension.Extension{{Identity: commonidentity.Identity{ID: tExtensionID}}},
			expectNumber: "2001",
			expectID:     tExtensionID,
		},
		{
			name:         "unknown number",
			target:       "2002",
			expectNumber: "2002",
			expectErr:    true,
		},
		{
			name:         "lookup error",
			target:       "2003",
			responseErr:  errors.New("registrar down"),
			expectNumber: "2003",
			expectErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)

			filters := map[rmextension.Field]any{
				rmextension.FieldCustomerID: tCustomerID,
				rmextension.FieldExtension:  tt.expectNumber,
				rmextension.FieldDeleted:    false,
			}
			e.req.EXPECT().RegistrarV1ExtensionList(gomock.Any(), "", uint64(1), filters).Return(tt.responseList, tt.responseErr)

			got, err := e.h.getExtensionID(ctx, byNumber(tt.target, tt.targetName))
			if (err != nil) != tt.expectErr || got != tt.expectID {
				t.Errorf("Wrong match. got: %v, %v", got, err)
			}
		})
	}
}

func Test_sipRegistered(t *testing.T) {
	ctx := context.Background()

	req := func(target, targetName string) *startRequest {
		return &startRequest{
			customerID:  tCustomerID,
			destination: commonaddress.Address{Type: commonaddress.TypeExtension, Target: target, TargetName: targetName},
		}
	}

	t.Run("the number comes from the extension when the destination has no target name", func(t *testing.T) {
		e := newEnv(t)
		e.req.EXPECT().RegistrarV1ExtensionGet(gomock.Any(), tExtensionID).Return(&rmextension.Extension{Extension: "2000"}, nil)

		if !e.h.sipRegistered(ctx, req(tExtensionID.String(), "")) {
			t.Errorf("expected registered")
		}
	})

	t.Run("an extension lookup error counts as registered", func(t *testing.T) {
		e := newEnv(t)
		e.contacts = nil
		e.req.EXPECT().RegistrarV1ExtensionGet(gomock.Any(), tExtensionID).Return(nil, errors.New("registrar down"))

		if !e.h.sipRegistered(ctx, req(tExtensionID.String(), "")) {
			t.Errorf("an unknown state must count as registered")
		}
	})

	t.Run("a number as the target", func(t *testing.T) {
		e := newEnv(t)
		e.contacts = nil

		if e.h.sipRegistered(ctx, req("2000", "")) {
			t.Errorf("no contact: expected not registered")
		}
	})

	t.Run("a contact list error counts as registered", func(t *testing.T) {
		e := newEnv(t)
		e.contacts = nil
		e.contactErr = errors.New("registrar down")

		if !e.h.sipRegistered(ctx, req("2000", "2000")) {
			t.Errorf("an unknown state must count as registered")
		}
	})
}

// Every notification failed explicitly.
func Test_Start_pushFailed(t *testing.T) {
	tests := []struct {
		name string

		contacts   bool
		sendResult func(kind pmnotification.Kind, deviceID uuid.UUID) (*pmnotification.SendResult, error)

		expectState  pushring.State
		expectHangup bool
	}{
		{
			name:     "every device failed and a SIP contact is registered: the gate stays open",
			contacts: true,
			sendResult: func(pmnotification.Kind, uuid.UUID) (*pmnotification.SendResult, error) {
				return &pmnotification.SendResult{DeviceCount: 1, SentCount: 0}, nil
			},
			expectState: pushring.StatePushFailed,
		},
		{
			name: "every device failed and no SIP contact: the call ends with no answer",
			sendResult: func(pmnotification.Kind, uuid.UUID) (*pmnotification.SendResult, error) {
				return &pmnotification.SendResult{DeviceCount: 1, SentCount: 0}, nil
			},
			expectState:  pushring.StatePushFailed,
			expectHangup: true,
		},
		{
			name: "an RPC error is unknown, not a failure",
			sendResult: func(pmnotification.Kind, uuid.UUID) (*pmnotification.SendResult, error) {
				return nil, errors.New("timeout")
			},
			expectState: pushring.StateRinging,
		},
		{
			name: "one device accepted: not a failure",
			sendResult: func(kind pmnotification.Kind, deviceID uuid.UUID) (*pmnotification.SendResult, error) {
				if deviceID == tDeviceA {
					return &pmnotification.SendResult{DeviceCount: 1, SentCount: 1}, nil
				}
				return &pmnotification.SendResult{DeviceCount: 1, SentCount: 0}, nil
			},
			expectState: pushring.StateRinging,
		},
		{
			name: "one device failed and the other is unknown: not a failure",
			sendResult: func(kind pmnotification.Kind, deviceID uuid.UUID) (*pmnotification.SendResult, error) {
				if deviceID == tDeviceA {
					return nil, errors.New("timeout")
				}
				return &pmnotification.SendResult{DeviceCount: 1, SentCount: 0}, nil
			},
			expectState: pushring.StateRinging,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			if !tt.contacts {
				e.contacts = nil
			}
			e.sendResult = tt.sendResult

			res := e.start()
			s := e.session(res.SessionID)
			if s.State != tt.expectState {
				t.Errorf("Wrong state. expect: %s, got: %s", tt.expectState, s.State)
			}

			hang := e.hangupIDs()
			if tt.expectHangup {
				if hang[tMasterCall] != call.HangupReasonNoanswer || len(hang) != 1 {
					t.Errorf("The master call must end with no answer. got: %v", hang)
				}
			} else if len(hang) != 0 {
				t.Errorf("Nothing may be hung up. got: %v", hang)
			}

			if tt.expectState == pushring.StatePushFailed {
				for _, k := range e.mr.Keys() {
					if len(k) > len("call:pushticket:") && k[:len("call:pushticket:")] == "call:pushticket:" {
						t.Errorf("The tickets must be deleted: %s", k)
					}
				}
			}
		})
	}
}
