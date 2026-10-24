package customer

import "testing"

func Test_IdentityVerificationStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status IdentityVerificationStatus
		expect bool
	}{
		{"none", IdentityVerificationStatusNone, true},
		{"pending", IdentityVerificationStatusPending, true},
		{"verified", IdentityVerificationStatusVerified, true},
		{"rejected", IdentityVerificationStatusRejected, true},
		{"empty", IdentityVerificationStatus(""), false},
		{"undefined", IdentityVerificationStatus("approved"), false},
		{"case differs", IdentityVerificationStatus("Verified"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.expect {
				t.Errorf("Wrong match. status: %q, expect: %v, got: %v", tt.status, tt.expect, got)
			}
		})
	}
}
