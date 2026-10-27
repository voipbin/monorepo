package confbridgehandler

import (
	"context"
	stderrors "errors"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/confbridge"
)

func Test_Kick_cov2(t *testing.T) {
	ctx := context.Background()

	cb := cov2CB(func(c *confbridge.Confbridge) {
		c.ChannelCallIDs = map[string]uuid.UUID{"c2-ch-1": cov2CallID}
	})

	tests := []struct {
		name string

		callID uuid.UUID
		setup  func(m *cov2Mocks)

		expectIs  error
		expectMsg string
	}{
		{
			name:   "confbridge get failed",
			callID: cov2CallID,
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
			expectIs: errCov2,
		},
		{
			name:   "hangup failed",
			callID: cov2CallID,
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cb, nil)
				m.channel.EXPECT().HangingUp(ctx, "c2-ch-1", ari.ChannelCauseNormalClearing).Return(nil, errCov2)
			},
			expectIs: errCov2,
		},
		{
			name:   "call is not in the confbridge",
			callID: cov2CallID2,
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cb, nil)
			},
			expectMsg: "joined call not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			err := h.Kick(ctx, cov2CBID, tt.callID)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
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
