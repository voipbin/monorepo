package callhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
)

func TestARIChannelStateChangeStatusProgressing(t *testing.T) {
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

	type test struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}

	tests := []test{
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
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallSetStatus(gomock.Any(), tt.call.ID, call.StatusProgressing, tt.channel.TMAnswer)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil).AnyTimes()

			mockNotfiy.EXPECT().PublishWebhookEvent(gomock.Any(), notifyhandler.EventTypeCallAnswered, tt.call.WebhookURI, tt.call)

			mockDB.EXPECT().CallSetStatus(gomock.Any(), tt.call.ID, call.StatusTerminating, gomock.Any())
			mockReq.EXPECT().AstChannelHangup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			if err := h.ARIChannelStateChange(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestARIChannelStateChangeStatusRinging(t *testing.T) {
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

	type test struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}

	tests := []test{
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
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallSetStatus(gomock.Any(), tt.call.ID, call.StatusRinging, tt.channel.TMRinging)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)

			mockNotfiy.EXPECT().PublishWebhookEvent(gomock.Any(), notifyhandler.EventTypeCallRinging, tt.call.WebhookURI, tt.call)

			if err := h.ARIChannelStateChange(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestARIChannelDestroyedContextTypeCall(t *testing.T) {
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
	}

	tests := []test{
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
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf("no call"))

			if err := h.ARIChannelDestroyed(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestARIChannelDestroyedContextTypeConference(t *testing.T) {
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
	}

	tests := []test{
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
			ctx := context.Background()
			if err := h.ARIChannelDestroyed(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestARIPlaybackFinished(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name       string
		channel    *channel.Channel
		call       *call.Call
		playbackID string
	}

	tests := []test{
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
				FlowID: uuid.FromStringOrNil("32c36bf4-156f-11ec-af17-87eb4aca917b"),
			},
			"77a82874-e7dd-11ea-9647-27054cd71830",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			returnAction := action.Action{
				Type:   action.TypeHangup,
				Option: []byte(`{}`),
			}

			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)

			// action next part.
			mockReq.EXPECT().FMV1ActvieFlowGetNextAction(gomock.Any(), tt.call.ID, tt.call.Action.ID).Return(&returnAction, nil)
			mockDB.EXPECT().CallSetAction(gomock.Any(), tt.call.ID, gomock.Any()).Return(nil)
			mockDB.EXPECT().CallSetStatus(gomock.Any(), tt.call.ID, call.StatusTerminating, gomock.Any()).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockReq.EXPECT().AstChannelHangup(gomock.Any(), tt.call.AsteriskID, tt.call.ChannelID, ari.ChannelCauseNormalClearing)

			if err := h.ARIPlaybackFinished(ctx, tt.channel, tt.playbackID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
