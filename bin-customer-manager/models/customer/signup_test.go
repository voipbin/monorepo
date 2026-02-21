package customer

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-customer-manager/models/accesskey"

	"github.com/gofrs/uuid"
)

func Test_SignupResult_ConvertWebhookMessage(t *testing.T) {
	tests := []struct {
		name      string
		input     *SignupResult
		expectRes *SignupResultWebhookMessage
	}{
		{
			name: "with customer",
			input: &SignupResult{
				Customer: &Customer{
					ID:                 uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
					Name:               "test",
					Email:              "test@example.com",
					TermsAgreedVersion: "2026-02-22T00:00:00Z",
					TermsAgreedIP:      "192.168.1.1",
				},
				TempToken: "tmp_abc123",
			},
			expectRes: &SignupResultWebhookMessage{
				Customer: &WebhookMessage{
					ID:    uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
					Name:  "test",
					Email: "test@example.com",
				},
				TempToken: "tmp_abc123",
			},
		},
		{
			name: "nil customer",
			input: &SignupResult{
				TempToken: "tmp_abc123",
			},
			expectRes: &SignupResultWebhookMessage{
				TempToken: "tmp_abc123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.input.ConvertWebhookMessage()
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SignupResult_ConvertWebhookMessage_excludes_terms_fields(t *testing.T) {
	input := &SignupResult{
		Customer: &Customer{
			ID:                 uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
			TermsAgreedVersion: "2026-02-22T00:00:00Z",
			TermsAgreedIP:      "192.168.1.1",
		},
		TempToken: "tmp_abc123",
	}

	res := input.ConvertWebhookMessage()

	// Serialize to JSON and verify terms fields are absent
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	customerRaw, ok := raw["customer"].(map[string]any)
	if !ok {
		t.Fatal("Expected customer field in JSON output")
	}

	if _, exists := customerRaw["terms_agreed_version"]; exists {
		t.Error("terms_agreed_version should not appear in WebhookMessage JSON")
	}
	if _, exists := customerRaw["terms_agreed_ip"]; exists {
		t.Error("terms_agreed_ip should not appear in WebhookMessage JSON")
	}
}

func Test_EmailVerifyResult_ConvertWebhookMessage(t *testing.T) {
	tmExpire := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		input     *EmailVerifyResult
		expectRes *EmailVerifyResultWebhookMessage
	}{
		{
			name: "with customer and accesskey",
			input: &EmailVerifyResult{
				Customer: &Customer{
					ID:                 uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
					Name:               "test",
					Email:              "test@example.com",
					TermsAgreedVersion: "2026-02-22T00:00:00Z",
					TermsAgreedIP:      "192.168.1.1",
				},
				Accesskey: &accesskey.Accesskey{
					ID:         uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
					Name:       "default",
					RawToken:   "tok_abc123",
					TMExpire:   &tmExpire,
				},
			},
			expectRes: &EmailVerifyResultWebhookMessage{
				Customer: &WebhookMessage{
					ID:    uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
					Name:  "test",
					Email: "test@example.com",
				},
				Accesskey: &accesskey.WebhookMessage{
					ID:         uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
					Name:       "default",
					Token:      "tok_abc123",
					TMExpire:   &tmExpire,
				},
			},
		},
		{
			name: "nil customer and accesskey",
			input: &EmailVerifyResult{},
			expectRes: &EmailVerifyResultWebhookMessage{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.input.ConvertWebhookMessage()
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_EmailVerifyResult_ConvertWebhookMessage_excludes_terms_fields(t *testing.T) {
	input := &EmailVerifyResult{
		Customer: &Customer{
			ID:                 uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
			TermsAgreedVersion: "2026-02-22T00:00:00Z",
			TermsAgreedIP:      "192.168.1.1",
		},
	}

	res := input.ConvertWebhookMessage()

	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	customerRaw, ok := raw["customer"].(map[string]any)
	if !ok {
		t.Fatal("Expected customer field in JSON output")
	}

	if _, exists := customerRaw["terms_agreed_version"]; exists {
		t.Error("terms_agreed_version should not appear in WebhookMessage JSON")
	}
	if _, exists := customerRaw["terms_agreed_ip"]; exists {
		t.Error("terms_agreed_ip should not appear in WebhookMessage JSON")
	}
}

func Test_CompleteSignupResult_ConvertWebhookMessage(t *testing.T) {
	tmExpire := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		input     *CompleteSignupResult
		expectRes *CompleteSignupResultWebhookMessage
	}{
		{
			name: "with accesskey",
			input: &CompleteSignupResult{
				CustomerID: "81133fc8-4a01-11ee-8dbf-4bbf6dd46254",
				Accesskey: &accesskey.Accesskey{
					ID:         uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
					Name:       "default",
					RawToken:   "tok_abc123",
					TokenHash:  "secret_hash_value",
					TMExpire:   &tmExpire,
				},
			},
			expectRes: &CompleteSignupResultWebhookMessage{
				CustomerID: "81133fc8-4a01-11ee-8dbf-4bbf6dd46254",
				Accesskey: &accesskey.WebhookMessage{
					ID:         uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("81133fc8-4a01-11ee-8dbf-4bbf6dd46254"),
					Name:       "default",
					Token:      "tok_abc123",
					TMExpire:   &tmExpire,
				},
			},
		},
		{
			name: "nil accesskey",
			input: &CompleteSignupResult{
				CustomerID: "81133fc8-4a01-11ee-8dbf-4bbf6dd46254",
			},
			expectRes: &CompleteSignupResultWebhookMessage{
				CustomerID: "81133fc8-4a01-11ee-8dbf-4bbf6dd46254",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.input.ConvertWebhookMessage()
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CompleteSignupResult_ConvertWebhookMessage_excludes_token_hash(t *testing.T) {
	input := &CompleteSignupResult{
		CustomerID: "81133fc8-4a01-11ee-8dbf-4bbf6dd46254",
		Accesskey: &accesskey.Accesskey{
			ID:        uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000001"),
			TokenHash: "secret_hash_value",
			RawToken:  "tok_abc123",
		},
	}

	res := input.ConvertWebhookMessage()

	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	akRaw, ok := raw["accesskey"].(map[string]any)
	if !ok {
		t.Fatal("Expected accesskey field in JSON output")
	}

	if _, exists := akRaw["token_hash"]; exists {
		t.Error("token_hash should not appear in WebhookMessage JSON")
	}
}
