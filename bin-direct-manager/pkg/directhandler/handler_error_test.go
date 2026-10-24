package directhandler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/models/direct"
	"monorepo/bin-direct-manager/pkg/cachehandler"
	"monorepo/bin-direct-manager/pkg/dbhandler"
)

var (
	errDBTest    = errors.New("database failure")
	testDirectID = uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5")
	testCustomer = uuid.FromStringOrNil("5c517950-2a4b-11ee-b280-7389d3585310")
	testResource = uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527")
)

type testMocks struct {
	util   *utilhandler.MockUtilHandler
	db     *dbhandler.MockDBHandler
	notify *notifyhandler.MockNotifyHandler
	cache  *cachehandler.MockCacheHandler
}

func newTestHandler(t *testing.T) (*directHandler, *testMocks) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &testMocks{
		util:   utilhandler.NewMockUtilHandler(mc),
		db:     dbhandler.NewMockDBHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		cache:  cachehandler.NewMockCacheHandler(mc),
	}

	h := &directHandler{
		utilHandler:   m.util,
		db:            m.db,
		notifyhandler: m.notify,
		cache:         m.cache,
	}

	return h, m
}

func newTestDirect(hash string) *direct.Direct {
	return &direct.Direct{
		Identity: commonidentity.Identity{
			ID:         testDirectID,
			CustomerID: testCustomer,
		},
		ResourceType: "extension",
		ResourceID:   testResource,
		Hash:         hash,
	}
}

func assertDirectNotFound(t *testing.T, err error) {
	t.Helper()

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("error = %v, want a typed VoipbinError", err)
	}
	if ve.Reason != "DIRECT_NOT_FOUND" {
		t.Errorf("error reason = %q, want DIRECT_NOT_FOUND", ve.Reason)
	}
	if ve.Domain != string(commonoutline.ServiceNameDirectManager) {
		t.Errorf("error domain = %q, want %q", ve.Domain, commonoutline.ServiceNameDirectManager)
	}
	if !errors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("error = %v, want the cause dbhandler.ErrNotFound to stay reachable", err)
	}
}

func Test_NewDirectHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	res := NewDirectHandler(mockReq, mockDB, mockNotify, mockCache)
	h, ok := res.(*directHandler)
	if !ok {
		t.Fatalf("NewDirectHandler() returned %T, want *directHandler", res)
	}

	if h.reqHandler != mockReq || h.db != mockDB || h.notifyhandler != mockNotify || h.cache != mockCache {
		t.Errorf("NewDirectHandler() did not store the injected dependencies in the matching fields")
	}
	if h.utilHandler == nil {
		t.Errorf("NewDirectHandler() did not create the util handler")
	}
}

func Test_Create_errors(t *testing.T) {
	ctx := context.Background()
	newID := uuid.FromStringOrNil("5c82c65e-2a4b-11ee-b4ae-c3cd00ea0c41")

	t.Run("fails after three hash collisions", func(t *testing.T) {
		h, m := newTestHandler(t)

		// every attempt draws a new id and a new hash and collides; there is no 4th try
		m.util.EXPECT().UUIDCreate().Return(newID).Times(3)
		m.db.EXPECT().DirectCreate(ctx, gomock.Any()).Return(errDBTest).Times(3)
		// no DirectGet, no event: the strict mocks fail the test on any other call

		res, err := h.Create(ctx, testCustomer, "extension", testResource)
		if err == nil {
			t.Fatalf("Create() error = nil, want an error")
		}
		if res != nil {
			t.Errorf("Create() = %v, want nil", res)
		}
	})

	t.Run("recovers when the second attempt succeeds", func(t *testing.T) {
		h, m := newTestHandler(t)
		created := newTestDirect("direct.abcdef123456")

		var hashes []string
		m.util.EXPECT().UUIDCreate().Return(newID).Times(2)
		gomock.InOrder(
			m.db.EXPECT().DirectCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, d *direct.Direct) error {
				hashes = append(hashes, d.Hash)
				return errDBTest
			}),
			m.db.EXPECT().DirectCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, d *direct.Direct) error {
				hashes = append(hashes, d.Hash)
				return nil
			}),
		)
		m.db.EXPECT().DirectGet(ctx, newID).Return(created, nil)
		m.notify.EXPECT().PublishEvent(ctx, direct.EventTypeDirectCreated, created)

		res, err := h.Create(ctx, testCustomer, "extension", testResource)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if res != created {
			t.Errorf("Create() = %v, want %v", res, created)
		}
		if len(hashes) != 2 || hashes[0] == hashes[1] {
			t.Errorf("each attempt must use a fresh hash, got %v", hashes)
		}
	})

	t.Run("get after create fails", func(t *testing.T) {
		h, m := newTestHandler(t)

		m.util.EXPECT().UUIDCreate().Return(newID)
		m.db.EXPECT().DirectCreate(ctx, gomock.Any()).Return(nil)
		m.db.EXPECT().DirectGet(ctx, newID).Return(nil, errDBTest)
		// no event may be published for a record that could not be read back

		res, err := h.Create(ctx, testCustomer, "extension", testResource)
		if !errors.Is(err, errDBTest) {
			t.Errorf("Create() error = %v, want %v", err, errDBTest)
		}
		if res != nil {
			t.Errorf("Create() = %v, want nil", res)
		}
	})
}

