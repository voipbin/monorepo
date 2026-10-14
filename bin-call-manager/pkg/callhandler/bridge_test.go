package callhandler

import (
	"context"
	"fmt"
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

			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNormalClearing).Return(&channel.Channel{}, nil)
			mockDB.EXPECT().CallSetConfbridgeID(ctx, tt.bridge.ReferenceID, uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.bridge.ReferenceID).Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.call.CustomerID, call.EventTypeCallUpdated, tt.call)
			mockReq.EXPECT().CallV1CallActionNext(ctx, tt.call.ID, false).Return(nil)

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

			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNormalClearing).Return(&channel.Channel{}, nil)

			if len(tt.bridge.ChannelIDs) == 0 {
				mockBridge.EXPECT().Destroy(ctx, tt.bridge.ID).Return(nil)
			} else {
				for _, channelID := range tt.bridge.ChannelIDs {
					mockBridge.EXPECT().ChannelKick(ctx, tt.bridge.ID, channelID).Return(nil)
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
				mockBridge.EXPECT().ChannelKick(ctx, tt.bridge.ID, channelID).Return(nil)
			}
			h.removeAllChannelsInBridge(ctx, tt.bridge)
		})
	}
}

func Test_BridgeLeftExternal_callBridge(t *testing.T) {
	tests := []struct {
		name    string
		channel *channel.Channel
		bridge  *bridge.Bridge
	}{
		{
			name: "call bridge with the call channel remaining: hangup only, no kick, no destroy",
			channel: &channel.Channel{
				ID:         "3e20f43c-151d-11ec-be7f-6b10f15c44b3",
				AsteriskID: "42:01:0a:a4:00:03",
				Type:       channel.TypeExternal,
			},
			bridge: &bridge.Bridge{
				ID:            "543a1b3a-151e-11ec-ac2a-ef955db1beeb",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"5c0bfe56-151d-11ec-b49b-cf370dddad9f"},
			},
		},
		{
			name: "call bridge with no channel remaining: no destroy either",
			channel: &channel.Channel{
				ID:         "be2ad3b4-151d-11ec-bf66-0fbf215234b3",
				AsteriskID: "42:01:0a:a4:00:03",
				Type:       channel.TypeExternal,
			},
			bridge: &bridge.Bridge{
				ID:            "543a1b3a-151e-11ec-ac2a-ef955db1beeb",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}
			ctx := context.Background()

			// only HangingUp is expected: any Destroy or ChannelKick call fails the test.
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNormalClearing).Return(&channel.Channel{}, nil)

			if err := h.bridgeLeftExternal(ctx, tt.channel, tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_hangupExternalMembers(t *testing.T) {
	tests := []struct {
		name string

		bridge *bridge.Bridge

		responseChannels   map[string]*channel.Channel
		responseGetErrIDs  map[string]bool
		responseHangupErrs map[string]bool

		expectHangupIDs []string
	}{
		{
			name: "external member is hung up, other members are not",
			bridge: &bridge.Bridge{
				ID:            "543a1b3a-151e-11ec-ac2a-ef955db1beeb",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"join-channel", "ext-channel"},
			},
			responseChannels: map[string]*channel.Channel{
				"join-channel": {ID: "join-channel", Type: channel.TypeJoin},
				"ext-channel":  {ID: "ext-channel", Type: channel.TypeExternal},
			},
			expectHangupIDs: []string{"ext-channel"},
		},
		{
			name: "no external member",
			bridge: &bridge.Bridge{
				ID:            "543a1b3a-151e-11ec-ac2a-ef955db1beeb",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"join-channel"},
			},
			responseChannels: map[string]*channel.Channel{
				"join-channel": {ID: "join-channel", Type: channel.TypeJoin},
			},
			expectHangupIDs: []string{},
		},
		{
			name: "no remaining channel",
			bridge: &bridge.Bridge{
				ID:            "543a1b3a-151e-11ec-ac2a-ef955db1beeb",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{},
			},
			expectHangupIDs: []string{},
		},
		{
			name: "get error on one member does not stop the next member",
			bridge: &bridge.Bridge{
				ID:            "543a1b3a-151e-11ec-ac2a-ef955db1beeb",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"broken-channel", "ext-channel"},
			},
			responseChannels: map[string]*channel.Channel{
				"ext-channel": {ID: "ext-channel", Type: channel.TypeExternal},
			},
			responseGetErrIDs: map[string]bool{"broken-channel": true},
			expectHangupIDs:   []string{"ext-channel"},
		},
		{
			name: "hangup error on one external member does not stop the next member",
			bridge: &bridge.Bridge{
				ID:            "543a1b3a-151e-11ec-ac2a-ef955db1beeb",
				ReferenceType: bridge.ReferenceTypeCall,
				ChannelIDs:    []string{"ext-channel-1", "ext-channel-2"},
			},
			responseChannels: map[string]*channel.Channel{
				"ext-channel-1": {ID: "ext-channel-1", Type: channel.TypeExternal},
				"ext-channel-2": {ID: "ext-channel-2", Type: channel.TypeExternal},
			},
			responseHangupErrs: map[string]bool{"ext-channel-1": true},
			expectHangupIDs:    []string{"ext-channel-1", "ext-channel-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{channelHandler: mockChannel}
			ctx := context.Background()

			for _, channelID := range tt.bridge.ChannelIDs {
				if tt.responseGetErrIDs[channelID] {
					mockChannel.EXPECT().Get(ctx, channelID).Return(nil, fmt.Errorf("not found"))
					continue
				}
				mockChannel.EXPECT().Get(ctx, channelID).Return(tt.responseChannels[channelID], nil)
			}
			for _, channelID := range tt.expectHangupIDs {
				if tt.responseHangupErrs[channelID] {
					mockChannel.EXPECT().HangingUp(ctx, channelID, ari.ChannelCauseNormalClearing).Return(nil, fmt.Errorf("hangup failed"))
					continue
				}
				mockChannel.EXPECT().HangingUp(ctx, channelID, ari.ChannelCauseNormalClearing).Return(&channel.Channel{}, nil)
			}

			h.hangupExternalMembers(ctx, tt.bridge)
		})
	}
}
