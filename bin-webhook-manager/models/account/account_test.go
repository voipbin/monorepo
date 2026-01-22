package account

import (
	"testing"

	"github.com/gofrs/uuid"

	cscustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-webhook-manager/models/webhook"
)

func TestAccountStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	acc := Account{
		ID:            id,
		WebhookMethod: webhook.MethodTypePOST,
		WebhookURI:    "https://example.com/webhook",
	}

	if acc.ID != id {
		t.Errorf("Account.ID = %v, expected %v", acc.ID, id)
	}
	if acc.WebhookMethod != webhook.MethodTypePOST {
		t.Errorf("Account.WebhookMethod = %v, expected %v", acc.WebhookMethod, webhook.MethodTypePOST)
	}
	if acc.WebhookURI != "https://example.com/webhook" {
		t.Errorf("Account.WebhookURI = %v, expected %v", acc.WebhookURI, "https://example.com/webhook")
	}
}

func TestCreateAccountFromCustomer(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	customer := &cscustomer.Customer{
		ID:            id,
		WebhookMethod: "POST",
		WebhookURI:    "https://callback.example.com/events",
	}

	result := CreateAccountFromCustomer(customer)

	if result.ID != id {
		t.Errorf("CreateAccountFromCustomer().ID = %v, expected %v", result.ID, id)
	}
	if result.WebhookMethod != webhook.MethodTypePOST {
		t.Errorf("CreateAccountFromCustomer().WebhookMethod = %v, expected %v", result.WebhookMethod, webhook.MethodTypePOST)
	}
	if result.WebhookURI != "https://callback.example.com/events" {
		t.Errorf("CreateAccountFromCustomer().WebhookURI = %v, expected %v", result.WebhookURI, "https://callback.example.com/events")
	}
}

func TestCreateAccountFromCustomerWithDifferentMethods(t *testing.T) {
	tests := []struct {
		name           string
		webhookMethod  cscustomer.WebhookMethod
		expectedMethod webhook.MethodType
	}{
		{"post_method", cscustomer.WebhookMethodPost, webhook.MethodTypePOST},
		{"put_method", cscustomer.WebhookMethodPut, webhook.MethodTypePUT},
		{"get_method", cscustomer.WebhookMethodGet, webhook.MethodTypeGET},
		{"delete_method", cscustomer.WebhookMethodDelete, webhook.MethodTypeDELETE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customer := &cscustomer.Customer{
				ID:            uuid.Must(uuid.NewV4()),
				WebhookMethod: tt.webhookMethod,
				WebhookURI:    "https://test.example.com",
			}

			result := CreateAccountFromCustomer(customer)

			if result.WebhookMethod != tt.expectedMethod {
				t.Errorf("CreateAccountFromCustomer().WebhookMethod = %v, expected %v", result.WebhookMethod, tt.expectedMethod)
			}
		})
	}
}