func Test_Get_errors(t *testing.T) {
	ctx := context.Background()

	t.Run("not found is typed", func(t *testing.T) {
		h, m := newTestHandler(t)
		m.db.EXPECT().DirectGet(ctx, testDirectID).Return(nil, dbhandler.ErrNotFound)

		res, err := h.Get(ctx, testDirectID)
		if res != nil {
			t.Errorf("Get() = %v, want nil", res)
		}
		assertDirectNotFound(t, err)
	})

	t.Run("other error passes through unchanged", func(t *testing.T) {
		h, m := newTestHandler(t)
		m.db.EXPECT().DirectGet(ctx, testDirectID).Return(nil, errDBTest)

		_, err := h.Get(ctx, testDirectID)
		if err != errDBTest {
			t.Errorf("Get() error = %v, want %v unchanged", err, errDBTest)
		}
		var ve *cerrors.VoipbinError
		if errors.As(err, &ve) {
			t.Errorf("Get() error = %v, a database failure must not be reported as typed", err)
		}
	})
}

func Test_GetByHash_errors(t *testing.T) {
	ctx := context.Background()
	hash := "direct.abcdef123456"

	t.Run("not found in cache and database is typed", func(t *testing.T) {
		h, m := newTestHandler(t)
		m.cache.EXPECT().DirectGetByHash(ctx, hash).Return(nil, errors.New("cache miss"))
		m.db.EXPECT().DirectGetByHash(ctx, hash).Return(nil, dbhandler.ErrNotFound)
		// a record that does not exist must not be cached

		res, err := h.GetByHash(ctx, hash)
		if res != nil {
			t.Errorf("GetByHash() = %v, want nil", res)
		}
		assertDirectNotFound(t, err)
	})

	t.Run("database error passes through unchanged", func(t *testing.T) {
		h, m := newTestHandler(t)
		m.cache.EXPECT().DirectGetByHash(ctx, hash).Return(nil, errors.New("cache miss"))
		m.db.EXPECT().DirectGetByHash(ctx, hash).Return(nil, errDBTest)

		_, err := h.GetByHash(ctx, hash)
		if err != errDBTest {
			t.Errorf("GetByHash() error = %v, want %v unchanged", err, errDBTest)
		}
	})

	t.Run("cache hit does not touch the database", func(t *testing.T) {
		h, m := newTestHandler(t)
		cached := newTestDirect(hash)
		m.cache.EXPECT().DirectGetByHash(ctx, hash).Return(cached, nil)

		res, err := h.GetByHash(ctx, hash)
		if err != nil || res != cached {
			t.Errorf("GetByHash() = %v, %v, want the cached record", res, err)
		}
	})

	t.Run("a cache write failure after a database hit is ignored", func(t *testing.T) {
		h, m := newTestHandler(t)
		fromDB := newTestDirect(hash)
		m.cache.EXPECT().DirectGetByHash(ctx, hash).Return(nil, errors.New("cache miss"))
		m.db.EXPECT().DirectGetByHash(ctx, hash).Return(fromDB, nil)
		m.cache.EXPECT().DirectSetByHash(ctx, hash, fromDB).Return(errors.New("cache write failure"))

		res, err := h.GetByHash(ctx, hash)
		if err != nil || res != fromDB {
			t.Errorf("GetByHash() = %v, %v, want the database record without an error", res, err)
		}
	})
}

func Test_Gets_error(t *testing.T) {
	h, m := newTestHandler(t)
	ctx := context.Background()
	filters := map[direct.Field]any{direct.FieldCustomerID: testCustomer}

	m.db.EXPECT().DirectGets(ctx, uint64(10), "token", filters).Return(nil, errDBTest)

	res, err := h.Gets(ctx, 10, "token", filters)
	if err != errDBTest || res != nil {
		t.Errorf("Gets() = %v, %v, want nil and %v", res, err, errDBTest)
	}
}

