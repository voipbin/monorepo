package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/providercall"
	"monorepo/bin-route-manager/pkg/cachehandler"
)

// newProviderCallTestHandler returns a handler backed by the shared sqlite test database.
func newProviderCallTestHandler(mc *gomock.Controller) (*handler, *utilhandler.MockUtilHandler) {
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := &handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       cachehandler.NewMockCacheHandler(mc),
	}
	return h, mockUtil
}

// cleanupProviderCalls removes the given rows before and after the test, because the sqlite
// database is shared by all tests of the package and the test ids are fixed.
func cleanupProviderCalls(t *testing.T, ids ...uuid.UUID) {
	t.Helper()

	del := func() {
		for _, id := range ids {
			if _, err := dbTest.Exec("DELETE FROM route_providercalls WHERE id = ?", id.Bytes()); err != nil {
				t.Errorf("could not clean up the row. id: %s, err: %v", id, err)
			}
		}
	}
	del()
	t.Cleanup(del)
}

func Test_ProviderCallCreate_Get(t *testing.T) {
	tests := []struct {
		name         string
		providerCall *providercall.ProviderCall

		responseCurTime *time.Time

		expectRes *providercall.ProviderCall
	}{
		{
			"nil slices are stored as empty arrays",
			&providercall.ProviderCall{
				ID:         uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000001"),
				ProviderID: uuid.FromStringOrNil("a3000000-0000-0000-0000-000000000001"),
			},

			timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)),

			&providercall.ProviderCall{
				ID:           uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001"),
				CustomerID:   uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000001"),
				ProviderID:   uuid.FromStringOrNil("a3000000-0000-0000-0000-000000000001"),
				Destinations: []commonaddress.Address{},
				CallIDs:      []uuid.UUID{},
				GroupcallIDs: []uuid.UUID{},
				TMCreate:     timePtr(time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)),
			},
		},
		{
			"all fields",
			&providercall.ProviderCall{
				ID:         uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000002"),
				CustomerID: uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000002"),
				ProviderID: uuid.FromStringOrNil("a3000000-0000-0000-0000-000000000002"),
				FlowID:     uuid.FromStringOrNil("a4000000-0000-0000-0000-000000000002"),
				Source:     &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"},
				Destinations: []commonaddress.Address{
					{Type: commonaddress.TypeTel, Target: "+15550000002"},
					{Type: commonaddress.TypeTel, Target: "+15550000003"},
				},
				Anonymous:    "auto",
				CallIDs:      []uuid.UUID{uuid.FromStringOrNil("a5000000-0000-0000-0000-000000000002")},
				GroupcallIDs: []uuid.UUID{uuid.FromStringOrNil("a6000000-0000-0000-0000-000000000002")},
				// timestamps supplied by the caller must be overwritten by Create
				TMUpdate: timePtr(time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)),
				TMDelete: timePtr(time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)),
			},

			timePtr(time.Date(2020, 5, 18, 3, 22, 17, 995000000, time.UTC)),

			&providercall.ProviderCall{
				ID:         uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000002"),
				CustomerID: uuid.FromStringOrNil("a2000000-0000-0000-0000-000000000002"),
				ProviderID: uuid.FromStringOrNil("a3000000-0000-0000-0000-000000000002"),
				FlowID:     uuid.FromStringOrNil("a4000000-0000-0000-0000-000000000002"),
				Source:     &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"},
				Destinations: []commonaddress.Address{
					{Type: commonaddress.TypeTel, Target: "+15550000002"},
					{Type: commonaddress.TypeTel, Target: "+15550000003"},
				},
				Anonymous:    "auto",
				CallIDs:      []uuid.UUID{uuid.FromStringOrNil("a5000000-0000-0000-0000-000000000002")},
				GroupcallIDs: []uuid.UUID{uuid.FromStringOrNil("a6000000-0000-0000-0000-000000000002")},
				TMCreate:     timePtr(time.Date(2020, 5, 18, 3, 22, 17, 995000000, time.UTC)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockUtil := newProviderCallTestHandler(mc)
			ctx := context.Background()
			cleanupProviderCalls(t, tt.providerCall.ID)

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.ProviderCallCreate(ctx, tt.providerCall); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.ProviderCallGet(ctx, tt.providerCall.ID)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ProviderCallGet_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newProviderCallTestHandler(mc)

	_, err := h.ProviderCallGet(context.Background(), uuid.FromStringOrNil("ffffffff-0000-0000-0000-00000000aa01"))
	if err != ErrNotFound {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}

func Test_ProviderCallCreate_DuplicateID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil := newProviderCallTestHandler(mc)
	ctx := context.Background()

	id := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000009")
	cleanupProviderCalls(t, id)
	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 18, 0, 0, 0, 0, time.UTC))).Times(2)

	if err := h.ProviderCallCreate(ctx, &providercall.ProviderCall{ID: id}); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := h.ProviderCallCreate(ctx, &providercall.ProviderCall{ID: id}); err == nil {
		t.Errorf("Wrong match. expect: error on duplicate primary key, got: nil")
	}
}

