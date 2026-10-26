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

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/cachehandler"
)

var errDriver = errors.New("simulated driver failure")

type mockedHandler struct {
	h     *handler
	sql   sqlmock.Sqlmock
	cache *cachehandler.MockCacheHandler
	util  *utilhandler.MockUtilHandler
}

// newMockedHandler returns a handler on an independent sqlmock database, so the shared
// sqlite test database is neither used nor closed.
func newMockedHandler(t *testing.T) *mockedHandler {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}

	mc := gomock.NewController(t)
	m := &mockedHandler{
		sql:   mock,
		cache: cachehandler.NewMockCacheHandler(mc),
		util:  utilhandler.NewMockUtilHandler(mc),
	}
	m.h = &handler{util: m.util, db: db, cache: m.cache}

	t.Cleanup(func() {
		_ = db.Close()
		mc.Finish()
		if errExpect := mock.ExpectationsWereMet(); errExpect != nil {
			t.Errorf("Unmet sql expectations: %v", errExpect)
		}
	})

	return m
}

// shortRow is a result row that has fewer columns than the model, so reading it into the model fails.
func shortRow() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id"}).AddRow([]byte("x"))
}

// entityOps holds the same set of operations of one entity, so the failure paths shared by the
// three entities are checked by one table.
type entityOps struct {
	name string

	create          func(h *handler, ctx context.Context, id uuid.UUID) error
	get             func(h *handler, ctx context.Context, id uuid.UUID) error
	getFromDB       func(h *handler, ctx context.Context, id uuid.UUID) error
	updateToCache   func(h *handler, ctx context.Context, id uuid.UUID) error
	deleteRecord    func(h *handler, ctx context.Context, id uuid.UUID) error
	update          func(h *handler, ctx context.Context, id uuid.UUID) error
	list            func(h *handler, ctx context.Context, token string) ([]any, error)
	expectCacheMiss func(m *mockedHandler, id uuid.UUID)
	expectCacheHit  func(m *mockedHandler, id uuid.UUID)
}

func entityOpsList() []entityOps {
	return []entityOps{
		{
			name: "campaign",
			create: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.CampaignCreate(ctx, &campaign.Campaign{Identity: commonidentity.Identity{ID: id}})
			},
			get: func(h *handler, ctx context.Context, id uuid.UUID) error {
				_, err := h.CampaignGet(ctx, id)
				return err
			},
			getFromDB: func(h *handler, ctx context.Context, id uuid.UUID) error {
				_, err := h.campaignGetFromDB(ctx, id)
				return err
			},
			updateToCache: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.campaignUpdateToCache(ctx, id)
			},
			deleteRecord: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.CampaignDelete(ctx, id)
			},
			update: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.CampaignUpdate(ctx, id, map[campaign.Field]any{})
			},
			list: func(h *handler, ctx context.Context, token string) ([]any, error) {
				res, err := h.CampaignList(ctx, token, 10, nil)
				return toAnySlice(res), err
			},
			expectCacheMiss: func(m *mockedHandler, id uuid.UUID) {
				m.cache.EXPECT().CampaignGet(gomock.Any(), id).Return(nil, errDriver)
			},
			expectCacheHit: func(m *mockedHandler, id uuid.UUID) {
				m.cache.EXPECT().CampaignGet(gomock.Any(), id).Return(&campaign.Campaign{Identity: commonidentity.Identity{ID: id}}, nil)
			},
		},
		{
			name: "campaigncall",
			create: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.CampaigncallCreate(ctx, &campaigncall.Campaigncall{Identity: commonidentity.Identity{ID: id}})
			},
			get: func(h *handler, ctx context.Context, id uuid.UUID) error {
				_, err := h.CampaigncallGet(ctx, id)
				return err
			},
			getFromDB: func(h *handler, ctx context.Context, id uuid.UUID) error {
				_, err := h.campaigncallGetFromDB(ctx, id)
				return err
			},
			updateToCache: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.campaigncallUpdateToCache(ctx, id)
			},
			deleteRecord: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.CampaigncallDelete(ctx, id)
			},
			update: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.CampaigncallUpdate(ctx, id, map[campaigncall.Field]any{})
			},
			list: func(h *handler, ctx context.Context, token string) ([]any, error) {
				res, err := h.CampaigncallList(ctx, token, 10, nil)
				return toAnySlice(res), err
			},
			expectCacheMiss: func(m *mockedHandler, id uuid.UUID) {
				m.cache.EXPECT().CampaigncallGet(gomock.Any(), id).Return(nil, errDriver)
			},
			expectCacheHit: func(m *mockedHandler, id uuid.UUID) {
				m.cache.EXPECT().CampaigncallGet(gomock.Any(), id).Return(&campaigncall.Campaigncall{Identity: commonidentity.Identity{ID: id}}, nil)
			},
		},
		{
			name: "outplan",
			create: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.OutplanCreate(ctx, &outplan.Outplan{Identity: commonidentity.Identity{ID: id}})
			},
			get: func(h *handler, ctx context.Context, id uuid.UUID) error {
				_, err := h.OutplanGet(ctx, id)
				return err
			},
			getFromDB: func(h *handler, ctx context.Context, id uuid.UUID) error {
				_, err := h.outplanGetFromDB(ctx, id)
				return err
			},
			updateToCache: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.outplanUpdateToCache(ctx, id)
			},
			deleteRecord: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.OutplanDelete(ctx, id)
			},
			update: func(h *handler, ctx context.Context, id uuid.UUID) error {
				return h.OutplanUpdate(ctx, id, map[outplan.Field]any{})
			},
			list: func(h *handler, ctx context.Context, token string) ([]any, error) {
				res, err := h.OutplanList(ctx, token, 10, nil)
				return toAnySlice(res), err
			},
			expectCacheMiss: func(m *mockedHandler, id uuid.UUID) {
				m.cache.EXPECT().OutplanGet(gomock.Any(), id).Return(nil, errDriver)
			},
			expectCacheHit: func(m *mockedHandler, id uuid.UUID) {
				m.cache.EXPECT().OutplanGet(gomock.Any(), id).Return(&outplan.Outplan{Identity: commonidentity.Identity{ID: id}}, nil)
			},
		},
	}
}

