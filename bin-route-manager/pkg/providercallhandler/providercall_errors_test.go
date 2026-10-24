package providercallhandler

import (
	"context"
	stderrors "errors"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/providercall"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

func newErrorTestHandler(mc *gomock.Controller) (*providerCallHandler, *dbhandler.MockDBHandler, *requesthandler.MockRequestHandler) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	return &providerCallHandler{db: mockDB, reqHandler: mockReq, notifyHandler: mockNotify}, mockDB, mockReq
}

func Test_Get_NotFoundBecomesTypedError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _ := newErrorTestHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-1111111111a1")

	mockDB.EXPECT().ProviderCallGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Get(ctx, id)
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("Wrong match. expect: *VoipbinError, got: %T %v", err, err)
	}
	if ve.Status != cerrors.StatusNotFound || ve.Reason != "PROVIDERCALL_NOT_FOUND" || ve.Domain != string(commonoutline.ServiceNameRouteManager) {
		t.Errorf("Wrong match. got: status %v, reason %v, domain %v", ve.Status, ve.Reason, ve.Domain)
	}
	if !stderrors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("Wrong match. expect: the cause ErrNotFound to stay reachable")
	}
}

func Test_Get_OtherErrorIsNotConverted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _ := newErrorTestHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-1111111111a2")

	dbErr := stderrors.New("db unavailable")
	mockDB.EXPECT().ProviderCallGet(ctx, id).Return(nil, dbErr)

	_, err := h.Get(ctx, id)
	if err != dbErr {
		t.Errorf("Wrong match. expect: the original error, got: %v", err)
	}
	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) {
		t.Errorf("Wrong match. expect: no typed error for a non NotFound failure, got: %v", ve)
	}
}

func Test_List_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _ := newErrorTestHandler(mc)
	ctx := context.Background()

	dbErr := stderrors.New("db unavailable")
	mockDB.EXPECT().ProviderCallList(ctx, "token", uint64(5), map[providercall.Field]any{}).Return(nil, dbErr)

	res, err := h.List(ctx, "token", 5, map[providercall.Field]any{})
	if res != nil || err != dbErr {
		t.Errorf("Wrong match. expect: nil and the original error, got: %v, %v", res, err)
	}
}

func Test_Delete_Errors(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-1111-1111-1111-1111111111a3")

	t.Run("delete failure stops before the re-read and publishes nothing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, _ := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().ProviderCallDelete(ctx, id).Return(dbhandler.ErrNotFound)

		res, err := h.Delete(ctx, id)
		if res != nil || err == nil {
			t.Fatalf("Wrong match. expect: nil response and an error, got: %v, %v", res, err)
		}
		if !stderrors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect: the cause to stay reachable, got: %v", err)
		}
	})

	t.Run("re-read failure after delete publishes nothing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, _ := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().ProviderCallDelete(ctx, id).Return(nil)
		readErr := stderrors.New("db unavailable")
		mockDB.EXPECT().ProviderCallGet(ctx, id).Return(nil, readErr)

		res, err := h.Delete(ctx, id)
		if res != nil || err == nil {
			t.Fatalf("Wrong match. expect: nil response and an error, got: %v, %v", res, err)
		}
		if !stderrors.Is(err, readErr) {
			t.Errorf("Wrong match. expect: the cause to stay reachable, got: %v", err)
		}
	})
}

func Test_Create_Errors(t *testing.T) {
	customerID := uuid.FromStringOrNil("a0000000-0000-0000-0000-0000000000e1")
	providerID := uuid.FromStringOrNil("b0000000-0000-0000-0000-0000000000e1")
	flowID := uuid.FromStringOrNil("e0000000-0000-0000-0000-0000000000e1")
	tempFlowID := uuid.FromStringOrNil("f0000000-0000-0000-0000-0000000000e1")
	actions := []fmaction.Action{{Type: fmaction.TypeHangup}}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+15550000002"}}
	createdCall := []*cmcall.Call{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("c0000000-0000-0000-0000-0000000000e1")}}}

	t.Run("temp flow creation failure makes no call and needs no cleanup", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockReq := newErrorTestHandler(mc)
		ctx := context.Background()

		mockReq.EXPECT().
			FlowV1FlowCreate(ctx, customerID, gomock.Any(), "tmp", gomock.Any(), actions, uuid.Nil, false).
			Return(nil, stderrors.New("flow create failed"))

		res, err := h.Create(ctx, customerID, providerID, uuid.Nil, actions, nil, destinations, "")
		if res != nil || err == nil {
			t.Errorf("Wrong match. expect: nil response and an error, got: %v, %v", res, err)
		}
	})

	t.Run("persist failure with a caller flow does not delete the flow", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, mockReq := newErrorTestHandler(mc)
		ctx := context.Background()

		mockReq.EXPECT().
			CallV1CallsCreate(ctx, customerID, flowID, uuid.Nil, nil, destinations, false, false, "", gomock.Any(), gomock.Any()).
			Return(createdCall, nil, nil)
		mockDB.EXPECT().ProviderCallCreate(ctx, gomock.Any()).Return(stderrors.New("insert failed"))
		// no FlowV1FlowDelete expectation: the flow belongs to the caller

		res, err := h.Create(ctx, customerID, providerID, flowID, nil, nil, destinations, "")
		if res != nil || err == nil {
			t.Errorf("Wrong match. expect: nil response and an error, got: %v, %v", res, err)
		}
	})

	t.Run("persist failure with a temp flow deletes the temp flow", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, mockReq := newErrorTestHandler(mc)
		ctx := context.Background()

		mockReq.EXPECT().
			FlowV1FlowCreate(ctx, customerID, gomock.Any(), "tmp", gomock.Any(), actions, uuid.Nil, false).
			Return(stubFlow(tempFlowID), nil)
		mockReq.EXPECT().
			CallV1CallsCreate(ctx, customerID, tempFlowID, uuid.Nil, nil, destinations, false, false, "", gomock.Any(), gomock.Any()).
			Return(createdCall, nil, nil)
		mockDB.EXPECT().ProviderCallCreate(ctx, gomock.Any()).Return(stderrors.New("insert failed"))
		mockReq.EXPECT().FlowV1FlowDelete(ctx, tempFlowID).Return(stubFlow(tempFlowID), nil)

		if _, err := h.Create(ctx, customerID, providerID, uuid.Nil, actions, nil, destinations, ""); err == nil {
			t.Errorf("Wrong match. expect: an error, got: nil")
		}
	})

	t.Run("re-read failure after persist returns an error and publishes nothing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, mockReq := newErrorTestHandler(mc)
		ctx := context.Background()

		mockReq.EXPECT().
			CallV1CallsCreate(ctx, customerID, flowID, uuid.Nil, nil, destinations, false, false, "", gomock.Any(), gomock.Any()).
			Return(createdCall, nil, nil)
		mockDB.EXPECT().ProviderCallCreate(ctx, gomock.Any()).Return(nil)
		mockDB.EXPECT().ProviderCallGet(ctx, gomock.Any()).Return(nil, stderrors.New("read failed"))

		res, err := h.Create(ctx, customerID, providerID, flowID, nil, nil, destinations, "")
		if res != nil || err == nil {
			t.Errorf("Wrong match. expect: nil response and an error, got: %v, %v", res, err)
		}
	})
}
