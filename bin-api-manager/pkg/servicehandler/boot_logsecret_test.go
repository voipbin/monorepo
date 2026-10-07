package servicehandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// The sentinel carries the DirectPrefix (otherwise AuthBoot returns before any
// log point) and a unique tail beyond the 12 chars that masking may keep.
const (
	bootSentinelHash = "direct.0123456789ab-SENTINEL-MUST-NOT-LEAK-9f3c"
	bootSentinelPart = "SENTINEL-MUST-NOT-LEAK"
)

func Test_AuthBoot_directHashNotInLogs(t *testing.T) {
	hook := newSecretLogHook(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := serviceHandler{reqHandler: mockReq}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	responseDirect := &dmdirect.Direct{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d1b2c3d4-0000-0000-0000-000000000001"),
			CustomerID: customerID,
		},
		ResourceType: dmdirect.ResourceTypeExtension,
		ResourceID:   uuid.FromStringOrNil("e1b2c3d4-0000-0000-0000-000000000001"),
		Hash:         bootSentinelHash,
	}

	mockReq.EXPECT().DirectV1DirectGetByHash(ctx, bootSentinelHash).Return(responseDirect, nil)
	// stop right after the "Retrieved direct info" log line
	mockReq.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(nil, fmt.Errorf("customer lookup failed"))

	if _, err := h.AuthBoot(ctx, bootSentinelHash); err == nil {
		t.Fatalf("expected an error from the customer lookup, got nil")
	}

	assertNoSecretInLogs(t, hook, "Retrieved direct info", bootSentinelHash, bootSentinelPart)
}
