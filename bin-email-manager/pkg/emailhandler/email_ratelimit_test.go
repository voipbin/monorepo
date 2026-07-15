package emailhandler

import (
	"context"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-email-manager/pkg/cachehandler"
	"monorepo/bin-email-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// Test_Create_RateLimitExceeded_FailClosed asserts that when the outbound
// email rate limit is exceeded (minute cap breached), Create returns a typed
// ResourceExhausted error and skips the create/send path entirely —
// mirroring the existing balance-check gating pattern. VOIP-1259.
func Test_Create_RateLimitExceeded_FailClosed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockSendgrid := NewMockEngineSendgrid(mc)
	mockMailgun := NewMockEngineMailgun(mc)

	h := &emailHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		utilHandler:   mockUtil,
		cache:         mockCache,

		engineSendgrid: mockSendgrid,
		engineMailgun:  mockMailgun,
	}

	ctx := context.Background()

	customerID := uuid.FromStringOrNil("e3000000-0000-4000-8000-000000000001")
	activeflowID := uuid.FromStringOrNil("e3000000-0000-4000-8000-000000000002")

	destinations := []commonaddress.Address{
		{
			Type:       commonaddress.TypeEmail,
			Target:     "test@voipbin.net",
			TargetName: "test name",
		},
	}

	mockReq.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(&cucustomer.Customer{
		ID:                         customerID,
		IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
	}, nil)
	mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeEmail, "", len(destinations)).Return(true, nil)

	// minute counter breaches the cap; the rate-limit gate must reject before
	// h.create/h.Send is ever reached.
	mockCache.EXPECT().RateLimitIncrement(ctx, gomock.Any(), gomock.Any()).Return(int64(999999), nil).Times(2)

	// No further interactions: EmailCreate, EmailGet, PublishWebhookEvent,
	// engineSendgrid.Send, etc. must NOT be invoked. gomock will fail the
	// test if any unexpected call lands.

	res, err := h.Create(ctx, customerID, activeflowID, destinations, "test subject", "test content", nil)

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
