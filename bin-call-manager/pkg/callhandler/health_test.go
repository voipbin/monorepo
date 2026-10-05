package callhandler

import (
	"context"
	"fmt"
	"testing"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/testhelper"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_HealthCheck(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		retryCount int

		responseCall       *call.Call
		responseChannel    *channel.Channel
		responseChannelErr error

		// expectHealth is false when the health check chain must end without a new health check request
		expectHealth     bool
		expectRetryCount int
	}{
		{
			name: "normal",

			id:         uuid.FromStringOrNil("8b0c11f9-ad03-4bd7-98a5-102f89877e2a"),
			retryCount: 0,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8b0c11f9-ad03-4bd7-98a5-102f89877e2a"),
				},
				ChannelID: "a5416ed3-5f61-4e1e-971f-0b3ff61ce19e",
				Status:    call.StatusProgressing,
				TMHangup:  nil,
				TMDelete:  nil,
			},
			responseChannel: &channel.Channel{
				ID:       "a5416ed3-5f61-4e1e-971f-0b3ff61ce19e",
				TMEnd:    nil,
				TMDelete: nil,
			},

			expectHealth:     true,
			expectRetryCount: 0,
		},
		{
			name: "alive channel resets the retry count",

			id:         uuid.FromStringOrNil("8b0c11f9-ad03-4bd7-98a5-102f89877e2a"),
			retryCount: 2,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8b0c11f9-ad03-4bd7-98a5-102f89877e2a"),
				},
				ChannelID: "a5416ed3-5f61-4e1e-971f-0b3ff61ce19e",
				Status:    call.StatusProgressing,
			},
			responseChannel: &channel.Channel{
				ID: "a5416ed3-5f61-4e1e-971f-0b3ff61ce19e",
			},

			expectHealth:     true,
			expectRetryCount: 0,
		},
		{
			name: "calll channel ended",

			id:         uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
			retryCount: 0,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
				},
				ChannelID: "cba7edf9-8586-40c0-992b-5885103228c1",
				Status:    call.StatusProgressing,
				TMHangup:  nil,
				TMDelete:  nil,
			},
			responseChannel: &channel.Channel{
				ID:       "cba7edf9-8586-40c0-992b-5885103228c1",
				TMEnd:    testhelper.TimePtr("2023-01-18T03:22:18.995000Z"),
				TMDelete: nil,
			},

			expectHealth:     true,
			expectRetryCount: 1,
		},
		{
			name: "calll channel deleted",

			id:         uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
			retryCount: 0,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
				},
				ChannelID: "cba7edf9-8586-40c0-992b-5885103228c1",
				Status:    call.StatusProgressing,
				TMHangup:  nil,
				TMDelete:  nil,
			},
			responseChannel: &channel.Channel{
				ID:       "cba7edf9-8586-40c0-992b-5885103228c1",
				TMEnd:    nil,
				TMDelete: testhelper.TimePtr("2023-01-18T03:22:18.995000Z"),
			},

			expectHealth:     true,
			expectRetryCount: 1,
		},
		{
			name: "channel row is missing: counted like an ended channel",

			id:         uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
			retryCount: 0,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
				},
				ChannelID: "cba7edf9-8586-40c0-992b-5885103228c1",
				Status:    call.StatusDialing,
			},
			responseChannelErr: dbhandler.ErrNotFound,

			expectHealth:     true,
			expectRetryCount: 1,
		},
		{
			name: "channel row is still missing at the third check",

			id:         uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
			retryCount: 2,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
				},
				ChannelID: "cba7edf9-8586-40c0-992b-5885103228c1",
				Status:    call.StatusDialing,
			},
			responseChannelErr: dbhandler.ErrNotFound,

			expectHealth:     true,
			expectRetryCount: 3,
		},
		{
			name: "channel read failure ends the chain",

			id:         uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
			retryCount: 2,

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d0760324-75d6-443d-aa6f-d3b8703bf78a"),
				},
				ChannelID: "cba7edf9-8586-40c0-992b-5885103228c1",
				Status:    call.StatusDialing,
			},
			responseChannelErr: fmt.Errorf("could not query. ChannelGet"),

			expectHealth: false,
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

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockDB.EXPECT().ChannelGet(ctx, tt.responseCall.ChannelID).Return(tt.responseChannel, tt.responseChannelErr)

			if tt.expectHealth {
				mockReq.EXPECT().CallV1CallHealth(ctx, tt.id, defaultHealthDelay, tt.expectRetryCount).Return(nil)
			}

			h.HealthCheck(ctx, tt.id, tt.retryCount)
		})
	}
}

