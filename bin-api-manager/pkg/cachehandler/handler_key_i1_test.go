package cachehandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

// Test_ProvisioningTokenSet_StoredKey pins the stored Redis key and value format, which other components
// read, so a prefix or value format change is detected.
func Test_ProvisioningTokenSet_StoredKey(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	token := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	extensionID := uuid.FromStringOrNil("6a1cca9e-80fb-11f0-92cd-2f8ae91ffc6e")

	if err := h.ProvisioningTokenSet(ctx, token, extensionID, time.Minute); err != nil {
		t.Fatalf("ProvisioningTokenSet: unexpected error: %v", err)
	}

	got, err := mr.Get("api-manager.provisioning_token." + token)
	if err != nil {
		t.Fatalf("expected the key api-manager.provisioning_token.<token> to exist: %v", err)
	}
	if got != extensionID.String() {
		t.Errorf("wrong stored value. expect: %s, got: %s", extensionID.String(), got)
	}
}
