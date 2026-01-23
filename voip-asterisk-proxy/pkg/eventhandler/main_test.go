package eventhandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"
)

func TestNewEventHandler(t *testing.T) {
	tests := []struct {
		name string

		ariAddr         string
		ariAccount      string
		ariSubscribeAll string
		ariApplication  string
		amiEventFilter  []string
	}{
		{
			name: "creates_handler_with_all_parameters",

			ariAddr:         "localhost:8088",
			ariAccount:      "asterisk:asterisk",
			ariSubscribeAll: "true",
			ariApplication:  "voipbin",
			amiEventFilter:  []string{"Registry", "PeerStatus"},
		},
		{
			name: "creates_handler_with_minimal_parameters",

			ariAddr:         "127.0.0.1:8088",
			ariAccount:      "user:pass",
			ariSubscribeAll: "false",
			ariApplication:  "test",
			amiEventFilter:  []string{},
		},
		{
			name: "creates_handler_with_empty_strings",

			ariAddr:         "",
			ariAccount:      "",
			ariSubscribeAll: "",
			ariApplication:  "",
			amiEventFilter:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)

			h := NewEventHandler(
				mockNotify,
				mockSock,
				"asterisk.all.event",
				tt.ariAddr,
				tt.ariAccount,
				tt.ariSubscribeAll,
				tt.ariApplication,
				nil, // amiSock can be nil for unit testing
				tt.amiEventFilter,
			)

			if h == nil {
				t.Errorf("Expected non-nil handler, got nil")
			}
		})
	}
}

func TestEventHandler_Run(t *testing.T) {
	tests := []struct {
		name string

		expectErr bool
	}{
		{
			name: "runs_without_error",

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)

			h := NewEventHandler(
				mockNotify,
				mockSock,
				"asterisk.all.event",
				"localhost:8088",
				"asterisk:asterisk",
				"true",
				"voipbin",
				nil,
				[]string{},
			)

			err := h.Run()

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
