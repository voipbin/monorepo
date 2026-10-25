package providercallhandler

import (
	"context"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/providercall"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

func Test_NewProviderCallHandler_WiresDependencies(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	res := NewProviderCallHandler(mockDB, mockReq, mockNotify)

	h, ok := res.(*providerCallHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *providerCallHandler, got: %T", res)
	}
	if h.db != mockDB || h.reqHandler != mockReq || h.notifyHandler != mockNotify {
		t.Errorf("Wrong match. expect: the given dependencies to be wired, got: %+v", h)
	}
}

func Test_Create_RecordsOnlyNonNilCallsAndGroupcalls(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, mockReq := newErrorTestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h.notifyHandler = mockNotify
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("a0000000-0000-0000-0000-0000000000f1")
	providerID := uuid.FromStringOrNil("b0000000-0000-0000-0000-0000000000f1")
	flowID := uuid.FromStringOrNil("e0000000-0000-0000-0000-0000000000f1")
	callID := uuid.FromStringOrNil("c0000000-0000-0000-0000-0000000000f1")
	groupcallID := uuid.FromStringOrNil("c1000000-0000-0000-0000-0000000000f1")
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+155****0003"}}
	persisted := &providercall.ProviderCall{ID: uuid.FromStringOrNil("d0000000-0000-0000-0000-0000000000f1")}

	// nil entries between valid ones must be skipped, not dereferenced
	mockReq.EXPECT().
		CallV1CallsCreate(ctx, customerID, flowID, uuid.Nil, nil, destinations, false, false, "", gomock.Any(), gomock.Any()).
		Return(
			[]*cmcall.Call{nil, {Identity: commonidentity.Identity{ID: callID}}},
			[]*cmgroupcall.Groupcall{nil, {Identity: commonidentity.Identity{ID: groupcallID}}},
			nil,
		)
	mockDB.EXPECT().ProviderCallCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, p *providercall.ProviderCall) error {
		if len(p.CallIDs) != 1 || p.CallIDs[0] != callID {
			t.Errorf("Wrong match. expect call ids: [%s], got: %v", callID, p.CallIDs)
		}
		if len(p.GroupcallIDs) != 1 || p.GroupcallIDs[0] != groupcallID {
			t.Errorf("Wrong match. expect groupcall ids: [%s], got: %v", groupcallID, p.GroupcallIDs)
		}
		return nil
	})
	mockDB.EXPECT().ProviderCallGet(ctx, gomock.Any()).Return(persisted, nil)
	mockNotify.EXPECT().PublishEvent(ctx, providercall.EventTypeProviderCallCreated, persisted)

	res, err := h.Create(ctx, customerID, providerID, flowID, nil, nil, destinations, "")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != persisted {
		t.Errorf("Wrong match. expect: %v, got: %v", persisted, res)
	}
}

func Test_Create_TempFlowCleanupFailure_ReturnsOriginalError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, mockReq := newErrorTestHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("a0000000-0000-0000-0000-0000000000f2")
	providerID := uuid.FromStringOrNil("b0000000-0000-0000-0000-0000000000f2")
	tempFlowID := uuid.FromStringOrNil("f0000000-0000-0000-0000-0000000000f2")
	actions := []fmaction.Action{{Type: fmaction.TypeHangup}}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+155****0004"}}

	mockReq.EXPECT().
		FlowV1FlowCreate(ctx, customerID, gomock.Any(), "tmp", gomock.Any(), actions, uuid.Nil, false).
		Return(stubFlow(tempFlowID), nil)
	mockReq.EXPECT().
		CallV1CallsCreate(ctx, customerID, tempFlowID, uuid.Nil, nil, destinations, false, false, "", gomock.Any(), gomock.Any()).
		Return(nil, nil, errTestFailure)
	// the cleanup is attempted exactly once and its own failure does not replace the original error
	mockReq.EXPECT().FlowV1FlowDelete(ctx, tempFlowID).Return(nil, errorsNew("flow delete failed")).Times(1)

	res, err := h.Create(ctx, customerID, providerID, uuid.Nil, actions, nil, destinations, "")
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
	if err == nil || err.Error() != "could not create calls: downstream rpc failure" {
		t.Errorf("Wrong match. expect: could not create calls: downstream rpc failure, got: %v", err)
	}
}
