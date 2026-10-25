package numberhandler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	bmaccount "monorepo/bin-billing-manager/models/account"
	bmbilling "monorepo/bin-billing-manager/models/billing"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/providernumber"
	"monorepo/bin-number-manager/pkg/dbhandler"
	"monorepo/bin-number-manager/pkg/numberhandlertelnyx"
	"monorepo/bin-number-manager/pkg/providerhandler"
)

type errTestMocks struct {
	util     *utilhandler.MockUtilHandler
	req      *requesthandler.MockRequestHandler
	db       *dbhandler.MockDBHandler
	notify   *notifyhandler.MockNotifyHandler
	telnyx   *numberhandlertelnyx.MockNumberHandlerTelnyx
	provider *providerhandler.MockProviderHandler
}

func newErrTestHandler(t *testing.T) (*numberHandler, *errTestMocks) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &errTestMocks{
		util:     utilhandler.NewMockUtilHandler(mc),
		req:      requesthandler.NewMockRequestHandler(mc),
		db:       dbhandler.NewMockDBHandler(mc),
		notify:   notifyhandler.NewMockNotifyHandler(mc),
		telnyx:   numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc),
		provider: providerhandler.NewMockProviderHandler(mc),
	}

	h := &numberHandler{
		utilHandler:     m.util,
		reqHandler:      m.req,
		db:              m.db,
		notifyHandler:   m.notify,
		providerHandler: m.provider,
	}

	return h, m
}

var (
	errTestCustomerID = uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000001")
	errTestNumberID   = uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000002")
	errTestNumber     = "+15550100009"
	errTestVirtual    = "+899123456789"
	errTestDBFailure  = fmt.Errorf("db failure")
)

func Test_Create_errors(t *testing.T) {
	ctx := context.Background()

	t.Run("reserved prefix is rejected before any external call", func(t *testing.T) {
		h, _ := newErrTestHandler(t)

		res, err := h.Create(ctx, errTestCustomerID, "+899000123456", uuid.Nil, uuid.Nil, "", "")
		if err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
	})

	t.Run("balance check failure stops the purchase", func(t *testing.T) {
		h, m := newErrTestHandler(t)

		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(false, fmt.Errorf("billing down"))

		if _, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", ""); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("insufficient balance stops the purchase", func(t *testing.T) {
		h, m := newErrTestHandler(t)

		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(false, nil)

		if _, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", ""); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("purchase failure is returned and nothing is registered", func(t *testing.T) {
		h, m := newErrTestHandler(t)

		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)
		m.db.EXPECT().NumberList(ctx, uint64(1), "", map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: errTestNumber}).Return([]*number.Number{}, nil)
		m.provider.EXPECT().PurchaseCandidates(ctx).Return([]*providerhandler.Candidate{{Name: number.ProviderNameTelnyx, Provider: m.telnyx}}, nil)
		m.telnyx.EXPECT().NumberPurchase(errTestNumber).Return(nil, fmt.Errorf("provider failure"))

		if _, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", ""); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("rollback failure still returns the register error", func(t *testing.T) {
		h, m := newErrTestHandler(t)

		purchased := &providernumber.ProviderNumber{ID: "provider-id-1", Status: number.StatusActive}
		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)
		m.db.EXPECT().NumberList(ctx, uint64(1), "", map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: errTestNumber}).Return([]*number.Number{}, nil)
		m.provider.EXPECT().PurchaseCandidates(ctx).Return([]*providerhandler.Candidate{{Name: number.ProviderNameTelnyx, Provider: m.telnyx}}, nil)
		m.telnyx.EXPECT().NumberPurchase(errTestNumber).Return(purchased, nil)
		m.db.EXPECT().NumberList(ctx, uint64(1), "", map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: errTestNumber}).Return(nil, errTestDBFailure)
		m.telnyx.EXPECT().NumberRelease(ctx, &number.Number{
			ProviderName:        number.ProviderNameTelnyx,
			ProviderReferenceID: "provider-id-1",
		}).Return(fmt.Errorf("release failure"))

		res, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", "")
		if err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
		if !errors.Is(err, errTestDBFailure) {
			t.Errorf("Wrong match. expect the register error to be returned, got: %v", err)
		}
	})

	t.Run("tag update failure does not fail the creation", func(t *testing.T) {
		h, m := newErrTestHandler(t)

		purchased := &providernumber.ProviderNumber{ID: "provider-id-2", Status: number.StatusActive}
		created := &number.Number{
			Identity: commonidentity.Identity{ID: errTestNumberID, CustomerID: errTestCustomerID},
			Number:   errTestNumber,
		}

		m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "", 1).Return(true, nil)
		m.db.EXPECT().NumberList(ctx, uint64(1), "", map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: errTestNumber}).Return([]*number.Number{}, nil)
		m.provider.EXPECT().PurchaseCandidates(ctx).Return([]*providerhandler.Candidate{{Name: number.ProviderNameTelnyx, Provider: m.telnyx}}, nil)
		m.telnyx.EXPECT().NumberPurchase(errTestNumber).Return(purchased, nil)
		m.db.EXPECT().NumberList(ctx, uint64(1), "", gomock.Any()).Return([]*number.Number{}, nil)
		m.util.EXPECT().UUIDCreate().Return(errTestNumberID)
		m.db.EXPECT().NumberCreate(ctx, gomock.Any()).Return(nil)
		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(created, nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, errTestCustomerID, number.EventTypeNumberCreated, created)
		m.telnyx.EXPECT().NumberUpdateTags(ctx, created, []string{
			"CustomerID_a0000000-0000-4000-8000-000000000001",
			"NumberID_a0000000-0000-4000-8000-000000000002",
		}).Return(fmt.Errorf("tag failure"))

		res, err := h.Create(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", "")
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
		if !reflect.DeepEqual(created, res) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", created, res)
		}
	})
}

