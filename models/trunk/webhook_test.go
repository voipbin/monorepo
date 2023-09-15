package trunk

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	type test struct {
		name string

		trunk Trunk

		expectRes *WebhookMessage
	}

	tests := []test{
		{
			name: "normal",
			trunk: Trunk{
				ID:         uuid.FromStringOrNil("dfd1c0d6-5210-11ee-9b01-8f98fa57b2ce"),
				CustomerID: uuid.FromStringOrNil("e01eb224-5210-11ee-893e-e33a13839d13"),
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test",
				AuthTypes:  []AuthType{AuthTypeBasic, AuthTypeIP},
				Username:   "testusername",
				Password:   "testpassword",
				AllowedIPs: []string{
					"1.2.3.4",
				},
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:18.995000",
				TMDelete: "2020-04-18 03:22:19.995000",
			},

			expectRes: &WebhookMessage{
				ID:         uuid.FromStringOrNil("dfd1c0d6-5210-11ee-9b01-8f98fa57b2ce"),
				CustomerID: uuid.FromStringOrNil("e01eb224-5210-11ee-893e-e33a13839d13"),
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test",
				AuthTypes:  []AuthType{AuthTypeBasic, AuthTypeIP},
				Username:   "testusername",
				Password:   "testpassword",
				AllowedIPs: []string{
					"1.2.3.4",
				},
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:18.995000",
				TMDelete: "2020-04-18 03:22:19.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := tt.trunk.ConvertWebhookMessage()
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
