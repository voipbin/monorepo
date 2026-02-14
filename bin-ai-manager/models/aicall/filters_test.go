package aicall

import (
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
				AIID:          uuid.Must(uuid.NewV4()),
				AIEngineType:  ai.EngineTypeNone,
				AIEngineModel: ai.EngineModelOpenaiGPT4O,
				ActiveflowID:  uuid.Must(uuid.NewV4()),
				ReferenceType: ReferenceTypeCall,
				ReferenceID:   uuid.Must(uuid.NewV4()),
				ConfbridgeID:  uuid.Must(uuid.NewV4()),
				PipecatcallID: uuid.Must(uuid.NewV4()),
				Status:        StatusProgressing,
				Gender:        GenderFemale,
				Language:      "en-US",
				Deleted:       false,
			},
		},
		{
			name: "creates_field_struct_with_empty_fields",
			fs: &FieldStruct{
				CustomerID:    uuid.Nil,
				AIID:          uuid.Nil,
				AIEngineType:  "",
				AIEngineModel: "",
				ActiveflowID:  uuid.Nil,
				ReferenceType: ReferenceTypeNone,
				ReferenceID:   uuid.Nil,
				ConfbridgeID:  uuid.Nil,
				PipecatcallID: uuid.Nil,
				Status:        "",
				Gender:        GenderNone,
				Language:      "",
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
