package confbridgehandler

import (
	"context"
	stderrors "errors"
	"testing"

	"monorepo/bin-call-manager/models/channel"
)

func Test_StartContextIncoming_cov2_errors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		stasisData map[channel.StasisDataType]string
		setup      func(m *cov2Mocks)

		expectIs error
	}{
		{
			name: "confbridge id is missing",
			stasisData: map[channel.StasisDataType]string{
				channel.StasisDataTypeCallID: cov2CallID.String(),
			},
		},
		{
			name: "call id is missing",
			stasisData: map[channel.StasisDataType]string{
				channel.StasisDataTypeConfbridgeID: cov2CBID.String(),
			},
		},
		{
			name: "confbridge get failed",
			stasisData: map[channel.StasisDataType]string{
				channel.StasisDataTypeConfbridgeID: cov2CBID.String(),
				channel.StasisDataTypeCallID:       cov2CallID.String(),
			},
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
			expectIs: errCov2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			err := h.StartContextIncoming(ctx, &channel.Channel{ID: "c2-ch-1", StasisData: tt.stasisData})
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if tt.expectIs != nil && !stderrors.Is(err, tt.expectIs) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectIs, err)
			}
		})
	}
}
