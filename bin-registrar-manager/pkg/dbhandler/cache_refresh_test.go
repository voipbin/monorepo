package dbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-registrar-manager/models/astaor"
	"monorepo/bin-registrar-manager/models/astauth"
	"monorepo/bin-registrar-manager/models/astcontact"
	"monorepo/bin-registrar-manager/models/astendpoint"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/sipauth"
	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

// cacheRefreshCase describes one cached entity for the cache refresh tests.
type cacheRefreshCase struct {
	name string

	// create inserts the row of key "present" through the public create method
	create func(ctx context.Context, h *handler, mockCache *cachehandler.MockCacheHandler, mockUtil *utilhandler.MockUtilHandler) error
	// expectSet registers the expectation of the cache write that follows a refresh
	expectSet func(mockCache *cachehandler.MockCacheHandler, err error) *gomock.Call
	// refresh runs the unexported cache refresh helper for the "present" or the "missing" key
	refresh func(ctx context.Context, h *handler, present bool) error
	// existsInDB reads the "present" row straight from the database
	existsInDB func(ctx context.Context, h *handler) error
}

func cacheRefreshCases() []cacheRefreshCase {
	curTime := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC)

	aorID := "cache-refresh-aor@example.test"
	authID := "cache-refresh-auth@example.test"
	endpointID := "cache-refresh-endpoint@example.test"
	extID := uuid.FromStringOrNil("7a1c0000-519a-11ee-809d-0397adb97529")
	trunkID := uuid.FromStringOrNil("7b2d0000-519a-11ee-809d-0397adb97529")
	missingUUID := uuid.FromStringOrNil("7c3e0000-519a-11ee-809d-0397adb97529")

	return []cacheRefreshCase{
		{
			name: "aor",
			create: func(ctx context.Context, h *handler, mockCache *cachehandler.MockCacheHandler, mockUtil *utilhandler.MockUtilHandler) error {
				return h.AstAORCreate(ctx, &astaor.AstAOR{ID: getStringPointer(aorID), MaxContacts: getIntegerPointer(1)})
			},
			expectSet: func(c *cachehandler.MockCacheHandler, err error) *gomock.Call {
				return c.EXPECT().AstAORSet(gomock.Any(), gomock.Any()).Return(err)
			},
			refresh: func(ctx context.Context, h *handler, present bool) error {
				if !present {
					return h.astAORUpdateToCache(ctx, "missing-aor@example.test")
				}
				return h.astAORUpdateToCache(ctx, aorID)
			},
			existsInDB: func(ctx context.Context, h *handler) error { _, err := h.astAORGetFromDB(ctx, aorID); return err },
		},
		{
			name: "auth",
			create: func(ctx context.Context, h *handler, mockCache *cachehandler.MockCacheHandler, mockUtil *utilhandler.MockUtilHandler) error {
				return h.AstAuthCreate(ctx, &astauth.AstAuth{ID: getStringPointer(authID), AuthType: getStringPointer("userpass")})
			},
			expectSet: func(c *cachehandler.MockCacheHandler, err error) *gomock.Call {
				return c.EXPECT().AstAuthSet(gomock.Any(), gomock.Any()).Return(err)
			},
			refresh: func(ctx context.Context, h *handler, present bool) error {
				if !present {
					return h.astAuthUpdateToCache(ctx, "missing-auth@example.test")
				}
				return h.astAuthUpdateToCache(ctx, authID)
			},
			existsInDB: func(ctx context.Context, h *handler) error { _, err := h.astAuthGetFromDB(ctx, authID); return err },
		},
		{
			name: "endpoint",
			create: func(ctx context.Context, h *handler, mockCache *cachehandler.MockCacheHandler, mockUtil *utilhandler.MockUtilHandler) error {
				return h.AstEndpointCreate(ctx, &astendpoint.AstEndpoint{
					ID:        getStringPointer(endpointID),
					AORs:      getStringPointer(endpointID),
					Auth:      getStringPointer(endpointID),
					Transport: getStringPointer("transport-tcp"),
				})
			},
			expectSet: func(c *cachehandler.MockCacheHandler, err error) *gomock.Call {
				return c.EXPECT().AstEndpointSet(gomock.Any(), gomock.Any()).Return(err)
			},
			refresh: func(ctx context.Context, h *handler, present bool) error {
				if !present {
					return h.astEndpointUpdateToCache(ctx, "missing-endpoint@example.test")
				}
				return h.astEndpointUpdateToCache(ctx, endpointID)
			},
			existsInDB: func(ctx context.Context, h *handler) error {
				_, err := h.astEndpointGetFromDB(ctx, endpointID)
				return err
			},
		},
		{
			name: "extension",
			create: func(ctx context.Context, h *handler, mockCache *cachehandler.MockCacheHandler, mockUtil *utilhandler.MockUtilHandler) error {
				mockUtil.EXPECT().TimeNow().Return(&curTime).AnyTimes()
				return h.ExtensionCreate(ctx, &extension.Extension{
					Identity:   commonidentity.Identity{ID: extID, CustomerID: uuid.FromStringOrNil("7d4f0000-519a-11ee-b9ae-0f71e4c28f81")},
					Name:       "cache refresh",
					EndpointID: "cr-ext@cr.example.test",
					AORID:      "cr-ext@cr.example.test",
					AuthID:     "cr-ext@cr.example.test",
					Extension:  "cr-ext",
					DomainName: "cr.example.test",
					Username:   "cr-ext",
				})
			},
			expectSet: func(c *cachehandler.MockCacheHandler, err error) *gomock.Call {
				return c.EXPECT().ExtensionSet(gomock.Any(), gomock.Any()).Return(err)
			},
			refresh: func(ctx context.Context, h *handler, present bool) error {
				if !present {
					return h.extensionUpdateToCache(ctx, missingUUID)
				}
				return h.extensionUpdateToCache(ctx, extID)
			},
			existsInDB: func(ctx context.Context, h *handler) error { _, err := h.extensionGetFromDB(ctx, extID); return err },
		},
		{
			name: "trunk",
			create: func(ctx context.Context, h *handler, mockCache *cachehandler.MockCacheHandler, mockUtil *utilhandler.MockUtilHandler) error {
				mockUtil.EXPECT().TimeNow().Return(&curTime).AnyTimes()
				return h.TrunkCreate(ctx, &trunk.Trunk{
					Identity:   commonidentity.Identity{ID: trunkID, CustomerID: uuid.FromStringOrNil("7d4f0000-519a-11ee-b9ae-0f71e4c28f81")},
					Name:       "cache refresh",
					DomainName: "cr-trunk",
					AuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic},
					Realm:      "cr-trunk.trunk.example.test",
					Username:   "cr-trunk",
				})
			},
			expectSet: func(c *cachehandler.MockCacheHandler, err error) *gomock.Call {
				return c.EXPECT().TrunkSet(gomock.Any(), gomock.Any()).Return(err)
			},
			refresh: func(ctx context.Context, h *handler, present bool) error {
				if !present {
					return h.trunkUpdateToCache(ctx, missingUUID)
				}
				return h.trunkUpdateToCache(ctx, trunkID)
			},
			existsInDB: func(ctx context.Context, h *handler) error { _, err := h.trunkGetFromDB(ctx, trunkID); return err },
		},
	}
}

