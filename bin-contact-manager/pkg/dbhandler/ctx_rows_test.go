package dbhandler

// Tests for VOIP-1615: every DB call receives the request context, and a
// row-iteration error is returned instead of being mistaken for "no row"
// (ErrNotFound, or a normal (nil, nil) / (false, nil) result).
//
// The cancelled-context tests deliberately never call ExpectationsWereMet:
// a correct implementation stops inside database/sql before the driver is
// reached, so the expectation stays unconsumed.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	mysql_driver "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

var (
	errCtxRowsBoom = errors.New("injected row iteration error")

	ctxRowsCustomerID = uuid.FromStringOrNil("c7000000-0000-0000-0000-000000000001")
	ctxRowsContactID  = uuid.FromStringOrNil("c7000000-0000-0000-0000-000000000002")
	ctxRowsID         = uuid.FromStringOrNil("c7000000-0000-0000-0000-000000000003")
)

func newCtxRowsHandler(t *testing.T) (*handler, sqlmock.Sqlmock, *cachehandler.MockCacheHandler) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mc := gomock.NewController(t)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	curTime := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&curTime).AnyTimes()

	return &handler{utilHandler: mockUtil, db: db, cache: mockCache}, mock, mockCache
}

// rowErrRows returns a result set whose first row iteration fails, so
// rows.Next() is false and rows.Err() is err.
func rowErrRows(err error) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"c"}).AddRow([]byte("x")).RowError(0, err)
}

func mysqlDeadlock() error {
	return &mysql_driver.MySQLError{Number: mysqlDeadlockErrno, Message: "Deadlock found"}
}

func assertRowsErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected the row iteration error, got nil")
	}
	if errors.Is(err, ErrNotFound) {
		t.Errorf("row iteration error must not be reported as ErrNotFound. err: %v", err)
	}
	if !errors.Is(err, errCtxRowsBoom) {
		t.Errorf("row iteration error must be preserved with %%w. err: %v", err)
	}
}

func Test_SingleGetter_RowsErr(t *testing.T) {
	tests := []struct {
		name  string
		setup func(mock sqlmock.Sqlmock, mockCache *cachehandler.MockCacheHandler)
		call  func(h *handler, ctx context.Context) error
	}{
		{
			name: "AddressGet",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCtxRowsBoom))
			},
			call: func(h *handler, ctx context.Context) error {
				_, err := h.AddressGet(ctx, ctxRowsCustomerID, ctxRowsID)
				return err
			},
		},
		{
			// addressTypeTargetContactByID (pre-lock read of AddressUpdate)
			name: "addressTypeTargetContactByID",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCtxRowsBoom))
			},
			call: func(h *handler, ctx context.Context) error {
				return h.AddressUpdate(ctx, ctxRowsID, map[string]any{"is_primary": false})
			},
		},
		{
			// addressCustomerID (second pre-lock read of AddressUpdate)
			name: "addressCustomerID",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectQuery("SELECT type, target, contact_id").WillReturnRows(
					sqlmock.NewRows([]string{"type", "target", "contact_id"}).AddRow(string(commonaddress.TypeTel), "+155****7001", ctxRowsContactID.Bytes()))
				mock.ExpectQuery("SELECT customer_id").WillReturnRows(rowErrRows(errCtxRowsBoom))
			},
			call: func(h *handler, ctx context.Context) error {
				return h.AddressUpdate(ctx, ctxRowsID, map[string]any{"is_primary": false})
			},
		},
		{
			name: "contactGetFromDB",
			setup: func(mock sqlmock.Sqlmock, mockCache *cachehandler.MockCacheHandler) {
				mockCache.EXPECT().ContactGet(gomock.Any(), ctxRowsContactID).Return(nil, fmt.Errorf("miss"))
				mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCtxRowsBoom))
			},
			call: func(h *handler, ctx context.Context) error {
				_, err := h.ContactGet(ctx, ctxRowsContactID)
				return err
			},
		},
		{
			name: "AddressLookupContactIDByTypeTarget",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCtxRowsBoom))
			},
			call: func(h *handler, ctx context.Context) error {
				_, err := h.AddressLookupContactIDByTypeTarget(ctx, ctxRowsCustomerID, commonaddress.TypeTel, "+155****7001")
				return err
			},
		},
		{
			name: "ContactLookupByPhone",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCtxRowsBoom))
			},
			call: func(h *handler, ctx context.Context) error {
				_, err := h.ContactLookupByPhone(ctx, ctxRowsCustomerID, "+155****7001")
				return err
			},
		},
		{
			name: "ContactLookupByEmail",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCtxRowsBoom))
			},
			call: func(h *handler, ctx context.Context) error {
				_, err := h.ContactLookupByEmail(ctx, ctxRowsCustomerID, "a@example.com")
				return err
			},
		},
		{
			name: "caseGetByIDExec",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCtxRowsBoom))
			},
			call: func(h *handler, ctx context.Context) error {
				_, err := h.CaseGetByID(ctx, ctxRowsID)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockCache := newCtxRowsHandler(t)
			tt.setup(mock, mockCache)

			assertRowsErr(t, tt.call(h, context.Background()))
		})
	}
}

