package dbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-registrar-manager/models/astaor"
	"monorepo/bin-registrar-manager/models/astauth"
	"monorepo/bin-registrar-manager/models/astcontact"
	"monorepo/bin-registrar-manager/models/astendpoint"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

var (
	errInjectedDB    = errors.New("injected db failure")
	errInjectedCache = errors.New("injected cache failure")
)

// newErrorPathHandler builds a handler on top of sqlmock and the cache mock.
func newErrorPathHandler(t *testing.T) (*handler, sqlmock.Sqlmock, *cachehandler.MockCacheHandler, *utilhandler.MockUtilHandler) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mc := gomock.NewController(t)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	return &handler{db: db, cache: mockCache, utilHandler: mockUtil}, mock, mockCache, mockUtil
}

// Test_CacheHit_SkipsDB verifies that a cache hit is returned without touching the DB.
// sqlmock has no expectations, so any query would fail the call.
func Test_CacheHit_SkipsDB(t *testing.T) {
	ctx := context.Background()
	extID := uuid.FromStringOrNil("5b1c7e3a-2f4d-11f1-9a6b-0f1d2c3b4a59")
	customerID := uuid.FromStringOrNil("6c2d8f4b-3a5e-11f1-8b7c-1a2e3d4c5b60")

	tests := []struct {
		name string
		fn   func(h *handler, c *cachehandler.MockCacheHandler) (any, error)
		want any
	}{
		{
			"aor",
			func(h *handler, c *cachehandler.MockCacheHandler) (any, error) {
				c.EXPECT().AstAORGet(gomock.Any(), "aor1").Return(&astaor.AstAOR{}, nil)
				return h.AstAORGet(ctx, "aor1")
			},
			&astaor.AstAOR{},
		},
		{
			"auth",
			func(h *handler, c *cachehandler.MockCacheHandler) (any, error) {
				c.EXPECT().AstAuthGet(gomock.Any(), "auth1").Return(&astauth.AstAuth{}, nil)
				return h.AstAuthGet(ctx, "auth1")
			},
			&astauth.AstAuth{},
		},
		{
			"endpoint",
			func(h *handler, c *cachehandler.MockCacheHandler) (any, error) {
				c.EXPECT().AstEndpointGet(gomock.Any(), "ep1").Return(&astendpoint.AstEndpoint{}, nil)
				return h.AstEndpointGet(ctx, "ep1")
			},
			&astendpoint.AstEndpoint{},
		},
		{
			"contacts by endpoint",
			func(h *handler, c *cachehandler.MockCacheHandler) (any, error) {
				c.EXPECT().AstContactsGet(gomock.Any(), "ep1").Return([]*astcontact.AstContact{}, nil)
				return h.AstContactGetsByEndpoint(ctx, "ep1")
			},
			[]*astcontact.AstContact{},
		},
		{
			"extension",
			func(h *handler, c *cachehandler.MockCacheHandler) (any, error) {
				c.EXPECT().ExtensionGet(gomock.Any(), extID).Return(&extension.Extension{}, nil)
				return h.ExtensionGet(ctx, extID)
			},
			&extension.Extension{},
		},
		{
			"extension by endpoint id",
			func(h *handler, c *cachehandler.MockCacheHandler) (any, error) {
				c.EXPECT().ExtensionGetByEndpointID(gomock.Any(), "ep1").Return(&extension.Extension{}, nil)
				return h.ExtensionGetByEndpointID(ctx, "ep1")
			},
			&extension.Extension{},
		},
		{
			"extension by extension",
			func(h *handler, c *cachehandler.MockCacheHandler) (any, error) {
				c.EXPECT().ExtensionGetByCustomerIDANDExtension(gomock.Any(), customerID, "1001").Return(&extension.Extension{}, nil)
				return h.ExtensionGetByExtension(ctx, customerID, "1001")
			},
			&extension.Extension{},
		},
		{
			"trunk",
			func(h *handler, c *cachehandler.MockCacheHandler) (any, error) {
				c.EXPECT().TrunkGet(gomock.Any(), extID).Return(&trunk.Trunk{}, nil)
				return h.TrunkGet(ctx, extID)
			},
			&trunk.Trunk{},
		},
		{
			"trunk by domain name",
			func(h *handler, c *cachehandler.MockCacheHandler) (any, error) {
				c.EXPECT().TrunkGetByDomainName(gomock.Any(), "t1").Return(&trunk.Trunk{}, nil)
				return h.TrunkGetByDomainName(ctx, "t1")
			},
			&trunk.Trunk{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockCache, _ := newErrorPathHandler(t)

			res, err := tt.fn(h, mockCache)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if res == nil {
				t.Fatalf("expected the cached value, got nil")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unexpected db interaction: %v", err)
			}
		})
	}
}

