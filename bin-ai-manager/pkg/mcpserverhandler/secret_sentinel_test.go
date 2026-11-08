package mcpserverhandler

import (
	"strings"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
)

// Test_NewMcpServerHandler_errorHoldsNoKeyText verifies that the error of a malformed key list does
// not repeat any part of the list: the caller logs it, and an entry without a version prefix is the
// key itself.
func Test_NewMcpServerHandler_errorHoldsNoKeyText(t *testing.T) {
	tests := []struct {
		name string
		keys string
	}{
		{"entry without a colon", "SENTINELNOCOLON"},
		{"version is not a number", "SENTINELVER:AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="},
		{"key is not base64", "1:SENTINELB64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res, err := NewMcpServerHandler(notifyhandler.NewMockNotifyHandler(mc), dbhandler.NewMockDBHandler(mc), tt.keys, mcpserver.DefaultCarrierTable())
			if err == nil || res != nil {
				t.Fatalf("expected an error and a nil handler, got %v / %v", res, err)
			}
			if strings.Contains(err.Error(), "SENTINEL") {
				t.Errorf("the error holds the key list text: %v", err)
			}
		})
	}
}

func Test_NewMcpServerHandler_blankEntriesMeanNoKey(t *testing.T) {
	for _, keys := range []string{"", "  ", ",", " , ,"} {
		mc := gomock.NewController(t)

		res, err := NewMcpServerHandler(notifyhandler.NewMockNotifyHandler(mc), dbhandler.NewMockDBHandler(mc), keys, mcpserver.DefaultCarrierTable())
		if err != nil || res == nil {
			t.Errorf("a list of blank entries (%q) must build a handler without a key, got %v / %v", keys, res, err)
		}
		mc.Finish()
	}
}