func Test_ProviderCallList(t *testing.T) {
	customerID := uuid.FromStringOrNil("b2000000-0000-0000-0000-000000000001")
	otherCustomerID := uuid.FromStringOrNil("b2000000-0000-0000-0000-000000000002")
	providerID := uuid.FromStringOrNil("b3000000-0000-0000-0000-000000000001")
	otherProviderID := uuid.FromStringOrNil("b3000000-0000-0000-0000-000000000002")

	idJan := uuid.FromStringOrNil("b1000000-0000-0000-0000-000000000001")
	idFeb := uuid.FromStringOrNil("b1000000-0000-0000-0000-000000000002")
	idMar := uuid.FromStringOrNil("b1000000-0000-0000-0000-000000000003")
	idOther := uuid.FromStringOrNil("b1000000-0000-0000-0000-000000000004")
	idDeleted := uuid.FromStringOrNil("b1000000-0000-0000-0000-000000000005")

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil := newProviderCallTestHandler(mc)
	ctx := context.Background()
	cleanupProviderCalls(t, idJan, idFeb, idMar, idOther, idDeleted)

	// the sqlite database is shared by all tests, so every row of this test uses its own customer ids
	seeds := []struct {
		pc *providercall.ProviderCall
		ts time.Time
	}{
		{&providercall.ProviderCall{ID: idJan, CustomerID: customerID, ProviderID: providerID}, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)},
		{&providercall.ProviderCall{ID: idFeb, CustomerID: customerID, ProviderID: otherProviderID}, time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)},
		{&providercall.ProviderCall{ID: idMar, CustomerID: customerID, ProviderID: providerID}, time.Date(2020, 3, 1, 0, 0, 0, 0, time.UTC)},
		{&providercall.ProviderCall{ID: idOther, CustomerID: otherCustomerID, ProviderID: providerID}, time.Date(2020, 3, 2, 0, 0, 0, 0, time.UTC)},
		{&providercall.ProviderCall{ID: idDeleted, CustomerID: customerID, ProviderID: providerID}, time.Date(2020, 3, 3, 0, 0, 0, 0, time.UTC)},
	}
	for _, s := range seeds {
		mockUtil.EXPECT().TimeNow().Return(timePtr(s.ts))
		if err := h.ProviderCallCreate(ctx, s.pc); err != nil {
			t.Fatalf("could not seed. err: %v", err)
		}
	}
	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 4, 1, 0, 0, 0, 0, time.UTC)))
	if err := h.ProviderCallDelete(ctx, idDeleted); err != nil {
		t.Fatalf("could not soft-delete the seed. err: %v", err)
	}

	tests := []struct {
		name string

		token   string
		limit   uint64
		filters map[providercall.Field]any

		expectIDs []uuid.UUID
	}{
		{
			"newest first and soft-deleted rows excluded",
			"2021-01-01T00:00:00.000000Z", 10,
			map[providercall.Field]any{providercall.FieldCustomerID: customerID},
			[]uuid.UUID{idMar, idFeb, idJan},
		},
		{
			"limit keeps the newest rows",
			"2021-01-01T00:00:00.000000Z", 2,
			map[providercall.Field]any{providercall.FieldCustomerID: customerID},
			[]uuid.UUID{idMar, idFeb},
		},
		{
			"token excludes rows created at or after it",
			"2020-02-15T00:00:00.000000Z", 10,
			map[providercall.Field]any{providercall.FieldCustomerID: customerID},
			[]uuid.UUID{idFeb, idJan},
		},
		{
			// the token is written in the format sqlite stores, so it equals the Feb row's tm_create exactly
			"row created exactly at the token is excluded",
			"2020-02-01 00:00:00+00:00", 10,
			map[providercall.Field]any{providercall.FieldCustomerID: customerID},
			[]uuid.UUID{idJan},
		},
		{
			"customer and provider filters combine",
			"2021-01-01T00:00:00.000000Z", 10,
			map[providercall.Field]any{
				providercall.FieldCustomerID: customerID,
				providercall.FieldProviderID: providerID,
			},
			[]uuid.UUID{idMar, idJan},
		},
		{
			"other customer is isolated",
			"2021-01-01T00:00:00.000000Z", 10,
			map[providercall.Field]any{providercall.FieldCustomerID: otherCustomerID},
			[]uuid.UUID{idOther},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := h.ProviderCallList(ctx, tt.token, tt.limit, tt.filters)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			got := []uuid.UUID{}
			for _, r := range res {
				got = append(got, r.ID)
			}
			if !reflect.DeepEqual(tt.expectIDs, got) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectIDs, got)
			}
		})
	}
}

