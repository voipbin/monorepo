package confbridgehandler

import (
	"context"
	stderrors "errors"
	"testing"

	"monorepo/bin-call-manager/models/confbridge"
)

func Test_Terminating_cov2_errors(t *testing.T) {
	ctx := context.Background()

	progressing := cov2CB(func(c *confbridge.Confbridge) { c.Status = confbridge.StatusProgressing })
	withBridge := cov2CB(func(c *confbridge.Confbridge) {
		c.Status = confbridge.StatusTerminating
		c.BridgeID = "c2-bridge-1"
	})

	tests := []struct {
		name string

		setup func(m *cov2Mocks)

		expectIs error
	}{
		{
			name: "get failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
			expectIs: errCov2,
		},
		{
			name: "update status failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(progressing, nil)
				m.db.EXPECT().ConfbridgeSetStatus(ctx, cov2CBID, confbridge.StatusTerminating).Return(errCov2)
			},
			expectIs: errCov2,
		},
		{
			name: "bridge destroy failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(progressing, nil)
				m.db.EXPECT().ConfbridgeSetStatus(ctx, cov2CBID, confbridge.StatusTerminating).Return(nil)
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(withBridge, nil)
				m.notify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeTerminating, withBridge)
				m.bridge.EXPECT().Destroy(ctx, "c2-bridge-1").Return(errCov2)
			},
			expectIs: errCov2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			res, err := h.Terminating(ctx, cov2CBID)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if !stderrors.Is(err, tt.expectIs) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectIs, err)
			}
		})
	}
}

func Test_Terminate_cov2(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		setup func(m *cov2Mocks)

		expectErr bool
	}{
		{
			name: "get failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
			expectErr: true,
		},
		{
			name: "not terminating does nothing",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cov2CB(func(c *confbridge.Confbridge) { c.Status = confbridge.StatusProgressing }), nil)
			},
		},
		{
			name: "update status failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cov2CB(func(c *confbridge.Confbridge) { c.Status = confbridge.StatusTerminating }), nil)
				m.db.EXPECT().ConfbridgeSetStatus(ctx, cov2CBID, confbridge.StatusTerminated).Return(errCov2)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			err := h.Terminate(ctx, cov2CBID)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr && !stderrors.Is(err, errCov2) {
				t.Errorf("Wrong match. expect: wrapped errCov2, got: %v", err)
			}
		})
	}
}
