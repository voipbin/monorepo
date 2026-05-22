package account

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestConvertWebhookMessage(t *testing.T) {
	tests := []struct {
		name string

		account *Account

		expectProviderData json.RawMessage
	}{
		{
			name: "forwards provider_data",

			account: &Account{
				Type:         TypeWhatsApp,
				Name:         "WhatsApp Support",
				ProviderData: json.RawMessage(`{"phone_number_id":"12345","app_secret":"s3cr3t"}`),
			},

			expectProviderData: json.RawMessage(`{"phone_number_id":"12345","app_secret":"s3cr3t"}`),
		},
		{
			name: "nil provider_data stays nil",

			account: &Account{
				Type: TypeLine,
				Name: "LINE Account",
			},

			expectProviderData: nil,
		},
		{
			name: "secret and token are not in WebhookMessage",

			account: &Account{
				Type:   TypeWhatsApp,
				Name:   "WhatsApp Account",
				Secret: "verify-token",
				Token:  "system-user-token",
			},

			expectProviderData: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wm := tt.account.ConvertWebhookMessage()

			if !bytes.Equal(wm.ProviderData, tt.expectProviderData) {
				t.Errorf("Wrong ProviderData. expect: %s, got: %s", tt.expectProviderData, wm.ProviderData)
			}

			// Secret and Token must never appear in WebhookMessage.
			b, err := json.Marshal(wm)
			if err != nil {
				t.Fatalf("Could not marshal WebhookMessage: %v", err)
			}
			var raw map[string]any
			if err := json.Unmarshal(b, &raw); err != nil {
				t.Fatalf("Could not unmarshal WebhookMessage JSON: %v", err)
			}
			if _, ok := raw["secret"]; ok {
				t.Error("WebhookMessage must not contain 'secret'")
			}
			if _, ok := raw["token"]; ok {
				t.Error("WebhookMessage must not contain 'token'")
			}
		})
	}
}

func TestCreateWebhookEvent(t *testing.T) {
	tests := []struct {
		name string

		account *Account

		expectProviderDataInJSON bool
	}{
		{
			name: "whatsapp account includes provider_data in JSON",

			account: &Account{
				Type:         TypeWhatsApp,
				Name:         "WhatsApp Support",
				ProviderData: json.RawMessage(`{"phone_number_id":"12345","app_secret":"s3cr3t"}`),
			},

			expectProviderDataInJSON: true,
		},
		{
			name: "line account without provider_data omits field",

			account: &Account{
				Type: TypeLine,
				Name: "LINE Account",
			},

			expectProviderDataInJSON: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := tt.account.CreateWebhookEvent()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			hasProviderData := strings.Contains(string(b), `"provider_data"`)
			if hasProviderData != tt.expectProviderDataInJSON {
				t.Errorf("provider_data in JSON: got %v, want %v. JSON: %s", hasProviderData, tt.expectProviderDataInJSON, b)
			}
		})
	}
}
