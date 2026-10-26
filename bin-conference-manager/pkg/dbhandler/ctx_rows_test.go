package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/cachehandler"
)

var errRowsBroken = errors.New("simulated failure while reading rows")

// Test_RowsErr_NotMisreported verifies that a failure while iterating the rows
// is returned as an error: a single-row get must not report it as ErrNotFound
// and a list must not return a partial (here empty) result as a success.
func Test_RowsErr_NotMisreported(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")

	tests := []struct {
		name string

		// wrapsCause is true when the returned error wraps the original cause
		// with %w. The pre-existing getters and the list functions wrap with %v.
		wrapsCause bool
		fn         func(h *handler) error
	}{
		{
			"get conference from db",
			false,
			func(h *handler) error { _, err := h.conferenceGetFromDB(context.Background(), id); return err },
		},
		{
			"get conferencecall from db",
			false,
			func(h *handler) error { _, err := h.conferencecallGetFromDB(context.Background(), id); return err },
		},
		{
			"get conference by confbridge id",
			true,
			func(h *handler) error {
				_, err := h.ConferenceGetByConfbridgeID(context.Background(), id)
				return err
			},
		},
		{
			"get conferencecall by reference id",
			true,
			func(h *handler) error {
				_, err := h.ConferencecallGetByReferenceID(context.Background(), id)
				return err
			},
		},
		{
			"list conference",
			false,
			func(h *handler) error {
				_, err := h.ConferenceList(context.Background(), 10, "2026-01-01 00:00:00.000000", map[conference.Field]any{})
				return err
			},
		},
		{
			"list conferencecall",
			false,
			func(h *handler) error {
				_, err := h.ConferencecallList(context.Background(), 10, "2026-01-01 00:00:00.000000", map[conferencecall.Field]any{})
				return err
			},
		},
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
			rows := sqlmock.NewRows([]string{"id"}).AddRow(id.Bytes()).RowError(0, errRowsBroken)
			mock.ExpectQuery("SELECT").WillReturnRows(rows)

			err = tt.fn(h)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Fatalf("an iteration failure must not be reported as ErrNotFound. err: %v", err)
			}
			if tt.wrapsCause && !errors.Is(err, errRowsBroken) {
				t.Errorf("the cause is not wrapped. err: %v", err)
			}
			if !strings.Contains(err.Error(), errRowsBroken.Error()) {
				t.Errorf("the cause is lost. err: %v", err)
			}
		})
	}
}

// Test_ExecContext_Canceled verifies that the json update statements honor the
// context: with a canceled context the statement must not be executed.
func Test_ExecContext_Canceled(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e78")
	otherID := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e79")

	tests := []struct {
		name string
		fn   func(ctx context.Context, h *handler) error
	}{
		{"add conferencecall id", func(ctx context.Context, h *handler) error { return h.ConferenceAddConferencecallID(ctx, id, otherID) }},
		{"remove conferencecall id", func(ctx context.Context, h *handler) error {
			return h.ConferenceRemoveConferencecallID(ctx, id, otherID)
		}},
		{"add recording id", func(ctx context.Context, h *handler) error { return h.ConferenceAddRecordingIDs(ctx, id, otherID) }},
		{"add transcribe id", func(ctx context.Context, h *handler) error { return h.ConferenceAddTranscribeIDs(ctx, id, otherID) }},
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockUtil.EXPECT().TimeNow().Return(&time.Time{}).AnyTimes()

			h := &handler{db: db, cache: cachehandler.NewMockCacheHandler(mc), utilHandler: mockUtil}

			// a statement that is executed would succeed, so only the context
			// can make this call fail. The expectation is deliberately not
			// verified because a canceled context never reaches the driver.
			mock.ExpectExec("update").WillReturnResult(sqlmock.NewResult(0, 1))

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err = tt.fn(ctx, h)
			if err == nil {
				t.Fatalf("expected an error for a canceled context, got nil")
			}
			if !strings.Contains(err.Error(), "context canceled") {
				t.Errorf("expected a context canceled error. err: %v", err)
			}
		})
	}
}

// Test_ConferencecallGetByReferenceID_queriesDBOnly verifies that the lookup by
// reference id goes straight to the database. The cache mock has no expectation,
// so any cache call fails the test.
func Test_ConferencecallGetByReferenceID_queriesDBOnly(t *testing.T) {
	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e78")

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &handler{db: db, cache: cachehandler.NewMockCacheHandler(mc), utilHandler: utilhandler.NewMockUtilHandler(mc)}

	mock.ExpectQuery("SELECT").WillReturnError(errRowsBroken)

	if _, err := h.ConferencecallGetByReferenceID(context.Background(), id); !errors.Is(err, errRowsBroken) {
		t.Errorf("Wrong match. expect: %v, got: %v", errRowsBroken, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
