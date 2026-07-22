package address

import (
	"errors"
	"testing"
)

func TestNormalizeTarget(t *testing.T) {
	tests := []struct {
		name        string
		addressType Type
		target      string
		expect      string
		expectErr   error // nil, ErrNotNormalizable, or ErrUnknownType
	}{
		// tel — canonicalizable
		{"tel punctuation", TypeTel, "+1 (555) 123-4567", "+15551234567", nil},
		{"tel idempotent", TypeTel, "+15551234567", "+15551234567", nil},
		{"tel dashes and spaces", TypeTel, "  +1-555-123-4567  ", "+15551234567", nil},
		{"tel leading ws keeps plus", TypeTel, "  +1555000  ", "+1555000", nil},
		{"tel plus not index 0 dropped", TypeTel, "00+15551230", "0015551230", nil},
		{"tel double plus", TypeTel, "++15551230", "+15551230", nil},
		{"tel arabic-indic digit stripped", TypeTel, "+1\u0665\u0665\u0665123", "+1123", nil},
		{"tel fullwidth digit stripped", TypeTel, "+1\uff15\uff15123", "+1123", nil},
		{"tel internal plus", TypeTel, "+15+55", "+1555", nil},
		{"tel tab newline ws", TypeTel, "\t+1555\n123\n", "+1555123", nil},
		// tel — NOT canonicalizable (loss-proof: original preserved + ErrNotNormalizable)
		{"tel anonymous sentinel", TypeTel, "anonymous", "anonymous", ErrNotNormalizable},
		{"tel restricted sentinel", TypeTel, "Restricted", "Restricted", ErrNotNormalizable},
		{"tel lone plus", TypeTel, "+", "+", ErrNotNormalizable},
		{"tel empty", TypeTel, "", "", ErrNotNormalizable},
		{"tel letters only", TypeTel, "abc", "abc", ErrNotNormalizable},

		// whatsapp (reuses tel)
		{"whatsapp spaces", TypeWhatsApp, "+1 555 123 4567", "+15551234567", nil},
		{"whatsapp idempotent", TypeWhatsApp, "+15551234567", "+15551234567", nil},
		{"whatsapp waID no plus preserved", TypeWhatsApp, "15551234567", "15551234567", nil},
		{"whatsapp alphanumeric sender", TypeWhatsApp, "VOIPBIN", "VOIPBIN", ErrNotNormalizable},
		{"whatsapp empty", TypeWhatsApp, "", "", ErrNotNormalizable},

		// email — lossless (always nil)
		{"email trim lower", TypeEmail, "  John@Example.COM ", "john@example.com", nil},
		{"email idempotent", TypeEmail, "john@example.com", "john@example.com", nil},
		{"email display name", TypeEmail, "Bob <Bob@Host.COM>", "bob <bob@host.com>", nil},
		{"email uppercase local", TypeEmail, "JOHN@example.com", "john@example.com", nil},

		// sip — lossless (always nil)
		{"sip host lower", TypeSIP, "User@Example.COM", "User@example.com", nil},
		{"sip with port", TypeSIP, "Alice@Host.com:5060", "Alice@host.com:5060", nil},
		{"sip transport param", TypeSIP, "Alice@Host.com;transport=TCP", "Alice@host.com;transport=TCP", nil},
		{"sip maddr param value preserved", TypeSIP, "Alice@Host.com;maddr=Relay.Example", "Alice@host.com;maddr=Relay.Example", nil},
		{"sip userinfo password preserved", TypeSIP, "user:Pass@Host.com", "user:Pass@host.com", nil},
		{"sip ipv6 host", TypeSIP, "Alice@[2001:DB8::1]:5060", "Alice@[2001:db8::1]:5060", nil},
		{"sip at inside header", TypeSIP, "Alice@Host.com?Replaces=Call@ID", "Alice@host.com?Replaces=Call@ID", nil},
		{"sip whole input trim", TypeSIP, " User@Host.com ", "User@host.com", nil},
		{"sip no at", TypeSIP, "nobody", "nobody", nil},
		{"sip empty", TypeSIP, "", "", nil},

		// identity types — always nil, unchanged
		{"line identity", TypeLine, "07d16b0a-302f-4db8-ae4a-a2c9a65f88b7", "07d16b0a-302f-4db8-ae4a-a2c9a65f88b7", nil},
		{"agent identity", TypeAgent, "a04a1f51-2495-48a5-9012-8081aa90b902", "a04a1f51-2495-48a5-9012-8081aa90b902", nil},
		{"ai identity", TypeAI, "some-ai-id", "some-ai-id", nil},
		{"ai_team identity", TypeAITeam, "some-team-id", "some-team-id", nil},
		{"conference identity", TypeConference, "34613ee5-5456-40fe-bb3b-395254270a9d", "34613ee5-5456-40fe-bb3b-395254270a9d", nil},
		{"extension identity", TypeExtension, "2000", "2000", nil},
		{"none identity", TypeNone, "anything", "anything", nil},
		{"webchat identity", TypeWebchat, "07d16b0a-302f-4db8-ae4a-a2c9a65f88b7", "07d16b0a-302f-4db8-ae4a-a2c9a65f88b7", nil},
		{"web_session identity", TypeWebSession, "07d16b0a-302f-4db8-ae4a-a2c9a65f88b7", "07d16b0a-302f-4db8-ae4a-a2c9a65f88b7", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeTarget(tt.addressType, tt.target)
			if got != tt.expect {
				t.Errorf("NormalizeTarget(%q, %q) = %q, want %q", tt.addressType, tt.target, got, tt.expect)
			}
			if tt.expectErr == nil {
				if err != nil {
					t.Errorf("NormalizeTarget(%q, %q) unexpected error: %v", tt.addressType, tt.target, err)
				}
			} else if !errors.Is(err, tt.expectErr) {
				t.Errorf("NormalizeTarget(%q, %q) error = %v, want errors.Is %v", tt.addressType, tt.target, err, tt.expectErr)
			}

			// Loss-proof property: result is either the canonical form or the
			// original input — never a blanked meaningful value. For a non-empty
			// input, the result must be non-empty UNLESS the input was only
			// whitespace fed to a lossless trim (email/sip).
			if tt.target != "" && got == "" && tt.addressType != TypeEmail && tt.addressType != TypeSIP {
				t.Errorf("NormalizeTarget(%q, %q) blanked a non-empty input", tt.addressType, tt.target)
			}

			// Idempotency: value AND error class are stable on a second apply.
			again, againErr := NormalizeTarget(tt.addressType, got)
			if again != got {
				t.Errorf("NormalizeTarget not value-idempotent for %q: first=%q second=%q", tt.addressType, got, again)
			}
			// Error class must match between the two passes for the SAME value.
			// (Note: the second pass runs on the already-canonical value, so a
			// once-canonicalizable tel like "+15551234567" yields nil on pass 2;
			// a never-canonicalizable tel like "anonymous" yields
			// ErrNotNormalizable on both passes because the value is unchanged.)
			if (err == nil) != (againErr == nil) && got == tt.target {
				t.Errorf("NormalizeTarget error-class not idempotent for unchanged value %q: first=%v second=%v", got, err, againErr)
			}
		})
	}
}

func TestNormalizeTarget_UnknownType(t *testing.T) {
	got, err := NormalizeTarget(Type("unknown"), "some-value")
	if !errors.Is(err, ErrUnknownType) {
		t.Errorf("NormalizeTarget(unknown) error = %v, want errors.Is ErrUnknownType", err)
	}
	// Loss-proof: original value preserved, NOT blanked.
	if got != "some-value" {
		t.Errorf("NormalizeTarget(unknown) result = %q, want original %q", got, "some-value")
	}
}
