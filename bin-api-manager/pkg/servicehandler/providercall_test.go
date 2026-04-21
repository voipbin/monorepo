package servicehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	cmcall "monorepo/bin-call-manager/models/call"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	rmprovider "monorepo/bin-route-manager/models/provider"
	rmprovidercall "monorepo/bin-route-manager/models/providercall"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

var adminAgent = auth.NewAgentIdentity(&amagent.Agent{
	Identity: commonidentity.Identity{
		ID:         uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
		CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
	},
	Permission: amagent.PermissionProjectSuperAdmin,
})

func Test_ProviderCallGet(t *testing.T) {
	id := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	resp := &rmprovidercall.ProviderCall{ID: id}
	expect := &rmprovidercall.WebhookMessage{ID: id}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
	ctx := context.Background()

	mockReq.EXPECT().RouteV1ProviderCallGet(ctx, id).Return(resp, nil)

	res, err := h.ProviderCallGet(ctx, adminAgent, id)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("wrong match.\nexpect: %v\ngot:    %v", expect, res)
	}
}

func Test_ProviderCallGet_NonAdmin_Denied(t *testing.T) {
	nonAdmin := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			CustomerID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
	ctx := context.Background()

	// No mock expectations — permission check short-circuits before any RPC.
	_, err := h.ProviderCallGet(ctx, nonAdmin, uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"))
	if err == nil {
		t.Error("expected permission error, got nil")
	}
}

func Test_ProviderCallGets(t *testing.T) {
	responses := []rmprovidercall.ProviderCall{
		{ID: uuid.FromStringOrNil("aaaaaaaa-1111-1111-1111-111111111111")},
		{ID: uuid.FromStringOrNil("bbbbbbbb-2222-2222-2222-222222222222")},
	}
	expect := []*rmprovidercall.WebhookMessage{
		{ID: uuid.FromStringOrNil("aaaaaaaa-1111-1111-1111-111111111111")},
		{ID: uuid.FromStringOrNil("bbbbbbbb-2222-2222-2222-222222222222")},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   mockDB,
		utilHandler: nil,
	}
	ctx := context.Background()

	// ProviderCallGets pulls a default token via utilHandler.TimeGetCurTime
	// when token is empty — pass a non-empty token to avoid that path in
	// this unit test.
	token := "2026-04-21T00:00:00.000000Z"

	// Filter must be empty (cross-customer by design) unless provider_id is given.
	expectedFilter := map[rmprovidercall.Field]any{}
	mockReq.EXPECT().RouteV1ProviderCallGets(ctx, token, uint64(10), expectedFilter).Return(responses, nil)

	res, err := h.ProviderCallGets(ctx, adminAgent, 10, token, uuid.Nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("wrong match.\nexpect: %v\ngot:    %v", expect, res)
	}
}

func Test_ProviderCallGets_WithProviderFilter(t *testing.T) {
	providerID := uuid.FromStringOrNil("cccccccc-3333-3333-3333-333333333333")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
	ctx := context.Background()

	token := "2026-04-21T00:00:00.000000Z"
	expectedFilter := map[rmprovidercall.Field]any{
		rmprovidercall.FieldProviderID: providerID,
	}
	mockReq.EXPECT().RouteV1ProviderCallGets(ctx, token, uint64(5), expectedFilter).Return([]rmprovidercall.ProviderCall{}, nil)

	res, err := h.ProviderCallGets(ctx, adminAgent, 5, token, providerID)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected empty slice, got %v", res)
	}
}

func Test_ProviderCallDelete(t *testing.T) {
	id := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")
	resp := &rmprovidercall.ProviderCall{ID: id}
	expect := &rmprovidercall.WebhookMessage{ID: id}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
	ctx := context.Background()

	mockReq.EXPECT().RouteV1ProviderCallDelete(ctx, id).Return(resp, nil)

	res, err := h.ProviderCallDelete(ctx, adminAgent, id)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("wrong match.\nexpect: %v\ngot:    %v", expect, res)
	}
}

func Test_ProviderCallCreate_ProviderNotFound(t *testing.T) {
	providerID := uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
	ctx := context.Background()

	mockReq.EXPECT().RouteV1ProviderGet(ctx, providerID).Return(nil, fmt.Errorf("not found"))
	// No CallV1CallsCreate / RouteV1ProviderCallCreate expected — the
	// provider-existence pre-check must fail fast before any side effect.

	_, err := h.ProviderCallCreate(
		ctx,
		adminAgent,
		providerID,
		uuid.Nil,
		nil,
		&commonaddress.Address{Type: commonaddress.TypeTel, Target: "+14155551234"},
		[]commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+821012345678"}},
		"auto",
	)
	if err == nil {
		t.Error("expected error when provider is not found, got nil")
	}
}

