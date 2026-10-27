package channel

import "testing"

func Test_GetTech_cov2(t *testing.T) {
	tests := []struct {
		name string

		channelName string

		expect Tech
	}{
		{name: "pjsip", channelName: "PJSIP/test-00000001", expect: TechPJSIP},
		{name: "snoop lowercase", channelName: "snoop/abc", expect: TechSnoop},
		{name: "unknown tech", channelName: "Unknown/abc", expect: TechNone},
		{name: "empty name", channelName: "", expect: TechNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := GetTech(tt.channelName); res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}