func Test_CreateVirtual_register_failure(t *testing.T) {
	h, m := newErrTestHandler(t)
	ctx := context.Background()

	m.req.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, errTestCustomerID, bmaccount.ResourceTypeVirtualNumber).Return(true, nil)
	m.db.EXPECT().NumberList(ctx, uint64(1), "", map[number.Field]any{number.FieldDeleted: false, number.FieldNumber: errTestVirtual}).Return(nil, errTestDBFailure)

	res, err := h.CreateVirtual(ctx, errTestCustomerID, errTestVirtual, uuid.Nil, uuid.Nil, "", "", false)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if !errors.Is(err, errTestDBFailure) {
		t.Errorf("Wrong match. expect the db error to be wrapped, got: %v", err)
	}
}

func Test_Register_dbCreate_failure(t *testing.T) {
	h, m := newErrTestHandler(t)
	ctx := context.Background()

	m.db.EXPECT().NumberList(ctx, uint64(1), "", gomock.Any()).Return([]*number.Number{}, nil)
	m.util.EXPECT().UUIDCreate().Return(errTestNumberID)
	m.db.EXPECT().NumberCreate(ctx, gomock.Any()).Return(errTestDBFailure)

	res, err := h.Register(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", "", number.TypeNormal, number.ProviderNameTelnyx, "ref", number.StatusActive, false, false)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
	if !errors.Is(err, errTestDBFailure) {
		t.Errorf("Wrong match. expect the db error to be wrapped, got: %v", err)
	}
}

func Test_dbCreate_get_failure(t *testing.T) {
	h, m := newErrTestHandler(t)
	ctx := context.Background()

	m.util.EXPECT().UUIDCreate().Return(errTestNumberID)
	m.db.EXPECT().NumberCreate(ctx, gomock.Any()).Return(nil)
	m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(nil, errTestDBFailure)

	// the webhook event must not be published when the created number cannot be read back
	res, err := h.dbCreate(ctx, errTestCustomerID, errTestNumber, uuid.Nil, uuid.Nil, "", "", number.TypeNormal, number.ProviderNameTelnyx, "ref", number.StatusActive, false, false)
	if !errors.Is(err, errTestDBFailure) {
		t.Errorf("Wrong match. expect: %v, got: %v", errTestDBFailure, err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_Delete_errors(t *testing.T) {
	ctx := context.Background()

	telnyxNumber := &number.Number{
		Identity:     commonidentity.Identity{ID: errTestNumberID, CustomerID: errTestCustomerID},
		Number:       errTestNumber,
		ProviderName: number.ProviderNameTelnyx,
	}
	twilioNumber := &number.Number{
		Identity:     commonidentity.Identity{ID: errTestNumberID, CustomerID: errTestCustomerID},
		Number:       errTestNumber,
		ProviderName: number.ProviderNameTwilio,
	}
	otherNumber := &number.Number{
		Identity:     commonidentity.Identity{ID: errTestNumberID, CustomerID: errTestCustomerID},
		Number:       errTestNumber,
		ProviderName: number.ProviderNameMessagebird,
	}

	t.Run("get failure", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(nil, errTestDBFailure)

		res, err := h.Delete(ctx, errTestNumberID)
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil, got: %v, %v", res, err)
		}
	})

	t.Run("telnyx release failure keeps the record", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(telnyxNumber, nil)
		m.provider.EXPECT().ForRelease(ctx, number.ProviderNameTelnyx).Return(m.telnyx, nil)
		m.telnyx.EXPECT().NumberRelease(ctx, telnyxNumber).Return(fmt.Errorf("release failure"))

		res, err := h.Delete(ctx, errTestNumberID)
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil, got: %v, %v", res, err)
		}
	})

	t.Run("twilio release failure keeps the record", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(twilioNumber, nil)
		m.provider.EXPECT().ForRelease(ctx, number.ProviderNameTwilio).Return(nil, fmt.Errorf("release failure"))

		res, err := h.Delete(ctx, errTestNumberID)
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil, got: %v, %v", res, err)
		}
	})

	t.Run("twilio release success deletes the record", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(twilioNumber, nil).Times(2)
		m.provider.EXPECT().ForRelease(ctx, number.ProviderNameTwilio).Return(m.telnyx, nil)
		m.telnyx.EXPECT().NumberRelease(ctx, twilioNumber).Return(nil)
		m.db.EXPECT().NumberDelete(ctx, errTestNumberID).Return(nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, errTestCustomerID, number.EventTypeNumberDeleted, twilioNumber)

		res, err := h.Delete(ctx, errTestNumberID)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
		if !reflect.DeepEqual(twilioNumber, res) {
			t.Errorf("Wrong match.\nexpect: %v\ngot: %v", twilioNumber, res)
		}
	})

	t.Run("unsupported provider", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(otherNumber, nil)
		m.provider.EXPECT().ForRelease(ctx, otherNumber.ProviderName).Return(nil, fmt.Errorf("unknown provider: %w", dbhandler.ErrNotFound))

		res, err := h.Delete(ctx, errTestNumberID)
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil, got: %v, %v", res, err)
		}
	})

	t.Run("db delete failure", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(telnyxNumber, nil)
		m.provider.EXPECT().ForRelease(ctx, number.ProviderNameTelnyx).Return(m.telnyx, nil)
		m.telnyx.EXPECT().NumberRelease(ctx, telnyxNumber).Return(nil)
		m.db.EXPECT().NumberDelete(ctx, errTestNumberID).Return(errTestDBFailure)

		res, err := h.Delete(ctx, errTestNumberID)
		if !errors.Is(err, errTestDBFailure) || res != nil {
			t.Errorf("Wrong match. expect: db failure and nil, got: %v, %v", res, err)
		}
	})

	t.Run("read back failure after the delete", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		gomock.InOrder(
			m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(telnyxNumber, nil),
			m.provider.EXPECT().ForRelease(ctx, number.ProviderNameTelnyx).Return(m.telnyx, nil),
			m.telnyx.EXPECT().NumberRelease(ctx, telnyxNumber).Return(nil),
			m.db.EXPECT().NumberDelete(ctx, errTestNumberID).Return(nil),
			m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(nil, errTestDBFailure),
		)

		res, err := h.Delete(ctx, errTestNumberID)
		if !errors.Is(err, errTestDBFailure) || res != nil {
			t.Errorf("Wrong match. expect: db failure and nil, got: %v, %v", res, err)
		}
	})
}

