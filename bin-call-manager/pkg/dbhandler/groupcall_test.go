package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/cachehandler"
)

func Test_GroupcallCreate(t *testing.T) {

	tests := []struct {
		name string

		data *groupcall.Groupcall

		responseCurTime string
		expectRes       *groupcall.Groupcall
	}{
		{
			name: "have all",

			data: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("39ee40d7-9f83-45bb-ba29-7bb9de62c93e"),
					CustomerID: uuid.FromStringOrNil("a8eaeb80-bd76-11ed-94db-7fe899d03ca7"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("bbc99a3e-2bf8-11ef-b744-db67144792ae"),
				},

				Status: groupcall.StatusProgressing,
				FlowID: uuid.FromStringOrNil("53cc3b24-0af3-4500-9764-1aa421ddbba3"),

				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				MasterCallID:      uuid.FromStringOrNil("259aa640-e11a-11ed-9a20-6755fa431b8b"),
				MasterGroupcallID: uuid.FromStringOrNil("d25e45b6-e2ba-11ed-a032-eb8d11c69ac9"),
				RingMethod:        groupcall.RingMethodRingAll,
				AnswerMethod:      groupcall.AnswerMethodHangupOthers,
				AnswerCallID:      uuid.Nil,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("a9127fec-bd76-11ed-9c0d-f79292b7b2a6"),
					uuid.FromStringOrNil("a939559a-bd76-11ed-ac28-4f58c3ed30f3"),
				},
				AnswerGroupcallID: uuid.FromStringOrNil("c4391238-e2b8-11ed-95a3-1b5b76d3b8c7"),
				GroupcallIDs: []uuid.UUID{
					uuid.FromStringOrNil("c486c690-e2b8-11ed-89b3-9bd17d2f40aa"),
					uuid.FromStringOrNil("c4cf6774-e2b8-11ed-b7f9-73a38e6b336c"),
				},
				CallCount:      2,
				GroupcallCount: 2,
				DialIndex:      1,
			},

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("39ee40d7-9f83-45bb-ba29-7bb9de62c93e"),
					CustomerID: uuid.FromStringOrNil("a8eaeb80-bd76-11ed-94db-7fe899d03ca7"),
				},
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("bbc99a3e-2bf8-11ef-b744-db67144792ae"),
				},

				Status: groupcall.StatusProgressing,
				FlowID: uuid.FromStringOrNil("53cc3b24-0af3-4500-9764-1aa421ddbba3"),

				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				MasterCallID:      uuid.FromStringOrNil("259aa640-e11a-11ed-9a20-6755fa431b8b"),
				MasterGroupcallID: uuid.FromStringOrNil("d25e45b6-e2ba-11ed-a032-eb8d11c69ac9"),
				RingMethod:        groupcall.RingMethodRingAll,
				AnswerMethod:      groupcall.AnswerMethodHangupOthers,
				AnswerCallID:      uuid.Nil,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("a9127fec-bd76-11ed-9c0d-f79292b7b2a6"),
					uuid.FromStringOrNil("a939559a-bd76-11ed-ac28-4f58c3ed30f3"),
				},
				AnswerGroupcallID: uuid.FromStringOrNil("c4391238-e2b8-11ed-95a3-1b5b76d3b8c7"),
				GroupcallIDs: []uuid.UUID{
					uuid.FromStringOrNil("c486c690-e2b8-11ed-89b3-9bd17d2f40aa"),
					uuid.FromStringOrNil("c4cf6774-e2b8-11ed-b7f9-73a38e6b336c"),
				},
				CallCount:      2,
				GroupcallCount: 2,
				DialIndex:      1,
				TMCreate:       "2023-01-18 03:22:18.995000",
				TMUpdate:       DefaultTimeStamp,
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any()).Return(nil)
			if errCreate := h.GroupcallCreate(ctx, tt.data); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}

			mockCache.EXPECT().GroupcallGet(ctx, tt.data.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
			res, err := h.GroupcallGet(ctx, tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_GroupcallSetAnswerCallID(t *testing.T) {

	tests := []struct {
		name string
		data *groupcall.Groupcall

		id           uuid.UUID
		answerCallID uuid.UUID

		responseCurTime string
		expectRes       *groupcall.Groupcall
	}{
		{
			name: "normal",
			data: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("feaf81a6-bd77-11ed-bd82-cba4c20d3477"),
					CustomerID: uuid.FromStringOrNil("a8eaeb80-bd76-11ed-94db-7fe899d03ca7"),
				},

				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				RingMethod:   groupcall.RingMethodRingAll,
				AnswerMethod: groupcall.AnswerMethodHangupOthers,
				AnswerCallID: uuid.Nil,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("49081344-bd78-11ed-aa51-0349af7b9f8b"),
					uuid.FromStringOrNil("4936e9f8-bd78-11ed-b921-37bb6ae97f98"),
				},
			},

			id:           uuid.FromStringOrNil("feaf81a6-bd77-11ed-bd82-cba4c20d3477"),
			answerCallID: uuid.FromStringOrNil("49081344-bd78-11ed-aa51-0349af7b9f8b"),

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("feaf81a6-bd77-11ed-bd82-cba4c20d3477"),
					CustomerID: uuid.FromStringOrNil("a8eaeb80-bd76-11ed-94db-7fe899d03ca7"),
				},

				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000003",
					},
				},
				RingMethod:   groupcall.RingMethodRingAll,
				AnswerMethod: groupcall.AnswerMethodHangupOthers,
				AnswerCallID: uuid.FromStringOrNil("49081344-bd78-11ed-aa51-0349af7b9f8b"),
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("49081344-bd78-11ed-aa51-0349af7b9f8b"),
					uuid.FromStringOrNil("4936e9f8-bd78-11ed-b921-37bb6ae97f98"),
				},
				GroupcallIDs: []uuid.UUID{},

				TMCreate: "2023-01-18 03:22:18.995000",
				TMUpdate: "2023-01-18 03:22:18.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any()).Return(nil)
			if errCreate := h.GroupcallCreate(ctx, tt.data); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
			if errSet := h.GroupcallSetAnswerCallID(ctx, tt.id, tt.answerCallID); errSet != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errSet)
			}

			mockCache.EXPECT().GroupcallGet(ctx, tt.data.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
			res, err := h.GroupcallGet(ctx, tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_GroupcallSetAnswerGroupcallID(t *testing.T) {

	tests := []struct {
		name string
		data *groupcall.Groupcall

		id                uuid.UUID
		answerGroupcallID uuid.UUID

		responseCurTime string
		expectRes       *groupcall.Groupcall
	}{
		{
			name: "normal",
			data: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("864ce138-e2b9-11ed-b321-333da5e2b527"),
				},
				Source:       &commonaddress.Address{},
				Destinations: []commonaddress.Address{},

				CallIDs:           []uuid.UUID{},
				AnswerGroupcallID: uuid.Nil,
				GroupcallIDs: []uuid.UUID{
					uuid.FromStringOrNil("b6fbd8d4-e2b9-11ed-b577-47ff74c6d9a5"),
					uuid.FromStringOrNil("4936e9f8-bd78-11ed-b921-37bb6ae97f98"),
				},
			},

			id:                uuid.FromStringOrNil("864ce138-e2b9-11ed-b321-333da5e2b527"),
			answerGroupcallID: uuid.FromStringOrNil("b6fbd8d4-e2b9-11ed-b577-47ff74c6d9a5"),

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("864ce138-e2b9-11ed-b321-333da5e2b527"),
				},
				Source:       &commonaddress.Address{},
				Destinations: []commonaddress.Address{},

				CallIDs:           []uuid.UUID{},
				AnswerGroupcallID: uuid.FromStringOrNil("b6fbd8d4-e2b9-11ed-b577-47ff74c6d9a5"),
				GroupcallIDs: []uuid.UUID{
					uuid.FromStringOrNil("b6fbd8d4-e2b9-11ed-b577-47ff74c6d9a5"),
					uuid.FromStringOrNil("4936e9f8-bd78-11ed-b921-37bb6ae97f98"),
				},

				TMCreate: "2023-01-18 03:22:18.995000",
				TMUpdate: "2023-01-18 03:22:18.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any()).Return(nil)
			if errCreate := h.GroupcallCreate(ctx, tt.data); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
			if errSet := h.GroupcallSetAnswerGroupcallID(ctx, tt.id, tt.answerGroupcallID); errSet != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errSet)
			}

			mockCache.EXPECT().GroupcallGet(ctx, tt.data.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
			res, err := h.GroupcallGet(ctx, tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_GroupcallDecreaseCallCount(t *testing.T) {

	tests := []struct {
		name string
		data *groupcall.Groupcall

		id uuid.UUID

		responseCurTime string
		expectRes       *groupcall.Groupcall
	}{
		{
			name: "normal",
			data: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("694c2b84-d913-11ed-82ca-8ffe9f085634"),
				},
				Source:       &commonaddress.Address{},
				Destinations: []commonaddress.Address{},
				CallIDs:      []uuid.UUID{},
				GroupcallIDs: []uuid.UUID{},
				CallCount:    2,
			},

			id: uuid.FromStringOrNil("694c2b84-d913-11ed-82ca-8ffe9f085634"),

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("694c2b84-d913-11ed-82ca-8ffe9f085634"),
				},

				Source:       &commonaddress.Address{},
				Destinations: []commonaddress.Address{},
				CallIDs:      []uuid.UUID{},
				GroupcallIDs: []uuid.UUID{},
				CallCount:    1,

				TMCreate: "2023-01-18 03:22:18.995000",
				TMUpdate: "2023-01-18 03:22:18.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any()).Return(nil)
			if errCreate := h.GroupcallCreate(ctx, tt.data); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
			if errSet := h.GroupcallDecreaseCallCount(ctx, tt.id); errSet != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errSet)
			}

			mockCache.EXPECT().GroupcallGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
			res, err := h.GroupcallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_GroupcallDecreaseGroupcallCount(t *testing.T) {

	tests := []struct {
		name string
		data *groupcall.Groupcall

		id uuid.UUID

		responseCurTime string
		expectRes       *groupcall.Groupcall
	}{
		{
			name: "normal",
			data: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("00731852-e2c3-11ed-99d8-53674cc4d92d"),
				},
				Source:         &commonaddress.Address{},
				Destinations:   []commonaddress.Address{},
				CallIDs:        []uuid.UUID{},
				GroupcallIDs:   []uuid.UUID{},
				GroupcallCount: 2,
			},

			id: uuid.FromStringOrNil("00731852-e2c3-11ed-99d8-53674cc4d92d"),

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("00731852-e2c3-11ed-99d8-53674cc4d92d"),
				},

				Source:         &commonaddress.Address{},
				Destinations:   []commonaddress.Address{},
				CallIDs:        []uuid.UUID{},
				GroupcallIDs:   []uuid.UUID{},
				GroupcallCount: 1,

				TMCreate: "2023-01-18 03:22:18.995000",
				TMUpdate: "2023-01-18 03:22:18.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any()).Return(nil)
			if errCreate := h.GroupcallCreate(ctx, tt.data); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
			if errSet := h.GroupcallDecreaseGroupcallCount(ctx, tt.id); errSet != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errSet)
			}

			mockCache.EXPECT().GroupcallGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
			res, err := h.GroupcallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_GroupcallSetStatus(t *testing.T) {

	tests := []struct {
		name string
		data *groupcall.Groupcall

		id     uuid.UUID
		status groupcall.Status

		responseCurTime string
		expectRes       *groupcall.Groupcall
	}{
		{
			name: "normal",
			data: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ee34fa3e-e123-11ed-92bf-a3c23e7dcb96"),
				},
			},

			id:     uuid.FromStringOrNil("ee34fa3e-e123-11ed-92bf-a3c23e7dcb96"),
			status: groupcall.StatusProgressing,

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ee34fa3e-e123-11ed-92bf-a3c23e7dcb96"),
				},
				Status: groupcall.StatusProgressing,

				Source:       &commonaddress.Address{},
				Destinations: []commonaddress.Address{},
				CallIDs:      []uuid.UUID{},
				GroupcallIDs: []uuid.UUID{},

				TMCreate: "2023-01-18 03:22:18.995000",
				TMUpdate: "2023-01-18 03:22:18.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any()).Return(nil)
			if errCreate := h.GroupcallCreate(ctx, tt.data); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
			if errSet := h.GroupcallSetStatus(ctx, tt.id, tt.status); errSet != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errSet)
			}

			mockCache.EXPECT().GroupcallGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
			res, err := h.GroupcallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_GroupcallSetCallIDsAndCallCountAndDialIndex(t *testing.T) {

	tests := []struct {
		name string
		data *groupcall.Groupcall

		id        uuid.UUID
		status    groupcall.Status
		callIDs   []uuid.UUID
		callCount int
		dialIndex int

		responseCurTime string
		expectRes       *groupcall.Groupcall
	}{
		{
			name: "normal",
			data: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e8c41e5a-e127-11ed-836a-4777b4880b93"),
				},
			},

			id: uuid.FromStringOrNil("e8c41e5a-e127-11ed-836a-4777b4880b93"),
			callIDs: []uuid.UUID{
				uuid.FromStringOrNil("01d039c4-e128-11ed-a7d6-8f88e8210e5a"),
				uuid.FromStringOrNil("02117f60-e128-11ed-a437-5f67f2f69ca2"),
			},
			callCount: 2,
			dialIndex: 3,

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e8c41e5a-e127-11ed-836a-4777b4880b93"),
				},

				Source:       &commonaddress.Address{},
				Destinations: []commonaddress.Address{},
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("01d039c4-e128-11ed-a7d6-8f88e8210e5a"),
					uuid.FromStringOrNil("02117f60-e128-11ed-a437-5f67f2f69ca2"),
				},
				GroupcallIDs: []uuid.UUID{},
				CallCount:    2,
				DialIndex:    3,

				TMCreate: "2023-01-18 03:22:18.995000",
				TMUpdate: "2023-01-18 03:22:18.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any()).Return(nil)
			if errCreate := h.GroupcallCreate(ctx, tt.data); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
			if errSet := h.GroupcallSetCallIDsAndCallCountAndDialIndex(ctx, tt.id, tt.callIDs, tt.callCount, tt.dialIndex); errSet != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errSet)
			}

			mockCache.EXPECT().GroupcallGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
			res, err := h.GroupcallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_GroupcallGets(t *testing.T) {

	type test struct {
		name       string
		groupcalls []*groupcall.Groupcall

		filters map[groupcall.Field]any

		responseCurTime string

		expectRes []*groupcall.Groupcall
	}

	tests := []test{
		{
			"normal",
			[]*groupcall.Groupcall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("fc555ffa-aef0-11ee-b94a-43b5d70aac44"),
						CustomerID: uuid.FromStringOrNil("fc86423c-aef0-11ee-a9c6-f7e96941fc95"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("fcb43250-aef0-11ee-9df0-db02730d68b3"),
						CustomerID: uuid.FromStringOrNil("fc86423c-aef0-11ee-a9c6-f7e96941fc95"),
					},
				},
			},

			map[groupcall.Field]any{
				groupcall.FieldCustomerID: uuid.FromStringOrNil("fc86423c-aef0-11ee-a9c6-f7e96941fc95"),
				groupcall.FieldDeleted:    false,
			},

			"2020-04-18 03:22:17.995000",

			[]*groupcall.Groupcall{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("fc555ffa-aef0-11ee-b94a-43b5d70aac44"),
						CustomerID: uuid.FromStringOrNil("fc86423c-aef0-11ee-a9c6-f7e96941fc95"),
					},

					Source:       &commonaddress.Address{},
					Destinations: []commonaddress.Address{},
					CallIDs:      []uuid.UUID{},
					GroupcallIDs: []uuid.UUID{},

					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("fcb43250-aef0-11ee-9df0-db02730d68b3"),
						CustomerID: uuid.FromStringOrNil("fc86423c-aef0-11ee-a9c6-f7e96941fc95"),
					},

					Source:       &commonaddress.Address{},
					Destinations: []commonaddress.Address{},
					CallIDs:      []uuid.UUID{},
					GroupcallIDs: []uuid.UUID{},

					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
			},
		},
		{
			"empty",
			[]*groupcall.Groupcall{},

			map[groupcall.Field]any{
				groupcall.FieldCustomerID: uuid.FromStringOrNil("fce1b9aa-aef0-11ee-b858-6ff6c7db63ee"),
				groupcall.FieldDeleted:    false,
			},

			"2020-04-18 03:22:17.995000",
			[]*groupcall.Groupcall{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// creates calls for test
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for _, gc := range tt.groupcalls {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().GroupcallSet(ctx, gomock.Any())
				_ = h.GroupcallCreate(ctx, gc)
			}

			res, err := h.GroupcallGets(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
