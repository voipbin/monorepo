package providercallhandler

import (
	"context"
	"reflect"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/providercall"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

// stubFlow builds a minimal *fmflow.Flow for use as a FlowV1FlowCreate return value.
func stubFlow(id uuid.UUID) *fmflow.Flow {
	return &fmflow.Flow{
		Identity: commonidentity.Identity{ID: id},
	}
}

// errTestFailure is a sentinel error for simulating downstream RPC failures in tests.
var errTestFailure = errorsNew("downstream rpc failure")

// errorsNew is a tiny local helper so we don't pull stdlib errors into imports.
func errorsNew(msg string) error { return &stringError{msg: msg} }

type stringError struct{ msg string }

func (e *stringError) Error() string { return e.msg }

func Test_Get(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseProviderCall *providercall.ProviderCall
	}{
		{
			"normal",
			uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			&providercall.ProviderCall{
				ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &providerCallHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ProviderCallGet(ctx, tt.id).Return(tt.responseProviderCall, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.responseProviderCall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseProviderCall, res)
			}
		})
	}
}

func Test_Create_NoActions_HappyPath(t *testing.T) {
	// No inline actions + explicit flow_id — handler skips temp-flow creation
	// and goes straight to CallV1CallsCreate, then persists the ProviderCall.
	customerID := uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001")
	providerID := uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000001")
	flowID := uuid.FromStringOrNil("e0000000-0000-0000-0000-000000000001")
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+14155551234"}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+821012345678"}}
	createdCallID := uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000001")
	persistedPC := &providercall.ProviderCall{
		ID:         uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000001"),
		CustomerID: customerID,
		ProviderID: providerID,
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &providerCallHandler{db: mockDB, reqHandler: mockReq, notifyHandler: mockNotify}
	ctx := context.Background()

	// No FlowV1FlowCreate — caller provided a flow_id.
	mockReq.EXPECT().
		CallV1CallsCreate(ctx, customerID, flowID, uuid.Nil, source, destinations, false, false, "auto", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _, _ uuid.UUID, _ *commonaddress.Address, _ []commonaddress.Address, _, _ bool, _ string, md map[string]any, _ map[string]string) ([]*cmcall.Call, []*cmgroupcall.Groupcall, error) {
			// Verify server-side metadata — both keys must be set by this handler.
			if md[string(cmcall.MetadataKeyRouteProviderIDs)] == nil {
				t.Errorf("expected metadata.route_provider_ids to be set, got %v", md)
			}
			if skip, ok := md[string(cmcall.MetadataKeySkipSourceValidation)].(bool); !ok || !skip {
				t.Errorf("expected metadata.skip_source_validation=true, got %v", md)
			}
			return []*cmcall.Call{{Identity: commonidentity.Identity{ID: createdCallID}}}, nil, nil
		})
	mockDB.EXPECT().ProviderCallCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ProviderCallGet(ctx, gomock.Any()).Return(persistedPC, nil)
	mockNotify.EXPECT().PublishEvent(ctx, providercall.EventTypeProviderCallCreated, persistedPC).Return()

	res, err := h.Create(ctx, customerID, providerID, flowID, nil, source, destinations, "auto")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !reflect.DeepEqual(res, persistedPC) {
		t.Errorf("wrong match.\nexpect: %v\ngot: %v", persistedPC, res)
	}
}

