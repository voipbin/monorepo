package message

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Message carries an explicit subscription address on the global topic exchange (VOIP-1419).
// The assertion pins the POINTER type: the event data reaches notifyhandler as a pointer and
// the interface assertion matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*Message)(nil)

// TestMessageEventSubscriptionIDUsesOwnIDNotCustomerID pins the address choice: an SMS message
// is an independent persistent resource, so its subscription address is its OWN id -- not the
// customer id, the other plausible field on the same struct.
func TestMessageEventSubscriptionIDUsesOwnIDNotCustomerID(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	m := &Message{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
	}

	res := m.EventSubscriptionID()
	if res != id.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", id.String(), res)
	}
	if res == customerID.String() {
		t.Errorf("Message must not be addressed by its customer id. got: %s", res)
	}
}

func TestMessageStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	m := Message{
		Type:                TypeSMS,
		ProviderName:        ProviderNameTelnyx,
		ProviderReferenceID: "ref-123",
		Text:                "Hello World",
		Medias:              []string{"media1.jpg", "media2.jpg"},
		Direction:           DirectionOutbound,
	}
	m.ID = id
	m.CustomerID = customerID

	if m.ID != id {
		t.Errorf("Message.ID = %v, expected %v", m.ID, id)
	}
	if m.CustomerID != customerID {
		t.Errorf("Message.CustomerID = %v, expected %v", m.CustomerID, customerID)
	}
	if m.Type != TypeSMS {
		t.Errorf("Message.Type = %v, expected %v", m.Type, TypeSMS)
	}
	if m.ProviderName != ProviderNameTelnyx {
		t.Errorf("Message.ProviderName = %v, expected %v", m.ProviderName, ProviderNameTelnyx)
	}
	if m.ProviderReferenceID != "ref-123" {
		t.Errorf("Message.ProviderReferenceID = %v, expected %v", m.ProviderReferenceID, "ref-123")
	}
	if m.Text != "Hello World" {
		t.Errorf("Message.Text = %v, expected %v", m.Text, "Hello World")
	}
	if len(m.Medias) != 2 {
		t.Errorf("Message.Medias length = %v, expected %v", len(m.Medias), 2)
	}
	if m.Direction != DirectionOutbound {
		t.Errorf("Message.Direction = %v, expected %v", m.Direction, DirectionOutbound)
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_sms", TypeSMS, "sms"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestDirectionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Direction
		expected string
	}{
		{"direction_outbound", DirectionOutbound, "outbound"},
		{"direction_inbound", DirectionInbound, "inbound"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestProviderNameConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ProviderName
		expected string
	}{
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
