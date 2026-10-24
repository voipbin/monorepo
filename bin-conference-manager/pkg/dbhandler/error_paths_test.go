package dbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/cachehandler"
)

var errTestFailure = errors.New("simulated failure")

func newSQLMockHandler(t *testing.T) (*handler, sqlmock.Sqlmock, *cachehandler.MockCacheHandler) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(&time.Time{}).AnyTimes()
	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000").AnyTimes()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	// every cache read misses so that the database is consulted.
	mockCache.EXPECT().ConferenceGet(gomock.Any(), gomock.Any()).Return(nil, errors.New("cache miss")).AnyTimes()
	mockCache.EXPECT().ConferencecallGet(gomock.Any(), gomock.Any()).Return(nil, errors.New("cache miss")).AnyTimes()
	mockCache.EXPECT().ConferencecallGetByReferenceID(gomock.Any(), gomock.Any()).Return(nil, errors.New("cache miss")).AnyTimes()

	return &handler{db: db, cache: mockCache, utilHandler: mockUtil}, mock, mockCache
}

// Test_DBFailures verifies that a failing statement is reported to the caller
// with the cause kept reachable, so that the business layer can classify it.
func Test_DBFailures(t *testing.T) {
	id := uuid.FromStringOrNil("8e6d4a77-0001-11f1-ad4b-9c5f6e7a8b01")
	otherID := uuid.FromStringOrNil("8e6d4a77-0002-11f1-ad4b-9c5f6e7a8b02")

	exec := func(h *handler, mock sqlmock.Sqlmock) { mock.ExpectExec(".").WillReturnError(errTestFailure) }
	query := func(h *handler, mock sqlmock.Sqlmock) { mock.ExpectQuery(".").WillReturnError(errTestFailure) }

	tests := []struct {
		name   string
		expect func(h *handler, mock sqlmock.Sqlmock)
		call   func(ctx context.Context, h *handler) error
	}{
		// statements
		{"conference create", exec, func(ctx context.Context, h *handler) error {
			return h.ConferenceCreate(ctx, &conference.Conference{Identity: commonidentity.Identity{ID: id}})
		}},
		{"conference update", exec, func(ctx context.Context, h *handler) error {
			return h.ConferenceUpdate(ctx, id, map[conference.Field]any{conference.FieldName: "name"})
		}},
		{"conference add conferencecall id", exec, func(ctx context.Context, h *handler) error {
			return h.ConferenceAddConferencecallID(ctx, id, otherID)
		}},
		{"conference remove conferencecall id", exec, func(ctx context.Context, h *handler) error {
			return h.ConferenceRemoveConferencecallID(ctx, id, otherID)
		}},
		{"conference add recording id", exec, func(ctx context.Context, h *handler) error {
			return h.ConferenceAddRecordingIDs(ctx, id, otherID)
		}},
		{"conference add transcribe id", exec, func(ctx context.Context, h *handler) error {
			return h.ConferenceAddTranscribeIDs(ctx, id, otherID)
		}},
		{"conference delete", exec, func(ctx context.Context, h *handler) error { return h.ConferenceDelete(ctx, id) }},
		{"conference end", exec, func(ctx context.Context, h *handler) error { return h.ConferenceEnd(ctx, id) }},
		{"conferencecall create", exec, func(ctx context.Context, h *handler) error {
			return h.ConferencecallCreate(ctx, &conferencecall.Conferencecall{Identity: commonidentity.Identity{ID: id}})
		}},
		{"conferencecall update", exec, func(ctx context.Context, h *handler) error {
			return h.ConferencecallUpdate(ctx, id, map[conferencecall.Field]any{conferencecall.FieldStatus: conferencecall.StatusJoined})
		}},
		{"conferencecall delete", exec, func(ctx context.Context, h *handler) error { return h.ConferencecallDelete(ctx, id) }},

		// queries
		{"conference count by customer", query, func(ctx context.Context, h *handler) error {
			_, err := h.ConferenceCountByCustomerID(ctx, id)
			return err
		}},
		{"conference get", query, func(ctx context.Context, h *handler) error {
			_, err := h.ConferenceGet(ctx, id)
			return err
		}},
		{"conference get by confbridge id", query, func(ctx context.Context, h *handler) error {
			_, err := h.ConferenceGetByConfbridgeID(ctx, id)
			return err
		}},
		{"conference list", query, func(ctx context.Context, h *handler) error {
			_, err := h.ConferenceList(ctx, 10, "", map[conference.Field]any{})
			return err
		}},
		{"conferencecall get", query, func(ctx context.Context, h *handler) error {
			_, err := h.ConferencecallGet(ctx, id)
			return err
		}},
		{"conferencecall get by reference id", query, func(ctx context.Context, h *handler) error {
			_, err := h.ConferencecallGetByReferenceID(ctx, id)
			return err
		}},
		{"conferencecall list", query, func(ctx context.Context, h *handler) error {
			_, err := h.ConferencecallList(ctx, 10, "", map[conferencecall.Field]any{})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newSQLMockHandler(t)
			tt.expect(h, mock)

			err := tt.call(context.Background(), h)
			if !errors.Is(err, errTestFailure) {
				t.Errorf("Wrong error. expect: the driver error is reachable, got: %v", err)
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong error. a driver failure must not be reported as not found: %v", err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("The statement was not executed. err: %v", errMet)
			}
		})
	}
}