func Test_Create_TempFlowCleanup_OnCallsCreateFailure(t *testing.T) {
	// Exercises the defer+returnErr cleanup path: FlowV1FlowCreate succeeds,
	// then CallV1CallsCreate fails, and the deferred cleanup must fire
	// FlowV1FlowDelete to avoid leaking the orphaned "tmp" flow.
	customerID := uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000003")
	providerID := uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000003")
	actions := []fmaction.Action{{Type: fmaction.TypeHangup}}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+821012345678"}}
	tempFlowID := uuid.FromStringOrNil("f0000000-0000-0000-0000-000000000002")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &providerCallHandler{db: mockDB, reqHandler: mockReq, notifyHandler: mockNotify}
	ctx := context.Background()

	// Temp flow is created.
	mockReq.EXPECT().
		FlowV1FlowCreate(ctx, customerID, gomock.Any(), "tmp", gomock.Any(), actions, uuid.Nil, false).
		Return(stubFlow(tempFlowID), nil)
	// Call creation fails downstream.
	mockReq.EXPECT().
		CallV1CallsCreate(ctx, customerID, tempFlowID, uuid.Nil, nil, destinations, false, false, "auto", gomock.Any(), gomock.Any()).
		Return(nil, nil, errTestFailure)
	// Cleanup of the orphaned temp flow MUST fire.
	mockReq.EXPECT().FlowV1FlowDelete(ctx, tempFlowID).Return(stubFlow(tempFlowID), nil)

	if _, err := h.Create(ctx, customerID, providerID, uuid.Nil, actions, nil, destinations, "auto"); err == nil {
		t.Fatal("expected error when CallV1CallsCreate fails")
	}
}

func Test_Create_WithActions_CreatesTempFlow(t *testing.T) {
	// Inline actions without a flow_id — handler must create a temp flow
	// via FlowV1FlowCreate and pass that flow's ID to CallV1CallsCreate.
	customerID := uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000002")
	providerID := uuid.FromStringOrNil("b0000000-0000-0000-0000-000000000002")
	actions := []fmaction.Action{{Type: fmaction.TypeHangup}}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+821012345678"}}
	tempFlowID := uuid.FromStringOrNil("f0000000-0000-0000-0000-000000000001")
	createdCallID := uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000002")
	persistedPC := &providercall.ProviderCall{ID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000002")}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &providerCallHandler{db: mockDB, reqHandler: mockReq, notifyHandler: mockNotify}
	ctx := context.Background()

	// Temp flow is created. fmflow.TypeFlow is "flow".
	mockReq.EXPECT().
		FlowV1FlowCreate(ctx, customerID, gomock.Any(), "tmp", gomock.Any(), actions, uuid.Nil, false).
		Return(stubFlow(tempFlowID), nil)
	// Call-create uses the temp flow's ID.
	mockReq.EXPECT().
		CallV1CallsCreate(ctx, customerID, tempFlowID, uuid.Nil, nil, destinations, false, false, "auto", gomock.Any(), gomock.Any()).
		Return([]*cmcall.Call{{Identity: commonidentity.Identity{ID: createdCallID}}}, nil, nil)
	mockDB.EXPECT().ProviderCallCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ProviderCallGet(ctx, gomock.Any()).Return(persistedPC, nil)
	mockNotify.EXPECT().PublishEvent(ctx, providercall.EventTypeProviderCallCreated, persistedPC).Return()

	if _, err := h.Create(ctx, customerID, providerID, uuid.Nil, actions, nil, destinations, "auto"); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func Test_List(t *testing.T) {
	tests := []struct {
		name string

		token     string
		limit     uint64
		filters   map[providercall.Field]any
		responses []*providercall.ProviderCall
	}{
		{
			"normal",
			"",
			10,
			map[providercall.Field]any{providercall.FieldCustomerID: uuid.FromStringOrNil("a0000000-0000-0000-0000-000000000001")},
			[]*providercall.ProviderCall{{ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")}},
		},
		{
			"empty result",
			"",
			10,
			map[providercall.Field]any{},
			[]*providercall.ProviderCall{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &providerCallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ProviderCallList(ctx, tt.token, tt.limit, tt.filters).Return(tt.responses, nil)

			res, err := h.List(ctx, tt.token, tt.limit, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.responses) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responses, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		responseDeleted *providercall.ProviderCall
	}{
		{
			"normal",
			uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			&providercall.ProviderCall{
				ID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &providerCallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ProviderCallDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().ProviderCallGet(ctx, tt.id).Return(tt.responseDeleted, nil)
			mockNotify.EXPECT().PublishEvent(ctx, providercall.EventTypeProviderCallDeleted, tt.responseDeleted).Return()

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.responseDeleted) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseDeleted, res)
			}
		})
	}
}
