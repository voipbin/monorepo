package callhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
)

func Test_Delete_cov2a(t *testing.T) {
	deleted := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name  string
		setup func(m *cov2aMocks, c *call.Call, res *call.Call)

		expectNil bool
		wantErr   string
	}{
		{
			name: "call lookup error",
			setup: func(m *cov2aMocks, c *call.Call, res *call.Call) {
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom)
			},
			expectNil: true,
			wantErr:   "boom",
		},
		{
			name: "already deleted call is returned as is",
			setup: func(m *cov2aMocks, c *call.Call, res *call.Call) {
				c.TMDelete = &deleted
				m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil)
			},
		},
		{
			name: "hanging up error",
			setup: func(m *cov2aMocks, c *call.Call, res *call.Call) {
				gomock.InOrder(
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil),
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(nil, errCov2aBoom),
				)
			},
			expectNil: true,
			wantErr:   "boom",
		},
		{
			name: "delete error after the call is already terminating",
			setup: func(m *cov2aMocks, c *call.Call, res *call.Call) {
				terminating := cov2aNewCall(nil)
				terminating.Status = call.StatusTerminating
				gomock.InOrder(
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil),
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(terminating, nil),
					m.db.EXPECT().CallDelete(gomock.Any(), c.ID).Return(errCov2aBoom),
				)
			},
			expectNil: true,
			wantErr:   "boom",
		},
		{
			name: "hung up call is deleted without hanging up again",
			setup: func(m *cov2aMocks, c *call.Call, res *call.Call) {
				c.Status = call.StatusHangup
				res.TMDelete = &deleted
				gomock.InOrder(
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil),
					m.db.EXPECT().CallDelete(gomock.Any(), c.ID).Return(nil),
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(res, nil),
					m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), res.CustomerID, call.EventTypeCallDeleted, res),
				)
			},
		},
		{
			name: "progressing call is hung up and then deleted",
			setup: func(m *cov2aMocks, c *call.Call, res *call.Call) {
				terminating := cov2aNewCall(nil)
				terminating.Status = call.StatusTerminating
				res.TMDelete = &deleted
				gomock.InOrder(
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(c, nil),
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(terminating, nil),
					m.db.EXPECT().CallDelete(gomock.Any(), c.ID).Return(nil),
					m.db.EXPECT().CallGet(gomock.Any(), c.ID).Return(res, nil),
					m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), res.CustomerID, call.EventTypeCallDeleted, res),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := cov2aNewHandler(t)

			c := cov2aNewCall(nil)
			res := cov2aNewCall(nil)
			tt.setup(m, c, res)

			got, err := h.Delete(context.Background(), c.ID)
			cov2aCheckErr(t, err, tt.wantErr)
			if tt.expectNil {
				if got != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", got)
				}
				return
			}

			// the already-deleted case returns the looked-up call, the others return the deleted one.
			if c.TMDelete != nil {
				if !reflect.DeepEqual(got, c) {
					t.Errorf("Wrong match. expect: %v, got: %v", c, got)
				}
				return
			}
			if !reflect.DeepEqual(got, res) {
				t.Errorf("Wrong match. expect: %v, got: %v", res, got)
			}
		})
	}
}
