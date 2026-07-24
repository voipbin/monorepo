package contacthandler

// Regression test for D5: crmIneligiblePeerTypes' doc-comment states
// "these types can never legitimately resolve to a contact... so they
// would otherwise sit in the unresolved queue forever" and "must not be
// created in the first place" -- but TypeNone ("") satisfied exactly
// that rule while being absent from the map. deriveEndpoints returns a
// zero commonaddress.Address (Type == TypeNone) whenever a webhook event
// carries an empty or unrecognized direction value, so before this fix
// an unknown-direction event silently created a permanently unresolved
// Interaction row.

import (
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
)

func Test_isCRMEligiblePeer(t *testing.T) {
	tests := []struct {
		name     string
		peerType commonaddress.Type
		want     bool
	}{
		{"tel is eligible", commonaddress.TypeTel, true},
		{"email is eligible", commonaddress.TypeEmail, true},
		{"line is eligible", commonaddress.TypeLine, true},
		{"whatsapp is eligible", commonaddress.TypeWhatsApp, true},
		{"none is ineligible (D5 fix)", commonaddress.TypeNone, false},
		{"agent is ineligible", commonaddress.TypeAgent, false},
		{"ai is ineligible", commonaddress.TypeAI, false},
		{"ai_team is ineligible", commonaddress.TypeAITeam, false},
		{"conference is ineligible", commonaddress.TypeConference, false},
		{"extension is ineligible", commonaddress.TypeExtension, false},
		{"sip is ineligible", commonaddress.TypeSIP, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCRMEligiblePeer(tt.peerType)
			if got != tt.want {
				t.Errorf("isCRMEligiblePeer(%q) = %v, want %v", tt.peerType, got, tt.want)
			}
		})
	}
}

// Test_deriveEndpoints_UnknownDirection_YieldsIneligiblePeer verifies the
// exact failure mode D5 fixes: an unrecognized/empty direction value
// yields a peer whose Type is TypeNone, which isCRMEligiblePeer must now
// reject. deriveEndpoints itself moved to the shared
// commonaddress.DeriveEndpoints authority (design doc §6); this test now
// exercises that shared function through the same call-site contract.
func Test_deriveEndpoints_UnknownDirection_YieldsIneligiblePeer(t *testing.T) {
	src := commonaddress.Address{Type: commonaddress.TypeTel, Target: "src"}
	dst := commonaddress.Address{Type: commonaddress.TypeTel, Target: "dst"}

	for _, direction := range []string{"", "unknown", "sideways"} {
		peer, local := commonaddress.DeriveEndpoints(direction, src, dst)
		if peer.Type != commonaddress.TypeNone {
			t.Fatalf("DeriveEndpoints(%q) peer.Type = %q, want TypeNone", direction, peer.Type)
		}
		if local.Type != commonaddress.TypeNone {
			t.Fatalf("DeriveEndpoints(%q) local.Type = %q, want TypeNone", direction, local.Type)
		}
		if isCRMEligiblePeer(peer.Type) {
			t.Errorf("isCRMEligiblePeer(peer from direction=%q) = true, want false (D5 regression)", direction)
		}
	}
}