func Test_ProviderCallList_DefaultToken(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil := newProviderCallTestHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("b2000000-0000-0000-0000-000000000010")
	id := uuid.FromStringOrNil("b1000000-0000-0000-0000-000000000010")
	cleanupProviderCalls(t, id)

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)))
	if err := h.ProviderCallCreate(ctx, &providercall.ProviderCall{ID: id, CustomerID: customerID}); err != nil {
		t.Fatalf("could not seed. err: %v", err)
	}

	// an empty token falls back to the current time provided by the util handler
	mockUtil.EXPECT().TimeGetCurTime().Return("2021-01-01T00:00:00.000000Z")
	res, err := h.ProviderCallList(ctx, "", 10, map[providercall.Field]any{providercall.FieldCustomerID: customerID})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 1 || res[0].ID != id {
		t.Errorf("Wrong match. expect: [%s], got: %v", id, res)
	}

	// a token older than the row hides it
	mockUtil.EXPECT().TimeGetCurTime().Return("2019-01-01T00:00:00.000000Z")
	res, err = h.ProviderCallList(ctx, "", 10, map[providercall.Field]any{providercall.FieldCustomerID: customerID})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Wrong match. expect: empty, got: %v", res)
	}
}

func Test_ProviderCallList_InvalidFilter(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newProviderCallTestHandler(mc)
	ctx := context.Background()

	// a column that does not exist fails at query time
	if _, err := h.ProviderCallList(ctx, "2021-01-01T00:00:00.000000Z", 10, map[providercall.Field]any{providercall.Field("no_such_column"): "x"}); err == nil {
		t.Errorf("Wrong match. expect: error for an unknown column, got: nil")
	}

	// a filter value type that cannot be bound is rejected while applying the filters
	if _, err := h.ProviderCallList(ctx, "2021-01-01T00:00:00.000000Z", 10, map[providercall.Field]any{providercall.FieldCustomerID: struct{}{}}); err == nil {
		t.Errorf("Wrong match. expect: error for an unsupported filter value type, got: nil")
	}
}

func Test_ProviderCallDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil := newProviderCallTestHandler(mc)
	ctx := context.Background()

	id := uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")
	cleanupProviderCalls(t, id)
	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	deletedAt := time.Date(2020, 6, 1, 12, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().TimeNow().Return(timePtr(createdAt))
	if err := h.ProviderCallCreate(ctx, &providercall.ProviderCall{ID: id, CustomerID: uuid.FromStringOrNil("c2000000-0000-0000-0000-000000000001")}); err != nil {
		t.Fatalf("could not seed. err: %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(timePtr(deletedAt))
	if err := h.ProviderCallDelete(ctx, id); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// Read the row back after the soft delete and check its timestamps.
	res, err := h.ProviderCallGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.TMDelete == nil || !res.TMDelete.Equal(deletedAt) {
		t.Errorf("Wrong match. expect tm_delete: %v, got: %v", deletedAt, res.TMDelete)
	}
	if res.TMUpdate == nil || !res.TMUpdate.Equal(deletedAt) {
		t.Errorf("Wrong match. expect tm_update: %v, got: %v", deletedAt, res.TMUpdate)
	}
	if res.TMCreate == nil || !res.TMCreate.Equal(createdAt) {
		t.Errorf("Wrong match. tm_create must be kept. expect: %v, got: %v", createdAt, res.TMCreate)
	}
}

func Test_ProviderCallDelete_NotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil := newProviderCallTestHandler(mc)

	mockUtil.EXPECT().TimeNow().Return(timePtr(time.Now()))
	err := h.ProviderCallDelete(context.Background(), uuid.FromStringOrNil("ffffffff-0000-0000-0000-00000000aa02"))
	if err != ErrNotFound {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}
