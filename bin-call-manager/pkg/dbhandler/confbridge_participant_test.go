package dbhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/cachehandler"
)

// Test_ConfbridgeParticipantClaim checks the claim and the release are passed through to the cache.
func Test_ConfbridgeParticipantClaim(t *testing.T) {
	id := uuid.FromStringOrNil("6c3ead74-3333-11ef-c001-000000000001")
	ttl := 90 * time.Second

	tests := []struct {
		name     string
		claimed  bool
		claimErr error
	}{
		{"claimed", true, nil},
		{"held by another", false, nil},
		{"cache error", false, fmt.Errorf("cache error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{cache: mockCache}
			ctx := context.Background()

			mockCache.EXPECT().ConfbridgeParticipantClaim(ctx, id, ttl).Return(tt.claimed, tt.claimErr)

			res, err := h.ConfbridgeParticipantClaim(ctx, id, ttl)
			if (err != nil) != (tt.claimErr != nil) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.claimErr, err)
			}
			if res != tt.claimed {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.claimed, res)
			}
		})
	}
}

func Test_ConfbridgeParticipantRelease(t *testing.T) {
	id := uuid.FromStringOrNil("6c3ead74-3333-11ef-c001-000000000002")

	for _, releaseErr := range []error{nil, fmt.Errorf("cache error")} {
		mc := gomock.NewController(t)

		mockCache := cachehandler.NewMockCacheHandler(mc)
		h := handler{cache: mockCache}
		ctx := context.Background()

		mockCache.EXPECT().ConfbridgeParticipantRelease(ctx, id).Return(releaseErr)

		if err := h.ConfbridgeParticipantRelease(ctx, id); (err != nil) != (releaseErr != nil) {
			t.Errorf("Wrong match. expect: %v, got: %v", releaseErr, err)
		}
		mc.Finish()
	}
}
