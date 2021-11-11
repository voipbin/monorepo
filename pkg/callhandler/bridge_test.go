package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestBridgeLeftJoin(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}

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
				ID:        uuid.FromStringOrNil("095658d8-151c-11ec-8aa9-1fca7824a72f"),
				ChannelID: "0820e474-151c-11ec-859d-0b3af329400f",
				Status:    call.StatusProgressing,
				ConfID:    uuid.FromStringOrNil("3d093cca-2022-11ec-9358-c7e3a147380e"),
			},
			&bridge.Bridge{
				ReferenceID: uuid.FromStringOrNil("095658d8-151c-11ec-8aa9-1fca7824a72f"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().AstChannelHangup(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, ari.ChannelCauseNormalClearing).Return(nil)
			mockDB.EXPECT().CallSetConferenceID(gomock.Any(), tt.bridge.ReferenceID, uuid.Nil).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.bridge.ReferenceID).Return(tt.call, nil)
			mockNotify.EXPECT().NotifyEvent(gomock.Any(), notifyhandler.EventTypeCallUpdated, tt.call.WebhookURI, tt.call)
			mockReq.EXPECT().CallCallActionNext(gomock.Any(), tt.call.ID).Return(nil)

			if err := h.bridgeLeftJoin(context.Background(), tt.channel, tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestBridgeLeftExternal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name    string
		channel *channel.Channel
		bridge  *bridge.Bridge
	}

	tests := []test{
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
			mockReq.EXPECT().AstChannelHangup(gomock.Any(), tt.channel.AsteriskID, tt.channel.ID, ari.ChannelCauseNormalClearing).Return(nil)

			if len(tt.bridge.ChannelIDs) == 0 {
				mockReq.EXPECT().AstBridgeDelete(gomock.Any(), tt.bridge.AsteriskID, tt.bridge.ID)
			} else {
				for _, channelID := range tt.bridge.ChannelIDs {
					mockReq.EXPECT().AstBridgeRemoveChannel(gomock.Any(), tt.bridge.AsteriskID, tt.bridge.ID, channelID).Return(nil)
				}
			}

			if err := h.bridgeLeftExternal(context.Background(), tt.channel, tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestRemoveAllChannelsInBridge(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name   string
		bridge *bridge.Bridge
	}

	tests := []test{
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
			for _, channelID := range tt.bridge.ChannelIDs {
				mockReq.EXPECT().AstBridgeRemoveChannel(gomock.Any(), tt.bridge.AsteriskID, tt.bridge.ID, channelID)
			}
			h.removeAllChannelsInBridge(context.Background(), tt.bridge)
		})
	}
}
