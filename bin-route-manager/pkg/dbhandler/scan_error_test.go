package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/models/providercall"
	"monorepo/bin-route-manager/models/route"
)

const (
	scanErrListToken = "2021-01-01T00:00:00.000000Z"
	scanErrTMCreate  = "2020-04-18 03:22:17.000000"
)

// insertBadRow inserts a row whose column value cannot be converted into the model field.
func insertBadRow(t *testing.T, query string, args ...any) {
	t.Helper()

	if _, err := dbTest.Exec(query, args...); err != nil {
		t.Fatalf("could not insert the row. err: %v", err)
	}
}

// assertScanError checks that the error carries the expected wrapping chain.
func assertScanError(t *testing.T, err error, expectContexts ...string) {
	t.Helper()

	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	for _, c := range expectContexts {
		if !strings.Contains(err.Error(), c) {
			t.Errorf("Wrong match. expect: message containing %q, got: %v", c, err)
		}
	}
	if errors.Unwrap(err) == nil {
		t.Errorf("Wrong match. expect: wrapped error, got: %v", err)
	}
}

func Test_Route_ScanError(t *testing.T) {
	badID := uuid.FromStringOrNil("c3000000-0000-0000-0000-000000000001")
	goodID := uuid.FromStringOrNil("c3000000-0000-0000-0000-000000000002")
	name := "scan-error-route"

	insert := func(t *testing.T) {
		cleanupRows(t, routesTable, badID, goodID)
		// the customer id is shorter than a uuid, so the row cannot be converted
		insertBadRow(t, "INSERT INTO "+routesTable+"(id, customer_id, name, tm_create) VALUES (?, ?, ?, ?)",
			badID.Bytes(), []byte{0x01, 0x02, 0x03}, name, scanErrTMCreate)
		insertBadRow(t, "INSERT INTO "+routesTable+"(id, customer_id, name, tm_create) VALUES (?, ?, ?, ?)",
			goodID.Bytes(), goodID.Bytes(), name, scanErrTMCreate)
	}

	t.Run("get returns the scan error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockCache := newLiveTestHandler(mc)
		insert(t)

		ctx := context.Background()
		mockCache.EXPECT().RouteGet(ctx, badID).Return(nil, errors.New("cache miss"))

		res, err := h.RouteGet(ctx, badID)
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
		assertScanError(t, err, "could not get data from row. routeGetFromDB", "copy to field failed")
	})

	t.Run("get of the readable row still works", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, _ := newLiveTestHandler(mc)
		insert(t)

		res, err := h.routeGetFromDB(context.Background(), goodID)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if res.ID != goodID || res.Name != name {
			t.Errorf("Wrong match. expect: id %s name %s, got: id %s name %s", goodID, name, res.ID, res.Name)
		}
	})

	t.Run("list fails as a whole when one row cannot be scanned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, _ := newLiveTestHandler(mc)
		insert(t)

		res, err := h.RouteList(context.Background(), scanErrListToken, 100, map[route.Field]any{route.FieldName: name})
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
		assertScanError(t, err, "could not get data. RouteGets", "copy to field failed")
	})
}

func Test_ProviderCall_ScanError(t *testing.T) {
	badID := uuid.FromStringOrNil("c4000000-0000-0000-0000-000000000001")
	goodID := uuid.FromStringOrNil("c4000000-0000-0000-0000-000000000002")
	providerID := uuid.FromStringOrNil("c4000000-0000-0000-0000-0000000000a1")

	insert := func(t *testing.T) {
		cleanupRows(t, providerCallsTable, badID, goodID)
		insertBadRow(t, "INSERT INTO "+providerCallsTable+"(id, customer_id, provider_id, tm_create) VALUES (?, ?, ?, ?)",
			badID.Bytes(), []byte{0x01, 0x02, 0x03}, providerID.Bytes(), scanErrTMCreate)
		insertBadRow(t, "INSERT INTO "+providerCallsTable+"(id, customer_id, provider_id, tm_create) VALUES (?, ?, ?, ?)",
			goodID.Bytes(), goodID.Bytes(), providerID.Bytes(), scanErrTMCreate)
	}

	t.Run("get returns the scan error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, _ := newLiveTestHandler(mc)
		insert(t)

		res, err := h.ProviderCallGet(context.Background(), badID)
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
		assertScanError(t, err, "could not get data from row. providerCallGetFromDB", "copy to field failed")
	})

	t.Run("list fails as a whole when one row cannot be scanned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, _ := newLiveTestHandler(mc)
		insert(t)

		res, err := h.ProviderCallList(context.Background(), scanErrListToken, 100, map[providercall.Field]any{providercall.FieldProviderID: providerID})
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
		assertScanError(t, err, "could not get data. ProviderCallList", "copy to field failed")
	})
}

