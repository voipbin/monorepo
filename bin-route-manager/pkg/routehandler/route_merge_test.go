package routehandler

import (
	"context"
	stderrors "errors"
	"reflect"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/route"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

func Test_Get_ErrNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &routeHandler{db: mockDB, notifyHandler: notifyhandler.NewMockNotifyHandler(mc)}
	ctx := context.Background()
	id := uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000001")

	mockDB.EXPECT().RouteGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Get(ctx, id)
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("Wrong match. expect: *VoipbinError, got: %T %v", err, err)
	}
	if ve.Status != cerrors.StatusNotFound || ve.Reason != "ROUTE_NOT_FOUND" || ve.Domain != string(commonoutline.ServiceNameRouteManager) {
		t.Errorf("Wrong match. got: status %v, reason %v, domain %v", ve.Status, ve.Reason, ve.Domain)
	}
	if !stderrors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("Wrong match. expect: the cause ErrNotFound to stay reachable")
	}
}

func Test_ListByTarget_SameProviderPrefersTargetRoute(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &routeHandler{db: mockDB, notifyHandler: notifyhandler.NewMockNotifyHandler(mc)}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("d2000000-0000-0000-0000-0000000000c1")
	providerA := uuid.FromStringOrNil("d2000000-0000-0000-0000-0000000000a1")
	providerB := uuid.FromStringOrNil("d2000000-0000-0000-0000-0000000000b1")

	routeTarget := &route.Route{ID: uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000011"), ProviderID: providerA}
	routeAllSameProvider := &route.Route{ID: uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000012"), ProviderID: providerA}
	routeAllOtherProvider := &route.Route{ID: uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000013"), ProviderID: providerB}

	mockDB.EXPECT().RouteList(ctx, "", uint64(1000), map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     "+82",
	}).Return([]*route.Route{routeTarget}, nil)
	mockDB.EXPECT().RouteList(ctx, "", uint64(1000), map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     route.TargetAll,
	}).Return([]*route.Route{routeAllSameProvider, routeAllOtherProvider}, nil)

	res, err := h.ListByTarget(ctx, customerID, "+82")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the "all" route of the provider that already has a target route is dropped
	expectRes := []*route.Route{routeTarget, routeAllOtherProvider}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_DialrouteList_SameProviderPrefersCustomerRoute(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &routeHandler{db: mockDB, notifyHandler: notifyhandler.NewMockNotifyHandler(mc)}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("d2000000-0000-0000-0000-0000000000c2")
	providerA := uuid.FromStringOrNil("d2000000-0000-0000-0000-0000000000a2")
	providerB := uuid.FromStringOrNil("d2000000-0000-0000-0000-0000000000b2")

	customerRoute := &route.Route{ID: uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000021"), CustomerID: customerID, ProviderID: providerA}
	defaultSameProvider := &route.Route{ID: uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000022"), CustomerID: route.CustomerIDBasicRoute, ProviderID: providerA}
	defaultOtherProvider := &route.Route{ID: uuid.FromStringOrNil("d2000000-0000-0000-0000-000000000023"), CustomerID: route.CustomerIDBasicRoute, ProviderID: providerB}

	// customer side: one route for the target, none for "all"
	mockDB.EXPECT().RouteList(ctx, "", uint64(1000), map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     "+82",
	}).Return([]*route.Route{customerRoute}, nil)
	mockDB.EXPECT().RouteList(ctx, "", uint64(1000), map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     route.TargetAll,
	}).Return([]*route.Route{}, nil)

	// default side: the first provider overlaps with the customer route, the second does not
	mockDB.EXPECT().RouteList(ctx, "", uint64(1000), map[route.Field]any{
		route.FieldCustomerID: route.CustomerIDBasicRoute,
		route.FieldTarget:     "+82",
	}).Return([]*route.Route{defaultSameProvider}, nil)
	mockDB.EXPECT().RouteList(ctx, "", uint64(1000), map[route.Field]any{
		route.FieldCustomerID: route.CustomerIDBasicRoute,
		route.FieldTarget:     route.TargetAll,
	}).Return([]*route.Route{defaultOtherProvider}, nil)

	res, err := h.DialrouteList(ctx, customerID, "+82", nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the default route of the provider that the customer already routes through is dropped
	expectRes := []*route.Route{customerRoute, defaultOtherProvider}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}