// Test_cacheRefresh_errors verifies the cache refresh helper of every cached entity:
// a read failure is returned and the cache is not written, a cache write failure is
// returned with its cause, and the create path treats the refresh as best-effort, so
// the row stays and the create call still succeeds when the cache is unavailable.
func Test_cacheRefresh_errors(t *testing.T) {
	ctx := context.Background()

	for _, tt := range cacheRefreshCases() {
		t.Run(tt.name+" read failure leaves the cache untouched", func(t *testing.T) {
			resetTestTables(t)

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}

			// the cache has no expectation: any write after the failed read fails the test
			err := tt.refresh(ctx, h, false)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
			}
		})

		t.Run(tt.name+" cache write failure is returned", func(t *testing.T) {
			resetTestTables(t)

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}

			// the create path refreshes the cache once with a working cache
			tt.expectSet(mockCache, nil)
			if err := tt.create(ctx, h, mockCache, mockUtil); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			tt.expectSet(mockCache, errInjectedCache)
			err := tt.refresh(ctx, h, true)
			if !errors.Is(err, errInjectedCache) {
				t.Errorf("Wrong match. expect: the cache error to stay reachable, got: %v", err)
			}
		})

		t.Run(tt.name+" create succeeds when the cache refresh fails", func(t *testing.T) {
			resetTestTables(t)

			mc := gomock.NewController(t)
			defer mc.Finish()
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}

			tt.expectSet(mockCache, errInjectedCache)
			if err := tt.create(ctx, h, mockCache, mockUtil); err != nil {
				t.Fatalf("Wrong match. expect: the create to succeed without the cache, got: %v", err)
			}

			if err := tt.existsInDB(ctx, h); err != nil {
				t.Errorf("Wrong match. expect: the row to be stored, got: %v", err)
			}
		})
	}
}

// Test_AstContactGetsByEndpoint_cacheMiss verifies that on a cache miss the contacts
// are read from the database and written to the cache for the same endpoint.
func Test_AstContactGetsByEndpoint_cacheMiss(t *testing.T) {
	resetTestTables(t)

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	endpoint := "1001@cache-miss.example.test"
	insertTestContact(t, "cache-miss-contact-1", endpoint)
	insertTestContact(t, "cache-miss-contact-2", "1002@cache-miss.example.test")

	mockCache.EXPECT().AstContactsGet(ctx, endpoint).Return(nil, errors.New("cache miss"))
	mockCache.EXPECT().AstContactsSet(ctx, endpoint, gomock.Any()).DoAndReturn(func(_ context.Context, _ string, contacts []*astcontact.AstContact) error {
		if len(contacts) != 1 || contacts[0].ID != "cache-miss-contact-1" {
			t.Errorf("Wrong match. expect: only the contact of the endpoint to be cached, got: %v", contacts)
		}
		return nil
	})

	res, err := h.AstContactGetsByEndpoint(ctx, endpoint)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 1 || res[0].ID != "cache-miss-contact-1" || res[0].Endpoint != endpoint {
		t.Errorf("Wrong match. expect: the contact of the endpoint, got: %v", res)
	}
}

// Test_SIPAuthCreate_execFailure verifies that a failing insert is returned with its
// cause.
func Test_SIPAuthCreate_execFailure(t *testing.T) {
	h, mock, _, mockUtil := newErrorPathHandler(t)
	ctx := context.Background()

	now := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now)
	mock.ExpectExec(".*").WillReturnError(errInjectedDB)

	err := h.SIPAuthCreate(ctx, &sipauth.SIPAuth{
		ID:       uuid.FromStringOrNil("7e500000-519a-11ee-809d-0397adb97529"),
		Realm:    "sa.example.test",
		Username: "sa-user",
	})
	if !errors.Is(err, errInjectedDB) {
		t.Errorf("Wrong match. expect: the cause to stay reachable, got: %v", err)
	}
	if mockErr := mock.ExpectationsWereMet(); mockErr != nil {
		t.Errorf("Wrong match. expect: the insert to be attempted once, got: %v", mockErr)
	}
}