// Test_CachedDeletedExtension_FallsBackToDB verifies that a soft-deleted cached
// extension is not served and the DB is consulted instead.
func Test_CachedDeletedExtension_FallsBackToDB(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("6c2d8f4b-3a5e-11f1-8b7c-1a2e3d4c5b60")
	deleted := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name  string
		setup func(c *cachehandler.MockCacheHandler)
		fn    func(h *handler) error
	}{
		{
			"by endpoint id",
			func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().ExtensionGetByEndpointID(gomock.Any(), "ep1").Return(&extension.Extension{TMDelete: &deleted}, nil)
			},
			func(h *handler) error {
				_, err := h.ExtensionGetByEndpointID(ctx, "ep1")
				return err
			},
		},
		{
			"by extension",
			func(c *cachehandler.MockCacheHandler) {
				c.EXPECT().ExtensionGetByCustomerIDANDExtension(gomock.Any(), customerID, "1001").Return(&extension.Extension{TMDelete: &deleted}, nil)
			},
			func(h *handler) error {
				_, err := h.ExtensionGetByExtension(ctx, customerID, "1001")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockCache, _ := newErrorPathHandler(t)
			tt.setup(mockCache)

			// the DB has no live row: the soft-deleted cache entry must not be served.
			mock.ExpectQuery("(?is)select.*from.*registrar_extensions").WillReturnRows(sqlmock.NewRows([]string{"id"}))

			err := tt.fn(h)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("expected ErrNotFound, got: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("the db must be queried: %v", err)
			}
		})
	}
}

