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

func TestGetTS(t *testing.T) {
	type test struct {
		name          string
		testTimestamp string
		expect        string
	}

	tests := []test{
		{
			"normal",
			"2020-04-19T14:38:00.363+0000",
			"2020-04-19T14:38:00.363",
		},
		{
			"without +0000",
			"2020-04-19T14:38:00.363",
			"2020-04-19T14:38:00.363",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := getTS(tt.testTimestamp)
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}
