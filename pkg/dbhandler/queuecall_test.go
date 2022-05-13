package dbhandler

import (
	context "context"
	"fmt"
	reflect "reflect"
	"testing"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/cachehandler"
)

func Test_QueuecallCreate(t *testing.T) {

	tests := []struct {
		name string

		data      *queuecall.Queuecall
		expectRes *queuecall.Queuecall
	}{
		{
			"normal",
			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("a90f81ba-5e5a-11ec-be17-5fbb9796c693"),
				CustomerID:      uuid.FromStringOrNil("9a965e7c-7f54-11ec-941d-f3ea2575f0b4"),
				QueueID:         uuid.FromStringOrNil("9b75a91c-5e5a-11ec-883b-ab05ca15277b"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("a875b472-5e5a-11ec-9467-8f2c600000f3"),
				ForwardActionID: uuid.FromStringOrNil("a89d0acc-5e5a-11ec-8f3b-274070e9fa26"),
				ExitActionID:    uuid.FromStringOrNil("a8bd43fa-5e5a-11ec-8e43-236c955d6691"),
				ConfbridgeID:    uuid.FromStringOrNil("a8dca420-5e5a-11ec-87e3-eff5c9e3d170"),

				Source: cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a8f7abf8-5e5a-11ec-b03a-0f722823a0ca"),
				},

				Status:         queuecall.StatusWaiting,
				ServiceAgentID: [16]byte{},
				TMCreate:       DefaultTimeStamp,
				TMService:      DefaultTimeStamp,
				TMUpdate:       DefaultTimeStamp,
				TMDelete:       DefaultTimeStamp,
			},
			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("a90f81ba-5e5a-11ec-be17-5fbb9796c693"),
				CustomerID:      uuid.FromStringOrNil("9a965e7c-7f54-11ec-941d-f3ea2575f0b4"),
				QueueID:         uuid.FromStringOrNil("9b75a91c-5e5a-11ec-883b-ab05ca15277b"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("a875b472-5e5a-11ec-9467-8f2c600000f3"),
				ForwardActionID: uuid.FromStringOrNil("a89d0acc-5e5a-11ec-8f3b-274070e9fa26"),
				ExitActionID:    uuid.FromStringOrNil("a8bd43fa-5e5a-11ec-8e43-236c955d6691"),
				ConfbridgeID:    uuid.FromStringOrNil("a8dca420-5e5a-11ec-87e3-eff5c9e3d170"),

				Source: cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a8f7abf8-5e5a-11ec-b03a-0f722823a0ca"),
				},

				Status:         queuecall.StatusWaiting,
				ServiceAgentID: [16]byte{},
				TMCreate:       DefaultTimeStamp,
				TMService:      DefaultTimeStamp,
				TMUpdate:       DefaultTimeStamp,
				TMDelete:       DefaultTimeStamp,
			},
		},
		{
			"added flow_id",
			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("e09db874-7686-11ec-8245-27e9d4dad47d"),
				CustomerID:      uuid.FromStringOrNil("a3aa4d8e-7f54-11ec-a892-e73fa57b7129"),
				QueueID:         uuid.FromStringOrNil("9b75a91c-5e5a-11ec-883b-ab05ca15277b"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("a875b472-5e5a-11ec-9467-8f2c600000f3"),
				FlowID:          uuid.FromStringOrNil("e0c80b2e-7686-11ec-9eed-6f64a02073fb"),
				ForwardActionID: uuid.FromStringOrNil("a89d0acc-5e5a-11ec-8f3b-274070e9fa26"),
				ExitActionID:    uuid.FromStringOrNil("a8bd43fa-5e5a-11ec-8e43-236c955d6691"),
				ConfbridgeID:    uuid.FromStringOrNil("a8dca420-5e5a-11ec-87e3-eff5c9e3d170"),

				Source: cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a8f7abf8-5e5a-11ec-b03a-0f722823a0ca"),
				},

				Status:         queuecall.StatusWaiting,
				ServiceAgentID: [16]byte{},
				TMCreate:       DefaultTimeStamp,
				TMService:      DefaultTimeStamp,
				TMUpdate:       DefaultTimeStamp,
				TMDelete:       DefaultTimeStamp,
			},
			&queuecall.Queuecall{
				ID:              uuid.FromStringOrNil("e09db874-7686-11ec-8245-27e9d4dad47d"),
				CustomerID:      uuid.FromStringOrNil("a3aa4d8e-7f54-11ec-a892-e73fa57b7129"),
				QueueID:         uuid.FromStringOrNil("9b75a91c-5e5a-11ec-883b-ab05ca15277b"),
				ReferenceType:   queuecall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("a875b472-5e5a-11ec-9467-8f2c600000f3"),
				FlowID:          uuid.FromStringOrNil("e0c80b2e-7686-11ec-9eed-6f64a02073fb"),
				ForwardActionID: uuid.FromStringOrNil("a89d0acc-5e5a-11ec-8f3b-274070e9fa26"),
				ExitActionID:    uuid.FromStringOrNil("a8bd43fa-5e5a-11ec-8e43-236c955d6691"),
				ConfbridgeID:    uuid.FromStringOrNil("a8dca420-5e5a-11ec-87e3-eff5c9e3d170"),

				Source: cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a8f7abf8-5e5a-11ec-b03a-0f722823a0ca"),
				},

				Status:         queuecall.StatusWaiting,
				ServiceAgentID: [16]byte{},
				TMCreate:       DefaultTimeStamp,
				TMService:      DefaultTimeStamp,
				TMUpdate:       DefaultTimeStamp,
				TMDelete:       DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallGet(gomock.Any(), tt.data.ID).Return(nil, fmt.Errorf("")).AnyTimes()

			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallGet(ctx, tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestQueuecallGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name       string
		customerID uuid.UUID
		data       []*queuecall.Queuecall
		size       uint64
		expectRes  []*queuecall.Queuecall
	}{
		{
			"normal",
			uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
			[]*queuecall.Queuecall{
				{
					ID:            uuid.FromStringOrNil("a8818302-5a7b-11ec-b948-a3d1ac87eeea"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
					TMCreate:      "2020-04-18T03:22:17.995000",
				},
				{
					ID:            uuid.FromStringOrNil("a8d4bff4-5a7b-11ec-b7f7-a3905f3d70e9"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
					TMCreate:      "2020-04-18T03:22:17.994000",
				},
			},
			2,
			[]*queuecall.Queuecall{
				{
					ID:            uuid.FromStringOrNil("a8818302-5a7b-11ec-b948-a3d1ac87eeea"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
					Source:        cmaddress.Address{},
					TagIDs:        []uuid.UUID{},

					TMCreate: "2020-04-18T03:22:17.995000",
				},
				{
					ID:            uuid.FromStringOrNil("a8d4bff4-5a7b-11ec-b7f7-a3905f3d70e9"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
					Source:        cmaddress.Address{},
					TagIDs:        []uuid.UUID{},

					TMCreate: "2020-04-18T03:22:17.994000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			for _, u := range tt.data {
				if err := h.QueuecallCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.QueuecallGetsByCustomerID(ctx, tt.customerID, tt.size, GetCurTime())
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
		name      string
		callID    uuid.UUID
		data      []*queuecall.Queuecall
		size      uint64
		expectRes []*queuecall.Queuecall
	}{
		{
			"normal",
			uuid.FromStringOrNil("66bb474c-5ab6-11ec-9636-0b7539c651ac"),
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
			2,
			[]*queuecall.Queuecall{
				{
					ID:            uuid.FromStringOrNil("66efdcbe-5ab6-11ec-953e-37514d9c9cb6"),
					CustomerID:    uuid.FromStringOrNil("c4002036-7f54-11ec-a49e-2fcd8498a049"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("66bb474c-5ab6-11ec-9636-0b7539c651ac"),
					Source:        cmaddress.Address{},
					TagIDs:        []uuid.UUID{},
					TMCreate:      "2020-04-18T03:22:17.995000",
				},
			},
		},
		{
			"2 queuecalls",
			uuid.FromStringOrNil("2690cc32-5ab6-11ec-b445-27ad1e0a543a"),
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
			2,
			[]*queuecall.Queuecall{
				{
					ID:            uuid.FromStringOrNil("2766dfd4-5ab6-11ec-afc2-f3a1937a9b34"),
					CustomerID:    uuid.FromStringOrNil("c4002036-7f54-11ec-a49e-2fcd8498a049"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("2690cc32-5ab6-11ec-b445-27ad1e0a543a"),
					Source:        cmaddress.Address{},
					TagIDs:        []uuid.UUID{},
					TMCreate:      "2020-04-18T03:22:17.995000",
				},
				{
					ID:            uuid.FromStringOrNil("279110b0-5ab6-11ec-afb8-e311840c5826"),
					CustomerID:    uuid.FromStringOrNil("c4002036-7f54-11ec-a49e-2fcd8498a049"),
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("2690cc32-5ab6-11ec-b445-27ad1e0a543a"),
					Source:        cmaddress.Address{},
					TagIDs:        []uuid.UUID{},
					TMCreate:      "2020-04-18T03:22:17.994000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			for _, u := range tt.data {
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

		queueID uuid.UUID
		status  queuecall.Status

		data      []*queuecall.Queuecall
		size      uint64
		expectRes []*queuecall.Queuecall
	}{
		{
			"normal",
			uuid.FromStringOrNil("76b0275a-d13d-11ec-9526-5f024469545f"),
			queuecall.StatusWaiting,

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
			2,
			[]*queuecall.Queuecall{
				{
					ID:            uuid.FromStringOrNil("9221109e-d13d-11ec-99e7-d7ab2832c12b"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					QueueID:       uuid.FromStringOrNil("76b0275a-d13d-11ec-9526-5f024469545f"),
					Status:        queuecall.StatusWaiting,
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
					Source:        cmaddress.Address{},
					TagIDs:        []uuid.UUID{},
					TMCreate:      "2020-04-18T03:22:17.994000",
				},
				{
					ID:            uuid.FromStringOrNil("91f4affe-d13d-11ec-999d-1b5c2c4dfa2b"),
					CustomerID:    uuid.FromStringOrNil("ae49986c-7f54-11ec-9a36-3ff9622d2952"),
					QueueID:       uuid.FromStringOrNil("76b0275a-d13d-11ec-9526-5f024469545f"),
					Status:        queuecall.StatusWaiting,
					ReferenceType: queuecall.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
					Source:        cmaddress.Address{},
					TagIDs:        []uuid.UUID{},
					TMCreate:      "2020-04-18T03:22:17.995000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			for _, u := range tt.data {
				if err := h.QueuecallCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.QueuecallGetsByQueueIDAndStatus(ctx, tt.queueID, tt.status, tt.size, GetCurTime())
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func TestQueuecallDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		data *queuecall.Queuecall

		expectRes *queuecall.Queuecall
	}{
		{
			"test normal",

			uuid.FromStringOrNil("240779f6-5ab7-11ec-8993-a74ac488bded"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("240779f6-5ab7-11ec-8993-a74ac488bded"),
			},

			&queuecall.Queuecall{
				ID:     uuid.FromStringOrNil("240779f6-5ab7-11ec-8993-a74ac488bded"),
				Source: cmaddress.Address{},
				TagIDs: []uuid.UUID{},
				Status: queuecall.StatusDone,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			err := h.QueuecallDelete(ctx, tt.id, queuecall.StatusDone)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}
			if res.TMDelete == "" {
				t.Errorf("Wrong match. expect: not empty, got: empty")
			}

			res.TMCreate = ""
			tt.expectRes.TMUpdate = res.TMUpdate
			tt.expectRes.TMDelete = res.TMDelete
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestQueuecallSetStatusConnecting(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		serviceAgentID uuid.UUID

		data *queuecall.Queuecall

		expectRes *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("7f82cb36-5ab8-11ec-9c95-5bb7be87064f"),
			uuid.FromStringOrNil("85b89f08-5ab8-11ec-94ea-5bed0069b7e9"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("7f82cb36-5ab8-11ec-9c95-5bb7be87064f"),
			},

			&queuecall.Queuecall{
				ID:             uuid.FromStringOrNil("7f82cb36-5ab8-11ec-9c95-5bb7be87064f"),
				Status:         queuecall.StatusConnecting,
				Source:         cmaddress.Address{},
				TagIDs:         []uuid.UUID{},
				ServiceAgentID: uuid.FromStringOrNil("85b89f08-5ab8-11ec-94ea-5bed0069b7e9"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

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

func TestQueuecallSetStatusService(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		data *queuecall.Queuecall

		expectRes *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("6eddc614-7624-11ec-a537-a358ff836d91"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("6eddc614-7624-11ec-a537-a358ff836d91"),
			},

			&queuecall.Queuecall{
				ID:     uuid.FromStringOrNil("6eddc614-7624-11ec-a537-a358ff836d91"),
				Status: queuecall.StatusService,
				Source: cmaddress.Address{},
				TagIDs: []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			err := h.QueuecallSetStatusService(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}
			if res.TMService == "" || res.TMUpdate == "" {
				t.Errorf("Wrong match. expect: not empty, got: empty")
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			tt.expectRes.TMService = res.TMService
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestQueuecallSetStatusKicking(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		data *queuecall.Queuecall

		expectRes *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("97222dd6-8a15-11ec-9cb1-eba575c6b180"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("97222dd6-8a15-11ec-9cb1-eba575c6b180"),
			},

			&queuecall.Queuecall{
				ID:     uuid.FromStringOrNil("97222dd6-8a15-11ec-9cb1-eba575c6b180"),
				Status: queuecall.StatusKicking,
				Source: cmaddress.Address{},
				TagIDs: []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			err := h.QueuecallSetStatusKicking(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueuecallSetStatusWaiting(t *testing.T) {

	tests := []struct {
		name string

		queuecallID uuid.UUID

		data *queuecall.Queuecall

		expectRes *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("6f83cec0-d1d1-11ec-9aa4-1764ad2da6d5"),

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("6f83cec0-d1d1-11ec-9aa4-1764ad2da6d5"),
			},

			&queuecall.Queuecall{
				ID:     uuid.FromStringOrNil("6f83cec0-d1d1-11ec-9aa4-1764ad2da6d5"),
				Status: queuecall.StatusWaiting,
				Source: cmaddress.Address{},
				TagIDs: []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			err := h.QueuecallSetStatusWaiting(ctx, tt.queuecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallGet(ctx, tt.queuecallID)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