// Test_QueryFailure_Propagates verifies that a failing query or exec is returned
// to the caller with the cause preserved.
func Test_QueryFailure_Propagates(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5b1c7e3a-2f4d-11f1-9a6b-0f1d2c3b4a59")
	customerID := uuid.FromStringOrNil("6c2d8f4b-3a5e-11f1-8b7c-1a2e3d4c5b60")
	miss := errors.New("cache miss")

	tests := []struct {
		name  string
		query bool // true: QueryContext fails, false: ExecContext fails
		table string
		setup func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler)
		fn    func(h *handler) error
	}{
		{
			"aor get", true, "ps_aors",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().AstAORGet(gomock.Any(), "a").Return(nil, miss)
			},
			func(h *handler) error { _, err := h.AstAORGet(ctx, "a"); return err },
		},
		{
			"contacts by endpoint", true, "ps_contacts",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().AstContactsGet(gomock.Any(), "e").Return(nil, miss)
			},
			func(h *handler) error { _, err := h.AstContactGetsByEndpoint(ctx, "e"); return err },
		},
		{
			"contact delete by endpoint", false, "ps_contacts",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {},
			func(h *handler) error { return h.AstContactDeleteByEndpoint(ctx, "e") },
		},
		{
			"extension get", true, "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().ExtensionGet(gomock.Any(), id).Return(nil, miss)
			},
			func(h *handler) error { _, err := h.ExtensionGet(ctx, id); return err },
		},
		{
			"extension get by endpoint id", true, "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().ExtensionGetByEndpointID(gomock.Any(), "e").Return(nil, miss)
			},
			func(h *handler) error { _, err := h.ExtensionGetByEndpointID(ctx, "e"); return err },
		},
		{
			"extension get by extension", true, "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().ExtensionGetByCustomerIDANDExtension(gomock.Any(), customerID, "1001").Return(nil, miss)
			},
			func(h *handler) error { _, err := h.ExtensionGetByExtension(ctx, customerID, "1001"); return err },
		},
		{
			"extension count", true, "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {},
			func(h *handler) error { _, err := h.ExtensionCountByCustomerID(ctx, customerID); return err },
		},
		{
			"extension create", false, "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().TimeNow().Return(nil)
			},
			func(h *handler) error {
				return h.ExtensionCreate(ctx, &extension.Extension{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}})
			},
		},
		{
			"extension delete", false, "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().TimeNow().Return(nil)
			},
			func(h *handler) error { return h.ExtensionDelete(ctx, id) },
		},
		{
			"extension list", true, "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {},
			func(h *handler) error {
				_, err := h.ExtensionList(ctx, 10, "2026-01-01T00:00:00.000000Z", nil)
				return err
			},
		},
		{
			"trunk get", true, "registrar_trunks",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().TrunkGet(gomock.Any(), id).Return(nil, miss)
			},
			func(h *handler) error { _, err := h.TrunkGet(ctx, id); return err },
		},
		{
			"trunk get by domain name", true, "registrar_trunks",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().TrunkGetByDomainName(gomock.Any(), "t").Return(nil, miss)
				u.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
			},
			func(h *handler) error { _, err := h.TrunkGetByDomainName(ctx, "t"); return err },
		},
		{
			"trunk count", true, "registrar_trunks",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {},
			func(h *handler) error { _, err := h.TrunkCountByCustomerID(ctx, customerID); return err },
		},
		{
			"trunk create", false, "registrar_trunks",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().TimeNow().Return(nil)
			},
			func(h *handler) error {
				return h.TrunkCreate(ctx, &trunk.Trunk{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}})
			},
		},
		{
			"trunk list", true, "registrar_trunks",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {},
			func(h *handler) error {
				_, err := h.TrunkList(ctx, 10, "2026-01-01T00:00:00.000000Z", nil)
				return err
			},
		},
		{
			"sipauth get", true, "registrar_sip_auths",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {},
			func(h *handler) error { _, err := h.SIPAuthGet(ctx, id); return err },
		},
		{
			"sipauth delete", false, "registrar_sip_auths",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {},
			func(h *handler) error { return h.SIPAuthDelete(ctx, id) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockCache, mockUtil := newErrorPathHandler(t)
			tt.setup(mockCache, mockUtil)

			if tt.query {
				mock.ExpectQuery("(?is)" + tt.table).WillReturnError(errInjectedDB)
			} else {
				mock.ExpectExec("(?is)" + tt.table).WillReturnError(errInjectedDB)
			}

			err := tt.fn(h)
			if !errors.Is(err, errInjectedDB) {
				t.Errorf("expected the db failure to be returned, got: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// Test_ScanFailure_Propagates verifies that a row that cannot be scanned is returned as an error.
func Test_ScanFailure_Propagates(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5b1c7e3a-2f4d-11f1-9a6b-0f1d2c3b4a59")
	customerID := uuid.FromStringOrNil("6c2d8f4b-3a5e-11f1-8b7c-1a2e3d4c5b60")
	miss := errors.New("cache miss")

	tests := []struct {
		name  string
		table string
		setup func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler)
		fn    func(h *handler) error
	}{
		{"aor", "ps_aors",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().AstAORGet(gomock.Any(), "a").Return(nil, miss)
			},
			func(h *handler) error { _, err := h.AstAORGet(ctx, "a"); return err }},
		{"auth", "ps_auths",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().AstAuthGet(gomock.Any(), "a").Return(nil, miss)
			},
			func(h *handler) error { _, err := h.AstAuthGet(ctx, "a"); return err }},
		{"endpoint", "ps_endpoints",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().AstEndpointGet(gomock.Any(), "a").Return(nil, miss)
			},
			func(h *handler) error { _, err := h.AstEndpointGet(ctx, "a"); return err }},
		{"contacts", "ps_contacts",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().AstContactsGet(gomock.Any(), "a").Return(nil, miss)
			},
			func(h *handler) error { _, err := h.AstContactGetsByEndpoint(ctx, "a"); return err }},
		{"extension", "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().ExtensionGet(gomock.Any(), id).Return(nil, miss)
			},
			func(h *handler) error { _, err := h.ExtensionGet(ctx, id); return err }},
		{"extension by endpoint id", "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().ExtensionGetByEndpointID(gomock.Any(), "a").Return(nil, miss)
			},
			func(h *handler) error { _, err := h.ExtensionGetByEndpointID(ctx, "a"); return err }},
		{"extension by extension", "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().ExtensionGetByCustomerIDANDExtension(gomock.Any(), customerID, "1001").Return(nil, miss)
			},
			func(h *handler) error { _, err := h.ExtensionGetByExtension(ctx, customerID, "1001"); return err }},
		{"extension list", "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {},
			func(h *handler) error {
				_, err := h.ExtensionList(ctx, 10, "2026-01-01T00:00:00.000000Z", nil)
				return err
			}},
		{"trunk", "registrar_trunks",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().TrunkGet(gomock.Any(), id).Return(nil, miss)
			},
			func(h *handler) error { _, err := h.TrunkGet(ctx, id); return err }},
		{"trunk by domain name", "registrar_trunks",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().TrunkGetByDomainName(gomock.Any(), "t").Return(nil, miss)
				u.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
			},
			func(h *handler) error { _, err := h.TrunkGetByDomainName(ctx, "t"); return err }},
		{"trunk list", "registrar_trunks",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {},
			func(h *handler) error {
				_, err := h.TrunkList(ctx, 10, "2026-01-01T00:00:00.000000Z", nil)
				return err
			}},
		{"sipauth", "registrar_sip_auths",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {},
			func(h *handler) error { _, err := h.SIPAuthGet(ctx, id); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockCache, mockUtil := newErrorPathHandler(t)
			tt.setup(mockCache, mockUtil)

			// one column only: scanning into the model fails on the column count.
			mock.ExpectQuery("(?is)" + tt.table).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("x"))

			err := tt.fn(h)
			if err == nil {
				t.Fatalf("expected a scan error, got nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("a scan failure must not be reported as ErrNotFound: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// Test_NotFound_ReturnsErrNotFound verifies that an empty result is reported as ErrNotFound.
func Test_NotFound_ReturnsErrNotFound(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5b1c7e3a-2f4d-11f1-9a6b-0f1d2c3b4a59")
	customerID := uuid.FromStringOrNil("6c2d8f4b-3a5e-11f1-8b7c-1a2e3d4c5b60")
	miss := errors.New("cache miss")

	tests := []struct {
		name  string
		table string
		setup func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler)
		fn    func(h *handler) error
	}{
		{"extension get", "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().ExtensionGet(gomock.Any(), id).Return(nil, miss)
			},
			func(h *handler) error { _, err := h.ExtensionGet(ctx, id); return err }},
		{"extension by endpoint id", "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().ExtensionGetByEndpointID(gomock.Any(), "a").Return(nil, miss)
			},
			func(h *handler) error { _, err := h.ExtensionGetByEndpointID(ctx, "a"); return err }},
		{"extension by extension", "registrar_extensions",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().ExtensionGetByCustomerIDANDExtension(gomock.Any(), customerID, "1001").Return(nil, miss)
			},
			func(h *handler) error { _, err := h.ExtensionGetByExtension(ctx, customerID, "1001"); return err }},
		{"trunk get", "registrar_trunks",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().TrunkGet(gomock.Any(), id).Return(nil, miss)
			},
			func(h *handler) error { _, err := h.TrunkGet(ctx, id); return err }},
		{"trunk by domain name", "registrar_trunks",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().TrunkGetByDomainName(gomock.Any(), "t").Return(nil, miss)
				u.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
			},
			func(h *handler) error { _, err := h.TrunkGetByDomainName(ctx, "t"); return err }},
		{"sipauth get", "registrar_sip_auths",
			func(c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {},
			func(h *handler) error { _, err := h.SIPAuthGet(ctx, id); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockCache, mockUtil := newErrorPathHandler(t)
			tt.setup(mockCache, mockUtil)
			mock.ExpectQuery("(?is)" + tt.table).WillReturnRows(sqlmock.NewRows([]string{"id"}))

			err := tt.fn(h)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("expected ErrNotFound, got: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// Test_AstContactDeleteByEndpoint_CacheFailure verifies that a failing cache
// invalidation is reported after the rows were deleted.
func Test_AstContactDeleteByEndpoint_CacheFailure(t *testing.T) {
	h, mock, mockCache, _ := newErrorPathHandler(t)

	mock.ExpectExec("(?is)delete from ps_contacts").WithArgs("ep1").WillReturnResult(sqlmock.NewResult(0, 2))
	mockCache.EXPECT().AstContactsDel(gomock.Any(), "ep1").Return(errInjectedCache)

	err := h.AstContactDeleteByEndpoint(context.Background(), "ep1")
	if !errors.Is(err, errInjectedCache) {
		t.Errorf("expected the cache failure, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

// Test_TrunkDeleteFromCache verifies the trunk cache invalidation helper.
func Test_TrunkDeleteFromCache(t *testing.T) {
	id := uuid.FromStringOrNil("5b1c7e3a-2f4d-11f1-9a6b-0f1d2c3b4a59")

	tests := []struct {
		name    string
		cacheOK bool
	}{
		{"deleted", true},
		{"cache failure is returned", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _, mockCache, _ := newErrorPathHandler(t)

			var cacheErr error
			if !tt.cacheOK {
				cacheErr = errInjectedCache
			}
			mockCache.EXPECT().TrunkDel(gomock.Any(), id, "t1").Return(cacheErr)

			err := h.trunkDeleteFromCache(context.Background(), id, "t1")
			if tt.cacheOK && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if !tt.cacheOK && !errors.Is(err, errInjectedCache) {
				t.Errorf("expected the cache failure, got: %v", err)
			}
		})
	}
}
