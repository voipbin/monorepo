package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
)

func Test_updateStatusRinging_cov2a(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *cov2aMocks, c *call.Call)

		wantErr string
	}{
		{
			name: "status update error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallSetStatusRinging(gomock.Any(), c.ID).Return(errCov2aBoom)
			},
			wantErr: "boom",
		},
		{
			name: "early execution runs the action next",
			setup: func(m *cov2aMocks, c *call.Call) {
				// the refetched call carries the early execution flag, so the next action is requested
				// from the flow and the failure of that request is tolerated.
				updated := cov2aNewCall(nil)
				updated.Status = call.StatusRinging
				updated.Data = map[call.DataType]string{call.DataTypeEarlyExecution: "true"}

				m.db.EXPECT().CallSetStatusRinging(gomock.Any(), c.ID).Return(nil)
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(updated, nil)
				m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), updated.CustomerID, call.EventTypeCallRinging, updated)
				m.db.EXPECT().CallSetActionNextHold(gomock.Any(), updated.ID, true).Return(nil)
				m.req.EXPECT().FlowV1ActiveflowGetNextAction(gomock.Any(), updated.ActiveflowID, updated.Action.ID).Return(nil, errCov2aBoom)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			c.Status = call.StatusDialing
			tt.setup(m, c)

			err := h.updateStatusRinging(context.Background(), &channel.Channel{ID: c.ChannelID}, c)
			cov2aCheckErr(t, err, tt.wantErr)
		})
	}
}

func Test_updateStatusProgressing_cov2a(t *testing.T) {
	groupcallID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000050")
	confbridgeID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000051")

	tests := []struct {
		name  string
		setup func(m *cov2aMocks, c *call.Call)

		wantErr string
	}{
		{
			name: "status update error",
			setup: func(m *cov2aMocks, c *call.Call) {
				m.db.EXPECT().CallSetStatusProgressing(gomock.Any(), c.ID).Return(errCov2aBoom)
			},
			wantErr: "boom",
		},
		{
			name: "groupcall and confbridge answer errors are only logged; early execution skips the action next",
			setup: func(m *cov2aMocks, c *call.Call) {
				c.Data = map[call.DataType]string{call.DataTypeEarlyExecution: "true"}

				updated := cov2aNewCall(nil)
				updated.Direction = call.DirectionOutgoing
				updated.GroupcallID = groupcallID
				updated.ConfbridgeID = confbridgeID

				m.db.EXPECT().CallSetStatusProgressing(gomock.Any(), c.ID).Return(nil)
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(updated, nil)
				m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), updated.CustomerID, call.EventTypeCallProgressing, updated)
				m.groupcall.EXPECT().AnswerCall(gomock.Any(), groupcallID, updated.ID).Return(errCov2aBoom)
				m.confbridge.EXPECT().Answer(gomock.Any(), confbridgeID).Return(errCov2aBoom)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			c.Status = call.StatusRinging
			tt.setup(m, c)

			err := h.updateStatusProgressing(context.Background(), &channel.Channel{ID: c.ChannelID}, c)
			cov2aCheckErr(t, err, tt.wantErr)
		})
	}
}
