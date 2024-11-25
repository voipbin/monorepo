package accesskey

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	type test struct {
		name string

		data Accesskey

		expectRes *WebhookMessage
	}

	tests := []test{
		{
			name: "normal",
			data: Accesskey{
				ID:         uuid.FromStringOrNil("4f6c0cb2-a756-11ef-a880-0b6b35a9a9b8"),
				CustomerID: uuid.FromStringOrNil("5014f0a2-a756-11ef-9589-e3afd497d2dd"),
				Name:       "test name",
				Detail:     "test detail",
				Token:      "test_token",
				TMExpire:   "2022-04-18 03:22:17.995000",
				TMCreate:   "2020-04-18 03:22:17.995000",
				TMUpdate:   "2020-04-18 03:22:18.995000",
				TMDelete:   "2020-04-18 03:22:19.995000",
			},

			expectRes: &WebhookMessage{
				ID:         uuid.FromStringOrNil("4f6c0cb2-a756-11ef-a880-0b6b35a9a9b8"),
				CustomerID: uuid.FromStringOrNil("5014f0a2-a756-11ef-9589-e3afd497d2dd"),
				Name:       "test name",
				Detail:     "test detail",
				Token:      "test_token",
				TMExpire:   "2022-04-18 03:22:17.995000",
				TMCreate:   "2020-04-18 03:22:17.995000",
				TMUpdate:   "2020-04-18 03:22:18.995000",
				TMDelete:   "2020-04-18 03:22:19.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := tt.data.ConvertWebhookMessage()
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
