package confbridgehandler

import (
	"context"
	stderrors "errors"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/recording"
)

func Test_RecordingStart_cov2_errors(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	plain := cov2CB(func(c *confbridge.Confbridge) { c.ActiveflowID = cov2CallID2 })
	recording1 := cov2CB(func(c *confbridge.Confbridge) { c.RecordingID = cov2RecID })
	deleted := cov2CB(func(c *confbridge.Confbridge) { c.TMDelete = &now })

	tests := []struct {
		name string

		setup func(m *cov2Mocks)

		expectIs  error
		expectMsg string
	}{
		{
			name: "get failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
			expectIs: errCov2,
		},
		{
			name: "recording already progressing",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(recording1, nil)
			},
			expectMsg: "recording is already progressing",
		},
		{
			name: "confbridge is deleted",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(deleted, nil)
			},
			expectMsg: "invalid status",
		},
		{
			name: "recording start failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(plain, nil)
				m.rec.EXPECT().Start(ctx, cov2CallID2, recording.ReferenceTypeConfbridge, cov2CBID, recording.FormatWAV, 1000, "#", 60, cov2ExtID).Return(nil, errCov2)
			},
			expectIs: errCov2,
		},
		{
			name: "update recording id failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(plain, nil)
				m.rec.EXPECT().Start(ctx, cov2CallID2, recording.ReferenceTypeConfbridge, cov2CBID, recording.FormatWAV, 1000, "#", 60, cov2ExtID).Return(&recording.Recording{Identity: identityOf(cov2RecID)}, nil)
				m.db.EXPECT().ConfbridgeSetRecordingID(ctx, cov2CBID, cov2RecID).Return(errCov2)
			},
			expectIs: errCov2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			res, err := h.RecordingStart(ctx, cov2CBID, recording.FormatWAV, 1000, "#", 60, cov2ExtID)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if tt.expectIs != nil && !stderrors.Is(err, tt.expectIs) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectIs, err)
			}
			if tt.expectMsg != "" && !reflect.DeepEqual(err.Error(), tt.expectMsg) {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectMsg, err.Error())
			}
		})
	}
}

func Test_RecordingStop_cov2_success(t *testing.T) {
	ctx := context.Background()
	h, m := newCov2Handler(t)

	recording1 := cov2CB(func(c *confbridge.Confbridge) { c.RecordingID = cov2RecID })
	updated := cov2CB(nil)

	m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(recording1, nil)
	m.rec.EXPECT().Stop(ctx, cov2RecID).Return(&recording.Recording{Identity: identityOf(cov2RecID)}, nil)
	m.db.EXPECT().ConfbridgeSetRecordingID(ctx, cov2CBID, uuidNil()).Return(nil)
	m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(updated, nil)

	res, err := h.RecordingStop(ctx, cov2CBID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, updated) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", updated, res)
	}
}
