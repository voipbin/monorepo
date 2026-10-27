package confbridgehandler

import (
	"context"
	stderrors "errors"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/externalmedia"
)

func Test_ExternalMediaStart_cov2_errors(t *testing.T) {
	ctx := context.Background()

	full := cov2CB(func(c *confbridge.Confbridge) {
		c.ExternalMediaIDs = make([]uuid.UUID, defaultMaxExternalMediaPerConfbridge)
	})
	plain := cov2CB(nil)

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
			name: "maximum external medias reached",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(full, nil)
			},
			expectMsg: "the confbridge has reached the maximum number of external medias",
		},
		{
			name: "external media start failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(plain, nil)
				m.ext.EXPECT().Start(ctx, cov2ExtID, externalmedia.ReferenceTypeConfbridge, cov2CBID, "192.0.2.1:5000",
					externalmedia.EncapsulationRTP, externalmedia.TransportUDP, "", "client", "ulaw",
					externalmedia.DirectionBoth, externalmedia.DirectionBoth).Return(nil, errCov2)
			},
			expectIs: errCov2,
		},
		{
			name: "add external media id failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(plain, nil)
				m.ext.EXPECT().Start(ctx, cov2ExtID, externalmedia.ReferenceTypeConfbridge, cov2CBID, "192.0.2.1:5000",
					externalmedia.EncapsulationRTP, externalmedia.TransportUDP, "", "client", "ulaw",
					externalmedia.DirectionBoth, externalmedia.DirectionBoth).Return(&externalmedia.ExternalMedia{ID: cov2ExtID}, nil)
				m.db.EXPECT().ConfbridgeAddExternalMediaID(ctx, cov2CBID, cov2ExtID).Return(errCov2)
			},
			expectIs: errCov2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			res, err := h.ExternalMediaStart(ctx, cov2CBID, cov2ExtID, "192.0.2.1:5000",
				externalmedia.EncapsulationRTP, externalmedia.TransportUDP, "", "client", "ulaw")
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

func Test_ExternalMediaStop_cov2_errors(t *testing.T) {
	ctx := context.Background()

	withMedia := cov2CB(func(c *confbridge.Confbridge) { c.ExternalMediaIDs = []uuid.UUID{cov2ExtID} })

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
			name: "external media is not in the confbridge",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cov2CB(nil), nil)
			},
			expectMsg: "the external media id is not associated with this confbridge",
		},
		{
			name: "external media stop failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(withMedia, nil)
				m.ext.EXPECT().Stop(ctx, cov2ExtID).Return(nil, errCov2)
			},
			expectIs: errCov2,
		},
		{
			name: "get updated confbridge failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(withMedia, nil)
				m.ext.EXPECT().Stop(ctx, cov2ExtID).Return(&externalmedia.ExternalMedia{ID: cov2ExtID}, nil)
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
			expectIs: errCov2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			res, err := h.ExternalMediaStop(ctx, cov2CBID, cov2ExtID)
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
