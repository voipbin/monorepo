package address

import "testing"

func Test_IsExternalEndpoint(t *testing.T) {
	tests := []struct {
		ty   Type
		want bool
	}{
		{TypeTel, true},
		{TypeSIP, true},
		{TypeEmail, true},
		{TypeLine, true},
		{TypeWhatsApp, true},
		// platform resource ids
		{TypeAgent, false},
		{TypeAI, false},
		{TypeAITeam, false},
		{TypeConference, false},
		{TypeExtension, false},
		{TypeWebchat, false},
		{TypeWebSession, false},
		// no type and an unknown type are resource-like until decided otherwise
		{TypeNone, false},
		{Type("a_type_added_later"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.ty), func(t *testing.T) {
			if got := IsExternalEndpoint(tt.ty); got != tt.want {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.want, got)
			}
		})
	}
}