func Test_TxSingleGetter_RowsErr(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		call func(h *handler, tx *sql.Tx) error
	}{
		{
			name: "ownershipAgreementHoldsTx",
			call: func(h *handler, tx *sql.Tx) error {
				_, err := h.ownershipAgreementHoldsTx(ctx, tx, ctxRowsCustomerID, ctxRowsContactID, commonaddress.TypeTel, "+155****7001")
				return err
			},
		},
		{
			name: "staleRowByTargetTx",
			call: func(_ *handler, tx *sql.Tx) error {
				_, err := staleRowByTargetTx(ctx, tx, ctxRowsCustomerID, commonaddress.TypeTel, "+155****7001")
				return err
			},
		},
		{
			name: "contactTombstoneTx",
			call: func(_ *handler, tx *sql.Tx) error {
				_, err := contactTombstoneTx(ctx, tx, ctxRowsContactID)
				return err
			},
		},
		{
			name: "latestOwnershipPeriodValidToTx",
			call: func(_ *handler, tx *sql.Tx) error {
				_, err := latestOwnershipPeriodValidToTx(ctx, tx, ctxRowsCustomerID, commonaddress.TypeTel, "+155****7001")
				return err
			},
		},
		{
			name: "CaseGetOpenByPeer",
			call: func(h *handler, tx *sql.Tx) error {
				_, err := h.CaseGetOpenByPeer(ctx, tx, ctxRowsCustomerID, commonaddress.TypeTel, "+155****7001", "call")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"/general", func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			mock.ExpectBegin()
			tx, err := h.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("could not begin: %v", err)
			}
			defer func() { _ = tx.Rollback() }()
			mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCtxRowsBoom))

			err = tt.call(h, tx)
			assertRowsErr(t, err)
			if err == ErrDeadlock {
				t.Errorf("a non-deadlock row error must not be mapped to ErrDeadlock")
			}
		})

		t.Run(tt.name+"/deadlock", func(t *testing.T) {
			h, mock, _ := newCtxRowsHandler(t)
			mock.ExpectBegin()
			tx, err := h.db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatalf("could not begin: %v", err)
			}
			defer func() { _ = tx.Rollback() }()
			mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(mysqlDeadlock()))

			// the callers compare with ==, so the bare sentinel is required.
			if err := tt.call(h, tx); err != ErrDeadlock {
				t.Errorf("a deadlock row error must be the bare ErrDeadlock sentinel. got: %v", err)
			}
		})
	}
}

