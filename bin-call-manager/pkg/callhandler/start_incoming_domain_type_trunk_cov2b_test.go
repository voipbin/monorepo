package callhandler

import (
	"context"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
)

func Test_startIncomingDomainTypeTrunkDestinationTypeTel_cov2b_flowCreateError(t *testing.T) {
	h, m := newCov2bHandler(t)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("cc1b2c3d-0000-4000-8000-00000000c2b1")
	cn := &channel.Channel{ID: "ch-cov2b-trunk"}

	m.req.EXPECT().FlowV1FlowCreate(ctx, customerID, gomock.Any(), "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, fmt.Errorf("rpc failure"))
	m.channel.EXPECT().HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder).Return(nil, nil)

	err := h.startIncomingDomainTypeTrunkDestinationTypeTel(ctx, cn, customerID,
		&commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230001"},
		&commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230002"})
	if err != nil {
		t.Errorf("Wrong error. expect: nil, got: %v", err)
	}
}
