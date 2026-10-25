package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

// the ciphertext and the secret never appear in the JSON of the models
func Test_json_hidesSecretMaterial(t *testing.T) {
	p := Provider{ProviderName: "telnyx", SecretCiphertext: []byte("fake-ciphertext-bytes"), KeyVersion: 2}
	c := Credentials{Secret: "fake-secret-value", Config: map[string]string{"connection_id": "c1"}}

	for _, v := range []any{p, &p, c, &c, Info{HasSecret: true, KeyVersion: 2}} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}

		out := string(b)
		if strings.Contains(out, "fake-ciphertext-bytes") || strings.Contains(out, "ZmFrZS1jaXBoZXJ0ZXh0") || strings.Contains(out, "fake-secret-value") {
			t.Errorf("The JSON holds secret material: %s", out)
		}
		if strings.Contains(out, "secret_ciphertext") {
			t.Errorf("The JSON has the ciphertext field: %s", out)
		}
	}
}
