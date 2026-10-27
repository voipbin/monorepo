package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"

	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/trunk"
)

const caseListToken = "2026-01-01T00:00:00.000000Z"

// Test_customerDomainCacheHelpers_returnCacheError verifies that the cache helpers
// return the cache failure to the caller (their callers decide to ignore it).
func Test_customerDomainCacheHelpers_returnCacheError(t *testing.T) {
	ctx := context.Background()
	cd := &customerdomain.CustomerDomain{CustomerID: uuid.FromStringOrNil("7d3e9a5c-4b6f-11f1-9c8d-2b3f4e5d6c71"), Realm: "r1"}

	t.Run("set", func(t *testing.T) {
		h, _, mockCache, _ := newErrorPathHandler(t)
		mockCache.EXPECT().CustomerDomainSet(ctx, cd).Return(errInjectedCache)

		if err := h.customerDomainSetToCache(ctx, cd); !errors.Is(err, errInjectedCache) {
			t.Errorf("Wrong match. expect: %v, got: %v", errInjectedCache, err)
		}
	})

	t.Run("delete by realm", func(t *testing.T) {
		h, _, mockCache, _ := newErrorPathHandler(t)
		mockCache.EXPECT().CustomerDomainDelByRealm(ctx, "r1").Return(errInjectedCache)

		if err := h.customerDomainDeleteFromCacheByRealm(ctx, "r1"); !errors.Is(err, errInjectedCache) {
			t.Errorf("Wrong match. expect: %v, got: %v", errInjectedCache, err)
		}
	})
}

// Test_CustomerDomainUpdate_genericExecFailure verifies that an exec failure that is not a
// duplicate entry is returned as it is and is not reported as ErrDuplicate.
func Test_CustomerDomainUpdate_genericExecFailure(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("8e4fab6d-5c70-11f1-ad9e-3c4a5f6e7d82")

	h, mock, _, mockUtil := newErrorPathHandler(t)

	mock.ExpectQuery("(?is)registrar_customer_domains").WillReturnRows(
		sqlmock.NewRows([]string{"customer_id", "domain_label", "realm", "tm_create", "tm_update"}).
			AddRow(customerID.Bytes(), "ab12", "ab12.reg.example.test", nil, nil))
	mockUtil.EXPECT().TimeNow().Return(nil)
	mock.ExpectExec("(?is)update registrar_customer_domains").WillReturnError(errInjectedDB)

	err := h.CustomerDomainUpdate(ctx, customerID, map[customerdomain.Field]any{customerdomain.FieldRealm: "r2"})
	if !errors.Is(err, errInjectedDB) {
		t.Errorf("Wrong match. expect: %v, got: %v", errInjectedDB, err)
	}
	if errors.Is(err, ErrDuplicate) {
		t.Errorf("Wrong match. a generic failure must not be reported as ErrDuplicate: %v", err)
	}
	if errMet := mock.ExpectationsWereMet(); errMet != nil {
		t.Errorf("unfulfilled expectations: %v", errMet)
	}
}

// Test_getByID_rowIterationError verifies that a failure while reading the first row is
// reported as an error and not as ErrNotFound.
func Test_getByID_rowIterationError(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("9f50bc7e-6d81-11f1-be0f-4d5b6a7f8e93")

	tests := []struct {
		name  string
		table string
		fn    func(h *handler) error
	}{
		{"extension", "registrar_extensions", func(h *handler) error { _, err := h.extensionGetFromDB(ctx, id); return err }},
		{"trunk", "registrar_trunks", func(h *handler) error { _, err := h.trunkGetFromDB(ctx, id); return err }},
		{"sipauth", "registrar_sip_auths", func(h *handler) error { _, err := h.SIPAuthGet(ctx, id); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _, _ := newErrorPathHandler(t)

			mock.ExpectQuery("(?is)select.*from.*" + tt.table).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("x").RowError(0, errRowsBroken))

			err := tt.fn(h)
			if !errors.Is(err, errRowsBroken) {
				t.Errorf("Wrong match. expect: %v, got: %v", errRowsBroken, err)
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong match. an iteration failure must not be reported as ErrNotFound: %v", err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("unfulfilled expectations: %v", errMet)
			}
		})
	}
}

// Test_list_rowsIterationError verifies that a failure while iterating the rows of a list
// is returned to the caller; the failure is only visible through rows.Err().
func Test_list_rowsIterationError(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		table string
		fn    func(h *handler) error
	}{
		{"extension", "registrar_extensions", func(h *handler) error {
			_, err := h.ExtensionList(ctx, 10, caseListToken, nil)
			return err
		}},
		{"trunk", "registrar_trunks", func(h *handler) error {
			_, err := h.TrunkList(ctx, 10, caseListToken, nil)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _, _ := newErrorPathHandler(t)

			// the single row fails when it is read, so no row is scanned
			mock.ExpectQuery("(?is)select.*from.*" + tt.table).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("x").RowError(0, errRowsBroken))

			err := tt.fn(h)
			if !errors.Is(err, errRowsBroken) {
				t.Errorf("Wrong match. expect: %v, got: %v", errRowsBroken, err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("unfulfilled expectations: %v", errMet)
			}
		})
	}
}

// Test_ExtensionList_emptyTokenUsesCurrentTime verifies that an empty page token is
// replaced by the current time of the util handler.
func Test_ExtensionList_emptyTokenUsesCurrentTime(t *testing.T) {
	ctx := context.Background()
	h, mock, _, mockUtil := newErrorPathHandler(t)

	const now = "2026-02-03T04:05:06.000000Z"
	customerID := uuid.FromStringOrNil("a061cd8f-7e92-11f1-8f10-5e6c7b8a9fa4")
	mockUtil.EXPECT().TimeGetCurTime().Return(now)
	mock.ExpectQuery("(?is)select.*from registrar_extensions.*tm_create < \\?").
		WithArgs(now, customerID.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// the filter limits the result to one customer
	res, err := h.ExtensionList(ctx, 10, "", map[extension.Field]any{extension.FieldCustomerID: customerID})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Wrong match. expect: no result, got: %d", len(res))
	}
	if errMet := mock.ExpectationsWereMet(); errMet != nil {
		t.Errorf("unfulfilled expectations: %v", errMet)
	}
}

// Test_list_unsupportedFilterType verifies that a filter value of an unsupported type is
// rejected before any query is sent (sqlmock has no expectations).
func Test_list_unsupportedFilterType(t *testing.T) {
	ctx := context.Background()
	unsupported := []string{"a", "b"}

	tests := []struct {
		name string
		fn   func(h *handler) error
	}{
		{"extension", func(h *handler) error {
			_, err := h.ExtensionList(ctx, 10, caseListToken, map[extension.Field]any{extension.FieldName: unsupported})
			return err
		}},
		{"trunk", func(h *handler) error {
			_, err := h.TrunkList(ctx, 10, caseListToken, map[trunk.Field]any{trunk.FieldName: unsupported})
			return err
		}},
		{"customer domain", func(h *handler) error {
			_, err := h.CustomerDomainList(ctx, 10, caseListToken, map[customerdomain.Field]any{customerdomain.FieldRealm: unsupported})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _, _ := newErrorPathHandler(t)

			err := tt.fn(h)
			if err == nil || !strings.Contains(err.Error(), "could not apply filters") || !strings.Contains(err.Error(), "unsupported filter type") {
				t.Errorf("Wrong match. expect the filter rejection, got: %v", err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("unfulfilled expectations: %v", errMet)
			}
		})
	}
}
