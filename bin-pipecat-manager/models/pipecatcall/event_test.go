package pipecatcall

import "testing"

func TestEventTypePipecatcallTerminated_value(t *testing.T) {
	if EventTypePipecatcallTerminated != "pipecatcall_terminated" {
		t.Fatalf("expected pipecatcall_terminated, got %q", EventTypePipecatcallTerminated)
	}
}
