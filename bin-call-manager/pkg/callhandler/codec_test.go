package callhandler

import (
	"testing"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/common"
)

func Test_embedCustomerCodecs(t *testing.T) {
	tests := []struct {
		name           string
		metadata       map[string]any
		outboundCodecs string
		expectCodecs   string
		expectSet      bool
	}{
		{"sets from customer when metadata empty", map[string]any{}, "PCMU,PCMA,G729", "PCMU,PCMA,G729", true},
		{"per-call override wins", map[string]any{call.MetadataKeyCodecs: "G722"}, "PCMU,PCMA", "G722", true},
		{"empty customer value — key not added", map[string]any{}, "", "", false},
		{"nil metadata with customer value — creates map", nil, "PCMU", "PCMU", true},
		{"nil metadata with empty customer value — no key added", nil, "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := embedCustomerCodecs(tt.metadata, tt.outboundCodecs)
			val, present := got[call.MetadataKeyCodecs]
			if present != tt.expectSet {
				t.Errorf("Key presence: got %v, expected %v", present, tt.expectSet)
			}
			if tt.expectSet {
				if s, ok := val.(string); !ok || s != tt.expectCodecs {
					t.Errorf("Codec value: got %v, expected %q", val, tt.expectCodecs)
				}
			}
		})
	}
}

func Test_setChannelVariableCodecs(t *testing.T) {
	headerKey := "PJSIP_HEADER(add," + common.SIPHeaderCodecs + ")"
	tests := []struct {
		name         string
		metadata     map[string]any
		expectHeader string
		expectSet    bool
	}{
		{"adds header when codecs set", map[string]any{call.MetadataKeyCodecs: "PCMU,PCMA,G729"}, "PCMU,PCMA,G729", true},
		{"no header when codecs key absent", map[string]any{}, "", false},
		{"no header when codecs value is empty string", map[string]any{call.MetadataKeyCodecs: ""}, "", false},
		{"CRLF in value rejected", map[string]any{call.MetadataKeyCodecs: "PCMU\r\nX-Inject: evil"}, "", false},
		{"CR alone in value rejected", map[string]any{call.MetadataKeyCodecs: "PCMU\rPCMA"}, "", false},
		{"LF alone in value rejected", map[string]any{call.MetadataKeyCodecs: "PCMU\nPCMA"}, "", false},
		{"non-string metadata value — no header", map[string]any{call.MetadataKeyCodecs: 42}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variables := map[string]string{}
			setChannelVariableCodecs(variables, tt.metadata)
			val, present := variables[headerKey]
			if present != tt.expectSet {
				t.Errorf("Header presence: got %v, expected %v. variables=%v", present, tt.expectSet, variables)
			}
			if tt.expectSet && val != tt.expectHeader {
				t.Errorf("Header value: got %q, expected %q", val, tt.expectHeader)
			}
		})
	}
}