func Test_ProviderCallCreate_NonAdmin_Denied(t *testing.T) {
	nonAdmin := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			CustomerID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
	ctx := context.Background()

	_, err := h.ProviderCallCreate(
		ctx,
		nonAdmin,
		uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
		uuid.Nil,
		nil,
		nil,
		[]commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+821012345678"}},
		"auto",
	)
	if err == nil {
		t.Error("expected permission error for non-admin, got nil")
	}
}

func Test_ProviderCallCreate_HappyPath(t *testing.T) {
	providerID := uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")
	flowID := uuid.Nil
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+14155551234"}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+821012345678"}}
	anonymous := "auto"

	createdCallID := uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777")
	createdProviderCallID := uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
	ctx := context.Background()

	// 1. Provider existence check passes.
	mockReq.EXPECT().
		RouteV1ProviderGet(ctx, providerID).
		Return(&rmprovider.Provider{ID: providerID}, nil)

	// 2. Call is created. Server-side metadata must include both keys.
	mockReq.EXPECT().
		CallV1CallsCreate(
			ctx,
			adminAgent.CustomerID,
			flowID,
			uuid.Nil, // master_call_id
			source,
			destinations,
			false, // early_execution
			false, // connect
			anonymous,
			gomock.AssignableToTypeOf(map[string]interface{}{}),
		).
		DoAndReturn(func(_ context.Context, _, _, _ uuid.UUID, _ *commonaddress.Address, _ []commonaddress.Address, _, _ bool, _ string, md map[string]interface{}) ([]*cmcall.Call, []interface{}, error) {
			// Assert the metadata shape: route_provider_ids and skip_source_validation.
			if md[string(cmcall.MetadataKeyRouteProviderIDs)] == nil {
				t.Errorf("expected metadata to contain route_provider_ids, got %v", md)
			}
			if skip, ok := md[string(cmcall.MetadataKeySkipSourceValidation)].(bool); !ok || !skip {
				t.Errorf("expected metadata.skip_source_validation=true, got %v", md)
			}
			return []*cmcall.Call{{Identity: commonidentity.Identity{ID: createdCallID}}}, nil, nil
		})

	// 3. ProviderCall record is persisted with the created call IDs.
	mockReq.EXPECT().
		RouteV1ProviderCallCreate(
			ctx,
			adminAgent.CustomerID,
			providerID,
			flowID,
			source,
			destinations,
			anonymous,
			[]uuid.UUID{createdCallID},
			gomock.Any(),
		).
		Return(&rmprovidercall.ProviderCall{
			ID:         createdProviderCallID,
			CustomerID: adminAgent.CustomerID,
			ProviderID: providerID,
			CallIDs:    []uuid.UUID{createdCallID},
		}, nil)

	res, err := h.ProviderCallCreate(ctx, adminAgent, providerID, flowID, nil, source, destinations, anonymous)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.ID != createdProviderCallID {
		t.Errorf("expected ID %s, got %s", createdProviderCallID, res.ID)
	}
	if len(res.CallIDs) != 1 || res.CallIDs[0] != createdCallID {
		t.Errorf("expected call_ids=[%s], got %v", createdCallID, res.CallIDs)
	}
}

func Test_ProviderCallCreate_PartialFailure_ProviderCallPersistErrors(t *testing.T) {
	// Documents the explicit trade-off: when CallV1CallsCreate succeeds but
	// RouteV1ProviderCallCreate fails, the created Call records are not
	// rolled back. The handler returns an error; the admin can still retrieve
	// the orphaned Calls via GET /v1/calls.
	providerID := uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+14155551234"}
	destinations := []commonaddress.Address{{Type: commonaddress.TypeTel, Target: "+821012345678"}}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &serviceHandler{reqHandler: mockReq, dbHandler: mockDB}
	ctx := context.Background()

	createdCallID := uuid.FromStringOrNil("77777777-7777-7777-7777-777777777777")

	mockReq.EXPECT().RouteV1ProviderGet(ctx, providerID).Return(&rmprovider.Provider{ID: providerID}, nil)
	mockReq.EXPECT().
		CallV1CallsCreate(ctx, adminAgent.CustomerID, uuid.Nil, uuid.Nil, source, destinations, false, false, "auto", gomock.Any()).
		Return([]*cmcall.Call{{Identity: commonidentity.Identity{ID: createdCallID}}}, nil, nil)
	mockReq.EXPECT().
		RouteV1ProviderCallCreate(ctx, adminAgent.CustomerID, providerID, uuid.Nil, source, destinations, "auto", []uuid.UUID{createdCallID}, gomock.Any()).
		Return(nil, fmt.Errorf("route-manager unavailable"))

	_, err := h.ProviderCallCreate(ctx, adminAgent, providerID, uuid.Nil, nil, source, destinations, "auto")
	if err == nil {
		t.Fatal("expected error when ProviderCall persistence fails")
	}
}
