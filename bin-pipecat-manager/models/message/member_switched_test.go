package message

import (
	"encoding/json"
	"testing"

	"monorepo/bin-common-handler/models/eventtopic"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

// MemberSwitchedEvent overrides the subscription address of the global topic exchange
// (VOIP-1404/1405). The assertion pins the POINTER receiver: notifyhandler asserts on the dynamic
// type of the event data, which is always a pointer, so a value receiver would silently never be
// picked up.
var _ eventtopic.SubscriptionIdentifier = (*MemberSwitchedEvent)(nil)

func TestMemberSwitchedEvent_JSONRoundTrip(t *testing.T) {
	evt := MemberSwitchedEvent{
		CustomerID:               uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000000"),
		PipecatcallID:            uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001"),
		PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000002"),
		TransitionFunctionName:   "transfer_to_sales",
		FromMember: MemberInfo{
			ID:          uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000001"),
			Name:        "Reception",
			EngineModel: "openai.gpt-5",
			TTSType:     "cartesia",
			TTSVoiceID:  "voice-123",
			STTType:     "deepgram",
		},
		ToMember: MemberInfo{
			ID:          uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000002"),
			Name:        "Sales Agent",
			EngineModel: "openai.gpt-5",
			TTSType:     "elevenlabs",
			TTSVoiceID:  "voice-456",
			STTType:     "deepgram",
		},
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var got MemberSwitchedEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if got.CustomerID != evt.CustomerID {
		t.Errorf("CustomerID = %q, want %q", got.CustomerID, evt.CustomerID)
	}
	if got.TransitionFunctionName != evt.TransitionFunctionName {
		t.Errorf("TransitionFunctionName = %q, want %q", got.TransitionFunctionName, evt.TransitionFunctionName)
	}
	if got.FromMember.Name != evt.FromMember.Name {
		t.Errorf("FromMember.Name = %q, want %q", got.FromMember.Name, evt.FromMember.Name)
	}
	if got.ToMember.Name != evt.ToMember.Name {
		t.Errorf("ToMember.Name = %q, want %q", got.ToMember.Name, evt.ToMember.Name)
	}
}

func TestMemberSwitchedEvent_EventSubscriptionID(t *testing.T) {
	pipecatcallID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001")

	tests := []struct {
		name   string
		evt    *MemberSwitchedEvent
		expect string
	}{
		{
			name: "normal",
			evt: &MemberSwitchedEvent{
				CustomerID:               uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000000"),
				PipecatcallID:            pipecatcallID,
				PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000002"),
				TransitionFunctionName:   "transfer_to_sales",
				FromMember:               MemberInfo{ID: uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000001")},
				ToMember:                 MemberInfo{ID: uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000002")},
			},
			expect: pipecatcallID.String(),
		},
		{
			name:   "empty pipecatcall id",
			evt:    &MemberSwitchedEvent{},
			expect: uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.evt.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestMemberSwitchedEvent_EventSubscriptionIDHasNoOwnID pins the failure mode the override
// prevents. Unlike Message, this payload has NO top-level `id` at all — the member ids live
// nested under from_member/to_member — so without the override the default JSON fallback resolves
// to the `-` placeholder and the event becomes unreachable by any instance binding. The address
// must be the pipecatcall-id and must not coincide with either member id.
func TestMemberSwitchedEvent_EventSubscriptionIDHasNoOwnID(t *testing.T) {
	evt := &MemberSwitchedEvent{
		PipecatcallID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001"),
		FromMember:    MemberInfo{ID: uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000001")},
		ToMember:      MemberInfo{ID: uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000002")},
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	d := struct {
		ID string `json:"id"`
	}{}
	if errUnmarshal := json.Unmarshal(data, &d); errUnmarshal != nil {
		t.Fatalf("Unmarshal failed: %v", errUnmarshal)
	}
	if d.ID != "" {
		t.Fatalf("MemberSwitchedEvent must not carry a top-level id. got: %s", d.ID)
	}

	if res := evt.EventSubscriptionID(); res != evt.PipecatcallID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", evt.PipecatcallID.String(), res)
	}
	if res := evt.EventSubscriptionID(); res == evt.FromMember.ID.String() || res == evt.ToMember.ID.String() {
		t.Errorf("Subscription address must not be a member id. got: %s", res)
	}
}
