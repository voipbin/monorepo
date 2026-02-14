package groupcall

import (
	"encoding/json"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestGroupcallStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	flowID := uuid.Must(uuid.NewV4())
	masterCallID := uuid.Must(uuid.NewV4())

	g := Groupcall{
		Status:           StatusProgressing,
		FlowID:           flowID,
		MasterCallID:     masterCallID,
		RingMethod:       RingMethodRingAll,
		AnswerMethod:     AnswerMethodHangupOthers,
		CallCount:        5,
		GroupcallCount:   2,
		DialIndex:        1,
	}
	g.ID = id

	if g.ID != id {
		t.Errorf("Groupcall.ID = %v, expected %v", g.ID, id)
	}
	if g.Status != StatusProgressing {
		t.Errorf("Groupcall.Status = %v, expected %v", g.Status, StatusProgressing)
	}
	if g.FlowID != flowID {
		t.Errorf("Groupcall.FlowID = %v, expected %v", g.FlowID, flowID)
	}
	if g.MasterCallID != masterCallID {
		t.Errorf("Groupcall.MasterCallID = %v, expected %v", g.MasterCallID, masterCallID)
	}
	if g.RingMethod != RingMethodRingAll {
		t.Errorf("Groupcall.RingMethod = %v, expected %v", g.RingMethod, RingMethodRingAll)
	}
	if g.AnswerMethod != AnswerMethodHangupOthers {
		t.Errorf("Groupcall.AnswerMethod = %v, expected %v", g.AnswerMethod, AnswerMethodHangupOthers)
	}
	if g.CallCount != 5 {
		t.Errorf("Groupcall.CallCount = %v, expected %v", g.CallCount, 5)
	}
	if g.GroupcallCount != 2 {
		t.Errorf("Groupcall.GroupcallCount = %v, expected %v", g.GroupcallCount, 2)
	}
	if g.DialIndex != 1 {
		t.Errorf("Groupcall.DialIndex = %v, expected %v", g.DialIndex, 1)
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_progressing", StatusProgressing, "progressing"},
		{"status_hangingup", StatusHangingup, "hangingup"},
		{"status_hangup", StatusHangup, "hangup"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestRingMethodConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant RingMethod
		expected string
	}{
		{"ring_method_none", RingMethodNone, ""},
		{"ring_method_ring_all", RingMethodRingAll, "ring_all"},
		{"ring_method_linear", RingMethodLinear, "linear"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestAnswerMethodConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant AnswerMethod
		expected string
	}{
		{"answer_method_none", AnswerMethodNone, ""},
		{"answer_method_hangup_others", AnswerMethodHangupOthers, "hangup_others"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestConvertWebhookMessage(t *testing.T) {
	now := time.Now()
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	ownerID := uuid.Must(uuid.NewV4())
	flowID := uuid.Must(uuid.NewV4())
	masterCallID := uuid.Must(uuid.NewV4())
	answerCallID := uuid.Must(uuid.NewV4())
	callID1 := uuid.Must(uuid.NewV4())
	callID2 := uuid.Must(uuid.NewV4())

	source := &commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: "+1234567890",
	}
	dest1 := commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: "+0987654321",
	}

	g := &Groupcall{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeAgent,
			OwnerID:   ownerID,
		},
		Status:         StatusProgressing,
		FlowID:         flowID,
		Source:         source,
		Destinations:   []commonaddress.Address{dest1},
		MasterCallID:   masterCallID,
		RingMethod:     RingMethodRingAll,
		AnswerMethod:   AnswerMethodHangupOthers,
		AnswerCallID:   answerCallID,
		CallIDs:        []uuid.UUID{callID1, callID2},
		CallCount:      2,
		GroupcallCount: 1,
		DialIndex:      0,
		TMCreate:       &now,
		TMUpdate:       &now,
	}

	webhook := g.ConvertWebhookMessage()

	if webhook.ID != id {
		t.Errorf("ConvertWebhookMessage ID = %v, expected %v", webhook.ID, id)
	}
	if webhook.CustomerID != customerID {
		t.Errorf("ConvertWebhookMessage CustomerID = %v, expected %v", webhook.CustomerID, customerID)
	}
	if webhook.Status != StatusProgressing {
		t.Errorf("ConvertWebhookMessage Status = %v, expected %v", webhook.Status, StatusProgressing)
	}
	if webhook.FlowID != flowID {
		t.Errorf("ConvertWebhookMessage FlowID = %v, expected %v", webhook.FlowID, flowID)
	}
	if webhook.Source.Target != "+1234567890" {
		t.Errorf("ConvertWebhookMessage Source = %v, expected +1234567890", webhook.Source.Target)
	}
	if len(webhook.Destinations) != 1 {
		t.Errorf("ConvertWebhookMessage Destinations length = %v, expected 1", len(webhook.Destinations))
	}
	if webhook.MasterCallID != masterCallID {
		t.Errorf("ConvertWebhookMessage MasterCallID = %v, expected %v", webhook.MasterCallID, masterCallID)
	}
	if webhook.AnswerCallID != answerCallID {
		t.Errorf("ConvertWebhookMessage AnswerCallID = %v, expected %v", webhook.AnswerCallID, answerCallID)
	}
	if len(webhook.CallIDs) != 2 {
		t.Errorf("ConvertWebhookMessage CallIDs length = %v, expected 2", len(webhook.CallIDs))
	}
	if webhook.CallCount != 2 {
		t.Errorf("ConvertWebhookMessage CallCount = %v, expected 2", webhook.CallCount)
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	flowID := uuid.Must(uuid.NewV4())

	g := &Groupcall{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{},
		Status: StatusProgressing,
		FlowID: flowID,
	}

	data, err := g.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("CreateWebhookEvent returned error: %v", err)
	}

	if len(data) == 0 {
		t.Error("CreateWebhookEvent returned empty data")
	}

	// Verify it's valid JSON
	var webhook WebhookMessage
	if err := json.Unmarshal(data, &webhook); err != nil {
		t.Errorf("CreateWebhookEvent returned invalid JSON: %v", err)
	}

	if webhook.ID != id {
		t.Errorf("Unmarshalled webhook ID = %v, expected %v", webhook.ID, id)
	}
	if webhook.Status != StatusProgressing {
		t.Errorf("Unmarshalled webhook Status = %v, expected %v", webhook.Status, StatusProgressing)
	}
}
