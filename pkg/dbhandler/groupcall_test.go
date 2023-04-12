package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
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
				ID:         uuid.FromStringOrNil("39ee40d7-9f83-45bb-ba29-7bb9de62c93e"),
				CustomerID: uuid.FromStringOrNil("a8eaeb80-bd76-11ed-94db-7fe899d03ca7"),

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
					uuid.FromStringOrNil("a9127fec-bd76-11ed-9c0d-f79292b7b2a6"),
					uuid.FromStringOrNil("a939559a-bd76-11ed-ac28-4f58c3ed30f3"),
				},
				CallCount: 2,
			},

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &groupcall.Groupcall{
				ID:         uuid.FromStringOrNil("39ee40d7-9f83-45bb-ba29-7bb9de62c93e"),
				CustomerID: uuid.FromStringOrNil("a8eaeb80-bd76-11ed-94db-7fe899d03ca7"),

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
					uuid.FromStringOrNil("a9127fec-bd76-11ed-9c0d-f79292b7b2a6"),
					uuid.FromStringOrNil("a939559a-bd76-11ed-ac28-4f58c3ed30f3"),
				},
				CallCount: 2,
				TMCreate:  "2023-01-18 03:22:18.995000",
				TMUpdate:  DefaultTimeStamp,
				TMDelete:  DefaultTimeStamp,
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

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
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
				ID:         uuid.FromStringOrNil("feaf81a6-bd77-11ed-bd82-cba4c20d3477"),
				CustomerID: uuid.FromStringOrNil("a8eaeb80-bd76-11ed-94db-7fe899d03ca7"),

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
				ID:         uuid.FromStringOrNil("feaf81a6-bd77-11ed-bd82-cba4c20d3477"),
				CustomerID: uuid.FromStringOrNil("a8eaeb80-bd76-11ed-94db-7fe899d03ca7"),

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

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any()).Return(nil)
			if errCreate := h.GroupcallCreate(ctx, tt.data); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
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
				ID:         uuid.FromStringOrNil("694c2b84-d913-11ed-82ca-8ffe9f085634"),
				CustomerID: uuid.FromStringOrNil("a8eaeb80-bd76-11ed-94db-7fe899d03ca7"),

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
				CallCount: 2,
			},

			id: uuid.FromStringOrNil("694c2b84-d913-11ed-82ca-8ffe9f085634"),

			responseCurTime: "2023-01-18 03:22:18.995000",
			expectRes: &groupcall.Groupcall{
				ID:         uuid.FromStringOrNil("694c2b84-d913-11ed-82ca-8ffe9f085634"),
				CustomerID: uuid.FromStringOrNil("a8eaeb80-bd76-11ed-94db-7fe899d03ca7"),

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
				CallCount: 1,

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

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().GroupcallSet(ctx, gomock.Any()).Return(nil)
			if errCreate := h.GroupcallCreate(ctx, tt.data); errCreate != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
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
