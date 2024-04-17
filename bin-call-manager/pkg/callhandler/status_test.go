package callhandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
)

func Test_UpdateStatusRinging(t *testing.T) {

	tests := []struct {
		name         string
		channel      *channel.Channel
		call         *call.Call
		responseCall *call.Call
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
			&call.Call{
				ID:     uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status: call.StatusRinging,
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
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallRinging, tt.responseCall)

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

		responseCall *call.Call
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
			&call.Call{
				ID:        uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status:    call.StatusProgressing,
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
			&call.Call{
				ID:        uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status:    call.StatusProgressing,
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
			&call.Call{
				ID:        uuid.FromStringOrNil("0c864f8e-c8a6-11ec-af3c-372ebc5b6d6d"),
				Status:    call.StatusProgressing,
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
			&call.Call{
				ID:        uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status:    call.StatusProgressing,
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
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.call.CustomerID, call.EventTypeCallProgressing, tt.responseCall)

			if tt.call.Direction != call.DirectionIncoming {

				mockDB.EXPECT().CallGet(ctx, tt.responseCall.ID).Return(tt.call, nil)
				mockDB.EXPECT().CallSetStatus(gomock.Any(), tt.responseCall.ID, gomock.Any())
				mockDB.EXPECT().CallGet(ctx, tt.responseCall.ID).Return(tt.responseCall, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
				mockChannel.EXPECT().HangingUp(gomock.Any(), gomock.Any(), gomock.Any()).Return(&channel.Channel{TMEnd: dbhandler.DefaultTimeStamp}, nil)
			}

			if err := h.updateStatusProgressing(ctx, tt.channel, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)
		})
	}
}

func Test_UpdateStatusProgressing_answerGroupcall(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel
		call    *call.Call

		responseCall      *call.Call
		responseGroupcall *groupcall.Groupcall
	}{
		{
			name: "normal",
			channel: &channel.Channel{
				TMAnswer: "2020-09-20T03:23:20.995000",
			},
			call: &call.Call{
				ID:          uuid.FromStringOrNil("1f29b106-30fb-4b30-b83b-62b7a7a76ef9"),
				Status:      call.StatusDialing,
				Direction:   call.DirectionOutgoing,
				GroupcallID: uuid.FromStringOrNil("e68fcdcd-401e-427e-a2e3-579a6c9c7dcd"),
			},

			responseCall: &call.Call{
				ID:          uuid.FromStringOrNil("1f29b106-30fb-4b30-b83b-62b7a7a76ef9"),
				Status:      call.StatusProgressing,
				Direction:   call.DirectionOutgoing,
				GroupcallID: uuid.FromStringOrNil("e68fcdcd-401e-427e-a2e3-579a6c9c7dcd"),
			},
			responseGroupcall: &groupcall.Groupcall{
				ID:           uuid.FromStringOrNil("e68fcdcd-401e-427e-a2e3-579a6c9c7dcd"),
				AnswerMethod: groupcall.AnswerMethodHangupOthers,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("1f29b106-30fb-4b30-b83b-62b7a7a76ef9"),
					uuid.FromStringOrNil("c6cc7416-19c5-48bf-ab93-d063681c9994"),
				},
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
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &callHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				notifyHandler:    mockNotify,
				db:               mockDB,
				channelHandler:   mockChannel,
				groupcallHandler: mockGroupcall,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallSetStatusProgressing(ctx, tt.call.ID).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.responseCall.CustomerID, call.EventTypeCallProgressing, tt.responseCall)

			mockGroupcall.EXPECT().AnswerCall(ctx, tt.responseCall.GroupcallID, tt.responseCall.ID).Return(nil)

			mockDB.EXPECT().CallGet(ctx, tt.responseCall.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallSetStatus(gomock.Any(), tt.responseCall.ID, gomock.Any())
			mockDB.EXPECT().CallGet(ctx, tt.responseCall.ID).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
			mockChannel.EXPECT().HangingUp(gomock.Any(), gomock.Any(), gomock.Any()).Return(&channel.Channel{TMEnd: dbhandler.DefaultTimeStamp}, nil)

			if err := h.updateStatusProgressing(ctx, tt.channel, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)
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
