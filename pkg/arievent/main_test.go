package arievent

import "testing"

func TestGetTech(t *testing.T) {
	type test struct {
		name     string
		testName string
		expect   string
	}

	tests := []test{
		{
			"pjsip normal",
			"PJSIP/in-voipbin-000002f6",
			"pjsip",
		},
		{
			"sip normal",
			"SIP/in-voipbin-00000006",
			"sip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := getTech(tt.testName)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}
