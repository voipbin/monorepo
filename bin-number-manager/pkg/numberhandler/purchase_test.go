package numberhandler

import (
	"context"
	"errors"
	"fmt"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/models/providernumber"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/providerhandler"
)

// newPurchaseMocks returns a handler with two providers, "first" and "second" (both mocks of
// the provider interface), for the purchase routing tests.
func newPurchaseMocks(t *testing.T) (*numberHandler, *errTestMocks, *providerhandler.MockProvider, *providerhandler.MockProvider) {
	t.Helper()

	h, m := newErrTestHandler(t)

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	return h, m, providerhandler.NewMockProvider(mc), providerhandler.NewMockProvider(mc)
}

func Test_Create_fallback(t *testing.T) {
	ctx := context.Background()

	second := number.ProviderName("second")
	purchased := &providernumber.ProviderNumber{ID: "provider-id-1", Status: number.StatusActive}
	created := &number.Number{
		Identity:     commonidentity.Identity{ID: errTestNumberID, CustomerID: errTestCustomerID},
		Number:       errTestNumber,
		ProviderName: second,
	}

	expectBefore := func(m *errTestMocks, p1, p2 providerhandler.Provider) {
		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)
		m.db.EXPECT().NumberList(ctx, uint64(1), "", map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: errTestNumber}).Return([]*number.Number{}, nil)
		m.provider.EXPECT().PurchaseCandidates(ctx).Return([]*providerhandler.Candidate{
			{Name: number.ProviderNameTelnyx, Provider: p1},
			{Name: second, Provider: p2},
		}, nil)
	}

	t.Run("a rejected purchase falls back to the next provider and records it", func(t *testing.T) {
		h, m, p1, p2 := newPurchaseMocks(t)
		expectBefore(m, p1, p2)

		p1.EXPECT().NumberPurchase(errTestNumber).Return(nil, fmt.Errorf("%w: no stock", provider.ErrPurchaseRejected))
		p2.EXPECT().NumberPurchase(errTestNumber).Return(purchased, nil)

		m.db.EXPECT().NumberList(ctx, uint64(1), "", map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: errTestNumber}).Return([]*number.Number{}, nil)
		m.util.EXPECT().UUIDCreate().Return(errTestNumberID)
		m.db.EXPECT().NumberCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, n *number.Number) error {
			if n.ProviderName != second || n.ProviderReferenceID != "provider-id-1" {
				t.Errorf("Wrong match. expect the number to be recorded with the provider that sold it, got: %s %s", n.ProviderName, n.ProviderReferenceID)
			}
			return nil
		})
		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(created, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, errTestCustomerID, number.EventTypeNumberCreated, created)
		// the tags go to the provider that sold the number
		p2.EXPECT().NumberUpdateTags(ctx, created, gomock.Any()).Return(nil)

		res, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", "")
		if err != nil || res != created {
			t.Errorf("Wrong match. expect: created, got: %v, %v", res, err)
		}
	})

	t.Run("an unconfirmed purchase stops without a fallback", func(t *testing.T) {
		h, m, p1, p2 := newPurchaseMocks(t)
		expectBefore(m, p1, p2)

		// p2 has no expectation: calling it fails the test
		p1.EXPECT().NumberPurchase(errTestNumber).Return(nil, fmt.Errorf("%w: timeout", provider.ErrPurchaseUnconfirmed))

		res, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", "")
		if res != nil || !errors.Is(err, provider.ErrPurchaseUnconfirmed) {
			t.Errorf("Wrong match. expect: unconfirmed error, got: %v, %v", res, err)
		}
	})

	t.Run("an error of an unknown kind is treated as unconfirmed", func(t *testing.T) {
		h, m, p1, p2 := newPurchaseMocks(t)
		expectBefore(m, p1, p2)

		p1.EXPECT().NumberPurchase(errTestNumber).Return(nil, fmt.Errorf("something else"))

		res, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", "")
		if res != nil || !errors.Is(err, provider.ErrPurchaseUnconfirmed) {
			t.Errorf("Wrong match. expect: unconfirmed error, got: %v, %v", res, err)
		}
	})

	t.Run("every provider rejecting fails the purchase", func(t *testing.T) {
		h, m, p1, p2 := newPurchaseMocks(t)
		expectBefore(m, p1, p2)

		p1.EXPECT().NumberPurchase(errTestNumber).Return(nil, fmt.Errorf("%w: a", provider.ErrPurchaseRejected))
		p2.EXPECT().NumberPurchase(errTestNumber).Return(nil, fmt.Errorf("%w: b", provider.ErrPurchaseRejected))

		res, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", "")
		if res != nil || !errors.Is(err, provider.ErrPurchaseRejected) {
			t.Errorf("Wrong match. expect: rejected error, got: %v, %v", res, err)
		}
	})

	t.Run("the rollback releases the number at the provider that sold it", func(t *testing.T) {
		h, m, p1, p2 := newPurchaseMocks(t)
		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)
		m.provider.EXPECT().PurchaseCandidates(ctx).Return([]*providerhandler.Candidate{
			{Name: number.ProviderNameTelnyx, Provider: p1},
			{Name: second, Provider: p2},
		}, nil)

		filter := map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: errTestNumber}
		gomock.InOrder(
			m.db.EXPECT().NumberList(ctx, uint64(1), "", filter).Return([]*number.Number{}, nil),
			m.db.EXPECT().NumberList(ctx, uint64(1), "", filter).Return(nil, errTestDBFailure),
		)
		p1.EXPECT().NumberPurchase(errTestNumber).Return(nil, fmt.Errorf("%w: a", provider.ErrPurchaseRejected))
		p2.EXPECT().NumberPurchase(errTestNumber).Return(purchased, nil)
		p2.EXPECT().NumberRelease(ctx, &number.Number{ProviderName: second, ProviderReferenceID: "provider-id-1"}).Return(nil)

		res, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", "")
		if res != nil || !errors.Is(err, errTestDBFailure) {
			t.Errorf("Wrong match. expect: db failure, got: %v, %v", res, err)
		}
	})
}

