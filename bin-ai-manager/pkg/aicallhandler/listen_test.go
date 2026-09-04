package aicallhandler

import (
	"testing"

	"monorepo/bin-ai-manager/models/aicall"
	cmcall "monorepo/bin-call-manager/models/call"

	"github.com/gofrs/uuid"
)

// Test_isListenableCallStatus pins the exact status set transcribe-manager's
// own isValidReference accepts for a call reference. Diverging in either
// direction is a real defect: a status this returns true for but
// transcribe-manager rejects means a listen-start that always fails, and a
// status it rejects that transcribe-manager would accept means listening
// silently never starts on a perfectly valid call.
func Test_isListenableCallStatus(t *testing.T) {
	tests := []struct {
		name   string
		status cmcall.Status
		expect bool
	}{
		{"dialing is listenable", cmcall.StatusDialing, true},
		{"ringing is listenable", cmcall.StatusRinging, true},
		{"progressing is listenable", cmcall.StatusProgressing, true},
		{"hangup is not", cmcall.StatusHangup, false},
		{"terminating is not", cmcall.StatusTerminating, false},
		{"canceling is not", cmcall.StatusCanceling, false},
		{"empty is not", cmcall.Status(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isListenableCallStatus(tt.status); got != tt.expect {
				t.Errorf("isListenableCallStatus(%q) mismatch. expected: %v, got: %v", tt.status, tt.expect, got)
			}
		})
	}
}

// Test_listenTranscribeIDFromMetadata pins the metadata reader's tolerance.
// Metadata round-trips through JSON, so a value written as a uuid.UUID comes
// back as a string -- and anything unexpected must read as uuid.Nil rather
// than panic, since this runs on the AIcall of every listen precondition check.
func Test_listenTranscribeIDFromMetadata(t *testing.T) {
	valid := uuid.FromStringOrNil("aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee")

	tests := []struct {
		name   string
		c      *aicall.AIcall
		expect uuid.UUID
	}{
		{"nil metadata", &aicall.AIcall{}, uuid.Nil},
		{"absent key", &aicall.AIcall{Metadata: map[string]any{"other": "x"}}, uuid.Nil},
		{"wrong type", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenTranscribeID: 42}}, uuid.Nil},
		{"unparseable string", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenTranscribeID: "not-a-uuid"}}, uuid.Nil},
		{"well-formed value", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenTranscribeID: valid.String()}}, valid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenTranscribeIDFromMetadata(tt.c); got != tt.expect {
				t.Errorf("mismatch. expected: %s, got: %s", tt.expect, got)
			}
		})
	}
}

// Test_listenOwnsTranscribeFromMetadata pins the ownership reader. Reading a
// missing or wrong-typed value as TRUE would let a non-owner stop a transcribe
// session another Case is still listening to, so the default must be false.
func Test_listenOwnsTranscribeFromMetadata(t *testing.T) {
	tests := []struct {
		name   string
		c      *aicall.AIcall
		expect bool
	}{
		{"nil metadata", &aicall.AIcall{}, false},
		{"absent key", &aicall.AIcall{Metadata: map[string]any{"other": "x"}}, false},
		{"wrong type", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenOwnsTranscribe: "true"}}, false},
		{"false", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenOwnsTranscribe: false}}, false},
		{"true", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenOwnsTranscribe: true}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenOwnsTranscribeFromMetadata(tt.c); got != tt.expect {
				t.Errorf("mismatch. expected: %v, got: %v", tt.expect, got)
			}
		})
	}
}
