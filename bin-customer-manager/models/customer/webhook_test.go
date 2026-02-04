package customer

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	type test struct {
		name string

		customer Customer

		expectRes *WebhookMessage
	}

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
				TMCreate:         "2020-04-18T03:22:17.995000Z",
				TMUpdate:         "2020-04-18T03:22:18.995000Z",
				TMDelete:         "2020-04-18T03:22:19.995000Z",
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
				TMCreate:         "2020-04-18T03:22:17.995000Z",
				TMUpdate:         "2020-04-18T03:22:18.995000Z",
				TMDelete:         "2020-04-18T03:22:19.995000Z",
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
