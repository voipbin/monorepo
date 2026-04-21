package routehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/models/route"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

func Test_DialrouteList(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		target     string

		responseRoutesCustomerTarget []*route.Route
		responseRoutesCustomerAll    []*route.Route

		responseRoutesDefaultTarget []*route.Route
		responseRoutesDefaultAll    []*route.Route

		expectRes []*route.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("ecfe86e4-e20f-4a06-890e-84b0e8ecfca4"),
			"+82",

			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("fd05574d-a43d-4d74-a9a0-301e6e875bb9"),
					ProviderID: uuid.FromStringOrNil("9e5b4d7d-ddd3-4a10-9873-3fab21f86645"),
				},
			},
			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("e1e6e844-4c84-4722-b1a7-26d43c193a26"),
					ProviderID: uuid.FromStringOrNil("f5a7c5b8-11b3-44c3-a87b-2224acecf8cf"),
				},
			},

			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("ce97f95d-5498-4ae2-a9c8-ae2a2f598a93"),
					ProviderID: uuid.FromStringOrNil("44eea3bb-46b9-456e-8dbc-1c28bc01f308"),
				},
			},
			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("0b71e856-f10e-40af-8a54-b05c2ae8bc81"),
					ProviderID: uuid.FromStringOrNil("0ed4e04a-9645-4002-b745-f11386af6305"),
				},
			},

			[]*route.Route{
				{
					ID:         uuid.FromStringOrNil("fd05574d-a43d-4d74-a9a0-301e6e875bb9"),
					ProviderID: uuid.FromStringOrNil("9e5b4d7d-ddd3-4a10-9873-3fab21f86645"),
				},
				{
					ID:         uuid.FromStringOrNil("e1e6e844-4c84-4722-b1a7-26d43c193a26"),
					ProviderID: uuid.FromStringOrNil("f5a7c5b8-11b3-44c3-a87b-2224acecf8cf"),
				},
				{
					ID:         uuid.FromStringOrNil("ce97f95d-5498-4ae2-a9c8-ae2a2f598a93"),
					ProviderID: uuid.FromStringOrNil("44eea3bb-46b9-456e-8dbc-1c28bc01f308"),
				},
				{
					ID:         uuid.FromStringOrNil("0b71e856-f10e-40af-8a54-b05c2ae8bc81"),
					ProviderID: uuid.FromStringOrNil("0ed4e04a-9645-4002-b745-f11386af6305"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &routeHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			// ListByTarget for customer route base
			// First call: filtersTarget with customerID and target
			filtersCustomerTarget := map[route.Field]any{
				route.FieldCustomerID: tt.customerID,
				route.FieldTarget:     tt.target,
			}
			mockDB.EXPECT().RouteList(ctx, "", uint64(1000), filtersCustomerTarget).Return(tt.responseRoutesCustomerTarget, nil)
			// Second call: filtersAll with customerID and TargetAll
			filtersCustomerAll := map[route.Field]any{
				route.FieldCustomerID: tt.customerID,
				route.FieldTarget:     route.TargetAll,
			}
			mockDB.EXPECT().RouteList(ctx, "", uint64(1000), filtersCustomerAll).Return(tt.responseRoutesCustomerAll, nil)

			// ListByTarget for default route base
			// First call: filtersTarget with CustomerIDBasicRoute and target
			filtersDefaultTarget := map[route.Field]any{
				route.FieldCustomerID: route.CustomerIDBasicRoute,
				route.FieldTarget:     tt.target,
			}
			mockDB.EXPECT().RouteList(ctx, "", uint64(1000), filtersDefaultTarget).Return(tt.responseRoutesDefaultTarget, nil)
			// Second call: filtersAll with CustomerIDBasicRoute and TargetAll
			filtersDefaultAll := map[route.Field]any{
				route.FieldCustomerID: route.CustomerIDBasicRoute,
				route.FieldTarget:     route.TargetAll,
			}
			mockDB.EXPECT().RouteList(ctx, "", uint64(1000), filtersDefaultAll).Return(tt.responseRoutesDefaultAll, nil)

			res, err := h.DialrouteList(ctx, tt.customerID, tt.target, nil)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_DialrouteList_CustomerRouteError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &routeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("ecfe86e4-e20f-4a06-890e-84b0e8ecfca4")

	filtersTarget := map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     "+82",
	}

	mockDB.EXPECT().RouteList(ctx, "", uint64(1000), filtersTarget).Return(nil, fmt.Errorf("database error"))

	res, err := h.DialrouteList(ctx, customerID, "+82", nil)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_DialrouteList_DefaultRouteError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &routeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("ecfe86e4-e20f-4a06-890e-84b0e8ecfca4")

	filtersCustomerTarget := map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     "+82",
	}
	filtersCustomerAll := map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     route.TargetAll,
	}
	filtersDefaultTarget := map[route.Field]any{
		route.FieldCustomerID: route.CustomerIDBasicRoute,
		route.FieldTarget:     "+82",
	}

	mockDB.EXPECT().RouteList(ctx, "", uint64(1000), filtersCustomerTarget).Return([]*route.Route{}, nil)
	mockDB.EXPECT().RouteList(ctx, "", uint64(1000), filtersCustomerAll).Return([]*route.Route{}, nil)
	mockDB.EXPECT().RouteList(ctx, "", uint64(1000), filtersDefaultTarget).Return(nil, fmt.Errorf("database error"))

	res, err := h.DialrouteList(ctx, customerID, "+82", nil)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_DialrouteList_WithTargetProviderIDs(t *testing.T) {
	tests := []struct {
		name string

		customerID        uuid.UUID
		target            string
		targetProviderIDs []uuid.UUID

		responseProviders map[uuid.UUID]*provider.Provider

		expectRes []*route.Route
	}{
		{
			name: "single provider override returns synthetic route with ID=ProviderID",

			customerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			target:     "+1",
			targetProviderIDs: []uuid.UUID{
				uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			},

			responseProviders: map[uuid.UUID]*provider.Provider{
				uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"): {
					ID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				},
			},

			expectRes: []*route.Route{
				{
					ID:         uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
					CustomerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
					ProviderID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
					Target:     "+1",
					Priority:   0,
					Name:       "synthetic-route",
					Detail:     "Synthetic route generated for route_provider_ids override. Not persisted.",
				},
			},
		},
		{
			name: "three providers preserve array order and priority",

			customerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			target:     "+82",
			targetProviderIDs: []uuid.UUID{
				uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
				uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
			},

			responseProviders: map[uuid.UUID]*provider.Provider{
				uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"): {
					ID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
				},
				uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"): {
					ID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
				},
				uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"): {
					ID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
				},
			},

			expectRes: []*route.Route{
				{
					ID:         uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
					CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
					ProviderID: uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
					Target:     "+82",
					Priority:   0,
					Name:       "synthetic-route",
					Detail:     "Synthetic route generated for route_provider_ids override. Not persisted.",
				},
				{
					ID:         uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
					CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
					ProviderID: uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
					Target:     "+82",
					Priority:   1,
					Name:       "synthetic-route",
					Detail:     "Synthetic route generated for route_provider_ids override. Not persisted.",
				},
				{
					ID:         uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
					CustomerID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
					ProviderID: uuid.FromStringOrNil("cccccccc-cccc-cccc-cccc-cccccccccccc"),
					Target:     "+82",
					Priority:   2,
					Name:       "synthetic-route",
					Detail:     "Synthetic route generated for route_provider_ids override. Not persisted.",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &routeHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			// Expect ProviderGet to be called for each targetProviderID in order.
			for _, pid := range tt.targetProviderIDs {
				mockDB.EXPECT().ProviderGet(ctx, pid).Return(tt.responseProviders[pid], nil)
			}

			res, err := h.DialrouteList(ctx, tt.customerID, tt.target, tt.targetProviderIDs)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

			// Critical (C1) assertion: every synthetic route ID must be unique and non-Nil.
			seen := make(map[uuid.UUID]bool)
			for _, r := range res {
				if r.ID == uuid.Nil {
					t.Errorf("synthetic route ID must not be Nil, got nil for route: %v", r)
				}
				if r.ID != r.ProviderID {
					t.Errorf("synthetic route ID must equal ProviderID. got ID=%s ProviderID=%s", r.ID, r.ProviderID)
				}
				if seen[r.ID] {
					t.Errorf("synthetic route IDs must be unique, got duplicate: %s", r.ID)
				}
				seen[r.ID] = true
			}
		})
	}
}

func Test_DialrouteList_UnknownProviderReturnsError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &routeHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	unknownPID := uuid.FromStringOrNil("deadbeef-dead-beef-dead-beefdeadbeef")

	// Provider lookup fails → must short-circuit before normal route merging.
	mockDB.EXPECT().ProviderGet(ctx, unknownPID).Return(nil, fmt.Errorf("provider not found"))
	// NOTE: No RouteList expectations — the normal merge path must NOT be invoked.

	res, err := h.DialrouteList(ctx, customerID, "+1", []uuid.UUID{unknownPID})
	if err == nil {
		t.Errorf("Expected error for unknown provider, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result on error, got %v", res)
	}
}
