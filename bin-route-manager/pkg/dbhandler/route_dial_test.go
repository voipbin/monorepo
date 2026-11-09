package dbhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-route-manager/models/route"
)

// insertRouteForDialTest inserts a route row directly. A nil priority stores SQL NULL.
func insertRouteForDialTest(t *testing.T, id, customerID uuid.UUID, target string, priority *int, tmCreate string) {
	t.Helper()

	_, err := dbTest.Exec(
		"INSERT INTO route_routes(id, customer_id, target, priority, tm_create) VALUES(?, ?, ?, ?, ?)",
		id.Bytes(), customerID.Bytes(), target, priority, tmCreate,
	)
	if err != nil {
		t.Fatalf("could not insert the route. err: %v", err)
	}
}

func intPtr(v int) *int {
	return &v
}

func Test_RouteListForDial_order(t *testing.T) {
	customerID := uuid.FromStringOrNil("a1000000-0000-4000-8000-0000000000c1")
	other := uuid.FromStringOrNil("a1000000-0000-4000-8000-0000000000c2")
	id := func(n int) uuid.UUID {
		return uuid.FromStringOrNil(fmt.Sprintf("a1000000-0000-4000-8000-0000000001%02d", n))
	}

	// id, priority, tm_create
	insertRouteForDialTest(t, id(1), customerID, "all", intPtr(5), "2020-04-18 03:22:10.000000")
	insertRouteForDialTest(t, id(2), customerID, "all", intPtr(5), "2020-04-18 03:22:12.000000") // newer, same priority
	insertRouteForDialTest(t, id(3), customerID, "all", intPtr(1), "2020-04-18 03:22:01.000000")
	insertRouteForDialTest(t, id(4), customerID, "all", nil, "2020-04-18 03:22:20.000000") // NULL, newest
	insertRouteForDialTest(t, id(5), customerID, "all", intPtr(0), "2020-04-18 03:22:02.000000")
	insertRouteForDialTest(t, id(6), customerID, "all", intPtr(-1), "2020-04-18 03:22:03.000000")
	insertRouteForDialTest(t, id(7), customerID, "all", intPtr(2147483647), "2020-04-18 03:22:15.000000") // ties with NULL
	insertRouteForDialTest(t, id(8), customerID, "all", intPtr(5), "2020-04-18 03:22:12.000000")          // same priority and tm_create as id(2), larger id
	insertRouteForDialTest(t, id(9), customerID, "+82", intPtr(1), "2020-04-18 03:22:01.000000")          // other target
	insertRouteForDialTest(t, id(10), other, "all", intPtr(1), "2020-04-18 03:22:01.000000")              // other customer

	h := handler{db: dbTest}
	res, err := h.RouteListForDial(context.Background(), 1000, map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     "all",
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// priority asc; ties: tm_create DESC, id DESC; NULL is read as 2147483647 and comes last
	expectIDs := []uuid.UUID{id(6), id(5), id(3), id(8), id(2), id(1), id(4), id(7)}
	expectPriority := []int{-1, 0, 1, 5, 5, 5, 2147483647, 2147483647}
	if len(res) != len(expectIDs) {
		t.Fatalf("Wrong match. expect: %d routes, got: %d", len(expectIDs), len(res))
	}
	for i := range expectIDs {
		if res[i].ID != expectIDs[i] || res[i].Priority != expectPriority[i] {
			t.Errorf("Wrong match at %d. expect: %s/%d, got: %s/%d", i, expectIDs[i], expectPriority[i], res[i].ID, res[i].Priority)
		}
	}
}

func Test_RouteListForDial_limit_keeps_lowest_priority(t *testing.T) {
	customerID := uuid.FromStringOrNil("a2000000-0000-4000-8000-0000000000c1")

	// 1001 routes: the one with the lowest priority number is the oldest, so a tm_create DESC cut would drop it
	oldest := uuid.FromStringOrNil("a2000000-0000-4000-8000-000000000001")
	insertRouteForDialTest(t, oldest, customerID, "all", intPtr(1), "2019-01-01 00:00:00.000000")
	for i := 0; i < 1000; i++ {
		rid := uuid.FromStringOrNil(fmt.Sprintf("a2000000-0000-4000-8000-%012x", 0x1000+i))
		insertRouteForDialTest(t, rid, customerID, "all", intPtr(9), fmt.Sprintf("2020-01-01 00:%02d:%02d.000000", i/60, i%60))
	}

	h := handler{db: dbTest}
	res, err := h.RouteListForDial(context.Background(), 1000, map[route.Field]any{
		route.FieldCustomerID: customerID,
		route.FieldTarget:     "all",
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 1000 {
		t.Fatalf("Wrong match. expect: 1000 routes, got: %d", len(res))
	}
	if res[0].ID != oldest {
		t.Errorf("Wrong match. expect the lowest priority route first even though it is the oldest. got: %s", res[0].ID)
	}
}

func Test_RouteListForDial_excludes_deleted(t *testing.T) {
	customerID := uuid.FromStringOrNil("a3000000-0000-4000-8000-0000000000c1")
	live := uuid.FromStringOrNil("a3000000-0000-4000-8000-000000000001")
	deleted := uuid.FromStringOrNil("a3000000-0000-4000-8000-000000000002")
	insertRouteForDialTest(t, live, customerID, "all", intPtr(2), "2020-04-18 03:22:01.000000")
	insertRouteForDialTest(t, deleted, customerID, "all", intPtr(1), "2020-04-18 03:22:02.000000")
	if _, err := dbTest.Exec("UPDATE route_routes SET tm_delete = '2020-05-01 00:00:00.000000' WHERE id = ?", deleted.Bytes()); err != nil {
		t.Fatalf("could not mark the route as deleted. err: %v", err)
	}

	h := handler{db: dbTest}
	res, err := h.RouteListForDial(context.Background(), 1000, map[route.Field]any{
		route.FieldCustomerID: customerID,
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 1 || res[0].ID != live {
		t.Errorf("Wrong match. expect only the live route, got: %v", res)
	}
}

func Test_RouteListForDial_invalid_filter(t *testing.T) {
	h := handler{db: dbTest}
	if _, err := h.RouteListForDial(context.Background(), 1000, map[route.Field]any{route.Field("no_such_field"): 1}); err == nil {
		t.Errorf("Wrong match. expect: error for an unknown filter field, got: nil")
	}
}
