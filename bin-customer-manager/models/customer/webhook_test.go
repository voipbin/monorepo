package customer

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	type test struct {
		name string

		customer Customer

		expectRes *WebhookMessage
	}

	tmCreate := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)
	tmUpdate := time.Date(2020, 4, 18, 3, 22, 18, 995000000, time.UTC)
	tmDelete := time.Date(2020, 4, 18, 3, 22, 19, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			customer: Customer{
				ID:               uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
				Name:             "test name",
				Detail:           "test detail",
				Email:            "test@test.com",
				PhoneNumber:      "+821100000001",
				Address:          "Copenhagen, Denmark",
				WebhookMethod:    WebhookMethodPost,
				WebhookURI:       "test.com",
				BillingAccountID: uuid.FromStringOrNil("1c61bf00-4a01-11ee-9e71-2b88ad09ca2f"),
				TMCreate:         &tmCreate,
				TMUpdate:         &tmUpdate,
				TMDelete:         &tmDelete,
			},

			expectRes: &WebhookMessage{
				ID:               uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
				Name:             "test name",
				Detail:           "test detail",
				Email:            "test@test.com",
				PhoneNumber:      "+821100000001",
				Address:          "Copenhagen, Denmark",
				WebhookMethod:    WebhookMethodPost,
				WebhookURI:       "test.com",
				BillingAccountID: uuid.FromStringOrNil("1c61bf00-4a01-11ee-9e71-2b88ad09ca2f"),
				TMCreate:         &tmCreate,
				TMUpdate:         &tmUpdate,
				TMDelete:         &tmDelete,
			},
		},
		{
			name: "terms fields excluded",
			customer: Customer{
				ID:                 uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
				Name:               "test name",
				TermsAgreedVersion: "2026-02-22T00:00:00Z",
				TermsAgreedIP:      "192.168.1.1",
			},

			expectRes: &WebhookMessage{
				ID:   uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
				Name: "test name",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := tt.customer.ConvertWebhookMessage()
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

// Test_ConvertWebhookMessage_NeverIncludesSecret is a regression test for a
// round-1 security review finding: the public conversion used for the
// customer_updated webhook event and every non-self-facing API response must
// never carry the webhook signing secret. WebhookMessage structurally has no
// WebhookSecret field, so this also guards against a future field-for-field
// copy mistake reintroducing it there.
func Test_ConvertWebhookMessage_NeverIncludesSecret(t *testing.T) {
	c := &Customer{
		ID:            uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
		Name:          "test name",
		WebhookSecret: "super-secret-value",
	}

	b, err := json.Marshal(c.ConvertWebhookMessage())
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if strings.Contains(string(b), "super-secret-value") || strings.Contains(string(b), "webhook_secret") {
		t.Errorf("Wrong match. WebhookSecret leaked via ConvertWebhookMessage: %s", b)
	}
}

// Test_ConvertWebhookMessageSelf_IncludesSecret verifies the one sanctioned
// path to expose the webhook signing secret: the self-view conversion used
// by CustomerSelfGet.
func Test_ConvertWebhookMessageSelf_IncludesSecret(t *testing.T) {
	c := &Customer{
		ID:            uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
		Name:          "test name",
		WebhookSecret: "super-secret-value",
	}

	res := c.ConvertWebhookMessageSelf()
	if res.WebhookSecret != "super-secret-value" {
		t.Errorf("Wrong match. expect: super-secret-value, got: %s", res.WebhookSecret)
	}

	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !strings.Contains(string(b), `"webhook_secret":"super-secret-value"`) {
		t.Errorf("Wrong match. expect webhook_secret in self-view JSON, got: %s", b)
	}
}

func Test_CreateWebhookEvent(t *testing.T) {
	tmCreate := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name     string
		customer Customer
	}{
		{
			name: "normal",
			customer: Customer{
				ID:            uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
				Name:          "test name",
				Detail:        "test detail",
				Email:         "test@test.com",
				PhoneNumber:   "+821100000001",
				Address:       "Copenhagen, Denmark",
				WebhookMethod: WebhookMethodPost,
				WebhookURI:    "test.com",
				TMCreate:      &tmCreate,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.customer.CreateWebhookEvent()
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil {
				t.Errorf("Wrong match. expect: webhook event, got: nil")
			}
		})
	}
}
