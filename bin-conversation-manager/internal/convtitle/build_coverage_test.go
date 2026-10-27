package convtitle

import (
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-conversation-manager/models/conversation"
)

func Test_Build_webchatAndUnknownTypes(t *testing.T) {
	tests := []struct {
		name       string
		convType   conversation.Type
		peer       commonaddress.Address
		wantName   string
		wantDetail string
	}{
		{
			name:       "webchat uses the webchat label",
			convType:   conversation.TypeWebchat,
			peer:       commonaddress.Address{Type: commonaddress.TypeLine, Target: "visitor-1", TargetName: "Visitor"},
			wantName:   "Webchat · Visitor",
			wantDetail: "Webchat conversation",
		},
		{
			name:       "a type without a label falls back to the type string",
			convType:   conversation.Type("custom"),
			peer:       commonaddress.Address{Type: commonaddress.TypeLine, Target: "user-1"},
			wantName:   "custom · user-1",
			wantDetail: "custom conversation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, detail := Build(tt.convType, tt.peer)
			if name != tt.wantName {
				t.Errorf("Build() name = %q, want %q", name, tt.wantName)
			}
			if detail != tt.wantDetail {
				t.Errorf("Build() detail = %q, want %q", detail, tt.wantDetail)
			}
		})
	}
}
