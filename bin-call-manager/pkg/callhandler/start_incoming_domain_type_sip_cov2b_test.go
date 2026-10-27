package callhandler

import (
	"context"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

func Test_startIncomingDomainTypeSIPDirect_cov2b(t *testing.T) {
	resourceID := uuid.FromStringOrNil("dd1b2c3d-0000-4000-8000-00000000c2b1")
	const chID = "ch-cov2b-direct"

	tests := []struct {
		name         string
		resourceType string
		setup        func(ctx context.Context, m *cov2bMocks)
	}{
		{
			name:         "extension resource lookup fails",
			resourceType: "extension",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().RegistrarV1ExtensionGet(ctx, resourceID).Return(nil, fmt.Errorf("rpc failure"))
			},
		},
		{
			name:         "conference resource lookup fails",
			resourceType: "conference",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().ConferenceV1ConferenceGet(ctx, resourceID).Return(nil, fmt.Errorf("rpc failure"))
			},
		},
		{
			name:         "queue resource lookup fails",
			resourceType: "queue",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().QueueV1QueueGet(ctx, resourceID).Return(nil, fmt.Errorf("rpc failure"))
			},
		},
		{
			name:         "flow resource lookup fails",
			resourceType: "flow",
			setup: func(ctx context.Context, m *cov2bMocks) {
				m.req.EXPECT().FlowV1FlowGet(ctx, resourceID).Return(nil, fmt.Errorf("rpc failure"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2bHandler(t)
			ctx := context.Background()
			cn := &channel.Channel{ID: chID}

			m.channel.EXPECT().AddressGetSource(cn, commonaddress.TypeTel).Return(&commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230001"})
			m.req.EXPECT().DirectV1DirectGetByHash(ctx, "hash-cov2b").Return(&dmdirect.Direct{ResourceType: tt.resourceType, ResourceID: resourceID}, nil)
			tt.setup(ctx, m)
			m.channel.EXPECT().HangingUp(ctx, chID, ari.ChannelCauseNoRouteDestination).Return(nil, nil)

			if err := h.startIncomingDomainTypeSIPDirect(ctx, cn, "hash-cov2b"); err != nil {
				t.Errorf("Wrong error. expect: nil, got: %v", err)
			}
		})
	}
}
