package confbridgehandler

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/confbridge"
)

func Test_Delete_cov2_errors(t *testing.T) {
	ctx := context.Background()

	terminated := cov2CB(func(c *confbridge.Confbridge) { c.Status = confbridge.StatusTerminated })

	tests := []struct {
		name  string
		setup func(m *cov2Mocks)
	}{
		{
			name: "terminating failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
		},
		{
			name: "db delete failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(terminated, nil)
				m.db.EXPECT().ConfbridgeDelete(ctx, cov2CBID).Return(errCov2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			res, err := h.Delete(ctx, cov2CBID)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if !stderrors.Is(err, errCov2) {
				t.Errorf("Wrong match. expect: wrapped errCov2, got: %v", err)
			}
		})
	}
}

func Test_Create_cov2_errors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		setup func(m *cov2Mocks)
	}{
		{
			name: "db create failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeCreate(ctx, cov2CB(func(c *confbridge.Confbridge) {
					c.CustomerID = cov2CallID
					c.ActiveflowID = cov2CallID2
					c.ReferenceType = confbridge.ReferenceTypeCall
					c.ReferenceID = cov2RecID
					c.Type = confbridge.TypeConnect
					c.Status = confbridge.StatusProgressing
					c.Flags = []confbridge.Flag{}
					c.ChannelCallIDs = map[string]uuid.UUID{}
					c.RecordingIDs = []uuid.UUID{}
					c.ExternalMediaIDs = []uuid.UUID{}
				})).Return(errCov2)
			},
		},
		{
			name: "get created confbridge failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			m.util.EXPECT().UUIDCreate().Return(cov2CBID)
			tt.setup(m)

			res, err := h.Create(ctx, cov2CallID, cov2CallID2, confbridge.ReferenceTypeCall, cov2RecID, confbridge.TypeConnect)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if !stderrors.Is(err, errCov2) {
				t.Errorf("Wrong match. expect: wrapped errCov2, got: %v", err)
			}
		})
	}
}

func Test_List_cov2_error(t *testing.T) {
	ctx := context.Background()
	h, m := newCov2Handler(t)

	m.db.EXPECT().ConfbridgeList(ctx, uint64(10), "token", map[confbridge.Field]any{confbridge.Field("customer_id"): "c2-customer"}).Return(nil, errCov2)

	res, err := h.List(ctx, 10, "token", map[string]string{"customer_id": "c2-customer"})
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %v", res)
	}
	if !stderrors.Is(err, errCov2) {
		t.Errorf("Wrong match. expect: wrapped errCov2, got: %v", err)
	}
}

func Test_UpdateStatus_cov2_terminatedWithDuration(t *testing.T) {
	ctx := context.Background()
	h, m := newCov2Handler(t)

	created := time.Now().Add(-time.Minute)
	res := cov2CB(func(c *confbridge.Confbridge) {
		c.Status = confbridge.StatusTerminated
		c.TMCreate = &created
	})

	m.db.EXPECT().ConfbridgeSetStatus(ctx, cov2CBID, confbridge.StatusTerminated).Return(nil)
	m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(res, nil)
	m.notify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeTerminated, res)

	got, err := h.UpdateStatus(ctx, cov2CBID, confbridge.StatusTerminated)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if got != res {
		t.Errorf("Wrong match. expect: %v, got: %v", res, got)
	}
}
