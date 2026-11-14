package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

// the secret never appears in the JSON of the models
func Test_json_hidesSecretMaterial(t *testing.T) {
	p := Provider{ProviderName: "telnyx"}
	c := Credentials{Secret: "fake-secret-value", Config: map[string]string{"connection_id": "c1"}}

	for _, v := range []any{p, &p, c, &c, Info{HasSecret: true}} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}

		out := string(b)
		if strings.Contains(out, "fake-secret-value") {
			t.Errorf("The JSON holds secret material: %s", out)
		}
		for _, name := range []string{"secret_ciphertext", "key_version", "secret_updated_by", "tm_secret_update"} {
			if strings.Contains(out, name) {
				t.Errorf("The JSON has the removed field %s: %s", name, out)
			}
		}
	}
}
