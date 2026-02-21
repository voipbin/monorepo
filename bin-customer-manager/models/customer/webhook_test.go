package customer

import (
	"reflect"
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