func Test_TagAssignmentListByContactID_RowsErr(t *testing.T) {
	h, mock, _ := newCtxRowsHandler(t)
	mock.ExpectQuery("SELECT").WillReturnRows(rowErrRows(errCtxRowsBoom))

	res, err := h.TagAssignmentListByContactID(context.Background(), ctxRowsContactID)
	if err == nil || !errors.Is(err, errCtxRowsBoom) {
		t.Errorf("expected the row iteration error, got res: %v, err: %v", res, err)
	}
}

func Test_CancelledContext_StopsDBCalls(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	empty := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"c"}) }

	tests := []struct {
		name  string
		setup func(mock sqlmock.Sqlmock, mockCache *cachehandler.MockCacheHandler)
		call  func(h *handler) error
	}{
		{
			name: "ContactCreate exec",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
			},
			call: func(h *handler) error {
				return h.ContactCreate(cancelled, &contact.Contact{Identity: commonidentity.Identity{ID: ctxRowsContactID, CustomerID: ctxRowsCustomerID}})
			},
		},
		{
			name: "CaseInsert exec via sqlExecutor",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
			},
			call: func(h *handler) error {
				return h.CaseInsert(cancelled, &kase.Case{
					ID:            ctxRowsID,
					CustomerID:    ctxRowsCustomerID,
					Peer:          commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****7001"},
					ReferenceType: "call",
					Status:        kase.StatusOpen,
				})
			},
		},
		{
			name: "CaseNoteCreate exec",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
			},
			call: func(h *handler) error {
				return h.CaseNoteCreate(cancelled, &casenote.CaseNote{ID: ctxRowsID, CustomerID: ctxRowsCustomerID})
			},
		},
		{
			name: "ContactGet query",
			setup: func(mock sqlmock.Sqlmock, mockCache *cachehandler.MockCacheHandler) {
				mockCache.EXPECT().ContactGet(gomock.Any(), ctxRowsContactID).Return(nil, fmt.Errorf("miss"))
				mock.ExpectQuery("SELECT").WillReturnRows(empty())
			},
			call: func(h *handler) error {
				_, err := h.ContactGet(cancelled, ctxRowsContactID)
				return err
			},
		},
		{
			name: "TagAssignmentListByContactID query",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectQuery("SELECT").WillReturnRows(empty())
			},
			call: func(h *handler) error {
				_, err := h.TagAssignmentListByContactID(cancelled, ctxRowsContactID)
				return err
			},
		},
		{
			name: "AddressList query",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectQuery("SELECT").WillReturnRows(empty())
			},
			call: func(h *handler) error {
				_, err := h.AddressList(cancelled, ctxRowsCustomerID, nil, "", 10)
				return err
			},
		},
		{
			name: "AddressListByContactID query",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectQuery("SELECT").WillReturnRows(empty())
			},
			call: func(h *handler) error {
				_, err := h.AddressListByContactID(cancelled, ctxRowsContactID)
				return err
			},
		},
		{
			name: "AddressListAllByContactID query",
			setup: func(mock sqlmock.Sqlmock, _ *cachehandler.MockCacheHandler) {
				mock.ExpectQuery("SELECT").WillReturnRows(empty())
			},
			call: func(h *handler) error {
				_, err := h.AddressListAllByContactID(cancelled, ctxRowsContactID)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, mockCache := newCtxRowsHandler(t)
			tt.setup(mock, mockCache)

			// the wrapping keeps the cause as text ("%v"), so assert the message.
			err := tt.call(h)
			if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
				t.Errorf("expected a context canceled error, got: %v", err)
			}
		})
	}
}

