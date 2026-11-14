package provider

import (
	"fmt"
	"strings"
	"testing"
)

// Test_format_redactsSecretMaterial checks that every formatting verb prints the redacted form
// of the credentials only, for values and pointers.
func Test_format_redactsSecretMaterial(t *testing.T) {
	c := Credentials{Secret: "fake-secret-value", Config: map[string]string{"connection_id": "c1"}}

	tests := []struct {
		name string

		value   any
		secrets []string

		expect string
	}{
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

func Test_Credentials_GoString(t *testing.T) {
	if got := (Credentials{}).GoString(); got != "Credentials{Secret:[REDACTED]}" {
		t.Errorf("Wrong match. got: %s", got)
	}
}
