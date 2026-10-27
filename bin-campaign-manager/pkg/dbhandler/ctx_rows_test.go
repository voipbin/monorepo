package dbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/cachehandler"

	gomock "go.uber.org/mock/gomock"
)

var errRowsBroken = errors.New("simulated failure while reading rows")

// Test_RowsErr_NotMisreported verifies that a failure while iterating the rows
// is returned as an error: a single-row get must not report it as ErrNotFound
// and a list must not return a partial (here empty) result as a success.
func Test_RowsErr_NotMisreported(t *testing.T) {
	resetTestDB(t)

	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e77")
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func(h *handler) error
	}{
		{"get campaign", func(h *handler) error { _, err := h.campaignGetFromDB(ctx, id); return err }},
		{"get campaigncall", func(h *handler) error { _, err := h.campaigncallGetFromDB(ctx, id); return err }},
		{"get campaigncall by reference id", func(h *handler) error {
			_, err := h.CampaigncallGetByReferenceID(ctx, id)
			return err
		}},
		{"get campaigncall by activeflow id", func(h *handler) error {
			_, err := h.CampaigncallGetByActiveflowID(ctx, id)
			return err
		}},
		{"get outplan", func(h *handler) error { _, err := h.outplanGetFromDB(ctx, id); return err }},
		{"list campaign", func(h *handler) error {
			_, err := h.CampaignList(ctx, "2026-01-01 00:00:00.000000", 10, nil)
			return err
		}},
		{"list campaign deleted since", func(h *handler) error {
			_, err := h.CampaignListDeletedSince(ctx, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 10)
			return err
		}},
		{"list campaigncall", func(h *handler) error {
			_, err := h.CampaigncallList(ctx, "2026-01-01 00:00:00.000000", 10, nil)
			return err
		}},
		{"list campaigncall ongoing by campaign id", func(h *handler) error {
			_, err := h.CampaigncallListOngoingByCampaignID(ctx, id, "2026-01-01 00:00:00.000000", 10)
			return err
		}},
		{"list outplan", func(h *handler) error {
			_, err := h.OutplanList(ctx, "2026-01-01 00:00:00.000000", 10, nil)
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

			h := &handler{db: db, cache: cachehandler.NewMockCacheHandler(mc)}

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
			if !errors.Is(err, errRowsBroken) {
				t.Errorf("expected the iteration failure in the error, got: %v", err)
			}
		})
	}
}

// Test_ContextCancel_UpdateAborted verifies that the request context reaches
// the database driver: a cancelled context aborts the three Update writes
// instead of the statement running to completion. The id does not exist, so
// the test does not depend on (or change) the shared database state.
func Test_ContextCancel_UpdateAborted(t *testing.T) {
	resetTestDB(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().TimeNow().Return(nil).AnyTimes()
	h := &handler{util: mockUtil, db: dbTest, cache: cachehandler.NewMockCacheHandler(mc)}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	id := uuid.FromStringOrNil("9a0f3c4e-1b2d-11f1-8a5e-0b3f6c1d2e78")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"campaign update", func() error {
			return h.CampaignUpdate(ctx, id, map[campaign.Field]any{campaign.FieldName: "x"})
		}},
		{"campaigncall update", func() error {
			return h.CampaigncallUpdate(ctx, id, map[campaigncall.Field]any{campaigncall.FieldStatus: campaigncall.StatusDone})
		}},
		{"outplan update", func() error {
			return h.OutplanUpdate(ctx, id, map[outplan.Field]any{outplan.FieldName: "x"})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, context.Canceled) {
				t.Errorf("expected context.Canceled in the error chain, got: %v", err)
			}
		})
	}
}
