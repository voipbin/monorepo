package campaigncall

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestCampaigncallStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	campaignID := uuid.Must(uuid.NewV4())
	outplanID := uuid.Must(uuid.NewV4())
	outdialID := uuid.Must(uuid.NewV4())
	outdialTargetID := uuid.Must(uuid.NewV4())
	queueID := uuid.Must(uuid.NewV4())
	activeflowID := uuid.Must(uuid.NewV4())
	flowID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())

	cc := Campaigncall{
		CampaignID:       campaignID,
		OutplanID:        outplanID,
		OutdialID:        outdialID,
		OutdialTargetID:  outdialTargetID,
		QueueID:          queueID,
		ActiveflowID:     activeflowID,
		FlowID:           flowID,
		ReferenceType:    ReferenceTypeCall,
		ReferenceID:      referenceID,
		Status:           StatusDialing,
		Result:           ResultNone,
		DestinationIndex: 0,
		TryCount:         1,
		TMCreate:         "2024-01-01T00:00:00.000000Z",
		TMUpdate:         "2024-01-01T00:00:00.000000Z",
		TMDelete:         "9999-01-01T00:00:00.000000Z",
	}
	cc.ID = id

	if cc.ID != id {
		t.Errorf("Campaigncall.ID = %v, expected %v", cc.ID, id)
	}
	if cc.CampaignID != campaignID {
		t.Errorf("Campaigncall.CampaignID = %v, expected %v", cc.CampaignID, campaignID)
	}
	if cc.Status != StatusDialing {
		t.Errorf("Campaigncall.Status = %v, expected %v", cc.Status, StatusDialing)
	}
	if cc.ReferenceType != ReferenceTypeCall {
		t.Errorf("Campaigncall.ReferenceType = %v, expected %v", cc.ReferenceType, ReferenceTypeCall)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{"reference_type_none", ReferenceTypeNone, "none"},
		{"reference_type_call", ReferenceTypeCall, "call"},
		{"reference_type_flow", ReferenceTypeFlow, "flow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_dialing", StatusDialing, "dialing"},
		{"status_progressing", StatusProgressing, "progressing"},
		{"status_done", StatusDone, "done"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestResultConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Result
		expected string
	}{
		{"result_none", ResultNone, ""},
		{"result_success", ResultSuccess, "success"},
		{"result_fail", ResultFail, "fail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
