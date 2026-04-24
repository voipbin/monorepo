package errors

import "testing"

func TestConstructors(t *testing.T) {
	tests := []struct {
		name       string
		build      func(string, string, string) *VoipbinError
		wantStatus Status
	}{
		{"invalid_argument", InvalidArgument, StatusInvalidArgument},
		{"unauthenticated", Unauthenticated, StatusUnauthenticated},
		{"payment_required", PaymentRequired, StatusPaymentRequired},
		{"permission_denied", PermissionDenied, StatusPermissionDenied},
		{"not_found", NotFound, StatusNotFound},
		{"already_exists", AlreadyExists, StatusAlreadyExists},
		{"failed_precondition", FailedPrecondition, StatusFailedPrecondition},
		{"resource_exhausted", ResourceExhausted, StatusResourceExhausted},
		{"unavailable", Unavailable, StatusUnavailable},
		{"internal", Internal, StatusInternal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tt.build("d", "R", "m")
			if e == nil {
				t.Fatal("nil VoipbinError")
			}
			if e.Status != tt.wantStatus {
				t.Errorf("wrong Status: got %q want %q", e.Status, tt.wantStatus)
			}
			if e.Domain != "d" || e.Reason != "R" || e.Message != "m" {
				t.Errorf("wrong fields: %+v", e)
			}
			if e.Cause != nil {
				t.Errorf("Cause should be nil by default")
			}
		})
	}
}
