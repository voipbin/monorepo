package number

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestNumberStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	callFlowID := uuid.Must(uuid.NewV4())
	messageFlowID := uuid.Must(uuid.NewV4())

	n := Number{
		Number:              "+15551234567",
		CallFlowID:          callFlowID,
		MessageFlowID:       messageFlowID,
		Name:                "Main Line",
		Detail:              "Primary business number",
		ProviderName:        ProviderNameTelnyx,
		ProviderReferenceID: "provider-ref-123",
		Status:              StatusActive,
		T38Enabled:          true,
		EmergencyEnabled:    false,
	}
	n.ID = id
	n.CustomerID = customerID

	if n.ID != id {
		t.Errorf("Number.ID = %v, expected %v", n.ID, id)
	}
	if n.CustomerID != customerID {
		t.Errorf("Number.CustomerID = %v, expected %v", n.CustomerID, customerID)
	}
	if n.Number != "+15551234567" {
		t.Errorf("Number.Number = %v, expected %v", n.Number, "+15551234567")
	}
	if n.CallFlowID != callFlowID {
		t.Errorf("Number.CallFlowID = %v, expected %v", n.CallFlowID, callFlowID)
	}
	if n.MessageFlowID != messageFlowID {
		t.Errorf("Number.MessageFlowID = %v, expected %v", n.MessageFlowID, messageFlowID)
	}
	if n.Name != "Main Line" {
		t.Errorf("Number.Name = %v, expected %v", n.Name, "Main Line")
	}
	if n.Detail != "Primary business number" {
		t.Errorf("Number.Detail = %v, expected %v", n.Detail, "Primary business number")
	}
	if n.ProviderName != ProviderNameTelnyx {
		t.Errorf("Number.ProviderName = %v, expected %v", n.ProviderName, ProviderNameTelnyx)
	}
	if n.ProviderReferenceID != "provider-ref-123" {
		t.Errorf("Number.ProviderReferenceID = %v, expected %v", n.ProviderReferenceID, "provider-ref-123")
	}
	if n.Status != StatusActive {
		t.Errorf("Number.Status = %v, expected %v", n.Status, StatusActive)
	}
	if n.T38Enabled != true {
		t.Errorf("Number.T38Enabled = %v, expected %v", n.T38Enabled, true)
	}
	if n.EmergencyEnabled != false {
		t.Errorf("Number.EmergencyEnabled = %v, expected %v", n.EmergencyEnabled, false)
	}
}

func TestProviderNameConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ProviderName
		expected string
	}{
		{"provider_name_none", ProviderNameNone, ""},
		{"provider_name_telnyx", ProviderNameTelnyx, "telnyx"},
		{"provider_name_twilio", ProviderNameTwilio, "twilio"},
		{"provider_name_messagebird", ProviderNameMessagebird, "messagebird"},
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
		{"status_none", StatusNone, ""},
		{"status_active", StatusActive, "active"},
		{"status_deleted", StatusDeleted, "deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
