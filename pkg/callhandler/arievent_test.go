package callhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_ARIChannelStateChangeStatusProgressing(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}{
		{
			"normal answer",
			&channel.Channel{
				ID:       "31384bbc-dd97-11ea-9e42-433e5113c783",
				Data:     map[string]interface{}{},
				State:    ari.ChannelStateUp,
				Type:     channel.TypeCall,
				TMAnswer: "2020-05-02 20:56:51.498",
			},
			&call.Call{
				ID:     uuid.FromStringOrNil("a4974832-4d3b-11ec-895b-0b7796863054"),
				Status: call.StatusRinging,
			},
		},
		{
			"update answer from dialing",
			&channel.Channel{
				ID:       "849f1e92-4d40-11ec-b40a-739fbc078d18",
				Data:     map[string]interface{}{},
				State:    ari.ChannelStateUp,
				Type:     channel.TypeCall,
				TMAnswer: "2020-05-02 20:56:51.498",
			},
			&call.Call{
				ID:     uuid.FromStringOrNil("84e77160-4d40-11ec-aa31-8b1d57a189d0"),
				Status: call.StatusDialing,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotfiy,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallSetStatusProgressing(gomock.Any(), tt.call.ID)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockNotfiy.EXPECT().PublishWebhookEvent(gomock.Any(), tt.call.CustomerID, call.EventTypeCallUpdated, tt.call)
			mockNotfiy.EXPECT().PublishWebhookEvent(gomock.Any(), tt.call.CustomerID, call.EventTypeCallAnswered, tt.call)
			if tt.call.Direction != call.DirectionIncoming {
				// handleSIPCallID
				mockReq.EXPECT().AstChannelVariableGet(ctx, tt.channel.AsteriskID, tt.channel.ID, `CHANNEL(pjsip,call-id)`).Return("test call id", nil).AnyTimes()
				mockReq.EXPECT().AstChannelVariableSet(ctx, tt.channel.AsteriskID, tt.channel.ID, "VB-SIP_CALLID", gomock.Any()).Return(nil).AnyTimes()

				// ActionNext
				// we hangup the call because the flow_id is not set
				mockDB.EXPECT().CallGet(ctx, gomock.Any()).Return(&call.Call{}, nil)
				mockDB.EXPECT().CallSetStatus(gomock.Any(), gomock.Any(), gomock.Any())
				mockDB.EXPECT().CallGet(ctx, gomock.Any()).Return(&call.Call{}, nil)
				mockNotfiy.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
				mockChannel.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)
				mockReq.EXPECT().AstChannelHangup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), 0).Return(nil)
			}

			if err := h.ARIChannelStateChange(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ARIChannelStateChangeStatusRinging(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}{
		{
			"normal ringing",
			&channel.Channel{
				ID:        "31384bbc-dd97-11ea-9e42-433e5113c783",
				Data:      map[string]interface{}{},
				State:     ari.ChannelStateRing,
				Type:      channel.TypeCall,
				TMRinging: "2020-05-02 20:56:51.498",
			},
			&call.Call{
				ID:     uuid.FromStringOrNil("a4974832-4d3b-11ec-895b-0b7796863054"),
				Status: call.StatusDialing,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotfiy,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallSetStatusRinging(gomock.Any(), tt.call.ID)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)

			mockNotfiy.EXPECT().PublishWebhookEvent(gomock.Any(), tt.call.CustomerID, call.EventTypeCallUpdated, tt.call)
			mockNotfiy.EXPECT().PublishWebhookEvent(gomock.Any(), tt.call.CustomerID, call.EventTypeCallRinging, tt.call)

			if err := h.ARIChannelStateChange(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestARIChannelDestroyedContextTypeCall(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
	}{
		{
			"call normal destroy",
			&channel.Channel{
				ID:          "31384bbc-dd97-11ea-9e42-433e5113c783",
				Data:        map[string]interface{}{},
				HangupCause: ari.ChannelCauseNormalClearing,
				Type:        channel.TypeCall,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf("no call"))

			if err := h.ARIChannelDestroyed(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestARIChannelDestroyedContextTypeConference(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
	}{
		{
			"conference normal destroy",
			&channel.Channel{
				ID:          "78ff0ed4-dd7b-11ea-9add-dbca62f7e8b9",
				Data:        map[string]interface{}{},
				Type:        channel.TypeConfbridge,
				HangupCause: ari.ChannelCauseNormalClearing,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()
			if err := h.ARIChannelDestroyed(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ARIPlaybackFinished(t *testing.T) {

	tests := []struct {
		name       string
		channel    *channel.Channel
		call       *call.Call
		playbackID string

		responseCall *call.Call
	}{
		{
			"normal",
			&channel.Channel{
				ID:   "1b8da938-e7dd-11ea-8e4a-1f2bd2b9f5b4",
				Data: map[string]interface{}{},
				Type: channel.TypeConfbridge,
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("66795a5a-e7dd-11ea-b2df-0757b438501c"),
				AsteriskID: "42:01:0a:a4:00:03",
				Action: action.Action{
					ID: uuid.FromStringOrNil("77a82874-e7dd-11ea-9647-27054cd71830"),
				},
				FlowID:       uuid.FromStringOrNil("32c36bf4-156f-11ec-af17-87eb4aca917b"),
				ActiveFlowID: uuid.FromStringOrNil("244d4566-a7bb-11ec-92eb-fbdbdda3d486"),
			},
			"77a82874-e7dd-11ea-9647-27054cd71830",

			&call.Call{
				ID:           uuid.FromStringOrNil("66795a5a-e7dd-11ea-b2df-0757b438501c"),
				AsteriskID:   "42:01:0a:a4:00:03",
				Action:       action.Action{},
				FlowID:       uuid.FromStringOrNil("32c36bf4-156f-11ec-af17-87eb4aca917b"),
				ActiveFlowID: uuid.FromStringOrNil("244d4566-a7bb-11ec-92eb-fbdbdda3d486"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)

			// action next part.
			mockDB.EXPECT().CallSetActionNextHold(ctx, tt.call.ID, true).Return(nil)
			mockReq.EXPECT().FlowV1ActiveflowGetNextAction(gomock.Any(), tt.call.ActiveFlowID, tt.call.Action.ID).Return(&action.Action{}, nil)
			mockUtil.EXPECT().GetCurTime().Return(utilhandler.GetCurTime()).AnyTimes()
			mockDB.EXPECT().CallSetActionAndActionNextHold(gomock.Any(), tt.call.ID, gomock.Any(), false).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.call.CustomerID, call.EventTypeCallUpdated, tt.responseCall)
			mockReq.EXPECT().CallV1CallActionNext(ctx, tt.call.ID, false).Return(nil)

			if err := h.ARIPlaybackFinished(ctx, tt.channel, tt.playbackID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
