package callhandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_HoldOn(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCall     *call.Call
		responseHoldOnErr error
		expectError      bool
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("97864554-cef3-11ed-9ba5-a7e641ec5c06"),
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("97864554-cef3-11ed-9ba5-a7e641ec5c06"),
				},
				ChannelID: "9a4086ec-cef3-11ed-b377-ef35b455442f",
			},
			responseHoldOnErr: nil,
			expectError:       false,
		},
		{
			name: "channel hold on error",

			id: uuid.FromStringOrNil("b1234554-cef3-11ed-9ba5-a7e641ec5c06"),
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b1234554-cef3-11ed-9ba5-a7e641ec5c06"),
				},
				ChannelID: "b2408600-cef3-11ed-b377-ef35b455442f",
			},
			responseHoldOnErr: fmt.Errorf("channel hold on error"),
			expectError:       true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockChannel.EXPECT().HoldOn(ctx, tt.responseCall.ChannelID).Return(tt.responseHoldOnErr)

			err := h.HoldOn(ctx, tt.id)
			if tt.expectError && err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_HoldOff(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCall      *call.Call
		responseHoldOffErr error
		expectError       bool
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("d9a5bc58-cef3-11ed-8296-4b4cc63de165"),
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d9a5bc58-cef3-11ed-8296-4b4cc63de165"),
				},
				ChannelID: "9a6e4122-cef3-11ed-b195-5b72e7449d60",
			},
			responseHoldOffErr: nil,
			expectError:        false,
		},
		{
			name: "channel hold off error",

			id: uuid.FromStringOrNil("e9a5bc58-cef3-11ed-8296-4b4cc63de165"),
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e9a5bc58-cef3-11ed-8296-4b4cc63de165"),
				},
				ChannelID: "ea6e4122-cef3-11ed-b195-5b72e7449d60",
			},
			responseHoldOffErr: fmt.Errorf("channel hold off error"),
			expectError:        true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockChannel.EXPECT().HoldOff(ctx, tt.responseCall.ChannelID).Return(tt.responseHoldOffErr)

			err := h.HoldOff(ctx, tt.id)
			if tt.expectError && err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