func Test_Get_errors(t *testing.T) {
	ctx := context.Background()

	t.Run("not found is converted to a typed error", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(nil, dbhandler.ErrNotFound)

		res, err := h.Get(ctx, errTestNumberID)
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}

		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) {
			t.Fatalf("Wrong match. expect: VoipbinError, got: %v", err)
		}
		if ve.Status != cerrors.StatusNotFound || ve.Reason != "NUMBER_NOT_FOUND" {
			t.Errorf("Wrong match. expect: NOT_FOUND/NUMBER_NOT_FOUND, got: %s/%s", ve.Status, ve.Reason)
		}
		if !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect the cause to be kept, got: %v", err)
		}
	})

	t.Run("other errors are wrapped without a typed error", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(nil, errTestDBFailure)

		res, err := h.Get(ctx, errTestNumberID)
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}

		var ve *cerrors.VoipbinError
		if errors.As(err, &ve) {
			t.Errorf("Wrong match. expect: untyped error, got: %v", err)
		}
		if !errors.Is(err, errTestDBFailure) {
			t.Errorf("Wrong match. expect the cause to be kept, got: %v", err)
		}
	})
}

func Test_List_Count_Update_errors(t *testing.T) {
	ctx := context.Background()

	t.Run("list", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		filters := map[number.Field]any{number.FieldDeleted: false}
		m.db.EXPECT().NumberList(ctx, uint64(5), "token", filters).Return(nil, errTestDBFailure)

		res, err := h.List(ctx, 5, "token", filters)
		if !errors.Is(err, errTestDBFailure) || res != nil {
			t.Errorf("Wrong match. expect: db failure and nil, got: %v, %v", res, err)
		}
	})

	t.Run("count virtual", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.db.EXPECT().NumberCountVirtualByCustomerID(ctx, errTestCustomerID).Return(0, errTestDBFailure)

		res, err := h.CountVirtualByCustomerID(ctx, errTestCustomerID)
		if !errors.Is(err, errTestDBFailure) || res != 0 {
			t.Errorf("Wrong match. expect: db failure and 0, got: %v, %v", res, err)
		}
	})

	t.Run("update", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		fields := map[number.Field]any{number.FieldName: "new name"}
		m.db.EXPECT().NumberUpdate(ctx, errTestNumberID, fields).Return(errTestDBFailure)

		res, err := h.Update(ctx, errTestNumberID, fields)
		if !errors.Is(err, errTestDBFailure) || res != nil {
			t.Errorf("Wrong match. expect: db failure and nil, got: %v, %v", res, err)
		}
	})

	t.Run("update read back failure", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		fields := map[number.Field]any{number.FieldName: "new name"}
		m.db.EXPECT().NumberUpdate(ctx, errTestNumberID, fields).Return(nil)
		m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(nil, errTestDBFailure)

		res, err := h.Update(ctx, errTestNumberID, fields)
		if !errors.Is(err, errTestDBFailure) || res != nil {
			t.Errorf("Wrong match. expect: db failure and nil, got: %v, %v", res, err)
		}
	})
}

