package routehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"monorepo/bin-route-manager/models/route"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

func newPriorityTestRoute(n int, providerN int, priority int) *route.Route {
	return &route.Route{
		ID:         uuid.FromStringOrNil(fmt.Sprintf("e1000000-0000-4000-8000-%012d", n)),
		ProviderID: uuid.FromStringOrNil(fmt.Sprintf("e2000000-0000-4000-8000-%012d", providerN)),
		Priority:   priority,
	}
}

func Test_ListByTarget_priority_order(t *testing.T) {
	customerID := uuid.FromStringOrNil("e3000000-0000-4000-8000-0000000000c1")

	// many routes with only a few distinct priorities: a non stable sort would reorder the ties
	manyTarget := []*route.Route{}
	manyAll := []*route.Route{}
	for i := 0; i < 40; i++ {
		manyTarget = append(manyTarget, newPriorityTestRoute(100+i, 100+i, i%3))
	}
	for i := 0; i < 40; i++ {
		manyAll = append(manyAll, newPriorityTestRoute(200+i, 200+i, i%3))
	}
	expectMany := []*route.Route{}
	for p := 0; p < 3; p++ {
		for _, r := range manyTarget {
			if r.Priority == p {
				expectMany = append(expectMany, r)
			}
		}
		for _, r := range manyAll {
			if r.Priority == p {
				expectMany = append(expectMany, r)
			}
		}
	}

	tests := []struct {
		name string

		responseTarget []*route.Route
		responseAll    []*route.Route

		expectRes []*route.Route
	}{
		{
			name: "ascending order across target and all",
			responseTarget: []*route.Route{
				newPriorityTestRoute(1, 1, 3),
				newPriorityTestRoute(2, 2, 1),
			},
			responseAll: []*route.Route{
				newPriorityTestRoute(3, 3, 2),
				newPriorityTestRoute(4, 4, 0),
			},
			expectRes: []*route.Route{
				newPriorityTestRoute(4, 4, 0),
				newPriorityTestRoute(2, 2, 1),
				newPriorityTestRoute(3, 3, 2),
				newPriorityTestRoute(1, 1, 3),
			},
		},
		{
			name: "negative zero and the largest value",
			responseTarget: []*route.Route{
				newPriorityTestRoute(1, 1, 2147483647),
				newPriorityTestRoute(2, 2, 0),
			},
			responseAll: []*route.Route{
				newPriorityTestRoute(3, 3, -5),
			},
			expectRes: []*route.Route{
				newPriorityTestRoute(3, 3, -5),
				newPriorityTestRoute(2, 2, 0),
				newPriorityTestRoute(1, 1, 2147483647),
			},
		},
		{
			name: "same priority keeps the input order and target comes before all",
			responseTarget: []*route.Route{
				newPriorityTestRoute(1, 1, 5),
				newPriorityTestRoute(2, 2, 5),
			},
			responseAll: []*route.Route{
				newPriorityTestRoute(3, 3, 5),
				newPriorityTestRoute(4, 4, 5),
			},
			expectRes: []*route.Route{
				newPriorityTestRoute(1, 1, 5),
				newPriorityTestRoute(2, 2, 5),
				newPriorityTestRoute(3, 3, 5),
				newPriorityTestRoute(4, 4, 5),
			},
		},
		{
			name: "same provider: the target route wins with its own priority",
			responseTarget: []*route.Route{
				newPriorityTestRoute(1, 7, 9),
			},
			responseAll: []*route.Route{
				newPriorityTestRoute(2, 7, 1),
				newPriorityTestRoute(3, 8, 5),
			},
			expectRes: []*route.Route{
				newPriorityTestRoute(3, 8, 5),
				newPriorityTestRoute(1, 7, 9),
			},
		},
		{
			name:           "same provider inside all: the one returned first wins",
			responseTarget: []*route.Route{},
			responseAll: []*route.Route{
				newPriorityTestRoute(1, 7, 1),
				newPriorityTestRoute(2, 7, 2),
			},
			expectRes: []*route.Route{
				newPriorityTestRoute(1, 7, 1),
			},
		},
		{
			name:           "many routes with ties",
			responseTarget: manyTarget,
			responseAll:    manyAll,
			expectRes:      expectMany,
		},
		{
			name:           "empty",
			responseTarget: []*route.Route{},
			responseAll:    []*route.Route{},
			expectRes:      []*route.Route{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &routeHandler{db: mockDB, notifyHandler: notifyhandler.NewMockNotifyHandler(mc)}
			ctx := context.Background()

			mockDB.EXPECT().RouteListForDial(ctx, uint64(1000), map[route.Field]any{
				route.FieldCustomerID: customerID,
				route.FieldTarget:     "+82",
			}).Return(tt.responseTarget, nil)
			mockDB.EXPECT().RouteListForDial(ctx, uint64(1000), map[route.Field]any{
				route.FieldCustomerID: customerID,
				route.FieldTarget:     route.TargetAll,
			}).Return(tt.responseAll, nil)

			res, err := h.ListByTarget(ctx, customerID, "+82")
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ListByTarget_priority_order_all_query_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &routeHandler{db: mockDB, notifyHandler: notifyhandler.NewMockNotifyHandler(mc)}
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("e3000000-0000-4000-8000-0000000000c2")

	mockDB.EXPECT().RouteListForDial(ctx, uint64(1000), map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     "+82",
	}).Return([]*route.Route{newPriorityTestRoute(1, 1, 4), newPriorityTestRoute(2, 2, 2)}, nil)
	mockDB.EXPECT().RouteListForDial(ctx, uint64(1000), map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     route.TargetAll,
	}).Return(nil, fmt.Errorf("database error"))

	// the error of the "all" query is only logged and the sorted target routes are returned
	res, err := h.ListByTarget(ctx, customerID, "+82")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	expectRes := []*route.Route{newPriorityTestRoute(2, 2, 2), newPriorityTestRoute(1, 1, 4)}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_ListByTarget_does_not_modify_db_result(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &routeHandler{db: mockDB, notifyHandler: notifyhandler.NewMockNotifyHandler(mc)}
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("e3000000-0000-4000-8000-0000000000c3")

	target := []*route.Route{newPriorityTestRoute(1, 1, 9), newPriorityTestRoute(2, 2, 3)}
	mockDB.EXPECT().RouteListForDial(ctx, uint64(1000), gomock.Any()).Return(target, nil).Times(2)

	if _, err := h.ListByTarget(ctx, customerID, "+82"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if target[0].ID != newPriorityTestRoute(1, 1, 9).ID || target[1].ID != newPriorityTestRoute(2, 2, 3).ID {
		t.Errorf("Wrong match. the slice returned by the database handler must keep its order. got: %v", target)
	}
}

func Test_DialrouteList_customer_routes_stay_before_default_routes(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &routeHandler{db: mockDB, notifyHandler: notifyhandler.NewMockNotifyHandler(mc)}
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("e3000000-0000-4000-8000-0000000000c4")

	customerA := newPriorityTestRoute(1, 1, 7)
	customerB := newPriorityTestRoute(2, 2, 3)
	defaultFirst := newPriorityTestRoute(3, 3, 0) // lower number than every customer route, still after them
	defaultSameProvider := newPriorityTestRoute(4, 1, 0)
	defaultLast := newPriorityTestRoute(5, 5, 4)

	mockDB.EXPECT().RouteListForDial(ctx, uint64(1000), map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     "+82",
	}).Return([]*route.Route{customerA, customerB}, nil)
	mockDB.EXPECT().RouteListForDial(ctx, uint64(1000), map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     route.TargetAll,
	}).Return([]*route.Route{}, nil)
	mockDB.EXPECT().RouteListForDial(ctx, uint64(1000), map[route.Field]any{
		route.FieldCustomerID: route.CustomerIDBasicRoute,
		route.FieldTarget:     "+82",
	}).Return([]*route.Route{defaultLast, defaultSameProvider}, nil)
	mockDB.EXPECT().RouteListForDial(ctx, uint64(1000), map[route.Field]any{
		route.FieldCustomerID: route.CustomerIDBasicRoute,
		route.FieldTarget:     route.TargetAll,
	}).Return([]*route.Route{defaultFirst}, nil)

	res, err := h.DialrouteList(ctx, customerID, "+82", nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// customer routes by priority, then default routes by priority; the default route of a provider the customer already uses is dropped
	expectRes := []*route.Route{customerB, customerA, defaultFirst, defaultLast}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_DialrouteList_synthetic_routes_keep_the_given_order(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &routeHandler{db: mockDB, notifyHandler: notifyhandler.NewMockNotifyHandler(mc)}
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("e3000000-0000-4000-8000-0000000000c5")

	providerIDs := []uuid.UUID{
		uuid.FromStringOrNil("e2000000-0000-4000-8000-0000000000f3"),
		uuid.FromStringOrNil("e2000000-0000-4000-8000-0000000000f1"),
		uuid.FromStringOrNil("e2000000-0000-4000-8000-0000000000f2"),
	}
	for _, pid := range providerIDs {
		mockDB.EXPECT().ProviderGet(ctx, pid).Return(nil, nil)
	}
	// RouteListForDial must not be called: the mock fails the test on an unexpected call

	res, err := h.DialrouteList(ctx, customerID, "+82", providerIDs)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 3 {
		t.Fatalf("Wrong match. expect: 3 routes, got: %d", len(res))
	}
	for i, r := range res {
		if r.ID != providerIDs[i] || r.Priority != i {
			t.Errorf("Wrong match at %d. expect: %s/%d, got: %s/%d", i, providerIDs[i], i, r.ID, r.Priority)
		}
	}
}

func Test_ListByTarget_priority_order_target_all(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &routeHandler{db: mockDB, notifyHandler: notifyhandler.NewMockNotifyHandler(mc)}
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("e3000000-0000-4000-8000-0000000000c6")

	// target "all" runs the same query twice; the second result is dropped by the provider de-duplication
	routes := []*route.Route{newPriorityTestRoute(1, 1, 3), newPriorityTestRoute(2, 2, 1)}
	mockDB.EXPECT().RouteListForDial(ctx, uint64(1000), map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     route.TargetAll,
	}).Return(routes, nil).Times(2)

	res, err := h.ListByTarget(ctx, customerID, route.TargetAll)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	expectRes := []*route.Route{newPriorityTestRoute(2, 2, 1), newPriorityTestRoute(1, 1, 3)}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}
