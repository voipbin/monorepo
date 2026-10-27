package provider

import (
	"fmt"
	"strings"
	"testing"
)

// Test_format_redactsSecretMaterial checks that every formatting verb prints the redacted form
// only, for values and pointers.
func Test_format_redactsSecretMaterial(t *testing.T) {
	p := Provider{ProviderName: "telnyx", SecretCiphertext: []byte("fake-ciphertext-bytes"), KeyVersion: 2}
	c := Credentials{Secret: "fake-secret-value", Config: map[string]string{"connection_id": "c1"}}

	tests := []struct {
		name string

		value   any
		secrets []string

		expect string
	}{
		{
			name:    "provider with %v",
			value:   p,
			secrets: []string{"fake-ciphertext-bytes"},
			expect:  "Provider{ProviderName:telnyx, SecretCiphertext:[REDACTED]}",
		},
		{
			name:    "provider pointer with %v",
			value:   &p,
			secrets: []string{"fake-ciphertext-bytes"},
			expect:  "Provider{ProviderName:telnyx, SecretCiphertext:[REDACTED]}",
		},
		{
			name:    "credentials with %v",
			value:   c,
			secrets: []string{"fake-secret-value"},
			expect:  "Credentials{Secret:[REDACTED]}",
		},
		{
			name:    "credentials pointer with %v",
			value:   &c,
			secrets: []string{"fake-secret-value"},
			expect:  "Credentials{Secret:[REDACTED]}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, verb := range []string{"%v", "%+v", "%#v", "%s"} {
				got := fmt.Sprintf(verb, tt.value)
				if got != tt.expect {
					t.Errorf("Wrong match. verb: %s, expect: %s, got: %s", verb, tt.expect, got)
				}
				for _, secret := range tt.secrets {
					if strings.Contains(got, secret) {
						t.Errorf("Wrong match. verb: %s, the output contains the secret material", verb)
					}
				}
			}
		})
	}
}

func Test_Provider_String_GoString(t *testing.T) {
	if got := (Provider{ProviderName: "twilio"}).String(); got != "Provider{ProviderName:twilio, SecretCiphertext:[REDACTED]}" {
		t.Errorf("Wrong match. got: %s", got)
	}
	if got := (Provider{ProviderName: "twilio"}).GoString(); got != "Provider{ProviderName:twilio, SecretCiphertext:[REDACTED]}" {
		t.Errorf("Wrong match. got: %s", got)
	}
	if got := (Credentials{}).GoString(); got != "Credentials{Secret:[REDACTED]}" {
		t.Errorf("Wrong match. got: %s", got)
	}
}
