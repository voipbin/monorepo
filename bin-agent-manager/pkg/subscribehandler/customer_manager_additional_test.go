package subscribehandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/pkg/agenthandler"
)

func Test_processEventCMCustomerDeleted(t *testing.T) {
	tests := []struct {
		name string

		event *sock.Event

		expectErr bool
	}{
		{
			name: "normal",

			event: &sock.Event{
				Type: "customer_deleted",
				Data: []byte(`{"id":"69434cfa-79a4-11ec-a7b1-6ba5b7016d83","name":"test customer"}`),
			},

			expectErr: false,
		},
		{
			name: "unmarshal error",

			event: &sock.Event{
				Type: "customer_deleted",
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
				mockAgent.EXPECT().EventCustomerDeleted(gomock.Any(), gomock.Any()).Return(nil)
			}

			err := h.processEventCMCustomerDeleted(ctx, tt.event)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong error match. expect error: %v, got error: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_processEventCMCustomerCreated(t *testing.T) {
	tests := []struct {
		name string

		event *sock.Event

		expectHeadless bool
		expectErr      bool
	}{
		{
			name: "normal - no headless field defaults to false",

			event: &sock.Event{
				Type: "customer_created",
				Data: []byte(`{"id":"69434cfa-79a4-11ec-a7b1-6ba5b7016d83","name":"test customer"}`),
			},

			expectHeadless: false,
			expectErr:      false,
		},
		{
			name: "headless true",

			event: &sock.Event{
				Type: "customer_created",
				Data: []byte(`{"id":"69434cfa-79a4-11ec-a7b1-6ba5b7016d83","name":"test customer","headless":true}`),
			},

			expectHeadless: true,
			expectErr:      false,
		},
		{
			name: "headless false explicitly",

			event: &sock.Event{
				Type: "customer_created",
				Data: []byte(`{"id":"69434cfa-79a4-11ec-a7b1-6ba5b7016d83","name":"test customer","headless":false}`),
			},

			expectHeadless: false,
			expectErr:      false,
		},
		{
			name: "unmarshal error",

			event: &sock.Event{
				Type: "customer_created",
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
				mockAgent.EXPECT().EventCustomerCreated(gomock.Any(), gomock.Any(), tt.expectHeadless).Return(nil)
			}

			err := h.processEventCMCustomerCreated(ctx, tt.event)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong error match. expect error: %v, got error: %v", tt.expectErr, err)
			}
		})
	}
}
