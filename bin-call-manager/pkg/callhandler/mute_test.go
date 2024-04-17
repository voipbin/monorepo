package callhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_MuteOn(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		direction call.MuteDirection

		responseCall *call.Call
	}{
		{
			name: "normal",

			id:        uuid.FromStringOrNil("17cb6616-d13a-11ed-ac5f-cf2e89d2d519"),
			direction: call.MuteDirectionBoth,

			responseCall: &call.Call{
				ID:        uuid.FromStringOrNil("17cb6616-d13a-11ed-ac5f-cf2e89d2d519"),
				ChannelID: "9a4086ec-cef3-11ed-b377-ef35b455442f",
			},
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
			mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotfiy,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockChannel.EXPECT().MuteOn(ctx, tt.responseCall.ChannelID, channel.MuteDirection(tt.direction)).Return(nil)
			mockDB.EXPECT().CallSetMuteDirection(ctx, tt.id, tt.direction).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockNotfiy.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallUpdated, tt.responseCall)

			if err := h.MuteOn(ctx, tt.id, tt.direction); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_MuteOff(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		direction call.MuteDirection

		responseCall        *call.Call
		expectMuteDirection call.MuteDirection
	}{
		{
			name: "normal",

			id:        uuid.FromStringOrNil("183303a2-d13a-11ed-a800-9b0f57c3143f"),
			direction: call.MuteDirectionBoth,

			responseCall: &call.Call{
				ID:        uuid.FromStringOrNil("183303a2-d13a-11ed-a800-9b0f57c3143f"),
				ChannelID: "9a6e4122-cef3-11ed-b195-5b72e7449d60",
			},
			expectMuteDirection: call.MuteDirectionNone,
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
			mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotfiy,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockChannel.EXPECT().MuteOff(ctx, tt.responseCall.ChannelID, channel.MuteDirection(tt.direction)).Return(nil)
			mockDB.EXPECT().CallSetMuteDirection(ctx, tt.id, tt.expectMuteDirection).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockNotfiy.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallUpdated, tt.responseCall)

			if err := h.MuteOff(ctx, tt.id, tt.direction); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
