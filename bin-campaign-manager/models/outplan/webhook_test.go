package outplan

import (
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {

	tests := []struct {
		name string

		data Outplan

		expectRes *WebhookMessage
	}{
		{
			name: "normal",

			data: Outplan{
				ID:         uuid.FromStringOrNil("036b16aa-7fbb-11ee-aff0-b322b7ffe078"),
				CustomerID: uuid.FromStringOrNil("03a64b4e-7fbb-11ee-b17b-7f37303b51ff"),
				Name:       "test name",
				Detail:     "test detail",
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DialTimeout:  100000,
				TryInterval:  100000,
				MaxTryCount0: 5,
				MaxTryCount1: 6,
				MaxTryCount2: 7,
				MaxTryCount3: 8,
				MaxTryCount4: 9,
				TMCreate:     "2020-10-10 03:30:17.000000",
				TMUpdate:     "2020-10-10 03:31:17.000000",
				TMDelete:     "9999-01-01 00:00:00.000000",
			},

			expectRes: &WebhookMessage{
				ID:         uuid.FromStringOrNil("036b16aa-7fbb-11ee-aff0-b322b7ffe078"),
				CustomerID: uuid.FromStringOrNil("03a64b4e-7fbb-11ee-b17b-7f37303b51ff"),
				Name:       "test name",
				Detail:     "test detail",
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DialTimeout:  100000,
				TryInterval:  100000,
				MaxTryCount0: 5,
				MaxTryCount1: 6,
				MaxTryCount2: 7,
				MaxTryCount3: 8,
				MaxTryCount4: 9,
				TMCreate:     "2020-10-10 03:30:17.000000",
				TMUpdate:     "2020-10-10 03:31:17.000000",
				TMDelete:     "9999-01-01 00:00:00.000000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := tt.data.ConvertWebhookMessage()
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
