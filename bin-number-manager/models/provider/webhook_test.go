package provider

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func Test_ConvertWebhookMessage(t *testing.T) {
	actor := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000")
	tm := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string
		info *Info
	}{
		{
			name: "all fields",
			info: &Info{
				ProviderName:     "telnyx",
				Enabled:          true,
				Priority:         10,
				Config:           map[string]string{"connection_id": "test-connection"},
				HasSecret:        true,
				KeyVersion:       2,
				CredentialSource: CredentialSourceDB,
				Usable:           true,
				UpdatedBy:        &actor,
				SecretUpdatedBy:  &actor,
				TMUpdate:         &tm,
				TMSecretUpdate:   &tm,
			},
		},
		{
			name: "zero values and no history",
			info: &Info{ProviderName: "twilio", CredentialSource: CredentialSourceNone},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.info.ConvertWebhookMessage()

			// every field of the message equals the same-named field of Info
			rv := reflect.ValueOf(*res)
			iv := reflect.ValueOf(*tt.info)
			for i := 0; i < rv.NumField(); i++ {
				name := rv.Type().Field(i).Name
				if !reflect.DeepEqual(rv.Field(i).Interface(), iv.FieldByName(name).Interface()) {
					t.Errorf("Wrong field %s. expect: %v, got: %v", name, iv.FieldByName(name).Interface(), rv.Field(i).Interface())
				}
			}
			if rv.NumField() != iv.NumField() {
				t.Errorf("Wrong field count. expect: %d, got: %d", iv.NumField(), rv.NumField())
			}

			b, err := json.Marshal(res)
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]any
			if errUnmarshal := json.Unmarshal(b, &m); errUnmarshal != nil {
				t.Fatal(errUnmarshal)
			}
			for _, key := range []string{"secret", "secret_ciphertext"} {
				if _, ok := m[key]; ok {
					t.Errorf("The JSON has the secret field %s: %s", key, string(b))
				}
			}
		})
	}
}
