package address

import (
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		address Address
		wantErr bool
	}{
		// TypeTel
		{"tel valid min", Address{Type: TypeTel, Target: "+1234567"}, false},
		{"tel valid max", Address{Type: TypeTel, Target: "+123456789012345"}, false},
		{"tel valid us", Address{Type: TypeTel, Target: "+14155551234"}, false},
		{"tel valid kr", Address{Type: TypeTel, Target: "+821012345678"}, false},
		{"tel missing plus", Address{Type: TypeTel, Target: "14155551234"}, true},
		{"tel too short", Address{Type: TypeTel, Target: "+123456"}, true},
		{"tel too long", Address{Type: TypeTel, Target: "+1234567890123456"}, true},
		{"tel with letters", Address{Type: TypeTel, Target: "+1415555abcd"}, true},
		{"tel empty", Address{Type: TypeTel, Target: ""}, true},

		// TypeEmail
		{"email valid", Address{Type: TypeEmail, Target: "user@example.com"}, false},
		{"email valid with name", Address{Type: TypeEmail, Target: "User <user@example.com>"}, false},
		{"email missing at", Address{Type: TypeEmail, Target: "userexample.com"}, true},
		{"email missing domain", Address{Type: TypeEmail, Target: "user@"}, true},
		{"email empty", Address{Type: TypeEmail, Target: ""}, true},

		// TypeSIP
		{"sip valid", Address{Type: TypeSIP, Target: "user@example.com"}, false},
		{"sip valid with port", Address{Type: TypeSIP, Target: "user@example.com:5060"}, false},
		{"sip missing at", Address{Type: TypeSIP, Target: "userexample.com"}, true},
		{"sip missing user", Address{Type: TypeSIP, Target: "@example.com"}, true},
		{"sip missing domain", Address{Type: TypeSIP, Target: "user@"}, true},
		{"sip empty", Address{Type: TypeSIP, Target: ""}, true},

		// TypeAgent (UUID)
		{"agent valid", Address{Type: TypeAgent, Target: "a04a1f51-2495-48a5-9012-8081aa90b902"}, false},
		{"agent invalid uuid", Address{Type: TypeAgent, Target: "not-a-uuid"}, true},
		{"agent empty", Address{Type: TypeAgent, Target: ""}, true},

		// TypeConference (UUID)
		{"conference valid", Address{Type: TypeConference, Target: "34613ee5-5456-40fe-bb3b-395254270a9d"}, false},
		{"conference invalid", Address{Type: TypeConference, Target: "invalid"}, true},

		// TypeLine (UUID)
		{"line valid", Address{Type: TypeLine, Target: "07d16b0a-302f-4db8-ae4a-a2c9a65f88b7"}, false},
		{"line invalid", Address{Type: TypeLine, Target: "invalid"}, true},

		// TypeExtension (UUID)
		{"extension valid", Address{Type: TypeExtension, Target: "c5e7f18c-fc5a-4520-8326-e534e2ca0b8f"}, false},
		{"extension invalid", Address{Type: TypeExtension, Target: "2000"}, true},

		// TypeNone
		{"none empty", Address{Type: TypeNone, Target: ""}, false},
		{"none with target", Address{Type: TypeNone, Target: "anything"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.address.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTarget(t *testing.T) {
	tests := []struct {
		name        string
		addressType Type
		target      string
		wantErr     bool
	}{
		// TypeTel
		{"tel valid", TypeTel, "+14155551234", false},
		{"tel invalid", TypeTel, "14155551234", true},

		// TypeEmail
		{"email valid", TypeEmail, "user@example.com", false},
		{"email invalid", TypeEmail, "invalid", true},

		// TypeSIP
		{"sip valid", TypeSIP, "user@domain.com", false},
		{"sip invalid", TypeSIP, "nodomain", true},

		// UUID types
		{"agent valid", TypeAgent, "a04a1f51-2495-48a5-9012-8081aa90b902", false},
		{"conference valid", TypeConference, "34613ee5-5456-40fe-bb3b-395254270a9d", false},
		{"line valid", TypeLine, "07d16b0a-302f-4db8-ae4a-a2c9a65f88b7", false},
		{"extension valid", TypeExtension, "c5e7f18c-fc5a-4520-8326-e534e2ca0b8f", false},

		// TypeNone
		{"none", TypeNone, "", false},

		// Unknown type
		{"unknown type", Type("unknown"), "target", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTarget(tt.addressType, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTarget() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