func toAnySlice[T any](in []*T) []any {
	if in == nil {
		return nil
	}
	res := make([]any, 0, len(in))
	for _, v := range in {
		res = append(res, v)
	}
	return res
}

func Test_entities_writeFailures(t *testing.T) {
	id := uuid.FromStringOrNil("d8000001-0000-4000-8000-000000000001")
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	for _, ops := range entityOpsList() {
		t.Run(ops.name+" create exec failure is returned and the cache is not touched", func(t *testing.T) {
			m := newMockedHandler(t)
			m.util.EXPECT().TimeNow().Return(&now)
			m.sql.ExpectExec("INSERT").WillReturnError(errDriver)

			if err := ops.create(m.h, context.Background(), id); !errors.Is(err, errDriver) {
				t.Errorf("Wrong match. expect: %v, got: %v", errDriver, err)
			}
		})

		t.Run(ops.name+" delete exec failure is returned and the cache is not touched", func(t *testing.T) {
			m := newMockedHandler(t)
			m.util.EXPECT().TimeNow().Return(&now)
			m.sql.ExpectExec("UPDATE").WillReturnError(errDriver)

			if err := ops.deleteRecord(m.h, context.Background(), id); !errors.Is(err, errDriver) {
				t.Errorf("Wrong match. expect: %v, got: %v", errDriver, err)
			}
		})

		t.Run(ops.name+" update with no fields is a no-op", func(t *testing.T) {
			// the sql mock has no expectation, so any statement fails the test
			m := newMockedHandler(t)

			if err := ops.update(m.h, context.Background(), id); err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
		})
	}
}

