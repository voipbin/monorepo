package callhandler

import (
	"context"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	cerrors "monorepo/bin-common-handler/models/errors"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	outboundconfighandler "monorepo/bin-call-manager/pkg/outboundconfighandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// Test_CreateCallOutgoing_RateLimitExceeded_FailClosed asserts that when the
// outbound call rate limit is exceeded (minute cap breached),
// CreateCallOutgoing returns a typed ResourceExhausted error and skips the
// outbound config fetch / channel start / call creation entirely — mirroring
// the existing balance-check gating pattern. VOIP-1259.
func Test_CreateCallOutgoing_RateLimitExceeded_FailClosed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockChannel := channelhandler.NewMockChannelHandler(mc)
	mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &callHandler{
		utilHandler:           mockUtil,
		reqHandler:            mockReq,
		notifyHandler:         mockNotify,
		db:                    mockDB,
		channelHandler:        mockChannel,
		outboundConfigHandler: mockOutboundConfig,
		cache:                 mockCache,
	}

	ctx := context.Background()

	id := uuid.FromStringOrNil("c1000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("c1000000-0000-4000-8000-000000000002")
	flowID := uuid.FromStringOrNil("c1000000-0000-4000-8000-000000000003")
	activeflowID := uuid.FromStringOrNil("c1000000-0000-4000-8000-000000000004")

	source := commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: "+141****0100",
	}
	destination := commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: "+141****0199",
	}

	mockReq.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(&cucustomer.Customer{
		ID:                         customerID,
		Status:                     cucustomer.StatusActive,
		IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
	}, nil)
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)

	// minute counter breaches the cap; the rate-limit gate must reject before
	// the outbound config fetch / channel start / call creation is ever
	// reached.
	mockCache.EXPECT().RateLimitIncrement(ctx, gomock.Any(), gomock.Any()).Return(int64(999999), nil).Times(2)

	// No further interactions: outbound config fetch, dialroutes, activeflow
	// create, channel start, call create, etc. must NOT be invoked. gomock
	// will fail the test if any unexpected call lands.

	res, err := h.CreateCallOutgoing(ctx, id, customerID, flowID, activeflowID, uuid.Nil, uuid.Nil, source, destination, false, false, "", nil, nil)

	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}

	verr, ok := err.(*cerrors.VoipbinError)
	if !ok {
		t.Fatalf("Wrong match. expect: *cerrors.VoipbinError, got: %T (%v)", err, err)
	}
	if verr.Reason != "RATE_LIMIT_EXCEEDED" {
		t.Errorf("Wrong match. expect reason: RATE_LIMIT_EXCEEDED, got: %s", verr.Reason)
	}
}
