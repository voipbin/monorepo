package dbhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/cachehandler"
)

func Test_CampaigncallLookup_failures(t *testing.T) {
	id := uuid.FromStringOrNil("d8000003-0000-4000-8000-000000000001")

	lookups := []struct {
		name string
		call func(h *handler) (*campaigncall.Campaigncall, error)
	}{
		{"by reference id", func(h *handler) (*campaigncall.Campaigncall, error) {
			return h.CampaigncallGetByReferenceID(context.Background(), id)
		}},
		{"by activeflow id", func(h *handler) (*campaigncall.Campaigncall, error) {
			return h.CampaigncallGetByActiveflowID(context.Background(), id)
		}},
	}

	for _, l := range lookups {
		t.Run(l.name+" query failure", func(t *testing.T) {
			m := newMockedHandler(t)
			m.sql.ExpectQuery("SELECT").WillReturnError(errDriver)

			res, err := l.call(m.h)
			if !errors.Is(err, errDriver) || res != nil {
				t.Errorf("Wrong match. expect: %v and no result, got: %v, %v", errDriver, err, res)
			}
		})

		t.Run(l.name+" no row is not found", func(t *testing.T) {
			m := newMockedHandler(t)
			m.sql.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))

			res, err := l.call(m.h)
			if !errors.Is(err, ErrNotFound) || res != nil {
				t.Errorf("Wrong match. expect: %v and no result, got: %v, %v", ErrNotFound, err, res)
			}
		})

		t.Run(l.name+" iteration failure is not reported as not found", func(t *testing.T) {
			m := newMockedHandler(t)
			m.sql.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow([]byte("x")).RowError(0, errDriver))

			res, err := l.call(m.h)
			if !errors.Is(err, errDriver) || errors.Is(err, ErrNotFound) || res != nil {
				t.Errorf("Wrong match. expect: %v and no result, got: %v, %v", errDriver, err, res)
			}
		})

		t.Run(l.name+" unreadable row is a scan failure", func(t *testing.T) {
			m := newMockedHandler(t)
			m.sql.ExpectQuery("SELECT").WillReturnRows(shortRow())

			res, err := l.call(m.h)
			if err == nil || errors.Is(err, ErrNotFound) || res != nil {
				t.Errorf("Wrong match. expect: a scan error and no result, got: %v, %v", err, res)
			}
		})
	}
}

func Test_CampaigncallListOngoingByCampaignID_failures(t *testing.T) {
	campaignID := uuid.FromStringOrNil("d8000004-0000-4000-8000-000000000001")

	t.Run("query failure", func(t *testing.T) {
		m := newMockedHandler(t)
		m.sql.ExpectQuery("SELECT").WillReturnError(errDriver)

		res, err := m.h.CampaigncallListOngoingByCampaignID(context.Background(), campaignID, "2026-01-01 00:00:00.000000", 10)
		if !errors.Is(err, errDriver) || res != nil {
			t.Errorf("Wrong match. expect: %v and no result, got: %v, %v", errDriver, err, res)
		}
	})

	t.Run("unreadable row gives no partial result", func(t *testing.T) {
		m := newMockedHandler(t)
		m.sql.ExpectQuery("SELECT").WillReturnRows(shortRow())

		res, err := m.h.CampaigncallListOngoingByCampaignID(context.Background(), campaignID, "2026-01-01 00:00:00.000000", 10)
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: an error and no partial result, got: %v, %v", err, res)
		}
	})

	t.Run("empty token starts from the current time", func(t *testing.T) {
		m := newMockedHandler(t)
		m.util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
		m.sql.ExpectQuery("SELECT").WithArgs(campaignID.Bytes(), campaigncall.StatusDialing, campaigncall.StatusProgressing, "2026-01-01 00:00:00.000000").
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		res, err := m.h.CampaigncallListOngoingByCampaignID(context.Background(), campaignID, "", 10)
		if err != nil || len(res) != 0 {
			t.Errorf("Wrong match. expect: empty result, got: %v, %v", res, err)
		}
	})
}

// Test_list_unsupportedFilter checks that a filter value of an unsupported type stops the list
// before any query is sent (the sql mock has no expectation).
func Test_list_unsupportedFilter(t *testing.T) {
	unsupported := map[string]int{"a": 1}

	tests := []struct {
		name string
		call func(h *handler) error
	}{
		{"campaign", func(h *handler) error {
			_, err := h.CampaignList(context.Background(), "2026-01-01 00:00:00.000000", 10, map[campaign.Field]any{campaign.FieldName: unsupported})
			return err
		}},
		{"campaigncall", func(h *handler) error {
			_, err := h.CampaigncallList(context.Background(), "2026-01-01 00:00:00.000000", 10, map[campaigncall.Field]any{campaigncall.FieldStatus: unsupported})
			return err
		}},
		{"outplan", func(h *handler) error {
			_, err := h.OutplanList(context.Background(), "2026-01-01 00:00:00.000000", 10, map[outplan.Field]any{outplan.FieldName: unsupported})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newMockedHandler(t)

			err := tt.call(m.h)
			if err == nil || !strings.Contains(err.Error(), "unsupported filter type") {
				t.Errorf("Wrong match. expect: the unsupported filter error, got: %v", err)
			}
		})
	}
}

