package callhandler

import (
	"context"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/channelhandler"
)

// The sentinel keeps the DirectPrefix and has a unique tail beyond the first 12
// chars that MaskHash retains.
const (
	sipSentinelHash = "direct.0123456789ab-SENTINEL-MUST-NOT-LEAK-9f3c"
	sipSentinelPart = "SENTINEL-MUST-NOT-LEAK"
)

func Test_startIncomingDomainTypeSIPDirect_directHashNotInLogs(t *testing.T) {
	hook := newSecretLogHook(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockChannel := channelhandler.NewMockChannelHandler(mc)
	h := &callHandler{
		reqHandler:     mockReq,
		channelHandler: mockChannel,
	}
	ctx := context.Background()

	cn := &channel.Channel{
		ID:                "asterisk-call-58f54b64c7-2kwmb-1675216038.301",
		DestinationNumber: sipSentinelHash,
	}
	responseDirect := &dmdirect.Direct{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d1b2c3d4-0000-0000-0000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001"),
		},
		// unsupported type: the function logs "Retrieved direct info" and then hangs up
		ResourceType: "unsupported-type",
		ResourceID:   uuid.FromStringOrNil("e1b2c3d4-0000-0000-0000-000000000001"),
		Hash:         sipSentinelHash,
	}

	mockChannel.EXPECT().AddressGetSource(cn, commonaddress.TypeTel).Return(&commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821****0002"})
	mockReq.EXPECT().DirectV1DirectGetByHash(ctx, sipSentinelHash).Return(responseDirect, nil)
	mockChannel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

	if err := h.startIncomingDomainTypeSIPDirect(ctx, cn, sipSentinelHash); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	assertNoSecretInLogs(t, hook, "Starting direct call handler", sipSentinelHash, sipSentinelPart)
	assertNoSecretInLogs(t, hook, "Retrieved direct info", sipSentinelHash, sipSentinelPart)
}
