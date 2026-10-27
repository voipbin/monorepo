package confbridgehandler

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/confbridge"
)

func Test_RingAnswer_cov2(t *testing.T) {
	ctx := context.Background()

	connectCB := cov2CB(func(c *confbridge.Confbridge) {
		c.Type = confbridge.TypeConnect
		c.ChannelCallIDs = map[string]uuid.UUID{"c2-ch-1": cov2CallID}
	})
	conferenceCB := cov2CB(func(c *confbridge.Confbridge) {
		c.Type = confbridge.TypeConference
		c.ChannelCallIDs = map[string]uuid.UUID{"c2-ch-1": cov2CallID}
	})

	tests := []struct {
		name string

		isRing bool
		setup  func(m *cov2Mocks)

		expectErr bool
	}{
		{
			name:   "ring: confbridge get failed",
			isRing: true,
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
			expectErr: true,
		},
		{
			name:   "ring: conference type does nothing",
			isRing: true,
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(conferenceCB, nil)
			},
		},
		{
			name:   "ring: channel ring failure is tolerated",
			isRing: true,
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(connectCB, nil)
				m.channel.EXPECT().Ring(ctx, "c2-ch-1").Return(errCov2)
			},
		},
		{
			name: "answer: confbridge get failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
			expectErr: true,
		},
		{
			name: "answer: conference type does nothing",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(conferenceCB, nil)
			},
		},
		{
			name: "answer: channel answer failure is tolerated",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(connectCB, nil)
				m.channel.EXPECT().Answer(ctx, "c2-ch-1").Return(errCov2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			var err error
			if tt.isRing {
				err = h.Ring(ctx, cov2CBID)
			} else {
				err = h.Answer(ctx, cov2CBID)
			}
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr && !stderrors.Is(err, errCov2) {
				t.Errorf("Wrong match. expect: wrapped errCov2, got: %v", err)
			}
		})
	}
}