func Test_Provider_ScanError(t *testing.T) {
	badID := uuid.FromStringOrNil("c5000000-0000-0000-0000-000000000001")
	goodID := uuid.FromStringOrNil("c5000000-0000-0000-0000-000000000002")
	name := "scan-error-provider"

	insert := func(t *testing.T) {
		cleanupRows(t, providersTable, badID, goodID)
		// tech_headers holds text that is not valid JSON
		insertBadRow(t, "INSERT INTO "+providersTable+"(id, name, tech_headers, metadata, tm_create) VALUES (?, ?, ?, ?, ?)",
			badID.Bytes(), name, "notjson{", "{}", scanErrTMCreate)
		insertBadRow(t, "INSERT INTO "+providersTable+"(id, name, tech_headers, metadata, tm_create) VALUES (?, ?, ?, ?, ?)",
			goodID.Bytes(), name, "{}", "{}", scanErrTMCreate)
	}

	t.Run("get returns the scan error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockCache := newLiveTestHandler(mc)
		insert(t)

		ctx := context.Background()
		mockCache.EXPECT().ProviderGet(ctx, badID).Return(nil, errors.New("cache miss"))

		res, err := h.ProviderGet(ctx, badID)
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
		assertScanError(t, err, "could not get data from row. providerGetFromDB", "copy to field failed")
	})

	t.Run("list fails as a whole when one row cannot be scanned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, _ := newLiveTestHandler(mc)
		insert(t)

		res, err := h.ProviderList(context.Background(), scanErrListToken, 100, map[provider.Field]any{provider.FieldName: name})
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
		assertScanError(t, err, "could not get data. ProviderGets", "copy to field failed")
	})
}

func Test_PrepareFieldsError(t *testing.T) {
	id := uuid.FromStringOrNil("c6000000-0000-0000-0000-000000000001")
	now := timePtr(time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC))
	// a channel inside a slice or a map cannot be marshaled
	unmarshalable := []any{make(chan int)}

	tests := []struct {
		name string

		run func(h *handler, ctx context.Context) error

		expectContext string
	}{
		{
			name: "provider create with unmarshalable metadata",
			run: func(h *handler, ctx context.Context) error {
				return h.ProviderCreate(ctx, &provider.Provider{ID: id, Metadata: map[string]interface{}{"k": unmarshalable}})
			},
			expectContext: "could not prepare fields. ProviderCreate",
		},
		{
			name: "provider update with unmarshalable field",
			run: func(h *handler, ctx context.Context) error {
				return h.ProviderUpdate(ctx, id, map[provider.Field]any{provider.FieldName: unmarshalable})
			},
			expectContext: "ProviderUpdate: prepare fields failed",
		},
		{
			name: "route update with unmarshalable field",
			run: func(h *handler, ctx context.Context) error {
				return h.RouteUpdate(ctx, id, map[route.Field]any{route.FieldName: unmarshalable})
			},
			expectContext: "RouteUpdate: prepare fields failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the closed database makes sure that nothing is executed: reaching it would change the error
			h, mockUtil, _ := newClosedDBHandler(t, mc)
			mockUtil.EXPECT().TimeNow().Return(now)

			err := tt.run(h, context.Background())
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !strings.Contains(err.Error(), tt.expectContext) || !strings.Contains(err.Error(), "JSON marshal failed") {
				t.Errorf("Wrong match. expect: %q and JSON marshal failed, got: %v", tt.expectContext, err)
			}
			if strings.Contains(err.Error(), closedDBMessage) {
				t.Errorf("Wrong match. the statement must not be executed, got: %v", err)
			}
		})
	}
}