// Test_HealthCheck_retryExceeded checks the retry-exceeded branch hands over to healthHangup: it reads the call from the
// database (not the cache) and never reads the cached call or reschedules.
func Test_HealthCheck_retryExceeded(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newRecoveryTestHandler(mc)
	ctx := context.Background()

	id := uuid.FromStringOrNil("5a1b8b4e-a1c2-4f6e-9a77-0d6c7a9e1b11")

	// a call that has finished meanwhile: healthHangup reads it and does nothing more.
	m.db.EXPECT().CallGetFromDB(ctx, id).Return(&call.Call{
		Identity: commonidentity.Identity{ID: id},
		Status:   call.StatusHangup,
	}, nil)

	h.HealthCheck(ctx, id, defaultHealthMaxRetryCount+1)
}

func Test_healthHangup(t *testing.T) {
	id := uuid.FromStringOrNil("0b9d0a42-6a0b-4d0c-8a15-9f5d3c2c7e21")
	activeflowID := uuid.FromStringOrNil("6c1e9a0f-7a50-4e2f-b0a1-2d4f8c3b9e32")
	channelID := "3f6a2d58-b1c4-4e7a-92d3-5a8e1c7f0b43"
	endedAt := testhelper.TimePtr("2023-01-18T03:22:18.995000Z")

	newCall := func(status call.Status, direction call.Direction) *call.Call {
		return &call.Call{
			Identity:     commonidentity.Identity{ID: id, CustomerID: uuid.FromStringOrNil("c0f4a1d2-0a5b-4e8f-9c3d-1b2a3c4d5e54")},
			ChannelID:    channelID,
			ActiveflowID: activeflowID,
			Status:       status,
			Direction:    direction,
		}
	}
	endedChannel := &channel.Channel{
		ID:          channelID,
		HangupCause: ari.ChannelCauseNormalClearing,
		TMEnd:       endedAt,
		TMDelete:    endedAt,
	}
	// a channel row that only has one of the two end markers is still an ended channel
	endedOnlyTMEnd := &channel.Channel{ID: channelID, HangupCause: ari.ChannelCauseNormalClearing, TMEnd: endedAt}
	endedOnlyTMDelete := &channel.Channel{ID: channelID, HangupCause: ari.ChannelCauseNormalClearing, TMDelete: endedAt}
	aliveChannel := &channel.Channel{ID: channelID}

	// expectHangup sets the expectations of Hangup for a call with the given status. reason and by are the values that
	// CalculateHangupReason and CalculateHangupBy return for it.
	expectHangup := func(ctx context.Context, m recoveryMocks, c *call.Call, reason call.HangupReason, by call.HangupBy) {
		hungup := *c
		hungup.Status = call.StatusHangup

		m.db.EXPECT().CallGetByChannelID(ctx, channelID).Return(c, nil)
		m.bridge.EXPECT().Destroy(ctx, c.BridgeID).Return(nil)
		m.db.EXPECT().CallSetHangup(ctx, id, reason, by).Return(nil)
		m.db.EXPECT().CallGet(ctx, id).Return(&hungup, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, hungup.CustomerID, call.EventTypeCallHangup, &hungup)
		m.req.EXPECT().FlowV1ActiveflowStop(ctx, activeflowID).Return(nil, nil)
	}

	// expectFailedCall sets the expectations of hangupFailedCall.
	expectFailedCall := func(ctx context.Context, m recoveryMocks, c *call.Call, stopActiveflow bool) {
		hungup := *c
		hungup.Status = call.StatusHangup

		m.db.EXPECT().CallSetHangup(ctx, id, call.HangupReasonFailed, call.HangupByLocal).Return(nil)
		m.db.EXPECT().CallGet(ctx, id).Return(&hungup, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, hungup.CustomerID, call.EventTypeCallHangup, &hungup)
		if stopActiveflow {
			m.req.EXPECT().FlowV1ActiveflowStop(ctx, c.ActiveflowID).Return(nil, nil)
		}
	}

	tests := []struct {
		name string

		responseCall *call.Call
		callErr      error

		responseChannel *channel.Channel
		channelErr      error

		expect func(ctx context.Context, m recoveryMocks, c *call.Call)
	}{
		{
			name:            "canceling call with an ended channel finishes through Hangup",
			responseCall:    newCall(call.StatusCanceling, call.DirectionOutgoing),
			responseChannel: endedChannel,
			expect: func(ctx context.Context, m recoveryMocks, c *call.Call) {
				expectHangup(ctx, m, c, call.HangupReasonCanceled, call.HangupByLocal)
			},
		},
		{
			name:            "terminating call with an ended channel finishes through Hangup",
			responseCall:    newCall(call.StatusTerminating, call.DirectionIncoming),
			responseChannel: endedChannel,
			expect: func(ctx context.Context, m recoveryMocks, c *call.Call) {
				expectHangup(ctx, m, c, call.HangupReasonNormal, call.HangupByLocal)
			},
		},
		{
			name:         "dialing call without a channel row is finalized as failed",
			responseCall: newCall(call.StatusDialing, call.DirectionOutgoing),
			channelErr:   dbhandler.ErrNotFound,
			expect: func(ctx context.Context, m recoveryMocks, c *call.Call) {
				expectFailedCall(ctx, m, c, true)
			},
		},
		{
			name:         "canceling call without a channel row is finalized as failed",
			responseCall: newCall(call.StatusCanceling, call.DirectionOutgoing),
			channelErr:   dbhandler.ErrNotFound,
			expect: func(ctx context.Context, m recoveryMocks, c *call.Call) {
				expectFailedCall(ctx, m, c, true)
			},
		},
		{
			name: "call with a dummy activeflow does not stop an activeflow",
			responseCall: func() *call.Call {
				c := newCall(call.StatusDialing, call.DirectionOutgoing)
				c.ActiveflowID = uuid.Nil
				return c
			}(),
			channelErr: dbhandler.ErrNotFound,
			expect: func(ctx context.Context, m recoveryMocks, c *call.Call) {
				expectFailedCall(ctx, m, c, false)
			},
		},
		{
			name:         "ringing call without a channel row is not finalized",
			responseCall: newCall(call.StatusRinging, call.DirectionOutgoing),
			channelErr:   dbhandler.ErrNotFound,
		},
		{
			name:         "progressing call without a channel row is not finalized",
			responseCall: newCall(call.StatusProgressing, call.DirectionOutgoing),
			channelErr:   dbhandler.ErrNotFound,
		},
		{
			name:         "terminating call without a channel row is not finalized",
			responseCall: newCall(call.StatusTerminating, call.DirectionOutgoing),
			channelErr:   dbhandler.ErrNotFound,
		},
		{
			name:            "alive channel restarts the health check",
			responseCall:    newCall(call.StatusCanceling, call.DirectionOutgoing),
			responseChannel: aliveChannel,
			expect: func(ctx context.Context, m recoveryMocks, c *call.Call) {
				m.req.EXPECT().CallV1CallHealth(ctx, id, defaultHealthDelay, 0).Return(nil)
			},
		},
		{
			name:            "canceling call whose channel only has tm_end finishes through Hangup",
			responseCall:    newCall(call.StatusCanceling, call.DirectionOutgoing),
			responseChannel: endedOnlyTMEnd,
			expect: func(ctx context.Context, m recoveryMocks, c *call.Call) {
				expectHangup(ctx, m, c, call.HangupReasonCanceled, call.HangupByLocal)
			},
		},
		{
			name:            "canceling call whose channel only has tm_delete finishes through Hangup",
			responseCall:    newCall(call.StatusCanceling, call.DirectionOutgoing),
			responseChannel: endedOnlyTMDelete,
			expect: func(ctx context.Context, m recoveryMocks, c *call.Call) {
				expectHangup(ctx, m, c, call.HangupReasonCanceled, call.HangupByLocal)
			},
		},
		{
			name:            "Hangup error is only logged",
			responseCall:    newCall(call.StatusCanceling, call.DirectionOutgoing),
			responseChannel: endedChannel,
			expect: func(ctx context.Context, m recoveryMocks, c *call.Call) {
				m.db.EXPECT().CallGetByChannelID(ctx, channelID).Return(nil, fmt.Errorf("could not get the call"))
			},
		},
		{
			name:         "failed call: a hangup write failure stops the activeflow request",
			responseCall: newCall(call.StatusDialing, call.DirectionOutgoing),
			channelErr:   dbhandler.ErrNotFound,
			expect: func(ctx context.Context, m recoveryMocks, c *call.Call) {
				m.db.EXPECT().CallSetHangup(ctx, id, call.HangupReasonFailed, call.HangupByLocal).Return(fmt.Errorf("could not write"))
			},
		},
		{
			name:         "call read failure",
			responseCall: nil,
			callErr:      fmt.Errorf("could not query. CallGet"),
		},
		{
			name:         "channel read failure",
			responseCall: newCall(call.StatusCanceling, call.DirectionOutgoing),
			channelErr:   fmt.Errorf("could not query. ChannelGet"),
		},
		{
			name: "call with only the hangup status is left alone",
			responseCall: func() *call.Call {
				return newCall(call.StatusHangup, call.DirectionOutgoing)
			}(),
		},
		{
			name: "call with only tm_hangup is left alone",
			responseCall: func() *call.Call {
				c := newCall(call.StatusCanceling, call.DirectionOutgoing)
				c.TMHangup = endedAt
				return c
			}(),
		},
		{
			name: "call with only tm_delete is left alone",
			responseCall: func() *call.Call {
				c := newCall(call.StatusCanceling, call.DirectionOutgoing)
				c.TMDelete = endedAt
				return c
			}(),
		},
		{
			name:            "dialing call with an ended channel keeps the HangingUp behavior",
			responseCall:    newCall(call.StatusDialing, call.DirectionOutgoing),
			responseChannel: endedChannel,
			expect: func(ctx context.Context, m recoveryMocks, c *call.Call) {
				// HangingUp reads the call from the cache first. a hangup status there ends it at once.
				done := *c
				done.Status = call.StatusHangup
				m.db.EXPECT().CallGet(ctx, id).Return(&done, nil)
			},
		},
		{
			name:            "ringing call with an ended channel keeps the HangingUp behavior",
			responseCall:    newCall(call.StatusRinging, call.DirectionOutgoing),
			responseChannel: endedChannel,
			expect: func(ctx context.Context, m recoveryMocks, c *call.Call) {
				done := *c
				done.Status = call.StatusHangup
				m.db.EXPECT().CallGet(ctx, id).Return(&done, nil)
			},
		},
		{
			name:            "progressing call with an ended channel keeps the HangingUp behavior",
			responseCall:    newCall(call.StatusProgressing, call.DirectionOutgoing),
			responseChannel: endedChannel,
			expect: func(ctx context.Context, m recoveryMocks, c *call.Call) {
				// the whole HangingUp sequence, to pin its arguments: the call goes to terminating and the channel
				// hangup is requested with the cause of the normal hangup reason.
				terminating := *c
				terminating.Status = call.StatusTerminating

				m.util.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime()).AnyTimes()
				m.db.EXPECT().CallGet(ctx, id).Return(c, nil)
				m.db.EXPECT().CallSetStatus(ctx, id, call.StatusTerminating).Return(nil)
				m.db.EXPECT().CallGet(ctx, id).Return(&terminating, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, terminating.CustomerID, call.EventTypeCallTerminating, &terminating)
				m.db.EXPECT().CallGetFromDB(ctx, id).Return(&terminating, nil)
				m.channel.EXPECT().HangingUp(ctx, channelID, call.ConvertHangupReasonToChannelCause(call.HangupReasonNormal)).
					Return(&channel.Channel{ID: channelID}, nil)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newRecoveryTestHandler(mc)
			ctx := context.Background()

			m.db.EXPECT().CallGetFromDB(ctx, id).Return(tt.responseCall, tt.callErr)

			// the channel is read only for a call that is not final (the same condition as healthHangup; keep them in sync)
			if tt.callErr == nil && tt.responseCall.Status != call.StatusHangup && tt.responseCall.TMHangup == nil && tt.responseCall.TMDelete == nil {
				m.db.EXPECT().ChannelGet(ctx, channelID).Return(tt.responseChannel, tt.channelErr)
			}

			if tt.expect != nil {
				tt.expect(ctx, m, tt.responseCall)
			}

			h.healthHangup(ctx, id, defaultHealthMaxRetryCount+1)
		})
	}
}
