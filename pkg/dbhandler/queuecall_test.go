package dbhandler

import (
	context "context"
	"fmt"
	reflect "reflect"
	"testing"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/cachehandler"
)

func Test_QueuecallCreate(t *testing.T) {

	tests := []struct {
		name string

		data *queuecall.Queuecall

		responseCurTime string
		expectRes       *queuecall.Queuecall
	}{
		{
			"have all",
			&queuecall.Queuecall{
				ID:                    uuid.FromStringOrNil("a90f81ba-5e5a-11ec-be17-5fbb9796c693"),
				CustomerID:            uuid.FromStringOrNil("9a965e7c-7f54-11ec-941d-f3ea2575f0b4"),
				QueueID:               uuid.FromStringOrNil("9b75a91c-5e5a-11ec-883b-ab05ca15277b"),
				ReferenceType:         queuecall.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("a875b472-5e5a-11ec-9467-8f2c600000f3"),
				ReferenceActiveflowID: uuid.FromStringOrNil("cea3d3a1-e5ff-4f9f-9738-94600767f8fd"),
				ForwardActionID:       uuid.FromStringOrNil("a89d0acc-5e5a-11ec-8f3b-274070e9fa26"),
				ExitActionID:          uuid.FromStringOrNil("a8bd43fa-5e5a-11ec-8e43-236c955d6691"),
				ConfbridgeID:          uuid.FromStringOrNil("a8dca420-5e5a-11ec-87e3-eff5c9e3d170"),
				Source:                commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821021656521"},
				RoutingMethod:         queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a8f7abf8-5e5a-11ec-b03a-0f722823a0ca"),
					uuid.FromStringOrNil("af30d0cf-e49a-49f6-b2e2-d1daeaa38556"),
				},
				Status:          queuecall.StatusWaiting,
				ServiceAgentID:  uuid.FromStringOrNil("4ea9c07d-fa5a-45e9-9056-5a7be14c634c"),
				TimeoutWait:     60000,
				TimeoutService:  50000,
				DurationWaiting: 40000,
				DurationService: 30000,
			},

			"2023-02-15 03:22:17.994000",
			&queuecall.Queuecall{
				ID:                    uuid.FromStringOrNil("a90f81ba-5e5a-11ec-be17-5fbb9796c693"),
				CustomerID:            uuid.FromStringOrNil("9a965e7c-7f54-11ec-941d-f3ea2575f0b4"),
				QueueID:               uuid.FromStringOrNil("9b75a91c-5e5a-11ec-883b-ab05ca15277b"),
				ReferenceType:         queuecall.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("a875b472-5e5a-11ec-9467-8f2c600000f3"),
				ReferenceActiveflowID: uuid.FromStringOrNil("cea3d3a1-e5ff-4f9f-9738-94600767f8fd"),
				ForwardActionID:       uuid.FromStringOrNil("a89d0acc-5e5a-11ec-8f3b-274070e9fa26"),
				ExitActionID:          uuid.FromStringOrNil("a8bd43fa-5e5a-11ec-8e43-236c955d6691"),
				ConfbridgeID:          uuid.FromStringOrNil("a8dca420-5e5a-11ec-87e3-eff5c9e3d170"),
				Source:                commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821021656521"},
				RoutingMethod:         queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a8f7abf8-5e5a-11ec-b03a-0f722823a0ca"),
					uuid.FromStringOrNil("af30d0cf-e49a-49f6-b2e2-d1daeaa38556"),
				},
				Status:          queuecall.StatusWaiting,
				ServiceAgentID:  uuid.FromStringOrNil("4ea9c07d-fa5a-45e9-9056-5a7be14c634c"),
				TimeoutWait:     60000,
				TimeoutService:  50000,
				DurationWaiting: 40000,
				DurationService: 30000,
				TMCreate:        "2023-02-15 03:22:17.994000",
				TMService:       DefaultTimeStamp,
				TMUpdate:        DefaultTimeStamp,
				TMEnd:           DefaultTimeStamp,
				TMDelete:        DefaultTimeStamp,
			},
		},
		{
			"empty",
			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("e09db874-7686-11ec-8245-27e9d4dad47d"),
			},

			"2023-02-15 03:22:17.994000",
			&queuecall.Queuecall{
				ID:        uuid.FromStringOrNil("e09db874-7686-11ec-8245-27e9d4dad47d"),
				TagIDs:    []uuid.UUID{},
				TMCreate:  "2023-02-15 03:22:17.994000",
				TMService: DefaultTimeStamp,
				TMUpdate:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,
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
			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallGet(gomock.Any(), tt.data.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallGet(ctx, tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestQueuecallGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string
		data []*queuecall.Queuecall

		customerID uuid.UUID
		size       uint64

		responseCurTime string
		expectRes       []*queuecall.Queuecall
	}{
		{
			"normal",
			[]*queuecall.Queuecall{
				{
					ID:            uuid.FromStringOrNil("a8818302-5a7b-11ec-b948-a3d1ac87eeea"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
				},
				{
					ID:            uuid.FromStringOrNil("a8d4bff4-5a7b-11ec-b7f7-a3905f3d70e9"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
				},
			},

			uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
			2,

			"2023-02-14 03:22:17.994000",
			[]*queuecall.Queuecall{
				{
					ID:            uuid.FromStringOrNil("a8818302-5a7b-11ec-b948-a3d1ac87eeea"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
					Source:        commonaddress.Address{},
					TagIDs:        []uuid.UUID{},

					TMCreate:  "2023-02-14 03:22:17.994000",
					TMService: DefaultTimeStamp,
					TMUpdate:  DefaultTimeStamp,
					TMEnd:     DefaultTimeStamp,
					TMDelete:  DefaultTimeStamp,
				},
				{
					ID:            uuid.FromStringOrNil("a8d4bff4-5a7b-11ec-b7f7-a3905f3d70e9"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
					Source:        commonaddress.Address{},
					TagIDs:        []uuid.UUID{},

					TMCreate:  "2023-02-14 03:22:17.994000",
					TMService: DefaultTimeStamp,
					TMUpdate:  DefaultTimeStamp,
					TMEnd:     DefaultTimeStamp,
					TMDelete:  DefaultTimeStamp,
				},
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

			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			for _, u := range tt.data {
				mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
				if err := h.QueuecallCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.QueuecallGetsByCustomerID(ctx, tt.customerID, tt.size, utilhandler.GetCurTime())
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func TestQueuecallGetsByReferenceID(t *testing.T) {

	tests := []struct {
		name string
		data []*queuecall.Queuecall

		callID uuid.UUID
		size   uint64

		responseCurTime string
		expectRes       []*queuecall.Queuecall
	}{
		{
			"normal",
			[]*queuecall.Queuecall{
				{
					ID:            uuid.FromStringOrNil("66efdcbe-5ab6-11ec-953e-37514d9c9cb6"),
					CustomerID:    uuid.FromStringOrNil("c4002036-7f54-11ec-a49e-2fcd8498a049"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("66bb474c-5ab6-11ec-9636-0b7539c651ac"),
					TMCreate:      "2020-04-18T03:22:17.995000",
				},
				{
					ID:            uuid.FromStringOrNil("671bc14e-5ab6-11ec-9c00-6b261a1357c1"),
					CustomerID:    uuid.FromStringOrNil("c4002036-7f54-11ec-a49e-2fcd8498a049"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("ee77af9c-5c85-11ec-bac4-d3a4e453df6e"),
					TMCreate:      "2020-04-18T03:22:17.994000",
				},
			},

			uuid.FromStringOrNil("66bb474c-5ab6-11ec-9636-0b7539c651ac"),
			2,

			"2023-02-14 03:22:17.994000",
			[]*queuecall.Queuecall{
				{
					ID:            uuid.FromStringOrNil("66efdcbe-5ab6-11ec-953e-37514d9c9cb6"),
					CustomerID:    uuid.FromStringOrNil("c4002036-7f54-11ec-a49e-2fcd8498a049"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("66bb474c-5ab6-11ec-9636-0b7539c651ac"),
					Source:        commonaddress.Address{},
					TagIDs:        []uuid.UUID{},
					TMCreate:      "2023-02-14 03:22:17.994000",
					TMService:     DefaultTimeStamp,
					TMUpdate:      DefaultTimeStamp,
					TMEnd:         DefaultTimeStamp,
					TMDelete:      DefaultTimeStamp,
				},
			},
		},
		{
			"2 queuecalls",
			[]*queuecall.Queuecall{
				{
					ID:            uuid.FromStringOrNil("2766dfd4-5ab6-11ec-afc2-f3a1937a9b34"),
					CustomerID:    uuid.FromStringOrNil("c4002036-7f54-11ec-a49e-2fcd8498a049"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("2690cc32-5ab6-11ec-b445-27ad1e0a543a"),
					TMCreate:      "2020-04-18T03:22:17.995000",
				},
				{
					ID:            uuid.FromStringOrNil("279110b0-5ab6-11ec-afb8-e311840c5826"),
					CustomerID:    uuid.FromStringOrNil("c4002036-7f54-11ec-a49e-2fcd8498a049"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("2690cc32-5ab6-11ec-b445-27ad1e0a543a"),
					TMCreate:      "2020-04-18T03:22:17.994000",
				},
			},

			uuid.FromStringOrNil("2690cc32-5ab6-11ec-b445-27ad1e0a543a"),
			2,

			"2023-02-14 03:22:17.994000",
			[]*queuecall.Queuecall{
				{
					ID:            uuid.FromStringOrNil("2766dfd4-5ab6-11ec-afc2-f3a1937a9b34"),
					CustomerID:    uuid.FromStringOrNil("c4002036-7f54-11ec-a49e-2fcd8498a049"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("2690cc32-5ab6-11ec-b445-27ad1e0a543a"),
					Source:        commonaddress.Address{},
					TagIDs:        []uuid.UUID{},
					TMCreate:      "2023-02-14 03:22:17.994000",
					TMService:     DefaultTimeStamp,
					TMUpdate:      DefaultTimeStamp,
					TMEnd:         DefaultTimeStamp,
					TMDelete:      DefaultTimeStamp,
				},
				{
					ID:            uuid.FromStringOrNil("279110b0-5ab6-11ec-afb8-e311840c5826"),
					CustomerID:    uuid.FromStringOrNil("c4002036-7f54-11ec-a49e-2fcd8498a049"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("2690cc32-5ab6-11ec-b445-27ad1e0a543a"),
					Source:        commonaddress.Address{},
					TagIDs:        []uuid.UUID{},
					TMCreate:      "2023-02-14 03:22:17.994000",
					TMService:     DefaultTimeStamp,
					TMUpdate:      DefaultTimeStamp,
					TMEnd:         DefaultTimeStamp,
					TMDelete:      DefaultTimeStamp,
				},
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

			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			for _, u := range tt.data {
				mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
				if err := h.QueuecallCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.QueuecallGetsByReferenceID(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_QueuecallGetsByQueueIDAndStatus(t *testing.T) {

	tests := []struct {
		name string
		data []*queuecall.Queuecall

		queueID uuid.UUID
		status  queuecall.Status
		size    uint64

		responseCurTime string
		expectRes       []*queuecall.Queuecall
	}{
		{
			"normal",
			[]*queuecall.Queuecall{
				{
					ID:            uuid.FromStringOrNil("91f4affe-d13d-11ec-999d-1b5c2c4dfa2b"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					QueueID:       uuid.FromStringOrNil("76b0275a-d13d-11ec-9526-5f024469545f"),
					Status:        queuecall.StatusWaiting,
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
					TMCreate:      "2020-04-18T03:22:17.995000",
				},
				{
					ID:            uuid.FromStringOrNil("9221109e-d13d-11ec-99e7-d7ab2832c12b"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					QueueID:       uuid.FromStringOrNil("76b0275a-d13d-11ec-9526-5f024469545f"),
					Status:        queuecall.StatusWaiting,
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
					TMCreate:      "2020-04-18T03:22:17.994000",
				},
			},

			uuid.FromStringOrNil("76b0275a-d13d-11ec-9526-5f024469545f"),
			queuecall.StatusWaiting,
			2,

			"2023-02-14 03:22:17.994000",
			[]*queuecall.Queuecall{
				{
					ID:            uuid.FromStringOrNil("91f4affe-d13d-11ec-999d-1b5c2c4dfa2b"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					QueueID:       uuid.FromStringOrNil("76b0275a-d13d-11ec-9526-5f024469545f"),
					Status:        queuecall.StatusWaiting,
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
					Source:        commonaddress.Address{},
					TagIDs:        []uuid.UUID{},
					TMCreate:      "2023-02-14 03:22:17.994000",
					TMService:     DefaultTimeStamp,
					TMUpdate:      DefaultTimeStamp,
					TMEnd:         DefaultTimeStamp,
					TMDelete:      DefaultTimeStamp,
				},
				{
					ID:            uuid.FromStringOrNil("9221109e-d13d-11ec-99e7-d7ab2832c12b"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					QueueID:       uuid.FromStringOrNil("76b0275a-d13d-11ec-9526-5f024469545f"),
					Status:        queuecall.StatusWaiting,
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
					Source:        commonaddress.Address{},
					TagIDs:        []uuid.UUID{},
					TMCreate:      "2023-02-14 03:22:17.994000",
					TMService:     DefaultTimeStamp,
					TMUpdate:      DefaultTimeStamp,
					TMEnd:         DefaultTimeStamp,
					TMDelete:      DefaultTimeStamp,
				},
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

			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			for _, u := range tt.data {
				mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
				if err := h.QueuecallCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.QueuecallGetsByQueueIDAndStatus(ctx, tt.queueID, tt.status, tt.size, utilhandler.GetCurTime())
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_QueuecallDelete(t *testing.T) {

	tests := []struct {
		name string
		data *queuecall.Queuecall

		queuecallID uuid.UUID

		responseCurTime string
		expectRes       *queuecall.Queuecall
	}{
		{
			"test normal",
			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("240779f6-5ab7-11ec-8993-a74ac488bded"),
			},

			uuid.FromStringOrNil("240779f6-5ab7-11ec-8993-a74ac488bded"),

			"2023-02-14 03:22:17.994000",
			&queuecall.Queuecall{
				ID:        uuid.FromStringOrNil("240779f6-5ab7-11ec-8993-a74ac488bded"),
				Source:    commonaddress.Address{},
				TagIDs:    []uuid.UUID{},
				TMCreate:  "2023-02-14 03:22:17.994000",
				TMService: DefaultTimeStamp,
				TMUpdate:  "2023-02-14 03:22:17.994000",
				TMEnd:     DefaultTimeStamp,
				TMDelete:  "2023-02-14 03:22:17.994000",
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
			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			err := h.QueuecallDelete(ctx, tt.queuecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallGet(ctx, tt.queuecallID)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueuecallSetStatusConnecting(t *testing.T) {

	tests := []struct {
		name string
		data *queuecall.Queuecall

		id             uuid.UUID
		serviceAgentID uuid.UUID

		responseCurTime string
		expectRes       *queuecall.Queuecall
	}{
		{
			"normal",
			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("7f82cb36-5ab8-11ec-9c95-5bb7be87064f"),
			},

			uuid.FromStringOrNil("7f82cb36-5ab8-11ec-9c95-5bb7be87064f"),
			uuid.FromStringOrNil("85b89f08-5ab8-11ec-94ea-5bed0069b7e9"),

			"2023-02-14 03:22:17.994000",
			&queuecall.Queuecall{
				ID:             uuid.FromStringOrNil("7f82cb36-5ab8-11ec-9c95-5bb7be87064f"),
				Status:         queuecall.StatusConnecting,
				Source:         commonaddress.Address{},
				TagIDs:         []uuid.UUID{},
				ServiceAgentID: uuid.FromStringOrNil("85b89f08-5ab8-11ec-94ea-5bed0069b7e9"),
				TMCreate:       "2023-02-14 03:22:17.994000",
				TMService:      DefaultTimeStamp,
				TMUpdate:       "2023-02-14 03:22:17.994000",
				TMEnd:          DefaultTimeStamp,
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

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			err := h.QueuecallSetStatusConnecting(ctx, tt.id, tt.serviceAgentID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			tt.expectRes.TMService = res.TMService
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueuecallSetStatusService(t *testing.T) {

	tests := []struct {
		name string
		data *queuecall.Queuecall

		id              uuid.UUID
		durationWaiting int
		timestamp       string

		responseCurTime string
		expectRes       *queuecall.Queuecall
	}{
		{
			"normal",
			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("6eddc614-7624-11ec-a537-a358ff836d91"),
			},

			uuid.FromStringOrNil("6eddc614-7624-11ec-a537-a358ff836d91"),
			10000,
			"2023-02-14 03:22:17.994000",

			"2023-02-14 03:22:17.994000",
			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("6eddc614-7624-11ec-a537-a358ff836d91"),
				Status:          queuecall.StatusService,
				Source:          commonaddress.Address{},
				TagIDs:          []uuid.UUID{},
				DurationWaiting: 10000,
				TMCreate:        "2023-02-14 03:22:17.994000",
				TMUpdate:        "2023-02-14 03:22:17.994000",
				TMService:       "2023-02-14 03:22:17.994000",
				TMEnd:           DefaultTimeStamp,
				TMDelete:        DefaultTimeStamp,
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
			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			err := h.QueuecallSetStatusService(ctx, tt.id, tt.durationWaiting, tt.timestamp)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueuecallSetStatusKicking(t *testing.T) {
	tests := []struct {
		name string
		data *queuecall.Queuecall

		id uuid.UUID

		responseCurTime string
		expectRes       *queuecall.Queuecall
	}{
		{
			"normal",
			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("97222dd6-8a15-11ec-9cb1-eba575c6b180"),
			},

			uuid.FromStringOrNil("97222dd6-8a15-11ec-9cb1-eba575c6b180"),

			"2023-02-14 03:22:17.994000",
			&queuecall.Queuecall{
				ID:        uuid.FromStringOrNil("97222dd6-8a15-11ec-9cb1-eba575c6b180"),
				Status:    queuecall.StatusKicking,
				Source:    commonaddress.Address{},
				TagIDs:    []uuid.UUID{},
				TMCreate:  "2023-02-14 03:22:17.994000",
				TMUpdate:  "2023-02-14 03:22:17.994000",
				TMService: DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,
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
			mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil)
			err := h.QueuecallSetStatusKicking(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.QueuecallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueuecallSetStatusAbandoned(t *testing.T) {

	tests := []struct {
		name string
		data *queuecall.Queuecall

		id              uuid.UUID
		durationWaiting int
		timestamp       string

		responseCurTime string
		expectRes       *queuecall.Queuecall
	}{
		{
			"normal",
			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("f3fce82c-518f-4fe9-ac78-d7b89c57c433"),
			},

			uuid.FromStringOrNil("f3fce82c-518f-4fe9-ac78-d7b89c57c433"),
			10000,
			"2023-02-14 03:22:17.994000",

			"2023-02-14 03:22:17.994000",
			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("f3fce82c-518f-4fe9-ac78-d7b89c57c433"),
				Status:          queuecall.StatusAbandoned,
				Source:          commonaddress.Address{},
				TagIDs:          []uuid.UUID{},
				DurationWaiting: 10000,
				TMCreate:        "2023-02-14 03:22:17.994000",
				TMUpdate:        "2023-02-14 03:22:17.994000",
				TMService:       DefaultTimeStamp,
				TMEnd:           "2023-02-14 03:22:17.994000",
				TMDelete:        DefaultTimeStamp,
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
			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			err := h.QueuecallSetStatusAbandoned(ctx, tt.id, tt.durationWaiting, tt.timestamp)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueuecallSetStatusDone(t *testing.T) {

	tests := []struct {
		name string
		data *queuecall.Queuecall

		id              uuid.UUID
		durationWaiting int
		timestamp       string

		responseCurTime string
		expectRes       *queuecall.Queuecall
	}{
		{
			"normal",
			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("aae34fc9-e298-401d-bfd4-d99eff5d5a43"),
			},

			uuid.FromStringOrNil("aae34fc9-e298-401d-bfd4-d99eff5d5a43"),
			10000,
			"2023-02-14 03:22:17.994000",

			"2023-02-14 03:22:17.994000",
			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("aae34fc9-e298-401d-bfd4-d99eff5d5a43"),
				Status:          queuecall.StatusDone,
				Source:          commonaddress.Address{},
				TagIDs:          []uuid.UUID{},
				DurationService: 10000,
				TMCreate:        "2023-02-14 03:22:17.994000",
				TMUpdate:        "2023-02-14 03:22:17.994000",
				TMService:       DefaultTimeStamp,
				TMEnd:           "2023-02-14 03:22:17.994000",
				TMDelete:        DefaultTimeStamp,
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
			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			err := h.QueuecallSetStatusDone(ctx, tt.id, tt.durationWaiting, tt.timestamp)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueuecallGetByReferenceID(t *testing.T) {

	type test struct {
		name string
		data *queuecall.Queuecall

		referenceID uuid.UUID

		responseCurTime string
		expectRes       *queuecall.Queuecall
	}

	tests := []test{
		{
			"normal",
			&queuecall.Queuecall{
				ID:          uuid.FromStringOrNil("d6c7fbcb-e206-4997-aab5-b23038e8f39b"),
				ReferenceID: uuid.FromStringOrNil("2c7a4abb-35f0-44b0-a646-74b302fef9f0"),
			},

			uuid.FromStringOrNil("2c7a4abb-35f0-44b0-a646-74b302fef9f0"),

			"2023-01-03 21:35:02.809",
			&queuecall.Queuecall{
				ID:          uuid.FromStringOrNil("d6c7fbcb-e206-4997-aab5-b23038e8f39b"),
				ReferenceID: uuid.FromStringOrNil("2c7a4abb-35f0-44b0-a646-74b302fef9f0"),
				TagIDs:      []uuid.UUID{},
				TMCreate:    "2023-01-03 21:35:02.809",
				TMService:   DefaultTimeStamp,
				TMUpdate:    DefaultTimeStamp,
				TMEnd:       DefaultTimeStamp,
				TMDelete:    DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil)
			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().QueuecallGetByReferenceID(ctx, tt.referenceID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.QueuecallGetByReferenceID(ctx, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
