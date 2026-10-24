package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-registrar-manager/models/astaor"
	"monorepo/bin-registrar-manager/models/astauth"
	"monorepo/bin-registrar-manager/models/astendpoint"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

var errRowsBroken = errors.New("simulated failure while reading rows")

// Test_RowsErr_NotMisreported verifies that a failure while iterating the rows
// is returned as an error: a single-row get must not report it as ErrNotFound.
func Test_RowsErr_NotMisreported(t *testing.T) {
	missErr := errors.New("cache miss")

	tests := []struct {
		name  string
		table string
		fn    func(h *handler, mockCache *cachehandler.MockCacheHandler) error
	}{
		{"ps_aors get", "ps_aors", func(h *handler, _ *cachehandler.MockCacheHandler) error {
			_, err := h.astAORGetFromDB(context.Background(), "id1")
			return err
		}},
		{"ps_auths get", "ps_auths", func(h *handler, _ *cachehandler.MockCacheHandler) error {
			_, err := h.astAuthGetFromDB(context.Background(), "id1")
			return err
		}},
		{"ps_endpoints get", "ps_endpoints", func(h *handler, _ *cachehandler.MockCacheHandler) error {
			_, err := h.astEndpointGetFromDB(context.Background(), "id1")
			return err
		}},
		{"extension get by endpoint id", "registrar_extensions", func(h *handler, mockCache *cachehandler.MockCacheHandler) error {
			mockCache.EXPECT().ExtensionGetByEndpointID(gomock.Any(), "ep1").Return(nil, missErr)
			_, err := h.ExtensionGetByEndpointID(context.Background(), "ep1")
			return err
		}},
		{"extension get by extension", "registrar_extensions", func(h *handler, mockCache *cachehandler.MockCacheHandler) error {
			customerID := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")
			mockCache.EXPECT().ExtensionGetByCustomerIDANDExtension(gomock.Any(), customerID, "1001").Return(nil, missErr)
			_, err := h.ExtensionGetByExtension(context.Background(), customerID, "1001")
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("could not create sqlmock: %v", err)
			}
			defer func() { _ = db.Close() }()

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{db: db, cache: mockCache, utilHandler: utilhandler.NewMockUtilHandler(mc)}

			// the first row fails when it is read, so Next returns false and
			// the cause is only reported by rows.Err().
			rows := sqlmock.NewRows([]string{"id"}).AddRow("id1").RowError(0, errRowsBroken)
			mock.ExpectQuery("(?is)select.*from.*" + tt.table).WillReturnRows(rows)

			err = tt.fn(h, mockCache)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Fatalf("an iteration failure must not be reported as ErrNotFound. err: %v", err)
			}
			if !errors.Is(err, errRowsBroken) {
				t.Errorf("expected the iteration failure in the error, got: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// Test_AstContactGetsByEndpoint_RowsErr verifies that a failure while iterating
// the contacts is returned as an error and the partial list is not cached.
func Test_AstContactGetsByEndpoint_RowsErr(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{db: db, cache: mockCache, utilHandler: utilhandler.NewMockUtilHandler(mc)}

	endpoint := "1001@realm.registrar.voipbin.net"
	mockCache.EXPECT().AstContactsGet(gomock.Any(), endpoint).Return(nil, errors.New("cache miss"))
	// AstContactsSet is intentionally not expected: gomock fails on a call.

	// the first row is read, the second one fails.
	rows := sqlmock.NewRows([]string{"id", "uri", "expiration_time", "qualify_frequency", "outbound_proxy", "path", "user_agent", "qualify_timeout", "reg_server", "authenticate_qualify", "via_addr", "via_port", "call_id", "endpoint", "prune_on_boot"}).
		AddRow("c1", "sip:a@1.2.3.4", 1, 0, "", "", "ua", 3.0, "srv", "no", "1.2.3.4", 5060, "cid1", endpoint, "no").
		AddRow("c2", "sip:b@1.2.3.4", 1, 0, "", "", "ua", 3.0, "srv", "no", "1.2.3.4", 5060, "cid2", endpoint, "no").
		RowError(1, errRowsBroken)
	mock.ExpectQuery("(?is)select.*from.*ps_contacts").WillReturnRows(rows)

	res, err := h.AstContactGetsByEndpoint(context.Background(), endpoint)
	if err == nil {
		t.Fatalf("expected an error, got the partial list: %v", res)
	}
	if !errors.Is(err, errRowsBroken) {
		t.Errorf("expected the iteration failure in the error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func countByID(t *testing.T, table string, id string) int {
	t.Helper()

	var count int
	if err := dbTest.QueryRow("select count(*) from "+table+" where id = ?", id).Scan(&count); err != nil {
		t.Fatalf("could not count the rows. err: %v", err)
	}
	return count
}

// Test_ContextCancel_QueryAborted verifies that the request context reaches the
// database driver for every converted call site: a cancelled context aborts the
// query (read, write and list paths) instead of the query running to completion.
func Test_ContextCancel_QueryAborted(t *testing.T) {
	resetTestTables(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		utilHandler: utilhandler.NewMockUtilHandler(mc),
		db:          dbTest,
		cache:       mockCache,
	}

	// the cache is only reached when a call ignores the cancelled context.
	missErr := errors.New("cache miss")
	mockCache.EXPECT().AstContactsGet(gomock.Any(), gomock.Any()).Return(nil, missErr).AnyTimes()
	mockCache.EXPECT().AstContactsSet(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().AstAORSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().AstAORDel(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().AstAuthSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().AstAuthDel(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().AstEndpointSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockCache.EXPECT().AstEndpointDel(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	insert := func(t *testing.T, q string, args ...any) {
		t.Helper()
		if _, err := dbTest.Exec(q, args...); err != nil {
			t.Fatalf("could not insert the fixture. err: %v", err)
		}
	}

	str := func(v string) *string { return &v }

	tests := []struct {
		name string
		// setup prepares the rows and returns a check run after the call.
		setup func(t *testing.T) func(t *testing.T)
		fn    func() error
	}{
		{"query ps_aors", nil, func() error { _, err := h.astAORGetFromDB(ctx, "cx-aor-q"); return err }},
		{"query ps_auths", nil, func() error { _, err := h.astAuthGetFromDB(ctx, "cx-auth-q"); return err }},
		{"query ps_endpoints", nil, func() error { _, err := h.astEndpointGetFromDB(ctx, "cx-ep-q"); return err }},
		{"query ps_contacts", nil, func() error { _, err := h.AstContactGetsByEndpoint(ctx, "cx-contact@realm"); return err }},
		{
			"exec create ps_aors",
			func(t *testing.T) func(t *testing.T) {
				return func(t *testing.T) {
					if c := countByID(t, "ps_aors", "cx-aor-c"); c != 0 {
						t.Errorf("the row must not be created by a cancelled call. count: %d", c)
					}
				}
			},
			func() error { return h.AstAORCreate(ctx, &astaor.AstAOR{ID: str("cx-aor-c")}) },
		},
		{
			"exec delete ps_aors",
			func(t *testing.T) func(t *testing.T) {
				insert(t, "insert into ps_aors(id) values('cx-aor-d')")
				return func(t *testing.T) {
					if c := countByID(t, "ps_aors", "cx-aor-d"); c != 1 {
						t.Errorf("the row must not be deleted by a cancelled call. count: %d", c)
					}
				}
			},
			func() error { return h.AstAORDelete(ctx, "cx-aor-d") },
		},
		{
			"exec create ps_auths",
			func(t *testing.T) func(t *testing.T) {
				return func(t *testing.T) {
					if c := countByID(t, "ps_auths", "cx-auth-c"); c != 0 {
						t.Errorf("the row must not be created by a cancelled call. count: %d", c)
					}
				}
			},
			func() error { return h.AstAuthCreate(ctx, &astauth.AstAuth{ID: str("cx-auth-c")}) },
		},
		{
			"exec delete ps_auths",
			func(t *testing.T) func(t *testing.T) {
				insert(t, "insert into ps_auths(id) values('cx-auth-d')")
				return func(t *testing.T) {
					if c := countByID(t, "ps_auths", "cx-auth-d"); c != 1 {
						t.Errorf("the row must not be deleted by a cancelled call. count: %d", c)
					}
				}
			},
			func() error { return h.AstAuthDelete(ctx, "cx-auth-d") },
		},
		{
			"exec update ps_auths",
			func(t *testing.T) func(t *testing.T) {
				insert(t, "insert into ps_auths(id, password) values('cx-auth-u', 'old')")
				return func(t *testing.T) {
					var pw string
					if err := dbTest.QueryRow("select password from ps_auths where id = 'cx-auth-u'").Scan(&pw); err != nil {
						t.Fatalf("could not read the password. err: %v", err)
					}
					if pw != "old" {
						t.Errorf("the row must not be updated by a cancelled call. password: %s", pw)
					}
				}
			},
			func() error {
				return h.AstAuthUpdate(ctx, &astauth.AstAuth{ID: str("cx-auth-u"), Password: str("new")})
			},
		},
		{
			"exec create ps_endpoints",
			func(t *testing.T) func(t *testing.T) {
				return func(t *testing.T) {
					if c := countByID(t, "ps_endpoints", "cx-ep-c"); c != 0 {
						t.Errorf("the row must not be created by a cancelled call. count: %d", c)
					}
				}
			},
			func() error { return h.AstEndpointCreate(ctx, &astendpoint.AstEndpoint{ID: str("cx-ep-c")}) },
		},
		{
			"exec delete ps_endpoints",
			func(t *testing.T) func(t *testing.T) {
				insert(t, "insert into ps_endpoints(id) values('cx-ep-d')")
				return func(t *testing.T) {
					if c := countByID(t, "ps_endpoints", "cx-ep-d"); c != 1 {
						t.Errorf("the row must not be deleted by a cancelled call. count: %d", c)
					}
				}
			},
			func() error { return h.AstEndpointDelete(ctx, "cx-ep-d") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var check func(t *testing.T)
			if tt.setup != nil {
				check = tt.setup(t)
			}

			err := tt.fn()
			if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
				t.Errorf("expected a context canceled error, got: %v", err)
			}

			if check != nil {
				check(t)
			}
		})
	}
}
