package message

import (
	"encoding/json"
	"testing"

	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

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