func Test_Delete_errors(t *testing.T) {
	ctx := context.Background()

	t.Run("get before delete fails", func(t *testing.T) {
		h, m := newTestHandler(t)
		m.db.EXPECT().DirectGet(ctx, testDirectID).Return(nil, errDBTest)
		// no DirectDelete, no cache invalidation, no event

		res, err := h.Delete(ctx, testDirectID)
		if !errors.Is(err, errDBTest) || res != nil {
			t.Errorf("Delete() = %v, %v, want nil and a wrapped %v", res, err, errDBTest)
		}
	})

	t.Run("delete fails", func(t *testing.T) {
		h, m := newTestHandler(t)
		m.db.EXPECT().DirectGet(ctx, testDirectID).Return(newTestDirect("direct.abcdef123456"), nil)
		m.db.EXPECT().DirectDelete(ctx, testDirectID).Return(errDBTest)
		// the record is still there, so the cache entry must stay and no event is published

		res, err := h.Delete(ctx, testDirectID)
		if !errors.Is(err, errDBTest) || res != nil {
			t.Errorf("Delete() = %v, %v, want nil and a wrapped %v", res, err, errDBTest)
		}
	})
}

func Test_Delete_invalidatesCacheAndPublishes(t *testing.T) {
	h, m := newTestHandler(t)
	ctx := context.Background()
	current := newTestDirect("direct.abcdef123456")

	gomock.InOrder(
		m.db.EXPECT().DirectGet(ctx, testDirectID).Return(current, nil),
		m.db.EXPECT().DirectDelete(ctx, testDirectID).Return(nil),
		m.cache.EXPECT().DirectDeleteByHash(ctx, "direct.abcdef123456").Return(errors.New("cache failure")),
		m.notify.EXPECT().PublishEvent(ctx, direct.EventTypeDirectDeleted, current),
	)

	// a cache failure must not fail the deletion
	res, err := h.Delete(ctx, testDirectID)
	if err != nil || res != current {
		t.Errorf("Delete() = %v, %v, want the deleted record without an error", res, err)
	}
}

func Test_Regenerate_errors(t *testing.T) {
	ctx := context.Background()
	oldHash := "direct.000000000001"

	t.Run("current record cannot be read", func(t *testing.T) {
		h, m := newTestHandler(t)
		m.db.EXPECT().DirectGet(ctx, testDirectID).Return(nil, errDBTest)

		res, err := h.Regenerate(ctx, testDirectID)
		if !errors.Is(err, errDBTest) || res != nil {
			t.Errorf("Regenerate() = %v, %v, want nil and a wrapped %v", res, err, errDBTest)
		}
	})

	t.Run("fails after three update attempts and keeps the old cache entry", func(t *testing.T) {
		h, m := newTestHandler(t)
		m.db.EXPECT().DirectGet(ctx, testDirectID).Return(newTestDirect(oldHash), nil)

		var hashes []string
		m.db.EXPECT().DirectUpdate(ctx, testDirectID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[direct.Field]any) error {
				hashes = append(hashes, fields[direct.FieldHash].(string))
				return errDBTest
			}).Times(3)
		// no cache invalidation and no event after a failed regeneration

		res, err := h.Regenerate(ctx, testDirectID)
		if err == nil || res != nil {
			t.Fatalf("Regenerate() = %v, %v, want nil and an error", res, err)
		}
		if len(hashes) != 3 {
			t.Errorf("DirectUpdate attempts = %d, want 3", len(hashes))
		}
	})

	t.Run("second attempt succeeds with a different hash", func(t *testing.T) {
		h, m := newTestHandler(t)
		regenerated := newTestDirect("direct.000000000002")

		m.db.EXPECT().DirectGet(ctx, testDirectID).Return(newTestDirect(oldHash), nil)
		var hashes []string
		gomock.InOrder(
			m.db.EXPECT().DirectUpdate(ctx, testDirectID, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[direct.Field]any) error {
				hashes = append(hashes, fields[direct.FieldHash].(string))
				return errDBTest
			}),
			m.db.EXPECT().DirectUpdate(ctx, testDirectID, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[direct.Field]any) error {
				hashes = append(hashes, fields[direct.FieldHash].(string))
				return nil
			}),
		)
		m.cache.EXPECT().DirectDeleteByHash(ctx, oldHash).Return(nil)
		m.db.EXPECT().DirectGet(ctx, testDirectID).Return(regenerated, nil)
		m.notify.EXPECT().PublishEvent(ctx, direct.EventTypeDirectRegenerated, regenerated)

		res, err := h.Regenerate(ctx, testDirectID)
		if err != nil || res != regenerated {
			t.Fatalf("Regenerate() = %v, %v, want the regenerated record", res, err)
		}
		if len(hashes) != 2 || hashes[0] == hashes[1] {
			t.Errorf("each attempt must use a fresh hash, got %v", hashes)
		}
	})

	t.Run("the regenerated record cannot be read back", func(t *testing.T) {
		h, m := newTestHandler(t)

		m.db.EXPECT().DirectGet(ctx, testDirectID).Return(newTestDirect(oldHash), nil)
		m.db.EXPECT().DirectUpdate(ctx, testDirectID, gomock.Any()).Return(nil)
		m.cache.EXPECT().DirectDeleteByHash(ctx, oldHash).Return(nil)
		m.db.EXPECT().DirectGet(ctx, testDirectID).Return(nil, errDBTest)
		// no event for a record that could not be read back

		res, err := h.Regenerate(ctx, testDirectID)
		if err != errDBTest || res != nil {
			t.Errorf("Regenerate() = %v, %v, want nil and %v", res, err, errDBTest)
		}
	})
}

