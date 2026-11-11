package notification

import (
	"reflect"
	"testing"
)

func Test_IsValidKind(t *testing.T) {
	tests := []struct {
		name   string
		kind   Kind
		expect bool
	}{
		{"call_incoming", KindCallIncoming, true},
		{"call_cancelled", KindCallCancelled, true},
		{"chat_mentioned", KindChatMentioned, true},
		{"chat_message", KindChatMessage, true},
		{"conversation_message", KindConversationMessage, true},
		{"case_assigned", KindCaseAssigned, true},
		{"case_note", KindCaseNote, true},
		{"unknown", Kind("unknown_kind"), false},
		{"empty", Kind(""), false},
		{"wrong case", Kind("Chat_Message"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidKind(tt.kind); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

// The wire values are a contract with the apps.
func Test_kindValues(t *testing.T) {
	tests := map[Kind]string{
		KindCallIncoming:        "call_incoming",
		KindCallCancelled:       "call_cancelled",
		KindChatMentioned:       "chat_mentioned",
		KindChatMessage:         "chat_message",
		KindConversationMessage: "conversation_message",
		KindCaseAssigned:        "case_assigned",
		KindCaseNote:            "case_note",
	}
	for k, expect := range tests {
		if string(k) != expect {
			t.Errorf("Wrong match. expect: %s, got: %s", expect, k)
		}
	}
}

func Test_SettingsKinds(t *testing.T) {
	expect := []Kind{KindChatMentioned, KindChatMessage, KindConversationMessage, KindCaseAssigned, KindCaseNote}
	if got := SettingsKinds(); !reflect.DeepEqual(got, expect) {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, got)
	}

	// the returned list is a copy
	got := SettingsKinds()
	got[0] = KindCallIncoming
	if again := SettingsKinds(); again[0] != KindChatMentioned {
		t.Errorf("Wrong match. the list must be a copy. got: %v", again)
	}

	for _, k := range expect {
		if !IsSettingsKind(k) {
			t.Errorf("Wrong match. %s must be a settings kind", k)
		}
	}
	for _, k := range []Kind{KindCallIncoming, KindCallCancelled, Kind(""), Kind("unknown_kind")} {
		if IsSettingsKind(k) {
			t.Errorf("Wrong match. %s must not be a settings kind", k)
		}
	}
}

func Test_PolicyOf(t *testing.T) {
	tests := []struct {
		kind   Kind
		expect Policy
	}{
		{KindCallIncoming, Policy{Priority: "HIGH", TTL: "60s"}},
		{KindCallCancelled, Policy{Priority: "HIGH", TTL: "60s"}},
		{KindChatMentioned, Policy{Priority: "HIGH", TTL: "3600s"}},
		{KindConversationMessage, Policy{Priority: "HIGH", TTL: "3600s"}},
		{KindCaseAssigned, Policy{Priority: "HIGH", TTL: "3600s"}},
		{KindChatMessage, Policy{Priority: "NORMAL", TTL: "3600s"}},
		{KindCaseNote, Policy{Priority: "NORMAL", TTL: "3600s"}},
		// a kind without a policy gets the strictest one: wake the device, keep it a minute
		{Kind("unknown_kind"), Policy{Priority: "HIGH", TTL: "60s"}},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			if got := PolicyOf(tt.kind); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}
