package monitoringhandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"go.uber.org/mock/gomock"
)

func TestNewMonitoringHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates_new_monitoring_handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := NewMonitoringHandler(mockReq, mockNotify, mockUtil)

			if h == nil {
				t.Errorf("Expected non-nil handler, got nil")
			}
		})
	}
}
