package dbhandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/interaction"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

func Test_InteractionCreate(t *testing.T) {
	tests := []struct {
		name        string
		interaction *interaction.Interaction
	}{
		{
			name: "normal incoming call",

			interaction: &interaction.Interaction{
				ID:            uuid.FromStringOrNil("a1b2c3d4-0001-0001-0001-000000000001"),
				CustomerID:    uuid.FromStringOrNil("a1b2c3d4-0001-0001-0001-000000000002"),
				Direction:     "incoming",
				PeerType:      "tel",
				PeerTarget:    "peerTarget-call-incoming",
				LocalType:     "tel",
				LocalTarget:   "localTarget-call-incoming",
				ReferenceType: "call",
				ReferenceID:   uuid.FromStringOrNil("a1b2c3d4-0001-0001-0001-000000000003"),
				TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC); return &t }(),
			},
		},
		{
			name: "outgoing conversation message LINE",

			interaction: &interaction.Interaction{
				ID:            uuid.FromStringOrNil("b1b2c3d4-0002-0002-0002-000000000001"),
				CustomerID:    uuid.FromStringOrNil("b1b2c3d4-0002-0002-0002-000000000002"),
				Direction:     "outgoing",
				PeerType:      "line",
				PeerTarget:    "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				LocalType:     "line",
				LocalTarget:   "",
				ReferenceType: "conversation_message",
				ReferenceID:   uuid.FromStringOrNil("b1b2c3d4-0002-0002-0002-000000000003"),
				TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 11, 0, 0, 0, time.UTC); return &t }(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// InteractionCreate does NOT call utilHandler or cache.
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			if err := h.InteractionCreate(ctx, tt.interaction); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_InteractionCreate_duplicate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	i := &interaction.Interaction{
		ID:            uuid.FromStringOrNil("c1b2c3d4-0003-0003-0003-000000000001"),
		CustomerID:    uuid.FromStringOrNil("c1b2c3d4-0003-0003-0003-000000000002"),
		Direction:     "incoming",
		PeerType:      "tel",
		PeerTarget:    "uniquePeerTarget-idem-001",
		LocalType:     "tel",
		LocalTarget:   "uniqueLocalTarget-idem-001",
		ReferenceType: "call",
		ReferenceID:   uuid.FromStringOrNil("c1b2c3d4-0003-0003-0003-000000000003"),
		TMCreate:      func() *time.Time { t := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC); return &t }(),
	}

	// first insert
	if err := h.InteractionCreate(ctx, i); err != nil {
		t.Fatalf("first insert failed: %v", err)
	}

	// duplicate insert — must NOT error (idempotent at-least-once guard)
	if err := h.InteractionCreate(ctx, i); err != nil {
		t.Errorf("duplicate insert should be idempotent (no error), got: %v", err)
	}
}
