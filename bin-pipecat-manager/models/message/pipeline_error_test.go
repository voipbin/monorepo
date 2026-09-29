package message

import (
	"encoding/json"
	"testing"

	"monorepo/bin-common-handler/models/eventtopic"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

// PipelineErrorEvent has no top-level id; the pointer type must satisfy SubscriptionIdentifier or
// the routing key would degrade to the `-` placeholder.
var _ eventtopic.SubscriptionIdentifier = (*PipelineErrorEvent)(nil)

func Test_PipelineErrorEvent_JSON(t *testing.T) {
	evt := &PipelineErrorEvent{
		CustomerID:               uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000000"),
		PipecatcallID:            uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001"),
		PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
		PipecatcallReferenceID:   uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000002"),
		ActiveflowID:             uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000003"),
		Category:                 ErrorCategoryAuthentication,
		Fatal:                    false,
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expect := `{"customer_id":"aaaaaaaa-0000-0000-0000-000000000000","pipecatcall_id":"aaaaaaaa-0000-0000-0000-000000000001","pipecatcall_reference_type":"ai_call","pipecatcall_reference_id":"aaaaaaaa-0000-0000-0000-000000000002","activeflow_id":"aaaaaaaa-0000-0000-0000-000000000003","category":"authentication","fatal":false}`
	if string(data) != expect {
		t.Errorf("Wrong match.\nexpect: %s\ngot:    %s", expect, string(data))
	}

	var got PipelineErrorEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if got != *evt {
		t.Errorf("Wrong match. expect: %v, got: %v", *evt, got)
	}
}

func Test_PipelineErrorEvent_EventSubscriptionID(t *testing.T) {
	evt := &PipelineErrorEvent{PipecatcallID: uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001")}
	if res := evt.EventSubscriptionID(); res != "aaaaaaaa-0000-0000-0000-000000000001" {
		t.Errorf("Wrong match. expect: pipecatcall id, got: %s", res)
	}
}

func Test_ErrorCategories(t *testing.T) {
	tests := map[ErrorCategory]string{
		ErrorCategoryAuthentication: "authentication",
		ErrorCategoryRateLimited:    "rate_limited",
		ErrorCategoryTimeout:        "timeout",
		ErrorCategoryUnknown:        "unknown",
		ErrorCategoryFunctionCall:   "function_call",
		ErrorCategoryInternal:       "internal",
	}
	for c, want := range tests {
		if string(c) != want {
			t.Errorf("Wrong match. expect: %s, got: %s", want, c)
		}
	}
}
