package callhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/testhelper"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func Test_HealthCheck_cov2b(t *testing.T) {
	id := uuid.FromStringOrNil("5a1b2c3d-0000-4000-8000-00000000c2b1")
	channelID := "5a1b2c3d-0000-4000-8000-00000000c2b2"
	ended := testhelper.TimePtr("2023-01-18T03:22:18.995000Z")

	activeCall := &call.Call{
		Identity:  commonidentity.Identity{ID: id},
		ChannelID: channelID,
		Status:    call.StatusProgressing,
	}

	tests := []struct {
		name string

		setup func(ctx context.Context, m *cov2bMocks)
	}{
		{
			name: "get call error stops the chain",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallGet(ctx, id).Return(nil, fmt.Errorf("db failure"))
			},
		},
		{
			name: "hung up call stops the chain",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallGet(ctx, id).Return(&call.Call{Identity: commonidentity.Identity{ID: id}, Status: call.StatusHangup}, nil)
			},
		},
		{
			name: "deleted call stops the chain",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallGet(ctx, id).Return(&call.Call{Identity: commonidentity.Identity{ID: id}, Status: call.StatusProgressing, TMDelete: ended}, nil)
			},
		},
		{
			name: "channel read failure stops the chain",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallGet(ctx, id).Return(activeCall, nil)
				m.db.EXPECT().ChannelGet(ctx, channelID).Return(nil, fmt.Errorf("db failure"))
			},
		},
		{
			name: "health request error ends quietly after counting a missing channel",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallGet(ctx, id).Return(activeCall, nil)
				m.db.EXPECT().ChannelGet(ctx, channelID).Return(nil, dbhandler.ErrNotFound)
				m.req.EXPECT().CallV1CallHealth(ctx, id, defaultHealthDelay, 2).Return(fmt.Errorf("rpc failure"))
			},
		},
		{
			name: "ended channel increments the retry count",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.db.EXPECT().CallGet(ctx, id).Return(activeCall, nil)
				m.db.EXPECT().ChannelGet(ctx, channelID).Return(&channel.Channel{ID: channelID, TMEnd: ended}, nil)
				m.req.EXPECT().CallV1CallHealth(ctx, id, defaultHealthDelay, 2).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()
			tt.setup(ctx, m)

			h.HealthCheck(ctx, id, 1)
		})
	}
}

func Test_healthHangup_cov2b(t *testing.T) {
	id := uuid.FromStringOrNil("5a1b2c3d-0000-4000-8000-00000000c2b3")
	activeflowID := uuid.FromStringOrNil("5a1b2c3d-0000-4000-8000-00000000c2b4")
	channelID := "5a1b2c3d-0000-4000-8000-00000000c2b5"

	dialing := &call.Call{
		Identity:     commonidentity.Identity{ID: id, CustomerID: uuid.FromStringOrNil("5a1b2c3d-0000-4000-8000-00000000c2b6")},
		ChannelID:    channelID,
		ActiveflowID: activeflowID,
		Status:       call.StatusDialing,
	}

	tests := []struct {
		name  string
		setup func(ctx context.Context, m *cov2bMocks)
	}{
		{
			name: "alive channel restarts the health check and a request error is only logged",
			setup: func(ctx context.Context, m *cov2bMocks) {
				progressing := *dialing
				progressing.Status = call.StatusProgressing
				m.db.EXPECT().CallGetFromDB(ctx, id).Return(&progressing, nil)
				m.db.EXPECT().ChannelGet(ctx, channelID).Return(&channel.Channel{ID: channelID}, nil)
				m.req.EXPECT().CallV1CallHealth(ctx, id, defaultHealthDelay, 0).Return(fmt.Errorf("rpc failure"))
			},
		},
		{
			name: "activeflow stop failure after a failed call is only logged",
			setup: func(ctx context.Context, m *cov2bMocks) {
				hungup := *dialing
				hungup.Status = call.StatusHangup
				m.db.EXPECT().CallGetFromDB(ctx, id).Return(dialing, nil)
				m.db.EXPECT().ChannelGet(ctx, channelID).Return(nil, dbhandler.ErrNotFound)
				m.db.EXPECT().CallSetHangup(ctx, id, call.HangupReasonFailed, call.HangupByLocal).Return(nil)
				m.db.EXPECT().CallGet(ctx, id).Return(&hungup, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, hungup.CustomerID, call.EventTypeCallHangup, &hungup)
				m.req.EXPECT().FlowV1ActiveflowStop(ctx, activeflowID).Return(nil, fmt.Errorf("rpc failure"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()
			tt.setup(ctx, m)

			h.healthHangup(ctx, id, defaultHealthMaxRetryCount+1)
		})
	}
}
