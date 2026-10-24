package dbhandler

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/sipauth"
	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

// Test_Update_EmptyFields_NoDBAccess verifies that an update without fields is a no-op.
// sqlmock has no expectations, so any statement would fail the call.
func Test_Update_EmptyFields_NoDBAccess(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5b1c7e3a-2f4d-11f1-9a6b-0f1d2c3b4a59")

	tests := []struct {
		name string
		fn   func(h *handler) error
	}{
		{"extension", func(h *handler) error { return h.ExtensionUpdate(ctx, id, map[extension.Field]any{}) }},
		{"trunk", func(h *handler) error { return h.TrunkUpdate(ctx, id, map[trunk.Field]any{}) }},
		{"sipauth", func(h *handler) error { return h.SIPAuthUpdate(ctx, id, map[sipauth.Field]any{}) }},
		{"customer domain", func(h *handler) error { return h.CustomerDomainUpdate(ctx, id, map[customerdomain.Field]any{}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _, _ := newErrorPathHandler(t)

			if err := tt.fn(h); err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unexpected db interaction: %v", err)
			}
		})
	}
}

// Test_Update_ExecFailure verifies that a failing UPDATE is returned and the cache is not touched.
func Test_Update_ExecFailure(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("5b1c7e3a-2f4d-11f1-9a6b-0f1d2c3b4a59")

	tests := []struct {
		name  string
		table string
		fn    func(h *handler) error
	}{
		{"extension", "registrar_extensions", func(h *handler) error {
			return h.ExtensionUpdate(ctx, id, map[extension.Field]any{extension.FieldName: "n"})
		}},
		{"trunk", "registrar_trunks", func(h *handler) error {
			return h.TrunkUpdate(ctx, id, map[trunk.Field]any{trunk.FieldName: "n"})
		}},
		{"sipauth", "registrar_sip_auths", func(h *handler) error {
			return h.SIPAuthUpdate(ctx, id, map[sipauth.Field]any{sipauth.FieldUsername: "u"})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _, mockUtil := newErrorPathHandler(t)

			mockUtil.EXPECT().TimeNow().Return(nil)
			// the cache mock has no expectations: a cache call after the failed UPDATE fails the test.
			mock.ExpectExec("(?is)update " + tt.table).WillReturnError(errInjectedDB)

			err := tt.fn(h)
			if !errors.Is(err, errInjectedDB) {
				t.Errorf("expected the exec failure, got: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

// Test_CustomerDomain_DBFailures covers the failure paths of the customer domain queries.
func Test_CustomerDomain_DBFailures(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("6c2d8f4b-3a5e-11f1-8b7c-1a2e3d4c5b60")
	miss := errors.New("cache miss")
	dupErr := errors.New("Duplicate entry 'x' for key 'realm'")

	tests := []struct {
		name  string
		setup func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler)
		fn    func(h *handler) error

		wantIs  error
		wantAny bool // any non-nil error that is not ErrNotFound
	}{
		{
			name: "get query failure",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				m.ExpectQuery("(?is)registrar_customer_domains").WillReturnError(errInjectedDB)
			},
			fn:     func(h *handler) error { _, err := h.CustomerDomainGet(ctx, customerID); return err },
			wantIs: errInjectedDB,
		},
		{
			name: "get iteration failure",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				m.ExpectQuery("(?is)registrar_customer_domains").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}).AddRow("x").RowError(0, errRowsBroken))
			},
			fn:     func(h *handler) error { _, err := h.CustomerDomainGet(ctx, customerID); return err },
			wantIs: errRowsBroken,
		},
		{
			name: "get not found",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				m.ExpectQuery("(?is)registrar_customer_domains").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}))
			},
			fn:     func(h *handler) error { _, err := h.CustomerDomainGet(ctx, customerID); return err },
			wantIs: ErrNotFound,
		},
		{
			name: "get scan failure",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				m.ExpectQuery("(?is)registrar_customer_domains").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}).AddRow("x"))
			},
			fn:      func(h *handler) error { _, err := h.CustomerDomainGet(ctx, customerID); return err },
			wantAny: true,
		},
		{
			name: "get by realm query failure",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().CustomerDomainGetByRealm(gomock.Any(), "r1").Return(nil, miss)
				m.ExpectQuery("(?is)registrar_customer_domains").WillReturnError(errInjectedDB)
			},
			fn:     func(h *handler) error { _, err := h.CustomerDomainGetByRealm(ctx, "r1"); return err },
			wantIs: errInjectedDB,
		},
		{
			name: "get by realm iteration failure",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().CustomerDomainGetByRealm(gomock.Any(), "r1").Return(nil, miss)
				m.ExpectQuery("(?is)registrar_customer_domains").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}).AddRow("x").RowError(0, errRowsBroken))
			},
			fn:     func(h *handler) error { _, err := h.CustomerDomainGetByRealm(ctx, "r1"); return err },
			wantIs: errRowsBroken,
		},
		{
			name: "get by realm scan failure",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				c.EXPECT().CustomerDomainGetByRealm(gomock.Any(), "r1").Return(nil, miss)
				m.ExpectQuery("(?is)registrar_customer_domains").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}).AddRow("x"))
			},
			fn:      func(h *handler) error { _, err := h.CustomerDomainGetByRealm(ctx, "r1"); return err },
			wantAny: true,
		},
		{
			name: "list query failure",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().TimeGetCurTime().Return("2026-01-01T00:00:00.000000Z")
				m.ExpectQuery("(?is)registrar_customer_domains").WillReturnError(errInjectedDB)
			},
			fn:     func(h *handler) error { _, err := h.CustomerDomainList(ctx, 10, "", nil); return err },
			wantIs: errInjectedDB,
		},
		{
			name: "list scan failure",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				m.ExpectQuery("(?is)registrar_customer_domains").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}).AddRow("x"))
			},
			fn: func(h *handler) error {
				_, err := h.CustomerDomainList(ctx, 10, "2026-01-01T00:00:00.000000Z", nil)
				return err
			},
			wantAny: true,
		},
		{
			name: "list iteration failure",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				m.ExpectQuery("(?is)registrar_customer_domains").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}).AddRow("x").RowError(0, errRowsBroken))
			},
			fn: func(h *handler) error {
				_, err := h.CustomerDomainList(ctx, 10, "2026-01-01T00:00:00.000000Z", nil)
				return err
			},
			wantIs: errRowsBroken,
		},
		{
			name: "create exec failure",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().TimeNow().Return(nil)
				m.ExpectExec("(?is)insert into registrar_customer_domains").WillReturnError(errInjectedDB)
			},
			fn: func(h *handler) error {
				return h.CustomerDomainCreate(ctx, &customerdomain.CustomerDomain{CustomerID: customerID, Realm: "r1"})
			},
			wantIs: errInjectedDB,
		},
		{
			name: "create duplicate is reported as ErrDuplicate",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().TimeNow().Return(nil)
				m.ExpectExec("(?is)insert into registrar_customer_domains").WillReturnError(dupErr)
			},
			fn: func(h *handler) error {
				return h.CustomerDomainCreate(ctx, &customerdomain.CustomerDomain{CustomerID: customerID, Realm: "r1"})
			},
			wantIs: ErrDuplicate,
		},
		{
			name: "update fails when the current row cannot be read",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				m.ExpectQuery("(?is)registrar_customer_domains").WillReturnError(errInjectedDB)
			},
			fn: func(h *handler) error {
				return h.CustomerDomainUpdate(ctx, customerID, map[customerdomain.Field]any{customerdomain.FieldRealm: "r2"})
			},
			wantIs: errInjectedDB,
		},
		{
			name: "delete fails when the current row cannot be read",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				m.ExpectQuery("(?is)registrar_customer_domains").WillReturnError(errInjectedDB)
			},
			fn:     func(h *handler) error { return h.CustomerDomainDelete(ctx, customerID) },
			wantIs: errInjectedDB,
		},
		{
			name: "delete exec failure is returned",
			setup: func(m sqlmock.Sqlmock, c *cachehandler.MockCacheHandler, u *utilhandler.MockUtilHandler) {
				m.ExpectQuery("(?is)registrar_customer_domains").WillReturnRows(
					sqlmock.NewRows([]string{"customer_id", "domain_label", "realm", "tm_create", "tm_update"}).
						AddRow(customerID.Bytes(), "ab12", "ab12.reg.example.test", nil, nil))
				m.ExpectExec("(?is)delete from registrar_customer_domains").WillReturnError(errInjectedDB)
			},
			fn:     func(h *handler) error { return h.CustomerDomainDelete(ctx, customerID) },
			wantIs: errInjectedDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockCache, mockUtil := newErrorPathHandler(t)
			tt.setup(mock, mockCache, mockUtil)

			err := tt.fn(h)
			switch {
			case tt.wantIs != nil:
				if !errors.Is(err, tt.wantIs) {
					t.Errorf("expected %v, got: %v", tt.wantIs, err)
				}
			case tt.wantAny:
				if err == nil || errors.Is(err, ErrNotFound) {
					t.Errorf("expected a failure that is not ErrNotFound, got: %v", err)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}