// The cache eviction after a committed delete must not depend on the request
// context. A cancellable (not cancelled) context is passed in, and the cache
// receives a context that can never be cancelled (Done() == nil).
func Test_ContactDelete_EvictsCacheWithoutCancel(t *testing.T) {
	assertDetached := func(t *testing.T) func(ctx context.Context, id uuid.UUID) error {
		return func(ctx context.Context, _ uuid.UUID) error {
			if ctx.Done() != nil {
				t.Errorf("the cache eviction must use a context detached from the request cancellation")
			}
			return nil
		}
	}

	t.Run("ContactDelete", func(t *testing.T) {
		h, mock, mockCache := newCtxRowsHandler(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
		mockCache.EXPECT().ContactDelete(gomock.Any(), ctxRowsContactID).DoAndReturn(assertDetached(t))

		if err := h.ContactDelete(ctx, ctxRowsContactID); err != nil {
			t.Fatalf("ContactDelete() error = %v", err)
		}
	})

	t.Run("ContactDeleteByCustomerID", func(t *testing.T) {
		h, mock, mockCache := newCtxRowsHandler(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mock.ExpectQuery("SELECT id").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(ctxRowsContactID.Bytes()))
		mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 1))
		mockCache.EXPECT().ContactDelete(gomock.Any(), ctxRowsContactID).DoAndReturn(assertDetached(t))

		if err := h.ContactDeleteByCustomerID(ctx, ctxRowsCustomerID); err != nil {
			t.Fatalf("ContactDeleteByCustomerID() error = %v", err)
		}
	})
}

var ctxRowsDBSeq atomic.Int64

// newCtxRowsContactDB returns a private in-memory sqlite DB with the test
// schema and one contact, so single related-table failures can be forced by
// dropping a table afterwards.
func newCtxRowsContactDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3_test", fmt.Sprintf("file:ctxrows%d?mode=memory&cache=private", ctxRowsDBSeq.Add(1)))
	if err != nil {
		t.Fatalf("could not open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	schema, err := os.ReadFile("../../scripts/database_scripts_test/contacts.sql")
	if err != nil {
		t.Fatalf("could not read the schema: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("could not load the schema: %v", err)
	}

	mc := gomock.NewController(t)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	curTime := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&curTime).AnyTimes()
	mockCache.EXPECT().ContactSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	h := &handler{utilHandler: mockUtil, db: db, cache: mockCache}
	c := &contact.Contact{Identity: commonidentity.Identity{ID: ctxRowsContactID, CustomerID: ctxRowsCustomerID}, FirstName: "a"}
	if err := h.ContactCreate(context.Background(), c); err != nil {
		t.Fatalf("could not create the contact: %v", err)
	}

	return db
}

// A failure while loading the addresses or tags must be returned, and the
// incomplete contact must not be written to the cache.
func Test_ContactRelatedDataError_IsReturnedAndNotCached(t *testing.T) {
	ctx := context.Background()

	calls := map[string]func(h *handler) error{
		"ContactGet": func(h *handler) error {
			_, err := h.ContactGet(ctx, ctxRowsContactID)
			return err
		},
		"ContactList": func(h *handler) error {
			_, err := h.ContactList(ctx, 10, "2099-01-01 00:00:00.000000", map[contact.Field]any{})
			return err
		},
		"contactUpdateToCache": func(h *handler) error {
			return h.contactUpdateToCache(ctx, ctxRowsContactID)
		},
	}

	for _, table := range []string{"contact_addresses", "contact_tag_assignments"} {
		for name, call := range calls {
			t.Run(name+"/"+table, func(t *testing.T) {
				db := newCtxRowsContactDB(t)
				if _, err := db.Exec("DROP TABLE " + table); err != nil {
					t.Fatalf("could not drop %s: %v", table, err)
				}

				mc := gomock.NewController(t)
				mockUtil := utilhandler.NewMockUtilHandler(mc)
				mockCache := cachehandler.NewMockCacheHandler(mc)
				mockCache.EXPECT().ContactGet(gomock.Any(), ctxRowsContactID).Return(nil, fmt.Errorf("miss")).AnyTimes()
				// no ContactSet expectation: a call fails the test.
				h := &handler{utilHandler: mockUtil, db: db, cache: mockCache}

				if err := call(h); err == nil {
					t.Errorf("expected the related data error, got nil")
				}
			})
		}
	}
}