// Test_ListUnsupportedFilterIsRejected verifies that a filter value of an unsupported
// type is reported before any statement is sent to the database.
func Test_ListUnsupportedFilterIsRejected(t *testing.T) {
	tests := []struct {
		name string
		call func(ctx context.Context, h *handler) error
	}{
		{"conference list", func(ctx context.Context, h *handler) error {
			_, err := h.ConferenceList(ctx, 10, "", map[conference.Field]any{conference.FieldName: struct{}{}})
			return err
		}},
		{"conferencecall list", func(ctx context.Context, h *handler) error {
			_, err := h.ConferencecallList(ctx, 10, "", map[conferencecall.Field]any{conferencecall.FieldStatus: struct{}{}})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newSQLMockHandler(t)
			// no statement is expected: sqlmock fails an unexpected query.

			if err := tt.call(context.Background(), h); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("Unexpected statement. err: %v", errMet)
			}
		})
	}
}

// Test_ExecSucceedsEvenIfCacheRefreshFails verifies that the cache refresh after
// a successful write is best effort: its failure is not returned to the caller.
func Test_ExecSucceedsEvenIfCacheRefreshFails(t *testing.T) {
	id := uuid.FromStringOrNil("8e6d4a77-0101-11f1-ad4b-9c5f6e7a8b01")
	otherID := uuid.FromStringOrNil("8e6d4a77-0102-11f1-ad4b-9c5f6e7a8b02")

	tests := []struct {
		name string
		call func(ctx context.Context, h *handler) error
	}{
		{"add conferencecall id", func(ctx context.Context, h *handler) error { return h.ConferenceAddConferencecallID(ctx, id, otherID) }},
		{"remove conferencecall id", func(ctx context.Context, h *handler) error {
			return h.ConferenceRemoveConferencecallID(ctx, id, otherID)
		}},
		{"add recording id", func(ctx context.Context, h *handler) error { return h.ConferenceAddRecordingIDs(ctx, id, otherID) }},
		{"add transcribe id", func(ctx context.Context, h *handler) error { return h.ConferenceAddTranscribeIDs(ctx, id, otherID) }},
		{"conference end", func(ctx context.Context, h *handler) error { return h.ConferenceEnd(ctx, id) }},
		{"conference delete", func(ctx context.Context, h *handler) error { return h.ConferenceDelete(ctx, id) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mock, _ := newSQLMockHandler(t)
			mock.ExpectExec(".").WillReturnResult(sqlmock.NewResult(0, 1))
			// the refresh read is attempted after the write and fails.
			mock.ExpectQuery(".").WillReturnError(errTestFailure)

			if err := tt.call(context.Background(), h); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if errMet := mock.ExpectationsWereMet(); errMet != nil {
				t.Errorf("The statement or the cache refresh was not attempted. err: %v", errMet)
			}
		})
	}
}

func Test_SetToCache_errors(t *testing.T) {
	h, _, mockCache := newSQLMockHandler(t)
	ctx := context.Background()

	cf := &conference.Conference{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("8e6d4a77-0201-11f1-ad4b-9c5f6e7a8b01")}}
	cc := &conferencecall.Conferencecall{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("8e6d4a77-0202-11f1-ad4b-9c5f6e7a8b02")}}

	mockCache.EXPECT().ConferenceSet(ctx, cf).Return(errTestFailure)
	if err := h.ConferenceSetToCache(ctx, cf); !errors.Is(err, errTestFailure) {
		t.Errorf("Wrong error. expect: simulated failure, got: %v", err)
	}

	mockCache.EXPECT().ConferencecallSet(ctx, cc).Return(errTestFailure)
	if err := h.conferencecallSetToCache(ctx, cc); !errors.Is(err, errTestFailure) {
		t.Errorf("Wrong error. expect: simulated failure, got: %v", err)
	}
}

func Test_UpdateToCache_errors(t *testing.T) {
	id := uuid.FromStringOrNil("8e6d4a77-0301-11f1-ad4b-9c5f6e7a8b01")

	t.Run("conference: the database read fails and the cache is not written", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)
		mock.ExpectQuery(".").WillReturnError(errTestFailure)

		if err := h.conferenceUpdateToCache(context.Background(), id); !errors.Is(err, errTestFailure) {
			t.Errorf("Wrong error. expect: simulated failure, got: %v", err)
		}
	})

	t.Run("conferencecall: the database read fails and the cache is not written", func(t *testing.T) {
		h, mock, _ := newSQLMockHandler(t)
		mock.ExpectQuery(".").WillReturnError(errTestFailure)

		if err := h.conferencecallUpdateToCache(context.Background(), id); !errors.Is(err, errTestFailure) {
			t.Errorf("Wrong error. expect: simulated failure, got: %v", err)
		}
	})
}

func Test_NewHandler(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h, ok := NewHandler(db, mockCache).(*handler)
	if !ok {
		t.Fatalf("Wrong type")
	}
	if h.db != db || h.cache != mockCache || h.utilHandler == nil {
		t.Errorf("Wrong handler wiring")
	}
}
