package account

import (
	"encoding/json"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

func TestAccount_ConvertWebhookMessage(t *testing.T) {
	tmCreate := time.Date(2023, 6, 7, 3, 22, 17, 995000000, time.UTC)
	tmUpdate := time.Date(2023, 6, 8, 10, 15, 30, 500000000, time.UTC)

	tests := []struct {
		name    string
		account *Account
	}{
		{
			name: "full account data",
			account: &Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ecdb856-0600-11ee-b746-d3ef5adc8ef7"),
					CustomerID: uuid.FromStringOrNil("6efc4a5e-0600-11ee-9aca-57553e6045e7"),
				},
				Name:          "Test Account",
				Detail:        "Test Detail",
				PlanType:      PlanTypeFree,
				Balance:       99.99,
				PaymentType:   PaymentTypePrepaid,
				PaymentMethod: PaymentMethodCreditCard,
				TMCreate:      &tmCreate,
				TMUpdate:      &tmUpdate,
				TMDelete:      nil,
			},
		},
		{
			name: "minimal account data",
			account: &Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9b219300-0600-11ee-bd0c-db57aac06783"),
				},
				TMCreate: &tmCreate,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.account.ConvertWebhookMessage()

			if result == nil {
				t.Fatal("ConvertWebhookMessage() returned nil")
			}

			if result.ID != tt.account.ID {
				t.Errorf("ID = %v, expected %v", result.ID, tt.account.ID)
			}

			if result.CustomerID != tt.account.CustomerID {
				t.Errorf("CustomerID = %v, expected %v", result.CustomerID, tt.account.CustomerID)
			}

			if result.Name != tt.account.Name {
				t.Errorf("Name = %s, expected %s", result.Name, tt.account.Name)
			}

			if result.Detail != tt.account.Detail {
				t.Errorf("Detail = %s, expected %s", result.Detail, tt.account.Detail)
			}

			if result.PlanType != tt.account.PlanType {
				t.Errorf("PlanType = %s, expected %s", result.PlanType, tt.account.PlanType)
			}

			if result.Balance != tt.account.Balance {
				t.Errorf("Balance = %f, expected %f", result.Balance, tt.account.Balance)
			}

			if result.PaymentType != tt.account.PaymentType {
				t.Errorf("PaymentType = %s, expected %s", result.PaymentType, tt.account.PaymentType)
			}

			if result.PaymentMethod != tt.account.PaymentMethod {
				t.Errorf("PaymentMethod = %s, expected %s", result.PaymentMethod, tt.account.PaymentMethod)
			}
		})
	}
}

func TestAccount_CreateWebhookEvent(t *testing.T) {
	tmCreate := time.Date(2023, 6, 7, 3, 22, 17, 995000000, time.UTC)
	tmUpdate := time.Date(2023, 6, 8, 10, 15, 30, 500000000, time.UTC)

	tests := []struct {
		name    string
		account *Account
		wantErr bool
	}{
		{
			name: "full account data",
			account: &Account{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ecdb856-0600-11ee-b746-d3ef5adc8ef7"),
					CustomerID: uuid.FromStringOrNil("6efc4a5e-0600-11ee-9aca-57553e6045e7"),
				},
				Name:          "Test Account",
				Detail:        "Test Detail",
				PlanType:      PlanTypeFree,
				Balance:       99.99,
				PaymentType:   PaymentTypePrepaid,
				PaymentMethod: PaymentMethodCreditCard,
				TMCreate:      &tmCreate,
				TMUpdate:      &tmUpdate,
				TMDelete:      nil,
			},
			wantErr: false,
		},
		{
			name: "minimal account data",
			account: &Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9b219300-0600-11ee-bd0c-db57aac06783"),
				},
				TMCreate: &tmCreate,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.account.CreateWebhookEvent()

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateWebhookEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("CreateWebhookEvent() returned nil bytes")
				}

				// Verify it's valid JSON
				var msg WebhookMessage
				if err := json.Unmarshal(result, &msg); err != nil {
					t.Errorf("CreateWebhookEvent() returned invalid JSON: %v", err)
				}

				// Verify the unmarshaled data matches
				if msg.ID != tt.account.ID {
					t.Errorf("Unmarshaled ID = %v, expected %v", msg.ID, tt.account.ID)
				}

				if msg.Name != tt.account.Name {
					t.Errorf("Unmarshaled Name = %s, expected %s", msg.Name, tt.account.Name)
				}
			}
		})
	}
}
