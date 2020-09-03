package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestUpdateStatusRinging(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}

	tests := []test{
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
		ctx := context.Background()

		mockDB.EXPECT().CallSetStatus(ctx, tt.call.ID, call.StatusRinging, tt.channel.TMRinging).Return(nil)

		if err := h.updateStatusRinging(ctx, tt.channel, tt.call); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}

func TestUpdateStatusRingingFail(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}

	tests := []test{
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
		ctx := context.Background()

		if err := h.updateStatusRinging(ctx, tt.channel, tt.call); err == nil {
			t.Errorf("Wrong match. expect: err, got: ok")
		}
	}
}

func TestUpdateStatusProgressing(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}

	tests := []test{
		{
			"call status dialing",
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
			"call status ringing",
			&channel.Channel{
				TMAnswer: "2020-09-20T03:23:20.995000",
			},
			&call.Call{
				ID:        uuid.FromStringOrNil("3ae0b538-edd6-11ea-bd23-d7e2d2e43f43"),
				Status:    call.StatusRinging,
				Direction: call.DirectionIncoming,
			},
		},
	}

	for _, tt := range tests {
		ctx := context.Background()

		mockDB.EXPECT().CallSetStatus(ctx, tt.call.ID, call.StatusProgressing, tt.channel.TMAnswer).Return(nil)

		if err := h.updateStatusProgressing(ctx, tt.channel, tt.call); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
}

func TestUpdateStatusProgressingFail(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
	}

	type test struct {
		name    string
		channel *channel.Channel
		call    *call.Call
	}

	tests := []test{
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
		ctx := context.Background()

		if err := h.updateStatusProgressing(ctx, tt.channel, tt.call); err == nil {
			t.Errorf("Wrong match. expect: err, got: ok")
		}
	}
}
