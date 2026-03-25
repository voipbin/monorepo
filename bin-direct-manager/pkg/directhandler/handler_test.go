package directhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/models/direct"
	"monorepo/bin-direct-manager/pkg/cachehandler"
	"monorepo/bin-direct-manager/pkg/dbhandler"
)

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseDirect *direct.Direct
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211"),

			responseDirect: &direct.Direct{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211"),
				},
				ResourceType: "extension",
				ResourceID:   uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
				Hash:         "abcdef123456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := directHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyhandler: mockNotify,
				cache:         mockCache,
			}
			ctx := context.Background()

			mockDB.EXPECT().DirectGet(ctx, tt.id).Return(tt.responseDirect, nil)
			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseDirect, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseDirect, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[direct.Field]any

		responseDirects []*direct.Direct
	}{
		{
			name: "normal",

			size:  10,
			token: "2020-04-18T03:22:17.995000Z",
			filters: map[direct.Field]any{
				direct.FieldCustomerID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
			},

			responseDirects: []*direct.Direct{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a0c95b3e-2a00-11ee-a3cd-3307849aa505"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := directHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyhandler: mockNotify,
				cache:         mockCache,
			}
			ctx := context.Background()

			mockDB.EXPECT().DirectGets(ctx, tt.size, tt.token, tt.filters).Return(tt.responseDirects, nil)
			res, err := h.Gets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseDirects, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseDirects, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseDirect *direct.Direct
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),

			responseDirect: &direct.Direct{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),
				},
				Hash: "abcdef123456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := directHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
				cache:         mockCache,
			}
			ctx := context.Background()

			mockDB.EXPECT().DirectGet(ctx, tt.id).Return(tt.responseDirect, nil)
			mockDB.EXPECT().DirectDelete(ctx, tt.id).Return(nil)
			mockCache.EXPECT().DirectDeleteByHash(ctx, tt.responseDirect.Hash).Return(nil)
			mockNotify.EXPECT().PublishEvent(ctx, direct.EventTypeDirectDeleted, tt.responseDirect)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseDirect, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseDirect, res)
			}
		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		resourceType string
		resourceID   uuid.UUID

		responseUUID   uuid.UUID
		responseDirect *direct.Direct
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("5c517950-2a4b-11ee-b280-7389d3585310"),
			resourceType: "extension",
			resourceID:   uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),

			responseUUID: uuid.FromStringOrNil("5c82c65e-2a4b-11ee-b4ae-c3cd00ea0c41"),
			responseDirect: &direct.Direct{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5c82c65e-2a4b-11ee-b4ae-c3cd00ea0c41"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := directHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyhandler: mockNotify,
				cache:         mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().DirectCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().DirectGet(ctx, tt.responseUUID).Return(tt.responseDirect, nil)
			mockNotify.EXPECT().PublishEvent(ctx, direct.EventTypeDirectCreated, tt.responseDirect)

			res, err := h.Create(ctx, tt.customerID, tt.resourceType, tt.resourceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseDirect, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseDirect, res)
			}
		})
	}
}

func Test_GetByHash(t *testing.T) {

	tests := []struct {
		name string
		hash string

		cacheHit       bool
		responseDirect *direct.Direct
	}{
		{
			name: "cache hit",
			hash: "abcdef123456",

			cacheHit: true,
			responseDirect: &direct.Direct{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211"),
				},
				Hash: "abcdef123456",
			},
		},
		{
			name: "cache miss",
			hash: "abcdef123456",

			cacheHit: false,
			responseDirect: &direct.Direct{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211"),
				},
				Hash: "abcdef123456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := directHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyhandler: mockNotify,
				cache:         mockCache,
			}
			ctx := context.Background()

			if tt.cacheHit {
				mockCache.EXPECT().DirectGetByHash(ctx, tt.hash).Return(tt.responseDirect, nil)
			} else {
				mockCache.EXPECT().DirectGetByHash(ctx, tt.hash).Return(nil, dbhandler.ErrNotFound)
				mockDB.EXPECT().DirectGetByHash(ctx, tt.hash).Return(tt.responseDirect, nil)
				mockCache.EXPECT().DirectSetByHash(ctx, tt.hash, tt.responseDirect).Return(nil)
			}

			res, err := h.GetByHash(ctx, tt.hash)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseDirect, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseDirect, res)
			}
		})
	}
}

func Test_Regenerate(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		currentDirect     *direct.Direct
		regeneratedDirect *direct.Direct
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),

			currentDirect: &direct.Direct{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),
				},
				Hash: "oldhash00001",
			},
			regeneratedDirect: &direct.Direct{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a6b3cf48-2a4b-11ee-b574-2bad4f039ce5"),
				},
				Hash: "newhash00001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := directHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyhandler: mockNotify,
				cache:         mockCache,
			}
			ctx := context.Background()

			// get current
			mockDB.EXPECT().DirectGet(ctx, tt.id).Return(tt.currentDirect, nil)
			// update with new hash
			mockDB.EXPECT().DirectUpdate(ctx, tt.id, gomock.Any()).Return(nil)
			// invalidate old cache
			mockCache.EXPECT().DirectDeleteByHash(ctx, tt.currentDirect.Hash).Return(nil)
			// get updated
			mockDB.EXPECT().DirectGet(ctx, tt.id).Return(tt.regeneratedDirect, nil)
			// publish event
			mockNotify.EXPECT().PublishEvent(ctx, direct.EventTypeDirectRegenerated, tt.regeneratedDirect)

			res, err := h.Regenerate(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.regeneratedDirect, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.regeneratedDirect, res)
			}
		})
	}
}
