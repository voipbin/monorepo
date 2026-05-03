package convtitle

import (
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-conversation-manager/models/conversation"
)

func Test_Build(t *testing.T) {
	tests := []struct {
		name       string
		convType   conversation.Type
		peer       commonaddress.Address
		wantName   string
		wantDetail string
	}{
		{
			name:     "sms with name and target",
			convType: conversation.TypeMessage,
			peer: commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+14155551234",
				TargetName: "Alice",
			},
			wantName:   "SMS · Alice (+14155551234)",
			wantDetail: "SMS conversation",
		},
		{
			name:     "sms with target only",
			convType: conversation.TypeMessage,
			peer: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+14155551234",
			},
			wantName:   "SMS · +14155551234",
			wantDetail: "SMS conversation",
		},
		{
			name:     "sms with neither",
			convType: conversation.TypeMessage,
			peer:     commonaddress.Address{Type: commonaddress.TypeTel},
			wantName:   "SMS · Unknown",
			wantDetail: "SMS conversation",
		},
		{
			name:     "line with name and opaque target",
			convType: conversation.TypeLine,
			peer: commonaddress.Address{
				Type:       commonaddress.TypeLine,
				Target:     "Uabcdef1234567890",
				TargetName: "Alice",
			},
			// Target is a LINE user ID — opaque, must NOT appear in name
			wantName:   "LINE · Alice",
			wantDetail: "LINE conversation",
		},
		{
			name:     "line with opaque target only",
			convType: conversation.TypeLine,
			peer: commonaddress.Address{
				Type:   commonaddress.TypeLine,
				Target: "Uabcdef1234567890",
			},
			// Target is opaque but it's the only info available, so it shows as fallback
			wantName:   "LINE · Uabcdef1234567890",
			wantDetail: "LINE conversation",
		},
		{
			name:     "line with neither",
			convType: conversation.TypeLine,
			peer:     commonaddress.Address{Type: commonaddress.TypeLine},
			wantName:   "LINE · Unknown",
			wantDetail: "LINE conversation",
		},
		{
			name:     "email with name and target",
			convType: conversation.TypeMessage,
			peer: commonaddress.Address{
				Type:       commonaddress.TypeEmail,
				Target:     "alice@example.com",
				TargetName: "Alice",
			},
			wantName:   "SMS · Alice (alice@example.com)",
			wantDetail: "SMS conversation",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotDetail := Build(tt.convType, tt.peer)
			if gotName != tt.wantName {
				t.Errorf("Build() name = %q, want %q", gotName, tt.wantName)
			}
			if gotDetail != tt.wantDetail {
				t.Errorf("Build() detail = %q, want %q", gotDetail, tt.wantDetail)
			}
		})
	}
}

func Test_humanReadableTarget(t *testing.T) {
	tests := []struct {
		addrType commonaddress.Type
		want     bool
	}{
		{commonaddress.TypeTel, true},
		{commonaddress.TypeEmail, true},
		{commonaddress.TypeSIP, true},
		{commonaddress.TypeExtension, true},
		{commonaddress.TypeLine, false},
		{commonaddress.TypeAgent, false},
		{commonaddress.TypeAI, false},
		{commonaddress.TypeConference, false},
		{commonaddress.TypeNone, false},
		{"unknown_future_type", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.addrType), func(t *testing.T) {
			if got := humanReadableTarget(tt.addrType); got != tt.want {
				t.Errorf("humanReadableTarget(%q) = %v, want %v", tt.addrType, got, tt.want)
			}
		})
	}
}
