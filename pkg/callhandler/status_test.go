package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_UpdateStatusRinging(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}{
		{
			"call status dialing",
			&channel.Channel{
				TMRinging: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:     uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status: call.StatusDialing,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &callHandler{
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallSetStatusRinging(ctx, tt.call.ID).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.call.CustomerID, call.EventTypeCallUpdated, tt.call)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.call.CustomerID, call.EventTypeCallRinging, tt.call)

			if err := h.updateStatusRinging(ctx, tt.channel, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateStatusRingingFail(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}{
		{
			"call status ringing",
			&channel.Channel{
				TMRinging: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:     uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status: call.StatusRinging,
			},
		},
		{
			"call status progressing",
			&channel.Channel{
				TMRinging: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:     uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status: call.StatusProgressing,
			},
		},
		{
			"call status terminating",
			&channel.Channel{
				TMRinging: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:     uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status: call.StatusTerminating,
			},
		},
		{
			"call status canceling",
			&channel.Channel{
				TMRinging: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:     uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status: call.StatusCanceling,
			},
		},
		{
			"call status hangup",
			&channel.Channel{
				TMRinging: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:     uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status: call.StatusHangup,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			if err := h.updateStatusRinging(ctx, tt.channel, tt.call); err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
			}
		})
	}
}

func Test_UpdateStatusProgressing(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}{
		{
			"call status dialing for incoming",
			&channel.Channel{
				TMAnswer: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:        uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status:    call.StatusDialing,
				Direction: call.DirectionIncoming,
			},
		},
		{
			"call status ringing for incoming",
			&channel.Channel{
				TMAnswer: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:        uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,
			},
		},
		{
			"call status dialing for outgoing",
			&channel.Channel{
				TMAnswer: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:        uuid.FromStringOrNil("0c864f8e-c8a6-11ec-af3c-372ebc5b6d6d"),
				Status:    call.StatusDialing,
				Direction: call.DirectionOutgoing,
			},
		},
		{
			"call status ringing for outgoing",
			&channel.Channel{
				TMAnswer: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:        uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status:    call.StatusRinging,
				Direction: call.DirectionOutgoing,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallSetStatusProgressing(ctx, tt.call.ID).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.call.CustomerID, call.EventTypeCallUpdated, tt.call)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.call.CustomerID, call.EventTypeCallAnswered, tt.call)

			if tt.call.Direction != call.DirectionIncoming {
				// handleSIPCallID
				mockReq.EXPECT().AstChannelVariableGet(ctx, tt.channel.AsteriskID, tt.channel.ID, `CHANNEL(pjsip,call-id)`).Return("test call id", nil).AnyTimes()
				mockReq.EXPECT().AstChannelVariableSet(ctx, tt.channel.AsteriskID, tt.channel.ID, "VB-SIP_CALLID", gomock.Any()).Return(nil).AnyTimes()

				mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
				mockDB.EXPECT().CallSetStatus(gomock.Any(), tt.call.ID, gomock.Any())
				mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
				mockChannel.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)
				mockReq.EXPECT().AstChannelHangup(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), 0).Return(nil)
			}

			if err := h.updateStatusProgressing(ctx, tt.channel, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateStatusProgressingFail(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}{
		{
			"call status progressing",
			&channel.Channel{
				TMRinging: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:        uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status:    call.StatusProgressing,
				Direction: call.DirectionIncoming,
			},
		},
		{
			"call status terminating",
			&channel.Channel{
				TMRinging: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:        uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status:    call.StatusTerminating,
				Direction: call.DirectionIncoming,
			},
		},
		{
			"call status canceling",
			&channel.Channel{
				TMRinging: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:        uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status:    call.StatusCanceling,
				Direction: call.DirectionIncoming,
			},
		},
		{
			"call status hangup",
			&channel.Channel{
				TMRinging: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:        uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status:    call.StatusHangup,
				Direction: call.DirectionIncoming,
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

			if err := h.updateStatusProgressing(ctx, tt.channel, tt.call); err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
			}
		})
	}
}