func Test_EventCustomerDeleted(t *testing.T) {
	ctx := context.Background()
	curTime := "2023-01-03 21:35:02.809000"
	first := newTestDirect("direct.000000000001")
	first.ID = uuid.FromStringOrNil("11111111-2a4b-11ee-b574-2bad4f039ce5")
	second := newTestDirect("direct.000000000002")
	second.ID = uuid.FromStringOrNil("22222222-2a4b-11ee-b574-2bad4f039ce5")
	third := newTestDirect("direct.000000000003")
	third.ID = uuid.FromStringOrNil("33333333-2a4b-11ee-b574-2bad4f039ce5")

	expectFilters := map[direct.Field]any{direct.FieldCustomerID: testCustomer}

	t.Run("deletes every direct of the customer and continues after a failure", func(t *testing.T) {
		h, m := newTestHandler(t)

		m.util.EXPECT().TimeGetCurTime().Return(curTime)
		m.db.EXPECT().DirectGets(ctx, uint64(999), curTime, expectFilters).Return([]*direct.Direct{first, second, third}, nil)

		// the first one is deleted completely
		m.db.EXPECT().DirectGet(ctx, first.ID).Return(first, nil)
		m.db.EXPECT().DirectDelete(ctx, first.ID).Return(nil)
		m.cache.EXPECT().DirectDeleteByHash(ctx, first.Hash).Return(nil)
		m.notify.EXPECT().PublishEvent(ctx, direct.EventTypeDirectDeleted, first)

		// the second one fails on delete and is skipped without cache invalidation or event
		m.db.EXPECT().DirectGet(ctx, second.ID).Return(second, nil)
		m.db.EXPECT().DirectDelete(ctx, second.ID).Return(errDBTest)

		// the third one is still processed
		m.db.EXPECT().DirectGet(ctx, third.ID).Return(third, nil)
		m.db.EXPECT().DirectDelete(ctx, third.ID).Return(nil)
		m.cache.EXPECT().DirectDeleteByHash(ctx, third.Hash).Return(nil)
		m.notify.EXPECT().PublishEvent(ctx, direct.EventTypeDirectDeleted, third)

		if err := h.EventCustomerDeleted(ctx, testCustomer); err != nil {
			t.Errorf("EventCustomerDeleted() error = %v, a failed delete of one record must not fail the event", err)
		}
	})

	t.Run("a customer without directs", func(t *testing.T) {
		h, m := newTestHandler(t)

		m.util.EXPECT().TimeGetCurTime().Return(curTime)
		m.db.EXPECT().DirectGets(ctx, uint64(999), curTime, expectFilters).Return([]*direct.Direct{}, nil)

		if err := h.EventCustomerDeleted(ctx, testCustomer); err != nil {
			t.Errorf("EventCustomerDeleted() error = %v", err)
		}
	})

	t.Run("list failure is returned and nothing is deleted", func(t *testing.T) {
		h, m := newTestHandler(t)

		m.util.EXPECT().TimeGetCurTime().Return(curTime)
		m.db.EXPECT().DirectGets(ctx, uint64(999), curTime, expectFilters).Return(nil, errDBTest)

		if err := h.EventCustomerDeleted(ctx, testCustomer); err != errDBTest {
			t.Errorf("EventCustomerDeleted() error = %v, want %v", err, errDBTest)
		}
	})
}

func Test_Get_typedNotFoundStatus(t *testing.T) {
	h, m := newTestHandler(t)
	m.db.EXPECT().DirectGet(gomock.Any(), testDirectID).Return(nil, fmt.Errorf("wrapped: %w", dbhandler.ErrNotFound))

	_, err := h.Get(context.Background(), testDirectID)
	assertDirectNotFound(t, err)

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("error = %v, want a typed VoipbinError", err)
	}
	resp, errResp := cerrors.ToResponse(ve)
	if errResp != nil {
		t.Fatalf("ToResponse() error = %v", errResp)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
