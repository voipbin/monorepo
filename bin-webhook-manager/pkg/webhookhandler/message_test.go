package webhookhandler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// Test_computeSignature verifies the HMAC-SHA256 signature computed for the
// X-VoIPBIN-Signature header matches the standard hmac.New(sha256.New, secret)
// construction used elsewhere in the monorepo for verifying inbound signatures.
func Test_computeSignature(t *testing.T) {

	type test struct {
		name   string
		secret string
		body   []byte
	}

	tests := []test{
		{
			name:   "normal",
			secret: "test-secret",
			body:   []byte(`{"type":"call_updated","data":{"id":"test"}}`),
		},
		{
			name:   "empty body",
			secret: "test-secret",
			body:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := computeSignature(tt.secret, tt.body)

			mac := hmac.New(sha256.New, []byte(tt.secret))
			mac.Write(tt.body)
			expect := "sha256=" + hex.EncodeToString(mac.Sum(nil))

			if res != expect {
				t.Errorf("Wrong match. expect: %s, got: %s", expect, res)
			}
		})
	}
}

// Test_computeSignature_differentSecretsDiffer ensures the signature is
// actually keyed by the secret -- a required property for receivers to be
// able to distinguish an authentic request from a forged one.
func Test_computeSignature_differentSecretsDiffer(t *testing.T) {
	body := []byte(`{"type":"call_updated"}`)

	sig1 := computeSignature("secret-a", body)
	sig2 := computeSignature("secret-b", body)

	if sig1 == sig2 {
		t.Errorf("Wrong match. expected different signatures for different secrets, got the same: %s", sig1)
	}
}