func Test_entities_readFailures(t *testing.T) {
	id := uuid.FromStringOrNil("d8000002-0000-4000-8000-000000000001")

	for _, ops := range entityOpsList() {
		t.Run(ops.name+" query failure", func(t *testing.T) {
			m := newMockedHandler(t)
			m.sql.ExpectQuery("SELECT").WillReturnError(errDriver)

			if err := ops.getFromDB(m.h, context.Background(), id); !errors.Is(err, errDriver) {
				t.Errorf("Wrong match. expect: %v, got: %v", errDriver, err)
			}
		})

		t.Run(ops.name+" no row is not found", func(t *testing.T) {
			m := newMockedHandler(t)
			m.sql.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))

			if err := ops.getFromDB(m.h, context.Background(), id); !errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
			}
		})

		t.Run(ops.name+" unreadable row is a scan failure", func(t *testing.T) {
			m := newMockedHandler(t)
			m.sql.ExpectQuery("SELECT").WillReturnRows(shortRow())

			err := ops.getFromDB(m.h, context.Background(), id)
			if err == nil || errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong match. expect: a scan error, got: %v", err)
			}
		})

		t.Run(ops.name+" cache refresh stops at a failed read", func(t *testing.T) {
			// the cache has no expectation, so a cache write after the failed read fails the test
			m := newMockedHandler(t)
			m.sql.ExpectQuery("SELECT").WillReturnError(errDriver)

			if err := ops.updateToCache(m.h, context.Background(), id); !errors.Is(err, errDriver) {
				t.Errorf("Wrong match. expect: %v, got: %v", errDriver, err)
			}
		})

		t.Run(ops.name+" get answers from the cache without a query", func(t *testing.T) {
			m := newMockedHandler(t)
			ops.expectCacheHit(m, id)

			if err := ops.get(m.h, context.Background(), id); err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
		})

		t.Run(ops.name+" get returns the database failure after a cache miss", func(t *testing.T) {
			m := newMockedHandler(t)
			ops.expectCacheMiss(m, id)
			m.sql.ExpectQuery("SELECT").WillReturnError(errDriver)

			if err := ops.get(m.h, context.Background(), id); !errors.Is(err, errDriver) {
				t.Errorf("Wrong match. expect: %v, got: %v", errDriver, err)
			}
		})
	}
}

func Test_entities_listFailures(t *testing.T) {
	for _, ops := range entityOpsList() {
		t.Run(ops.name+" query failure", func(t *testing.T) {
			m := newMockedHandler(t)
			m.sql.ExpectQuery("SELECT").WillReturnError(errDriver)

			res, err := ops.list(m.h, context.Background(), "2026-01-01 00:00:00.000000")
			if !errors.Is(err, errDriver) || res != nil {
				t.Errorf("Wrong match. expect: %v and no result, got: %v, %v", errDriver, err, res)
			}
		})

		t.Run(ops.name+" unreadable row", func(t *testing.T) {
			m := newMockedHandler(t)
			m.sql.ExpectQuery("SELECT").WillReturnRows(shortRow())

			res, err := ops.list(m.h, context.Background(), "2026-01-01 00:00:00.000000")
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: an error and no partial result, got: %v, %v", err, res)
			}
		})

		t.Run(ops.name+" empty token starts from the current time", func(t *testing.T) {
			m := newMockedHandler(t)
			m.util.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
			m.sql.ExpectQuery("SELECT").
				WithArgs("2026-01-01 00:00:00.000000").
				WillReturnRows(sqlmock.NewRows([]string{"id"}))

			res, err := ops.list(m.h, context.Background(), "")
			if err != nil || len(res) != 0 {
				t.Errorf("Wrong match. expect: empty result, got: %v, %v", res, err)
			}
		})
	}
}

func Test_CampaignListDeletedSince_failures(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("query failure", func(t *testing.T) {
		m := newMockedHandler(t)
		m.sql.ExpectQuery("SELECT").WillReturnError(errDriver)

		res, err := m.h.CampaignListDeletedSince(context.Background(), since, 10)
		if !errors.Is(err, errDriver) || res != nil {
			t.Errorf("Wrong match. expect: %v and no result, got: %v, %v", errDriver, err, res)
		}
	})

	t.Run("unreadable row", func(t *testing.T) {
		m := newMockedHandler(t)
		m.sql.ExpectQuery("SELECT").WillReturnRows(shortRow())

		res, err := m.h.CampaignListDeletedSince(context.Background(), since, 10)
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: an error and no partial result, got: %v, %v", err, res)
		}
	})
}

func Test_GetCurTimeAdd(t *testing.T) {
	const layout = "2006-01-02 15:04:05.999999999"

	tests := []struct {
		name   string
		offset time.Duration
	}{
		{"now", 0},
		{"one hour later", time.Hour},
		{"one hour earlier", -time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now().UTC().Add(tt.offset)
			res := GetCurTimeAdd(tt.offset)
			after := time.Now().UTC().Add(tt.offset)

			got, err := time.Parse(layout, res)
			if err != nil {
				t.Fatalf("Wrong match. the result must be a utc timestamp without zone suffix, got: %q, err: %v", res, err)
			}
			if got.Before(before.Add(-time.Microsecond)) || got.After(after.Add(time.Microsecond)) {
				t.Errorf("Wrong match. expect: between %v and %v, got: %v", before, after, got)
			}
		})
	}
}