func Test_Create_noPurchase(t *testing.T) {
	ctx := context.Background()

	t.Run("no usable provider", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)
		m.db.EXPECT().NumberList(ctx, uint64(1), "", gomock.Any()).Return([]*number.Number{}, nil)
		m.provider.EXPECT().PurchaseCandidates(ctx).Return([]*providerhandler.Candidate{}, nil)

		if res, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", ""); res != nil || err == nil {
			t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
		}
	})

	t.Run("candidate lookup failure", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)
		m.db.EXPECT().NumberList(ctx, uint64(1), "", gomock.Any()).Return([]*number.Number{}, nil)
		m.provider.EXPECT().PurchaseCandidates(ctx).Return(nil, errTestDBFailure)

		if _, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", ""); !errors.Is(err, errTestDBFailure) {
			t.Errorf("Wrong match. expect: db failure, got: %v", err)
		}
	})

	t.Run("an already registered number is not purchased", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)
		m.db.EXPECT().NumberList(ctx, uint64(1), "", gomock.Any()).Return([]*number.Number{{Number: errTestNumber}}, nil)

		if _, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", ""); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("duplicate check failure", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)
		m.db.EXPECT().NumberList(ctx, uint64(1), "", gomock.Any()).Return(nil, errTestDBFailure)

		if _, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", ""); !errors.Is(err, errTestDBFailure) {
			t.Errorf("Wrong match. expect: db failure, got: %v", err)
		}
	})

	t.Run("a canceled context stops before the first provider", func(t *testing.T) {
		h, m, p1, _ := newPurchaseMocks(t)
		canceled, cancel := context.WithCancel(ctx)
		cancel()

		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(canceled, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)
		m.db.EXPECT().NumberList(canceled, uint64(1), "", gomock.Any()).Return([]*number.Number{}, nil)
		m.provider.EXPECT().PurchaseCandidates(canceled).Return([]*providerhandler.Candidate{{Name: number.ProviderNameTelnyx, Provider: p1}}, nil)

		if _, err := h.Create(canceled, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", ""); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})
}

func Test_GetAvailableNumbers_providerFailure(t *testing.T) {
	ctx := context.Background()
	h, m := newErrTestHandler(t)

	m.provider.EXPECT().ForSearch(ctx).Return(nil, fmt.Errorf("no credential"))

	if res, err := h.GetAvailableNumbers(ctx, "US", 1); res != nil || err == nil {
		t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
	}
}

func Test_Delete_providerResolution(t *testing.T) {
	ctx := context.Background()

	t.Run("a number of a disabled provider is released through the provider resolution", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		n := &number.Number{Identity: commonidentity.Identity{ID: errTestNumberID, CustomerID: errTestCustomerID}, ProviderName: number.ProviderNameTelnyx}

		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(n, nil).Times(2)
		m.provider.EXPECT().ForRelease(ctx, number.ProviderNameTelnyx).Return(m.telnyx, nil)
		m.telnyx.EXPECT().NumberRelease(ctx, n).Return(nil)
		m.db.EXPECT().NumberDelete(ctx, errTestNumberID).Return(nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, errTestCustomerID, number.EventTypeNumberDeleted, n)

		if _, err := h.Delete(ctx, errTestNumberID); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("a provider without a credential keeps the record", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		n := &number.Number{Identity: commonidentity.Identity{ID: errTestNumberID, CustomerID: errTestCustomerID}, ProviderName: number.ProviderNameTelnyx}

		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(n, nil)
		m.provider.EXPECT().ForRelease(ctx, number.ProviderNameTelnyx).Return(nil, fmt.Errorf("no credential"))

		if res, err := h.Delete(ctx, errTestNumberID); res != nil || err == nil {
			t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
		}
	})

	t.Run("a virtual number never asks the provider handler", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		n := &number.Number{Identity: commonidentity.Identity{ID: errTestNumberID, CustomerID: errTestCustomerID}, ProviderName: number.ProviderNameNone}

		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(n, nil).Times(2)
		m.db.EXPECT().NumberDelete(ctx, errTestNumberID).Return(nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, errTestCustomerID, number.EventTypeNumberDeleted, n)

		if _, err := h.Delete(ctx, errTestNumberID); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("an unknown provider is unsupported", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		n := &number.Number{Identity: commonidentity.Identity{ID: errTestNumberID}, ProviderName: "unknown"}

		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(n, nil)
		m.provider.EXPECT().ForRelease(ctx, number.ProviderName("unknown")).Return(nil, fmt.Errorf("unknown: %w", dbhandler.ErrNotFound))

		if _, err := h.Delete(ctx, errTestNumberID); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})
}

// the search is telnyx only, so the number a customer picked is a telnyx stock. A twilio that
// holds no such number refuses the purchase, which is the normal case and falls back.
func Test_Create_twilioFirstCandidate(t *testing.T) {
	ctx := context.Background()

	twilioNumber := &providernumber.ProviderNumber{ID: "PN00000000000000000000000000000000", Status: number.StatusActive}
	telnyxNumber := &providernumber.ProviderNumber{ID: "telnyx-id-1", Status: number.StatusActive}
	created := &number.Number{
		Identity:     commonidentity.Identity{ID: errTestNumberID, CustomerID: errTestCustomerID},
		Number:       errTestNumber,
		ProviderName: number.ProviderNameTelnyx,
	}

	expectBefore := func(m *errTestMocks, twilio, telnyx providerhandler.Provider) {
		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)
		m.db.EXPECT().NumberList(ctx, uint64(1), "", map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: errTestNumber}).Return([]*number.Number{}, nil)
		m.provider.EXPECT().PurchaseCandidates(ctx).Return([]*providerhandler.Candidate{
			{Name: number.ProviderNameTwilio, Provider: twilio},
			{Name: number.ProviderNameTelnyx, Provider: telnyx},
		}, nil)
	}

	t.Run("twilio without the number rejects and telnyx sells it", func(t *testing.T) {
		h, m, twilio, telnyx := newPurchaseMocks(t)
		expectBefore(m, twilio, telnyx)

		twilio.EXPECT().NumberPurchase(errTestNumber).Return(nil, fmt.Errorf("%w: not available", provider.ErrPurchaseRejected))
		telnyx.EXPECT().NumberPurchase(errTestNumber).Return(telnyxNumber, nil)

		m.db.EXPECT().NumberList(ctx, uint64(1), "", map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: errTestNumber}).Return([]*number.Number{}, nil)
		m.util.EXPECT().UUIDCreate().Return(errTestNumberID)
		m.db.EXPECT().NumberCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, n *number.Number) error {
			if n.ProviderName != number.ProviderNameTelnyx || n.ProviderReferenceID != "telnyx-id-1" {
				t.Errorf("Wrong match. expect: telnyx, got: %s %s", n.ProviderName, n.ProviderReferenceID)
			}
			return nil
		})
		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(created, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, errTestCustomerID, number.EventTypeNumberCreated, created)
		telnyx.EXPECT().NumberUpdateTags(ctx, created, gomock.Any()).Return(nil)

		if res, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", ""); err != nil || res != created {
			t.Errorf("Wrong match. expect: created, got: %v, %v", res, err)
		}
	})

	t.Run("an unconfirmed twilio purchase never reaches telnyx", func(t *testing.T) {
		h, m, twilio, telnyx := newPurchaseMocks(t)
		expectBefore(m, twilio, telnyx)

		// telnyx has no expectation: a call fails the test
		twilio.EXPECT().NumberPurchase(errTestNumber).Return(nil, fmt.Errorf("%w: http 503", provider.ErrPurchaseUnconfirmed))

		if res, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", ""); res != nil || !errors.Is(err, provider.ErrPurchaseUnconfirmed) {
			t.Errorf("Wrong match. expect: unconfirmed, got: %v, %v", res, err)
		}
	})

	t.Run("a twilio sale is recorded as twilio and a failed registration releases it at twilio", func(t *testing.T) {
		h, m, twilio, telnyx := newPurchaseMocks(t)
		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)
		m.provider.EXPECT().PurchaseCandidates(ctx).Return([]*providerhandler.Candidate{
			{Name: number.ProviderNameTwilio, Provider: twilio},
			{Name: number.ProviderNameTelnyx, Provider: telnyx},
		}, nil)

		filter := map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: errTestNumber}
		gomock.InOrder(
			m.db.EXPECT().NumberList(ctx, uint64(1), "", filter).Return([]*number.Number{}, nil),
			m.db.EXPECT().NumberList(ctx, uint64(1), "", filter).Return(nil, errTestDBFailure),
		)
		twilio.EXPECT().NumberPurchase(errTestNumber).Return(twilioNumber, nil)
		twilio.EXPECT().NumberRelease(ctx, &number.Number{ProviderName: number.ProviderNameTwilio, ProviderReferenceID: twilioNumber.ID}).Return(nil)

		if res, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", ""); res != nil || !errors.Is(err, errTestDBFailure) {
			t.Errorf("Wrong match. expect: db failure, got: %v, %v", res, err)
		}
	})
}
