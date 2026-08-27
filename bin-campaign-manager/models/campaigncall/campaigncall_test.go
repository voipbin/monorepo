package campaigncall

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
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
		TMCreate: ptrTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		TMUpdate: ptrTime(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
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

func ptrTime(t time.Time) *time.Time {
	return &t
}

// Campaigncall overrides the subscription address of the global topic exchange (VOIP-1404/1405).
// The assertion pins the POINTER receiver: notifyhandler asserts on the dynamic type of the event
// data, which is always a pointer, so a value receiver would silently never be picked up.
var _ eventtopic.SubscriptionIdentifier = (*Campaigncall)(nil)

func TestCampaigncallEventSubscriptionID(t *testing.T) {
	campaignID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name         string
		campaigncall *Campaigncall
		expect       string
	}{
		{
			"normal",
			&Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				CampaignID: campaignID,
				Status:     StatusDialing,
			},
			campaignID.String(),
		},
		{
			"empty campaign id",
			&Campaigncall{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.campaigncall.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestCampaigncallEventSubscriptionIDIsNotOwnID pins the property the override exists for: every
// campaigncall of one campaign resolves to the SAME address, and that address is never the
// campaigncall's own id. Two distinct campaigncalls of one campaign must converge, so a consumer
// binds `campaign-manager.campaigncall.<campaign-id>.#` once and follows the whole campaign.
func TestCampaigncallEventSubscriptionIDIsNotOwnID(t *testing.T) {
	campaignID := uuid.Must(uuid.NewV4())

	first := &Campaigncall{
		Identity:   commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		CampaignID: campaignID,
	}
	second := &Campaigncall{
		Identity:   commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		CampaignID: campaignID,
	}

	if first.ID == second.ID {
		t.Fatalf("Campaigncall ids are expected to differ. id: %s", first.ID)
	}
	if first.EventSubscriptionID() != second.EventSubscriptionID() {
		t.Errorf("Wrong match. expect: %s, got: %s", first.EventSubscriptionID(), second.EventSubscriptionID())
	}
	if first.EventSubscriptionID() != campaignID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", campaignID.String(), first.EventSubscriptionID())
	}
	if first.EventSubscriptionID() == first.ID.String() {
		t.Errorf("Subscription address must not be the campaigncall own id. id: %s", first.ID)
	}
}