func Test_dbListByTMRenew_failure(t *testing.T) {
	h, m := newErrTestHandler(t)
	ctx := context.Background()

	m.db.EXPECT().NumberGetsByTMRenew(ctx, "2024-01-01 00:00:00.000000", uint64(100), map[number.Field]any{number.FieldDeleted: false}, gomock.Any()).Return(nil, errTestDBFailure)

	res, err := h.dbListByTMRenew(ctx, "2024-01-01 00:00:00.000000", nil)
	if !errors.Is(err, errTestDBFailure) || res != nil {
		t.Errorf("Wrong match. expect: db failure and nil, got: %v, %v", res, err)
	}
}

func Test_EventCustomerDeleted_errors(t *testing.T) {
	ctx := context.Background()
	customer := &cmcustomer.Customer{ID: errTestCustomerID}
	filters := map[number.Field]any{
		number.FieldCustomerID: errTestCustomerID,
		number.FieldDeleted:    false,
	}

	t.Run("list failure is returned", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.db.EXPECT().NumberList(ctx, uint64(10000), gomock.Any(), filters).Return(nil, errTestDBFailure)

		if err := h.EventCustomerDeleted(ctx, customer); !errors.Is(err, errTestDBFailure) {
			t.Errorf("Wrong match. expect: db failure, got: %v", err)
		}
	})

	t.Run("a failed delete does not stop the remaining numbers", func(t *testing.T) {
		h, m := newErrTestHandler(t)

		firstID := uuid.FromStringOrNil("a0000000-0000-4000-8000-0000000000f1")
		secondID := uuid.FromStringOrNil("a0000000-0000-4000-8000-0000000000f2")
		first := &number.Number{Identity: commonidentity.Identity{ID: firstID, CustomerID: errTestCustomerID}, ProviderName: number.ProviderNameTelnyx}
		second := &number.Number{Identity: commonidentity.Identity{ID: secondID, CustomerID: errTestCustomerID}, ProviderName: number.ProviderNameTelnyx}

		m.db.EXPECT().NumberList(ctx, uint64(10000), gomock.Any(), filters).Return([]*number.Number{first, second}, nil)

		// the first number cannot be read, so its delete fails
		m.db.EXPECT().NumberGet(ctx, firstID).Return(nil, errTestDBFailure)

		// the second number is deleted completely
		gomock.InOrder(
			m.db.EXPECT().NumberGet(ctx, secondID).Return(second, nil),
			m.provider.EXPECT().ForRelease(ctx, number.ProviderNameTelnyx).Return(m.telnyx, nil),
			m.telnyx.EXPECT().NumberRelease(ctx, second).Return(nil),
			m.db.EXPECT().NumberDelete(ctx, secondID).Return(nil),
			m.db.EXPECT().NumberGet(ctx, secondID).Return(second, nil),
			m.notify.EXPECT().PublishWebhookEvent(ctx, errTestCustomerID, number.EventTypeNumberDeleted, second),
		)

		if err := h.EventCustomerDeleted(ctx, customer); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}

func newFlowDeletedNumbers(count int, flowField string, flowID uuid.UUID) []*number.Number {
	res := make([]*number.Number, 0, count)
	for i := 0; i < count; i++ {
		n := &number.Number{
			Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		}
		switch flowField {
		case "call":
			n.CallFlowID = flowID
		default:
			n.MessageFlowID = flowID
		}
		res = append(res, n)
	}
	return res
}

func Test_EventFlowDeleted_errors(t *testing.T) {
	ctx := context.Background()
	flowID := uuid.FromStringOrNil("a0000000-0000-4000-8000-0000000000aa")
	flow := &fmflow.Flow{Identity: commonidentity.Identity{ID: flowID}}

	callFilters := map[number.Field]any{number.FieldCallFlowID: flowID, number.FieldDeleted: false}
	messageFilters := map[number.Field]any{number.FieldMessageFlowID: flowID, number.FieldDeleted: false}

	t.Run("list failures end each loop without updates", func(t *testing.T) {
		h, m := newErrTestHandler(t)

		m.util.EXPECT().TimeGetCurTime().Return("2024-01-01 00:00:00.000000").Times(2)
		m.db.EXPECT().NumberList(ctx, uint64(1000), gomock.Any(), callFilters).Return(nil, errTestDBFailure)
		m.db.EXPECT().NumberList(ctx, uint64(1000), gomock.Any(), messageFilters).Return(nil, errTestDBFailure)

		if err := h.EventFlowDeleted(ctx, flow); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("update failures do not stop the loops", func(t *testing.T) {
		h, m := newErrTestHandler(t)

		callNumbers := newFlowDeletedNumbers(2, "call", flowID)
		messageNumbers := newFlowDeletedNumbers(2, "message", flowID)

		m.util.EXPECT().TimeGetCurTime().Return("2024-01-01 00:00:00.000000").Times(2)
		m.db.EXPECT().NumberList(ctx, uint64(1000), gomock.Any(), callFilters).Return(callNumbers, nil)
		m.db.EXPECT().NumberList(ctx, uint64(1000), gomock.Any(), messageFilters).Return(messageNumbers, nil)

		for _, n := range callNumbers {
			m.db.EXPECT().NumberUpdate(ctx, n.ID, map[number.Field]any{number.FieldCallFlowID: uuid.Nil}).Return(errTestDBFailure)
		}
		for _, n := range messageNumbers {
			m.db.EXPECT().NumberUpdate(ctx, n.ID, map[number.Field]any{number.FieldMessageFlowID: uuid.Nil}).Return(errTestDBFailure)
		}

		if err := h.EventFlowDeleted(ctx, flow); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("a full page of 1000 fetches the next page for both loops", func(t *testing.T) {
		h, m := newErrTestHandler(t)

		callNumbers := newFlowDeletedNumbers(1000, "call", flowID)
		messageNumbers := newFlowDeletedNumbers(1000, "message", flowID)

		m.util.EXPECT().TimeGetCurTime().Return("2024-01-01 00:00:00.000000").Times(4)
		gomock.InOrder(
			m.db.EXPECT().NumberList(ctx, uint64(1000), gomock.Any(), callFilters).Return(callNumbers, nil),
			m.db.EXPECT().NumberList(ctx, uint64(1000), gomock.Any(), callFilters).Return([]*number.Number{}, nil),
			m.db.EXPECT().NumberList(ctx, uint64(1000), gomock.Any(), messageFilters).Return(messageNumbers, nil),
			m.db.EXPECT().NumberList(ctx, uint64(1000), gomock.Any(), messageFilters).Return([]*number.Number{}, nil),
		)
		m.db.EXPECT().NumberUpdate(ctx, gomock.Any(), map[number.Field]any{number.FieldCallFlowID: uuid.Nil}).Return(nil).Times(1000)
		m.db.EXPECT().NumberUpdate(ctx, gomock.Any(), map[number.Field]any{number.FieldMessageFlowID: uuid.Nil}).Return(nil).Times(1000)

		if err := h.EventFlowDeleted(ctx, flow); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("a page of 999 does not fetch another page", func(t *testing.T) {
		h, m := newErrTestHandler(t)

		callNumbers := newFlowDeletedNumbers(999, "call", flowID)
		messageNumbers := newFlowDeletedNumbers(999, "message", flowID)

		m.util.EXPECT().TimeGetCurTime().Return("2024-01-01 00:00:00.000000").Times(2)
		m.db.EXPECT().NumberList(ctx, uint64(1000), gomock.Any(), callFilters).Return(callNumbers, nil).Times(1)
		m.db.EXPECT().NumberList(ctx, uint64(1000), gomock.Any(), messageFilters).Return(messageNumbers, nil).Times(1)
		m.db.EXPECT().NumberUpdate(ctx, gomock.Any(), map[number.Field]any{number.FieldCallFlowID: uuid.Nil}).Return(nil).Times(999)
		m.db.EXPECT().NumberUpdate(ctx, gomock.Any(), map[number.Field]any{number.FieldMessageFlowID: uuid.Nil}).Return(nil).Times(999)

		if err := h.EventFlowDeleted(ctx, flow); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}

func Test_RenewNumbers_errors(t *testing.T) {
	ctx := context.Background()
	tmRenew := "2024-01-01 00:00:00.000000"
	listFilters := map[number.Field]any{number.FieldDeleted: false}

	t.Run("no renew time given", func(t *testing.T) {
		h, _ := newErrTestHandler(t)

		res, err := h.RenewNumbers(ctx, 0, 0, "")
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil, got: %v, %v", res, err)
		}
	})

	t.Run("list failure by tm_renew", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.db.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), listFilters, gomock.Any()).Return(nil, errTestDBFailure)

		res, err := h.RenewNumbers(ctx, 0, 0, tmRenew)
		if !errors.Is(err, errTestDBFailure) || res != nil {
			t.Errorf("Wrong match. expect: db failure and nil, got: %v, %v", res, err)
		}
	})

	t.Run("list failure by days", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.util.EXPECT().TimeGetCurTimeAdd(gomock.Any()).Return(tmRenew)
		m.db.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), listFilters, gomock.Any()).Return(nil, errTestDBFailure)

		res, err := h.RenewNumbers(ctx, 3, 0, "")
		if !errors.Is(err, errTestDBFailure) || res != nil {
			t.Errorf("Wrong match. expect: db failure and nil, got: %v, %v", res, err)
		}
	})

	t.Run("list failure by hours", func(t *testing.T) {
		h, m := newErrTestHandler(t)
		m.util.EXPECT().TimeGetCurTimeAdd(gomock.Any()).Return(tmRenew)
		m.db.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), listFilters, gomock.Any()).Return(nil, errTestDBFailure)

		res, err := h.RenewNumbers(ctx, 0, 5, "")
		if !errors.Is(err, errTestDBFailure) || res != nil {
			t.Errorf("Wrong match. expect: db failure and nil, got: %v, %v", res, err)
		}
	})

	t.Run("insufficient balance with a failed delete ends without renewing", func(t *testing.T) {
		h, m := newErrTestHandler(t)

		n := &number.Number{
			Identity:     commonidentity.Identity{ID: errTestNumberID, CustomerID: errTestCustomerID},
			Number:       errTestNumber,
			ProviderName: number.ProviderNameTelnyx,
		}

		gomock.InOrder(
			m.db.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), listFilters, gomock.Any()).Return([]*number.Number{n}, nil),
			m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(false, nil),
			// Delete cannot read the number
			m.db.EXPECT().NumberGet(ctx, errTestNumberID).Return(nil, errTestDBFailure),
			// the next page leaves the number that failed out, and it is empty
			m.db.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), listFilters, []uuid.UUID{errTestNumberID}).Return([]*number.Number{}, nil),
		)

		res, err := h.RenewNumbers(ctx, 0, 0, tmRenew)
		if err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
		if len(res) != 0 {
			t.Errorf("Wrong match. expect: no renewed numbers, got: %v", res)
		}
	})

	t.Run("failed deletes accumulate in the exclusion list and the loop ends", func(t *testing.T) {
		h, m := newErrTestHandler(t)

		firstID := uuid.FromStringOrNil("a0000000-0000-4000-8000-0000000000c1")
		secondID := uuid.FromStringOrNil("a0000000-0000-4000-8000-0000000000c2")
		first := &number.Number{Identity: commonidentity.Identity{ID: firstID, CustomerID: errTestCustomerID}, ProviderName: number.ProviderNameTelnyx}
		second := &number.Number{Identity: commonidentity.Identity{ID: secondID, CustomerID: errTestCustomerID}, ProviderName: number.ProviderNameTelnyx}

		gomock.InOrder(
			m.db.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), listFilters, []uuid.UUID{}).Return([]*number.Number{first}, nil),
			m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(false, nil),
			m.db.EXPECT().NumberGet(ctx, firstID).Return(nil, errTestDBFailure),
			// every number of the first page failed, yet the query goes on with the next numbers
			m.db.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), listFilters, []uuid.UUID{firstID}).Return([]*number.Number{second}, nil),
			m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(false, nil),
			m.db.EXPECT().NumberGet(ctx, secondID).Return(nil, errTestDBFailure),
			m.db.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), listFilters, []uuid.UUID{firstID, secondID}).Return([]*number.Number{}, nil),
		)

		if _, err := h.RenewNumbers(ctx, 0, 0, tmRenew); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("a page of numbers that only fail on billing ends the loop", func(t *testing.T) {
		h, m := newErrTestHandler(t)

		n := &number.Number{Identity: commonidentity.Identity{ID: errTestNumberID, CustomerID: errTestCustomerID}, ProviderName: number.ProviderNameTelnyx}

		gomock.InOrder(
			m.db.EXPECT().NumberGetsByTMRenew(ctx, tmRenew, uint64(100), listFilters, gomock.Any()).Return([]*number.Number{n}, nil),
			m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, errTestCustomerID, bmbilling.ReferenceTypeNumber, "us", 1).Return(false, errTestDBFailure),
		)

		if _, err := h.RenewNumbers(ctx, 0, 0, tmRenew); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})
}
