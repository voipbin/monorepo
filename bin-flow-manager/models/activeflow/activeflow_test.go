package activeflow

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-flow-manager/models/action"
)

func TestActiveflowStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	flowID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	onCompleteFlowID := uuid.Must(uuid.NewV4())

	a := Activeflow{
		FlowID:            flowID,
		Status:            StatusRunning,
		ReferenceType:     ReferenceTypeCall,
		ReferenceID:       referenceID,
		OnCompleteFlowID:  onCompleteFlowID,
		ExecuteCount:      5,
	}
	a.ID = id
	a.CustomerID = customerID

	if a.ID != id {
		t.Errorf("Activeflow.ID = %v, expected %v", a.ID, id)
	}
	if a.CustomerID != customerID {
		t.Errorf("Activeflow.CustomerID = %v, expected %v", a.CustomerID, customerID)
	}
	if a.FlowID != flowID {
		t.Errorf("Activeflow.FlowID = %v, expected %v", a.FlowID, flowID)
	}
	if a.Status != StatusRunning {
		t.Errorf("Activeflow.Status = %v, expected %v", a.Status, StatusRunning)
	}
	if a.ReferenceType != ReferenceTypeCall {
		t.Errorf("Activeflow.ReferenceType = %v, expected %v", a.ReferenceType, ReferenceTypeCall)
	}
	if a.ReferenceID != referenceID {
		t.Errorf("Activeflow.ReferenceID = %v, expected %v", a.ReferenceID, referenceID)
	}
	if a.OnCompleteFlowID != onCompleteFlowID {
		t.Errorf("Activeflow.OnCompleteFlowID = %v, expected %v", a.OnCompleteFlowID, onCompleteFlowID)
	}
	if a.ExecuteCount != 5 {
		t.Errorf("Activeflow.ExecuteCount = %v, expected %v", a.ExecuteCount, 5)
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_none", StatusNone, ""},
		{"status_running", StatusRunning, "running"},
		{"status_ended", StatusEnded, "ended"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{"reference_type_none", ReferenceTypeNone, ""},
		{"reference_type_ai", ReferenceTypeAI, "ai"},
		{"reference_type_api", ReferenceTypeAPI, "api"},
		{"reference_type_call", ReferenceTypeCall, "call"},
		{"reference_type_campaign", ReferenceTypeCampaign, "campaign"},
		{"reference_type_conversation", ReferenceTypeConversation, "conversation"},
		{"reference_type_transcribe", ReferenceTypeTranscribe, "transcribe"},
		{"reference_type_recording", ReferenceTypeRecording, "recording"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestMapActionMediaTypeByReferenceType(t *testing.T) {
	tests := []struct {
		name          string
		referenceType ReferenceType
		expectedMedia action.MediaType
	}{
		{"none", ReferenceTypeNone, action.MediaTypeNone},
		{"call", ReferenceTypeCall, action.MediaTypeRealTimeCommunication},
		{"ai", ReferenceTypeAI, action.MediaTypeNonRealTimeCommunication},
		{"api", ReferenceTypeAPI, action.MediaTypeNonRealTimeCommunication},
		{"conversation", ReferenceTypeConversation, action.MediaTypeNonRealTimeCommunication},
		{"recording", ReferenceTypeRecording, action.MediaTypeNonRealTimeCommunication},
		{"transcribe", ReferenceTypeTranscribe, action.MediaTypeNonRealTimeCommunication},
		{"campaign", ReferenceTypeCampaign, action.MediaTypeNonRealTimeCommunication},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mediaType, exists := MapActionMediaTypeByReferenceType[tt.referenceType]
			if !exists {
				t.Errorf("MapActionMediaTypeByReferenceType[%s] does not exist", tt.referenceType)
				return
			}
			if mediaType != tt.expectedMedia {
				t.Errorf("MapActionMediaTypeByReferenceType[%s] = %v, expected %v", tt.referenceType, mediaType, tt.expectedMedia)
			}
		})
	}
}

func TestActiveflowString(t *testing.T) {
	a := Activeflow{
		Status: StatusRunning,
	}
	a.ID = uuid.Must(uuid.NewV4())

	s := a.String()
	if s == "" {
		t.Error("Activeflow.String() returned empty string")
	}
}

func TestActiveflowMatches(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	flowID := uuid.Must(uuid.NewV4())

	a1 := &Activeflow{
		FlowID:   flowID,
		Status:   StatusRunning,
		TMCreate: "2024-01-01T00:00:00.000000Z",
		TMUpdate: "2024-01-01T00:00:00.000000Z",
	}
	a1.ID = id

	a2 := &Activeflow{
		FlowID:   flowID,
		Status:   StatusRunning,
		TMCreate: "2024-01-02T00:00:00.000000Z",
		TMUpdate: "2024-01-02T00:00:00.000000Z",
	}
	a2.ID = id

	if !a1.Matches(a2) {
		t.Error("Activeflow.Matches() should return true for matching activeflows (timestamps ignored)")
	}
}

func TestActiveflowMatchesDifferent(t *testing.T) {
	a1 := &Activeflow{
		Status: StatusRunning,
	}
	a1.ID = uuid.Must(uuid.NewV4())

	a2 := &Activeflow{
		Status: StatusEnded,
	}
	a2.ID = uuid.Must(uuid.NewV4())

	if a1.Matches(a2) {
		t.Error("Activeflow.Matches() should return false for different activeflows")
	}
}
