package trunk

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-registrar-manager/models/sipauth"
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
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("dfd1c0d6-5210-11ee-9b01-8f98fa57b2ce"),
					CustomerID: uuid.FromStringOrNil("e01eb224-5210-11ee-893e-e33a13839d13"),
				},
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test",
				AuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic, sipauth.AuthTypeIP},
				Username:   "testusername",
				Password:   "testpassword",
				AllowedIPs: []string{
					"1.2.3.4",
				},
				TMCreate: func() *time.Time { t := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC); return &t }(),
				TMUpdate: func() *time.Time { t := time.Date(2020, 4, 18, 3, 22, 18, 995000000, time.UTC); return &t }(),
				TMDelete: func() *time.Time { t := time.Date(2020, 4, 18, 3, 22, 19, 995000000, time.UTC); return &t }(),
			},

			expectRes: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("dfd1c0d6-5210-11ee-9b01-8f98fa57b2ce"),
					CustomerID: uuid.FromStringOrNil("e01eb224-5210-11ee-893e-e33a13839d13"),
				},
				Name:       "test name",
				Detail:     "test detail",
				DomainName: "test",
				AuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic, sipauth.AuthTypeIP},
				Username:   "testusername",
				Password:   "testpassword",
				AllowedIPs: []string{
					"1.2.3.4",
				},
				TMCreate: func() *time.Time { t := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC); return &t }(),
				TMUpdate: func() *time.Time { t := time.Date(2020, 4, 18, 3, 22, 18, 995000000, time.UTC); return &t }(),
				TMDelete: func() *time.Time { t := time.Date(2020, 4, 18, 3, 22, 19, 995000000, time.UTC); return &t }(),
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

func Test_CreateWebhookEvent(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	now := time.Now()

	tests := []struct {
		name    string
		trunk   *Trunk
		wantErr bool
	}{
		{
			name: "valid_trunk",
			trunk: &Trunk{
				Identity: commonidentity.Identity{
					ID:         id,
					CustomerID: customerID,
				},
				Name:       "Test",
				DomainName: "test.trunk.voipbin.net",
				AuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic},
				Username:   "user",
				Password:   "pass",
				AllowedIPs: []string{"1.2.3.4"},
				TMCreate:   &now,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.trunk.CreateWebhookEvent()
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateWebhookEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && data == nil {
				t.Error("CreateWebhookEvent() returned nil data")
			}
			if !tt.wantErr {
				// Verify it's valid JSON
				var wm WebhookMessage
				if err := json.Unmarshal(data, &wm); err != nil {
					t.Errorf("Failed to unmarshal webhook event: %v", err)
				}
			}
		})
	}
}
