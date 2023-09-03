package customer

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"
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
				ID:            uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
				Username:      "test",
				PasswordHash:  "passwordHash",
				Name:          "test name",
				Detail:        "test detail",
				Email:         "test@test.com",
				PhoneNumber:   "+821100000001",
				Address:       "Copenhagen, Denmark",
				WebhookMethod: WebhookMethodPost,
				WebhookURI:    "test.com",
				PermissionIDs: []uuid.UUID{
					permission.PermissionAdmin.ID,
				},
				BillingAccountID: uuid.FromStringOrNil("1c61bf00-4a01-11ee-9e71-2b88ad09ca2f"),
				TMCreate:         "2020-04-18 03:22:17.995000",
				TMUpdate:         "2020-04-18 03:22:18.995000",
				TMDelete:         "2020-04-18 03:22:19.995000",
			},

			expectRes: &WebhookMessage{
				ID:            uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
				Username:      "test",
				Name:          "test name",
				Detail:        "test detail",
				Email:         "test@test.com",
				PhoneNumber:   "+821100000001",
				Address:       "Copenhagen, Denmark",
				WebhookMethod: WebhookMethodPost,
				WebhookURI:    "test.com",
				PermissionIDs: []uuid.UUID{
					permission.PermissionAdmin.ID,
				},
				BillingAccountID: uuid.FromStringOrNil("1c61bf00-4a01-11ee-9e71-2b88ad09ca2f"),
				TMCreate:         "2020-04-18 03:22:17.995000",
				TMUpdate:         "2020-04-18 03:22:18.995000",
				TMDelete:         "2020-04-18 03:22:19.995000",
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
