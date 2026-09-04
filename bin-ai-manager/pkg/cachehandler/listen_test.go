package cachehandler

import (
	"testing"

	uuid "github.com/gofrs/uuid"
)

// Test_listenKeys pins every listen Redis key format. These keys are the
// contract between the intake path (which resolves ownership by SMEMBERS on a
// key built from a transcribe id) and the listen lifecycle (which writes and
// removes membership on the same key). A format drift between the two would
// silently stop every transcript segment from ever being matched -- with no
// error and no metric, since "not one of ours" is the overwhelmingly common
// case the intake path is designed to drop cheaply.
func Test_listenKeys(t *testing.T) {
	transcribeID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	aicallID := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa")

	tests := []struct {
		name   string
		got    string
		expect string
	}{
		{"transcribe resolver set", listenTranscribeKey(transcribeID), "ai:listen:transcribe:11111111-2222-3333-4444-555555555555"},
		{"pending buffer list", listenPendingKey(aicallID), "ai:listen:pending:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		{"rolling window list", listenWindowKey(aicallID), "ai:listen:window:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		{"debounce lock", listenLockKey(aicallID), "ai:listen:lock:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		{"turn counter", listenTurnsKey(aicallID), "ai:listen:turns:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		{"turn pipecatcall id set", listenTurnPipecatcallIDKey(aicallID), "ai:listen:turnpcid:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		// The start lock's key format is built in exactly ONE place, inside
		// ListenStartLockAcquire/Release (design §5.2.2, review round 16
		// finding LOW-6). Pinning it here is what keeps a future caller from
		// re-deriving it inline and drifting.
		{"start lock", listenStartLockKey(aicallID), "ai:listen:startlock:66666666-7777-8888-9999-aaaaaaaaaaaa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expect {
				t.Errorf("key mismatch.\nexpected: %s\ngot:      %s", tt.expect, tt.got)
			}
		})
	}
}

// Test_listenPendingPopMax pins the atomic-drain bound. LPOP key count (Redis
// >= 6.2) is what makes draining the pending buffer a single atomic command, so
// a concurrent appender can never lose a line between a read and a trim. The
// count argument is REQUIRED by go-redis v8's LPopCount, so there must be a
// bound; it exists to cap one turn's context, not to be tuned.
func Test_listenPendingPopMax(t *testing.T) {
	if listenPendingPopMax != 500 {
		t.Errorf("listenPendingPopMax mismatch. expected: 500, got: %d", listenPendingPopMax)
	}
}
