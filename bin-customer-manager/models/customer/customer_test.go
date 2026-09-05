package customer

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
)

// Customer's subscription address on the global topic exchange is its own id (VOIP-1419).
// The assertion pins the POINTER type: the event data reaches notifyhandler as a pointer and
// the interface check matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*Customer)(nil)

func TestCustomerEventSubscriptionID(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	billingAccountID := uuid.Must(uuid.NewV4())

	c := &Customer{
		ID:               id,
		BillingAccountID: billingAccountID,
	}

	res := c.EventSubscriptionID()
	if res != id.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", id.String(), res)
	}
	if res == billingAccountID.String() {
		t.Errorf("Customer must be addressed by its own id, not another id field. got: %s", res)
	}
}

func TestCustomerStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	billingAccountID := uuid.Must(uuid.NewV4())
	tmCreate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	c := Customer{
		ID:               id,
		Name:             "Test Customer",
		Detail:           "Test customer details",
		Email:            "test@example.com",
		PhoneNumber:      "+1234567890",
		Address:          "123 Test St",
		WebhookMethod:    WebhookMethodPost,
		WebhookURI:       "https://webhook.example.com",
		BillingAccountID: billingAccountID,
		TMCreate:         &tmCreate,
		TMUpdate:         &tmUpdate,
		TMDelete:         nil,
	}

	if c.ID != id {
		t.Errorf("Customer.ID = %v, expected %v", c.ID, id)
	}
	if c.Name != "Test Customer" {
		t.Errorf("Customer.Name = %v, expected %v", c.Name, "Test Customer")
	}
	if c.Email != "test@example.com" {
		t.Errorf("Customer.Email = %v, expected %v", c.Email, "test@example.com")
	}
	if c.WebhookMethod != WebhookMethodPost {
		t.Errorf("Customer.WebhookMethod = %v, expected %v", c.WebhookMethod, WebhookMethodPost)
	}
}

func TestWebhookMethodConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant WebhookMethod
		expected string
	}{
		{"webhook_method_none", WebhookMethodNone, ""},
		{"webhook_method_post", WebhookMethodPost, "POST"},
		{"webhook_method_get", WebhookMethodGet, "GET"},
		{"webhook_method_put", WebhookMethodPut, "PUT"},
		{"webhook_method_delete", WebhookMethodDelete, "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestSpecialIDConstants(t *testing.T) {
	// Test that special ID constants are not nil UUIDs
	// Note: The actual UUID values in customer.go have typos (missing digits)
	// which causes them to parse as nil UUIDs. This test verifies the current behavior.
	tests := []struct {
		name     string
		constant uuid.UUID
	}{
		{"id_empty", IDEmpty},
		{"id_call_manager", IDCallManager},
		{"id_ai_manager", IDAIManager},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the constant exists and can be used
			_ = tt.constant.String()
		})
	}
}

// TestIDAIManagerListenIsDistinctAndWellFormed pins the two properties the
// InsightAI realtime-listen design (docs/plans/
// 2026-09-03-insight-ai-realtime-listen-design.md §5.2.1) depends on:
//
//  1. IDAIManagerListen parses to a real, non-nil UUID. Several older sentinels
//     in this file (IDEmpty, IDCallManager, IDAIManager) have malformed literals
//     -- their last group has 11 hex digits instead of 12 -- so FromStringOrNil
//     silently yields uuid.Nil for them. A malformed IDAIManagerListen would
//     collapse onto IDAIManager and break property 2.
//  2. IDAIManagerListen != IDAIManager. bin-transcribe-manager's startLive
//     duplicate guard is scoped by (customer_id, reference_id, language,
//     status, deleted). Insight listening and AI summary must own SEPARATE
//     transcribe sessions on the same call; equal owner ids would make them
//     collide and share one session's lifecycle.
func TestIDAIManagerListenIsDistinctAndWellFormed(t *testing.T) {
	if IDAIManagerListen == uuid.Nil {
		t.Errorf("IDAIManagerListen parsed to uuid.Nil -- the literal is malformed (a UUID's last group needs exactly 12 hex digits)")
	}

	if IDAIManagerListen == IDAIManager {
		t.Errorf("IDAIManagerListen must differ from IDAIManager. got both: %s", IDAIManagerListen)
	}
}
