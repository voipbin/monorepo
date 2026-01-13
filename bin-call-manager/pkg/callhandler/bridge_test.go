package callhandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func TestBridgeLeftJoin(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		call    *call.Call
		bridge  *bridge.Bridge
	}{
		{
			"call normal destroy",
			&channel.Channel{
				ID:          "0820e474-151c-11ec-859d-0b3af329400f",
				AsteriskID:  "42:01:0a:a4:00:03",
				Data:        map[string]interface{}{},
				HangupCause: ari.ChannelCauseNormalClearing,
				Type:        channel.TypeCall,
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("095658d8-151c-11ec-8aa9-1fca7824a72f"),
				},
				ChannelID:    "0820e474-151c-11ec-859d-0b3af329400f",
				Status:       call.StatusProgressing,
				ConfbridgeID: uuid.FromStringOrNil("3d093cca-2022-11ec-9358-c7e3a147380e"),
			},
			&bridge.Bridge{
				ReferenceID: uuid.FromStringOrNil("095658d8-151c-11ec-8aa9-1fca7824a72f"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNormalClearing.Return(&channel.Channel{}, nil)
			mockDB.EXPECT().CallSetConfbridgeID(ctx, tt.bridge.ReferenceID, uuid.Nil.Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.bridge.ReferenceID.Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.call.CustomerID, call.EventTypeCallUpdated, tt.call)
			mockReq.EXPECT().CallV1CallActionNext(ctx, tt.call.ID, false.Return(nil)

			if err := h.bridgeLeftJoin(ctx, tt.channel, tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_BridgeLeftExternal(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		bridge  *bridge.Bridge
	}{
		{
			"normal external channel leftbridge",
			&channel.Channel{
				ID:          "3e20f43c-151d-11ec-be7f-6b10f15c44b3",
				AsteriskID:  "42:01:0a:a4:00:03",
				Data:        map[string]interface{}{},
				HangupCause: ari.ChannelCauseNormalClearing,
				Type:        channel.TypeCall,
			},
			&bridge.Bridge{
				ReferenceID: uuid.FromStringOrNil("3e01f064-151d-11ec-bbba-0b568fed9a16"),
				ChannelIDs: []string{
					"5c0bfe56-151d-11ec-b49b-cf370dddad9f",
				},
			},
		},
		{
			"empty bridge",
			&channel.Channel{
				ID:          "be2ad3b4-151d-11ec-bf66-0fbf215234b3",
				AsteriskID:  "42:01:0a:a4:00:03",
				Data:        map[string]interface{}{},
				HangupCause: ari.ChannelCauseNormalClearing,
				Type:        channel.TypeCall,
			},
			&bridge.Bridge{
				AsteriskID:  "42:01:0a:a4:00:03",
				ID:          "543a1b3a-151e-11ec-ac2a-ef955db1beeb",
				ReferenceID: uuid.FromStringOrNil("be0399d4-151d-11ec-bb2e-774604f45fa3"),
				ChannelIDs:  []string{},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				db:             mockDB,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNormalClearing.Return(&channel.Channel{}, nil)

			if len(tt.bridge.ChannelIDs) == 0 {
				mockBridge.EXPECT().Destroy(ctx, tt.bridge.ID.Return(nil)
			} else {
				for _, channelID := range tt.bridge.ChannelIDs {
					mockBridge.EXPECT().ChannelKick(ctx, tt.bridge.ID, channelID.Return(nil)
				}
			}

			if err := h.bridgeLeftExternal(ctx, tt.channel, tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestRemoveAllChannelsInBridge(t *testing.T) {

	tests := []struct {
		name   string
		bridge *bridge.Bridge
	}{
		{
			"normal",
			&bridge.Bridge{
				ReferenceID: uuid.FromStringOrNil("b051d674-151e-11ec-9602-934100bd5a16"),
				ChannelIDs: []string{
					"b074cae4-151e-11ec-a6df-f38c7fc949ad",
					"b094059e-151e-11ec-90bd-cbaa99091559",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				bridgeHandler: mockBridge,
			}

			ctx := context.Background()

			for _, channelID := range tt.bridge.ChannelIDs {
				mockBridge.EXPECT().ChannelKick(ctx, tt.bridge.ID, channelID.Return(nil)
			}
			h.removeAllChannelsInBridge(ctx, tt.bridge)
		})
	}
}
