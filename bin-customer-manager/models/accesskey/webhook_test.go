package accesskey

import (
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	type test struct {
		name string

		data Accesskey

		expectRes *WebhookMessage
	}

	tmExpire := time.Date(2022, 4, 18, 3, 22, 17, 995000000, time.UTC)
	tmCreate := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)
	tmUpdate := time.Date(2020, 4, 18, 3, 22, 18, 995000000, time.UTC)
	tmDelete := time.Date(2020, 4, 18, 3, 22, 19, 995000000, time.UTC)

	tests := []test{
		{
			name: "normal",
			data: Accesskey{
				ID:          uuid.FromStringOrNil("4f6c0cb2-a756-11ef-a880-0b6b35a9a9b8"),
				CustomerID:  uuid.FromStringOrNil("5014f0a2-a756-11ef-9589-e3afd497d2dd"),
				Name:        "test name",
				Detail:      "test detail",
				TokenPrefix: "vb_testpref",
				RawToken:    "vb_testprefixtoken123",
				TMExpire:    &tmExpire,
				TMCreate:    &tmCreate,
				TMUpdate:    &tmUpdate,
				TMDelete:    &tmDelete,
			},

			expectRes: &WebhookMessage{
				ID:          uuid.FromStringOrNil("4f6c0cb2-a756-11ef-a880-0b6b35a9a9b8"),
				CustomerID:  uuid.FromStringOrNil("5014f0a2-a756-11ef-9589-e3afd497d2dd"),
				Name:        "test name",
				Detail:      "test detail",
				Token:       "vb_testprefixtoken123",
				TokenPrefix: "vb_testpref",
				TMExpire:    &tmExpire,
				TMCreate:    &tmCreate,
				TMUpdate:    &tmUpdate,
				TMDelete:    &tmDelete,
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

func Test_CreateWebhookEvent(t *testing.T) {
	tmCreate := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string
		data Accesskey
	}{
		{
			name: "normal",
			data: Accesskey{
				ID:          uuid.FromStringOrNil("4f6c0cb2-a756-11ef-a880-0b6b35a9a9b8"),
				CustomerID:  uuid.FromStringOrNil("5014f0a2-a756-11ef-9589-e3afd497d2dd"),
				Name:        "test name",
				Detail:      "test detail",
				TokenPrefix: "vb_testpref",
				RawToken:    "vb_testprefixtoken123",
				TMCreate:    &tmCreate,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.data.CreateWebhookEvent()
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil {
				t.Errorf("Wrong match. expect: webhook event, got: nil")
			}
		})
	}
}
