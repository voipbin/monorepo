package aicall

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"monorepo/bin-ai-manager/models/ai"
)

func TestFieldStruct(t *testing.T) {
	tests := []struct {
		name string
		fs   *FieldStruct
	}{
		{
			name: "creates_field_struct_with_all_fields",
			fs: &FieldStruct{
				CustomerID:    uuid.Must(uuid.NewV4()),
				AssistanceType: AssistanceTypeAI,
				AssistanceID:   uuid.Must(uuid.NewV4()),
				AIEngineModel: ai.EngineModelOpenaiGPT5,
				ActiveflowID:  uuid.Must(uuid.NewV4()),
				ReferenceType: ReferenceTypeCall,
				ReferenceID:   uuid.Must(uuid.NewV4()),
				ConfbridgeID:  uuid.Must(uuid.NewV4()),
				PipecatcallID: uuid.Must(uuid.NewV4()),
				Status:        StatusProgressing,
				STTLanguage:   "en-US",
				Deleted:       false,
			},
		},
		{
			name: "creates_field_struct_with_empty_fields",
			fs: &FieldStruct{
				CustomerID:    uuid.Nil,
				AssistanceType: AssistanceTypeAI,
				AssistanceID:   uuid.Nil,
				AIEngineModel: "",
				ActiveflowID:  uuid.Nil,
				ReferenceType: ReferenceTypeNone,
				ReferenceID:   uuid.Nil,
				ConfbridgeID:  uuid.Nil,
				PipecatcallID: uuid.Nil,
				Status:        "",
				STTLanguage:   "",
				Deleted:       true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.fs == nil {
				t.Error("FieldStruct should not be nil")
			}
		})
	}
}

// Test_FieldStruct_ListenCallID pins that listen_call_id is a filterable field.
// Without the struct tag, ConvertFilters drops it and stopListenByCallID's
// AIcallList silently returns every contact_case AIcall on the platform instead
// of the ones listening to the hung-up call.
func Test_FieldStruct_ListenCallID(t *testing.T) {
	field, ok := reflect.TypeOf(FieldStruct{}).FieldByName("ListenCallID")
	if !ok {
		t.Fatalf("FieldStruct has no ListenCallID member")
	}

	if got := field.Tag.Get("filter"); got != "listen_call_id" {
		t.Errorf("filter tag mismatch. expected: %q, got: %q", "listen_call_id", got)
	}
}
