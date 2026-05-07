package outboundconfighandler

import (
	"strings"
	"testing"

	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "go.uber.org/mock/gomock"
)

func Test_validateWhitelist(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		wantErr bool
	}{
		{
			name:    "empty list is valid",
			entries: []string{},
			wantErr: false,
		},
		{
			name:    "valid lowercase codes",
			entries: []string{"us", "gb", "kr"},
			wantErr: false,
		},
		{
			name:    "valid uppercase codes - normalised to lowercase",
			entries: []string{"US"},
			wantErr: false,
		},
		{
			name:    "duplicate after normalisation",
			entries: []string{"us", "US"},
			wantErr: true,
		},
		{
			name:    "invalid country code",
			entries: []string{"xx"},
			wantErr: true,
		},
		{
			name:    "empty string entry",
			entries: []string{""},
			wantErr: true,
		},
		{
			name:    "whitespace-only entry",
			entries: []string{"  "},
			wantErr: true,
		},
		{
			name:    "mixed valid and invalid",
			entries: []string{"us", "xx"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWhitelist(tt.entries)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWhitelist(%v) error = %v, wantErr %v", tt.entries, err, tt.wantErr)
			}
		})
	}
}

func Test_outboundConfigHandler_validateUpdateRequest(t *testing.T) {
	validWL := []string{"us", "gb"}
	invalidWL := []string{"xx"}
	validCodecs := "PCMU,G729"
	validHyphenCodecs := "PCMU,PCMA,G729,AMR,AMR-WB,GSM,GSM-EFR,GSM-HR-08,opus,telephone-event"
	invalidCodecs := "PCMU;G729"                       // semicolon not allowed
	invalidLeadingHyphenCodecs := "-PCMU"              // token must start with alnum
	invalidTrailingHyphenCodecs := "PCMU-"             // token must end with alnum
	invalidDoubleHyphenCodecs := "AMR--WB"             // hyphens may not be adjacent
	invalidSecondTokenLeadingHyphen := "PCMU,-AMR"     // second token may not start with hyphen
	emptyCodecs := ""

	// 255-char boundary: build a comma-separated string of exactly 255 chars and one of 256.
	// Each "PCMU," is 5 bytes; 51 copies = 255, 51+"P" = 256, 52 copies (with trailing trim) etc.
	// Use exact lengths so the boundary check is deterministic.
	codecsAt255 := strings.Repeat("PCMU,", 50) + "PCMU"        // 5*50 + 4 = 254 — adjust to 255
	codecsAt255 = codecsAt255 + "U"                            // 255
	codecsAt256 := codecsAt255 + "U"                           // 256
	if len(codecsAt255) != 255 || len(codecsAt256) != 256 {
		t.Fatalf("test setup error: codecsAt255=%d, codecsAt256=%d", len(codecsAt255), len(codecsAt256))
	}

	tests := []struct {
		name    string
		req     *outboundconfig.UpdateRequest
		wantErr bool
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: false,
		},
		{
			name:    "valid whitelist",
			req:     &outboundconfig.UpdateRequest{DestinationWhitelist: &validWL},
			wantErr: false,
		},
		{
			name:    "invalid whitelist",
			req:     &outboundconfig.UpdateRequest{DestinationWhitelist: &invalidWL},
			wantErr: true,
		},
		{
			name:    "valid codecs",
			req:     &outboundconfig.UpdateRequest{Codecs: &validCodecs},
			wantErr: false,
		},
		{
			name:    "invalid codecs - semicolon separator",
			req:     &outboundconfig.UpdateRequest{Codecs: &invalidCodecs},
			wantErr: true,
		},
		{
			name:    "valid codecs with hyphens (AMR-WB, GSM-EFR, telephone-event)",
			req:     &outboundconfig.UpdateRequest{Codecs: &validHyphenCodecs},
			wantErr: false,
		},
		{
			name:    "invalid codecs - leading hyphen",
			req:     &outboundconfig.UpdateRequest{Codecs: &invalidLeadingHyphenCodecs},
			wantErr: true,
		},
		{
			name:    "invalid codecs - trailing hyphen",
			req:     &outboundconfig.UpdateRequest{Codecs: &invalidTrailingHyphenCodecs},
			wantErr: true,
		},
		{
			name:    "invalid codecs - double hyphen",
			req:     &outboundconfig.UpdateRequest{Codecs: &invalidDoubleHyphenCodecs},
			wantErr: true,
		},
		{
			name:    "invalid codecs - second token leading hyphen",
			req:     &outboundconfig.UpdateRequest{Codecs: &invalidSecondTokenLeadingHyphen},
			wantErr: true,
		},
		{
			name:    "valid codecs at 255-char boundary",
			req:     &outboundconfig.UpdateRequest{Codecs: &codecsAt255},
			wantErr: false,
		},
		{
			name:    "invalid codecs at 256-char boundary",
			req:     &outboundconfig.UpdateRequest{Codecs: &codecsAt256},
			wantErr: true,
		},
		{
			name:    "empty codecs string - server default",
			req:     &outboundconfig.UpdateRequest{Codecs: &emptyCodecs},
			wantErr: false,
		},
		{
			name: "valid whitelist and codecs together",
			req: &outboundconfig.UpdateRequest{
				DestinationWhitelist: &validWL,
				Codecs:               &validCodecs,
			},
			wantErr: false,
		},
		{
			name: "valid whitelist but invalid codecs",
			req: &outboundconfig.UpdateRequest{
				DestinationWhitelist: &validWL,
				Codecs:               &invalidCodecs,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &outboundConfigHandler{
				utilHandler:  utilhandler.NewMockUtilHandler(mc),
				db:           dbhandler.NewMockDBHandler(mc),
				cacheHandler: cachehandler.NewMockCacheHandler(mc),
			}

			err := h.validateUpdateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