func Test_update_execFailure(t *testing.T) {
	id := uuid.FromStringOrNil("d8000005-0000-4000-8000-000000000001")
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string
		call func(h *handler) error
	}{
		{"campaign", func(h *handler) error {
			return h.CampaignUpdate(context.Background(), id, map[campaign.Field]any{campaign.FieldName: "name-a"})
		}},
		{"campaigncall", func(h *handler) error {
			return h.CampaigncallUpdate(context.Background(), id, map[campaigncall.Field]any{campaigncall.FieldStatus: campaigncall.StatusDone})
		}},
		{"outplan", func(h *handler) error {
			return h.OutplanUpdate(context.Background(), id, map[outplan.Field]any{outplan.FieldName: "name-a"})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// the cache mock has no expectation, so a cache refresh after the failure fails the test
			m := newMockedHandler(t)
			m.util.EXPECT().TimeNow().Return(&now)
			m.sql.ExpectExec("UPDATE").WillReturnError(errDriver)

			if err := tt.call(m.h); !errors.Is(err, errDriver) {
				t.Errorf("Wrong match. expect: %v, got: %v", errDriver, err)
			}
		})
	}
}

// cacheOps holds the cache write operations of one entity on a real (sqlite) record.
type cacheOps struct {
	name string

	create        func(h *handler, ctx context.Context, id uuid.UUID) error
	expectCreate  func(m *cachehandler.MockCacheHandler)
	expectSetFail func(m *cachehandler.MockCacheHandler)
	updateToCache func(h *handler, ctx context.Context, id uuid.UUID) error
	setToCache    func(h *handler, ctx context.Context, id uuid.UUID) error
}

func cacheOpsList() []cacheOps {
	return []cacheOps{
		{
			name: "campaign",
			create: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.CampaignCreate(ctx, &campaign.Campaign{Identity: commonidentity.Identity{ID: id}})
			},
			expectCreate: func(m *cachehandler.MockCacheHandler) {
				m.EXPECT().CampaignSet(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectSetFail: func(m *cachehandler.MockCacheHandler) {
				m.EXPECT().CampaignSet(gomock.Any(), gomock.Any()).Return(errDriver)
			},
			updateToCache: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.campaignUpdateToCache(ctx, id)
			},
			setToCache: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.campaignSetToCache(ctx, &campaign.Campaign{Identity: commonidentity.Identity{ID: id}})
			},
		},
		{
			name: "campaigncall",
			create: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.CampaigncallCreate(ctx, &campaigncall.Campaigncall{Identity: commonidentity.Identity{ID: id}})
			},
			expectCreate: func(m *cachehandler.MockCacheHandler) {
				m.EXPECT().CampaigncallSet(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectSetFail: func(m *cachehandler.MockCacheHandler) {
				m.EXPECT().CampaigncallSet(gomock.Any(), gomock.Any()).Return(errDriver)
			},
			updateToCache: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.campaigncallUpdateToCache(ctx, id)
			},
			setToCache: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.campaigncallSetToCache(ctx, &campaigncall.Campaigncall{Identity: commonidentity.Identity{ID: id}})
			},
		},
		{
			name: "outplan",
			create: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.OutplanCreate(ctx, &outplan.Outplan{Identity: commonidentity.Identity{ID: id}})
			},
			expectCreate: func(m *cachehandler.MockCacheHandler) {
				m.EXPECT().OutplanSet(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectSetFail: func(m *cachehandler.MockCacheHandler) {
				m.EXPECT().OutplanSet(gomock.Any(), gomock.Any()).Return(errDriver)
			},
			updateToCache: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.outplanUpdateToCache(ctx, id)
			},
			setToCache: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.outplanSetToCache(ctx, &outplan.Outplan{Identity: commonidentity.Identity{ID: id}})
			},
		},
	}
}

func Test_cacheWrite_failures(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	for _, ops := range cacheOpsList() {
		t.Run(ops.name+" set to cache returns the cache failure", func(t *testing.T) {
			id := uuid.Must(uuid.NewV4())

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{util: utilhandler.NewMockUtilHandler(mc), db: dbTest, cache: mockCache}
			ops.expectSetFail(mockCache)

			if err := ops.setToCache(h, context.Background(), id); !errors.Is(err, errDriver) {
				t.Errorf("Wrong match. expect: %v, got: %v", errDriver, err)
			}
		})

		t.Run(ops.name+" cache refresh returns the cache failure after a database read", func(t *testing.T) {
			// the shared test database keeps its rows, so every run uses a new id
			id := uuid.Must(uuid.NewV4())

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &handler{util: mockUtil, db: dbTest, cache: mockCache}
			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(&now)
			ops.expectCreate(mockCache)
			if err := ops.create(h, ctx, id); err != nil {
				t.Fatalf("could not create the record: %v", err)
			}

			ops.expectSetFail(mockCache)
			if err := ops.updateToCache(h, ctx, id); !errors.Is(err, errDriver) {
				t.Errorf("Wrong match. expect: %v, got: %v", errDriver, err)
			}
		})
	}
}
