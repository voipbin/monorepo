package customer

import (
	"encoding/json"
	"testing"
)

func TestMetadata_OutboundCodecs_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect Metadata
	}{
		{
			"outbound_codecs set",
			`{"rtp_debug":false,"outbound_codecs":"PCMU,PCMA,G729"}`,
			Metadata{RTPDebug: false, OutboundCodecs: "PCMU,PCMA,G729"},
		},
		{
			"outbound_codecs empty",
			`{"rtp_debug":false,"outbound_codecs":""}`,
			Metadata{RTPDebug: false, OutboundCodecs: ""},
		},
		{
			"outbound_codecs absent (zero value)",
			`{"rtp_debug":true}`,
			Metadata{RTPDebug: true, OutboundCodecs: ""},
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
