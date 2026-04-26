package pipecatcallhandler

import (
	"context"
	"testing"
	"time"
)

func Test_pipecatcallHandler_Ping(t *testing.T) {
	tests := []struct {
		name   string
		hostID string
	}{
		{"non-empty host id", "10.4.2.18"},
		{"empty host id (defensive)", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &pipecatcallHandler{hostID: tt.hostID}
			before := time.Now().UTC()

			res, err := h.Ping(context.Background())

			if err != nil {
				t.Fatalf("Ping returned error: %v", err)
			}
			if res == nil {
				t.Fatalf("Ping returned nil result")
			}
			if res.HostID != tt.hostID {
				t.Errorf("HostID = %q, want %q", res.HostID, tt.hostID)
			}
			if res.Timestamp.Before(before) {
				t.Errorf("Timestamp %v is before test start %v", res.Timestamp, before)
			}
			if time.Since(res.Timestamp) > 5*time.Second {
				t.Errorf("Timestamp %v is too old", res.Timestamp)
			}
		})
	}
}
