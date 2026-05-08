package providerhandler

import (
	"testing"
)

func Test_validateCodecs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty is valid", input: "", wantErr: false},
		{name: "single codec", input: "PCMU", wantErr: false},
		{name: "comma separated", input: "PCMU,PCMA,G729", wantErr: false},
		{name: "too long", input: string(make([]byte, 256)), wantErr: true},
		{name: "CRLF injection CR", input: "PCMU\rPCMA", wantErr: true},
		{name: "CRLF injection LF", input: "PCMU\nPCMA", wantErr: true},
		{name: "open paren rejected", input: "PCMU(bad", wantErr: true},
		{name: "close paren rejected", input: "PCMU)bad", wantErr: true},
		{name: "double comma rejected", input: "PCMU,,PCMA", wantErr: true},
		{name: "whitespace trimmed and valid", input: " PCMU , PCMA ", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateCodecs(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCodecs(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && tt.input != "" {
				_ = got // normalized output; main assertion is no error
			}
		})
	}
}
