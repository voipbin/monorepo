package agent

import (
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	tmCreate := time.Now()
	tmUpdate := time.Now()

	tests := []struct {
		name      string
		agent     *Agent
		expectRes *WebhookMessage
	}{
		{
			name: "normal",
			agent: &Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4f6a7348-4b42-11ec-80ba-13dbc38fe32c"),
					CustomerID: uuid.FromStringOrNil("33f9ca84-7fde-11ec-a186-9f2e8c3a62aa"),
				},
				Username:     "test@voipbin.net",
				PasswordHash: "hash",
				Name:         "test name",
				Detail:       "test detail",
				RingMethod:   RingMethodRingAll,
				Status:       StatusAvailable,
				Permission:   PermissionCustomerAdmin,
				TagIDs:       []uuid.UUID{uuid.FromStringOrNil("700c10b4-4b4e-11ec-959b-bb95248c693f")},
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821021656521",
					},
				},
				TMCreate: &tmCreate,
				TMUpdate: &tmUpdate,
				TMDelete: nil,
			},
			expectRes: &WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4f6a7348-4b42-11ec-80ba-13dbc38fe32c"),
					CustomerID: uuid.FromStringOrNil("33f9ca84-7fde-11ec-a186-9f2e8c3a62aa"),
				},
				Username:   "test@voipbin.net",
				Name:       "test name",
				Detail:     "test detail",
				RingMethod: RingMethodRingAll,
				Status:     StatusAvailable,
				Permission: PermissionCustomerAdmin,
				TagIDs:     []uuid.UUID{uuid.FromStringOrNil("700c10b4-4b4e-11ec-959b-bb95248c693f")},
				Addresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821021656521",
					},
				},
				TMCreate: &tmCreate,
				TMUpdate: &tmUpdate,
				TMDelete: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.agent.ConvertWebhookMessage()
			if res.ID != tt.expectRes.ID {
				t.Errorf("Wrong ID. expect: %v, got: %v", tt.expectRes.ID, res.ID)
			}
			if res.Username != tt.expectRes.Username {
				t.Errorf("Wrong Username. expect: %v, got: %v", tt.expectRes.Username, res.Username)
			}
			if res.Name != tt.expectRes.Name {
				t.Errorf("Wrong Name. expect: %v, got: %v", tt.expectRes.Name, res.Name)
			}
		})
	}
}

func Test_CreateWebhookEvent(t *testing.T) {
	tmCreate := time.Now()

	tests := []struct {
		name      string
		agent     *Agent
		expectErr bool
	}{
		{
			name: "normal",
			agent: &Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4f6a7348-4b42-11ec-80ba-13dbc38fe32c"),
					CustomerID: uuid.FromStringOrNil("33f9ca84-7fde-11ec-a186-9f2e8c3a62aa"),
				},
				Username:     "test@voipbin.net",
				PasswordHash: "hash",
				Name:         "test name",
				Detail:       "test detail",
				RingMethod:   RingMethodRingAll,
				Status:       StatusAvailable,
				Permission:   PermissionCustomerAdmin,
				TagIDs:       []uuid.UUID{},
				Addresses:    []commonaddress.Address{},
				TMCreate:     &tmCreate,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.agent.CreateWebhookEvent()
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong error. expect error: %v, got error: %v", tt.expectErr, err)
			}
			if !tt.expectErr && len(res) == 0 {
				t.Errorf("Expected non-empty result")
			}
		})
	}
}
