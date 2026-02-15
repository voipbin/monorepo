package subscribehandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/pkg/agenthandler"
)

func Test_processEventCMGroupcallCreated(t *testing.T) {
	tests := []struct {
		name string

		event *sock.Event

		expectErr bool
	}{
		{
			name: "normal",

			event: &sock.Event{
				Type: "groupcall_created",
				Data: []byte(`{"id":"69434cfa-79a4-11ec-a7b1-6ba5b7016d83","customer_id":"91aed1d4-7fe2-11ec-848d-97c8e986acfc","status":"ring"}`),
			},

			expectErr: false,
		},
		{
			name: "unmarshal error",

			event: &sock.Event{
				Type: "groupcall_created",
				Data: []byte(`invalid json`),
			},

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &subscribeHandler{
				agentHandler: mockAgent,
			}
			ctx := context.Background()

			if !tt.expectErr {
				mockAgent.EXPECT().EventGroupcallCreated(gomock.Any(), gomock.Any()).Return(nil)
			}

			err := h.processEventCMGroupcallCreated(ctx, tt.event)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong error match. expect error: %v, got error: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_processEventCMGroupcallProgressing(t *testing.T) {
	tests := []struct {
		name string

		event *sock.Event

		expectErr bool
	}{
		{
			name: "normal",

			event: &sock.Event{
				Type: "groupcall_progressing",
				Data: []byte(`{"id":"69434cfa-79a4-11ec-a7b1-6ba5b7016d83","customer_id":"91aed1d4-7fe2-11ec-848d-97c8e986acfc","status":"ring"}`),
			},

			expectErr: false,
		},
		{
			name: "unmarshal error",

			event: &sock.Event{
				Type: "groupcall_progressing",
				Data: []byte(`invalid json`),
			},

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAgent := agenthandler.NewMockAgentHandler(mc)
			h := &subscribeHandler{
				agentHandler: mockAgent,
			}
			ctx := context.Background()

			if !tt.expectErr {
				mockAgent.EXPECT().EventGroupcallProgressing(gomock.Any(), gomock.Any()).Return(nil)
			}

			err := h.processEventCMGroupcallProgressing(ctx, tt.event)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong error match. expect error: %v, got error: %v", tt.expectErr, err)
			}
		})
	}
}
