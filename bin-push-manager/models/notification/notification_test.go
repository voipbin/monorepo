package notification

import "testing"

func Test_IsValidKind(t *testing.T) {
	tests := []struct {
		name   string
		kind   Kind
		expect bool
	}{
		{"call_incoming", KindCallIncoming, true},
		{"call_cancelled", KindCallCancelled, true},
		{"unknown", Kind("chat_message"), false},
		{"empty", Kind(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidKind(tt.kind); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}
