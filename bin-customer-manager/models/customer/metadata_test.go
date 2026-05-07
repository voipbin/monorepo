package customer

import (
	"encoding/json"
	"testing"
)

func TestMetadata_RTPDebug_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect Metadata
	}{
		{
			"rtp_debug true",
			`{"rtp_debug":true}`,
			Metadata{RTPDebug: true},
		},
		{
			"rtp_debug false",
			`{"rtp_debug":false}`,
			Metadata{RTPDebug: false},
		},
		{
			"rtp_debug absent (zero value)",
			`{}`,
			Metadata{RTPDebug: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Metadata
			if err := json.Unmarshal([]byte(tt.input), &got); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if got != tt.expect {
				t.Errorf("Got %+v, expected %+v", got, tt.expect)
			}

			b, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			var got2 Metadata
			if err := json.Unmarshal(b, &got2); err != nil {
				t.Fatalf("Second unmarshal failed: %v", err)
			}
			if got2 != tt.expect {
				t.Errorf("Round-trip mismatch. Got %+v, expected %+v", got2, tt.expect)
			}
		})
	}
}
