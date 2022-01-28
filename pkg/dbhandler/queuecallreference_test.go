package dbhandler

import (
	context "context"
	"fmt"
	reflect "reflect"
	"testing"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecallreference"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/cachehandler"
)

func TestQueuecallReferenceCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		data      *queuecallreference.QueuecallReference
		expectRes *queuecallreference.QueuecallReference
	}{
		{
			"normal",
			&queuecallreference.QueuecallReference{
				ID:           uuid.FromStringOrNil("93aeb602-6016-11ec-909b-d30e283ea6cb"),
				CustomerID:   uuid.FromStringOrNil("2d2c0a14-7ffc-11ec-a6f5-ef525309e5e6"),
				Type:         queuecall.ReferenceTypeCall,
				QueuecallIDs: []uuid.UUID{},
				TMCreate:     DefaultTimeStamp,
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
			},
			&queuecallreference.QueuecallReference{
				ID:           uuid.FromStringOrNil("93aeb602-6016-11ec-909b-d30e283ea6cb"),
				CustomerID:   uuid.FromStringOrNil("2d2c0a14-7ffc-11ec-a6f5-ef525309e5e6"),
				Type:         queuecall.ReferenceTypeCall,
				QueuecallIDs: []uuid.UUID{},
				TMCreate:     DefaultTimeStamp,
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().QueuecallReferenceSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallReferenceGet(gomock.Any(), tt.data.ID).Return(nil, fmt.Errorf("")).AnyTimes()

			if err := h.QueuecallReferenceCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallReferenceGet(ctx, tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestQueuecallReferenceDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	tests := []struct {
		name string

		id uuid.UUID

		data      *queuecallreference.QueuecallReference
		expectRes *queuecallreference.QueuecallReference
	}{
		{
			"test normal",

			uuid.FromStringOrNil("447196b0-762b-11ec-9010-375c278aee23"),

			&queuecallreference.QueuecallReference{
				ID:           uuid.FromStringOrNil("447196b0-762b-11ec-9010-375c278aee23"),
				CustomerID:   uuid.FromStringOrNil("378a2662-7ffc-11ec-8b4b-9b0bb8c4670a"),
				Type:         queuecall.ReferenceTypeCall,
				QueuecallIDs: []uuid.UUID{},
				TMCreate:     DefaultTimeStamp,
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
			},
			&queuecallreference.QueuecallReference{
				ID:           uuid.FromStringOrNil("447196b0-762b-11ec-9010-375c278aee23"),
				CustomerID:   uuid.FromStringOrNil("378a2662-7ffc-11ec-8b4b-9b0bb8c4670a"),
				Type:         queuecall.ReferenceTypeCall,
				QueuecallIDs: []uuid.UUID{},
				TMCreate:     DefaultTimeStamp,
				TMUpdate:     DefaultTimeStamp,
				TMDelete:     DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().QueuecallReferenceSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueuecallReferenceGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueuecallReferenceCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			err := h.QueuecallReferenceDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueuecallReferenceGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
			}
			if res.TMDelete == "" {
				t.Errorf("Wrong match. expect: not empty, got: empty")
			}

			tt.expectRes.TMCreate = res.TMCreate
			tt.expectRes.TMUpdate = res.TMUpdate
			tt.expectRes.TMDelete = res.TMDelete
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// func TestQueuecallGets(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockCache := cachehandler.NewMockCacheHandler(mc)
// 	h := NewHandler(dbTest, mockCache)

// 	tests := []struct {
// 		name      string
// 		userID    uint64
// 		data      []*queuecall.Queuecall
// 		size      uint64
// 		expectRes []*queuecall.Queuecall
// 	}{
// 		{
// 			"normal",
// 			11,
// 			[]*queuecall.Queuecall{
// 				{
// 					ID:            uuid.FromStringOrNil("a8818302-5a7b-11ec-b948-a3d1ac87eeea"),
// 					UserID:        11,
// 					ReferenceType: queuecall.ReferenceTypeCall,
// 					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
// 					TMCreate:      "2020-04-18T03:22:17.995000",
// 				},
// 				{
// 					ID:            uuid.FromStringOrNil("a8d4bff4-5a7b-11ec-b7f7-a3905f3d70e9"),
// 					UserID:        11,
// 					ReferenceType: queuecall.ReferenceTypeCall,
// 					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
// 					TMCreate:      "2020-04-18T03:22:17.994000",
// 				},
// 			},
// 			2,
// 			[]*queuecall.Queuecall{
// 				{
// 					ID:            uuid.FromStringOrNil("a8818302-5a7b-11ec-b948-a3d1ac87eeea"),
// 					UserID:        11,
// 					ReferenceType: queuecall.ReferenceTypeCall,
// 					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
// 					Source:        cmaddress.Address{},
// 					TagIDs:        []uuid.UUID{},

// 					TMCreate: "2020-04-18T03:22:17.995000",
// 				},
// 				{
// 					ID:            uuid.FromStringOrNil("a8d4bff4-5a7b-11ec-b7f7-a3905f3d70e9"),
// 					UserID:        11,
// 					ReferenceType: queuecall.ReferenceTypeCall,
// 					ReferenceID:   uuid.FromStringOrNil("c77f9fbe-5a7b-11ec-9191-97cb390509e2"),
// 					Source:        cmaddress.Address{},
// 					TagIDs:        []uuid.UUID{},

// 					TMCreate: "2020-04-18T03:22:17.994000",
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			ctx := context.Background()

// 			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
// 			for _, u := range tt.data {
// 				if err := h.QueuecallCreate(ctx, u); err != nil {
// 					t.Errorf("Wrong match. expect: ok, got: %v", err)
// 				}
// 			}

// 			res, err := h.QueuecallGets(ctx, tt.userID, tt.size, GetCurTime())
// 			if err != nil {
// 				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
// 			}

// 			if reflect.DeepEqual(tt.expectRes, res) == false {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
// 			}
// 		})
// 	}
// }

// func TestQueuecallGetsByReferenceID(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockCache := cachehandler.NewMockCacheHandler(mc)
// 	h := NewHandler(dbTest, mockCache)

// 	tests := []struct {
// 		name      string
// 		callID    uuid.UUID
// 		data      []*queuecall.Queuecall
// 		size      uint64
// 		expectRes []*queuecall.Queuecall
// 	}{
// 		{
// 			"normal",
// 			uuid.FromStringOrNil("66bb474c-5ab6-11ec-9636-0b7539c651ac"),
// 			[]*queuecall.Queuecall{
// 				{
// 					ID:            uuid.FromStringOrNil("66efdcbe-5ab6-11ec-953e-37514d9c9cb6"),
// 					UserID:        11,
// 					ReferenceType: queuecall.ReferenceTypeCall,
// 					ReferenceID:   uuid.FromStringOrNil("66bb474c-5ab6-11ec-9636-0b7539c651ac"),
// 					TMCreate:      "2020-04-18T03:22:17.995000",
// 				},
// 				{
// 					ID:            uuid.FromStringOrNil("671bc14e-5ab6-11ec-9c00-6b261a1357c1"),
// 					UserID:        11,
// 					ReferenceType: queuecall.ReferenceTypeCall,
// 					ReferenceID:   uuid.FromStringOrNil("ee77af9c-5c85-11ec-bac4-d3a4e453df6e"),
// 					TMCreate:      "2020-04-18T03:22:17.994000",
// 				},
// 			},
// 			2,
// 			[]*queuecall.Queuecall{
// 				{
// 					ID:            uuid.FromStringOrNil("66efdcbe-5ab6-11ec-953e-37514d9c9cb6"),
// 					UserID:        11,
// 					ReferenceType: queuecall.ReferenceTypeCall,
// 					ReferenceID:   uuid.FromStringOrNil("66bb474c-5ab6-11ec-9636-0b7539c651ac"),
// 					Source:        cmaddress.Address{},
// 					TagIDs:        []uuid.UUID{},
// 					TMCreate:      "2020-04-18T03:22:17.995000",
// 				},
// 			},
// 		},
// 		{
// 			"2 queuecalls",
// 			uuid.FromStringOrNil("2690cc32-5ab6-11ec-b445-27ad1e0a543a"),
// 			[]*queuecall.Queuecall{
// 				{
// 					ID:            uuid.FromStringOrNil("2766dfd4-5ab6-11ec-afc2-f3a1937a9b34"),
// 					UserID:        11,
// 					ReferenceType: queuecall.ReferenceTypeCall,
// 					ReferenceID:   uuid.FromStringOrNil("2690cc32-5ab6-11ec-b445-27ad1e0a543a"),
// 					TMCreate:      "2020-04-18T03:22:17.995000",
// 				},
// 				{
// 					ID:            uuid.FromStringOrNil("279110b0-5ab6-11ec-afb8-e311840c5826"),
// 					UserID:        11,
// 					ReferenceType: queuecall.ReferenceTypeCall,
// 					ReferenceID:   uuid.FromStringOrNil("2690cc32-5ab6-11ec-b445-27ad1e0a543a"),
// 					TMCreate:      "2020-04-18T03:22:17.994000",
// 				},
// 			},
// 			2,
// 			[]*queuecall.Queuecall{
// 				{
// 					ID:            uuid.FromStringOrNil("2766dfd4-5ab6-11ec-afc2-f3a1937a9b34"),
// 					UserID:        11,
// 					ReferenceType: queuecall.ReferenceTypeCall,
// 					ReferenceID:   uuid.FromStringOrNil("2690cc32-5ab6-11ec-b445-27ad1e0a543a"),
// 					Source:        cmaddress.Address{},
// 					TagIDs:        []uuid.UUID{},
// 					TMCreate:      "2020-04-18T03:22:17.995000",
// 				},
// 				{
// 					ID:            uuid.FromStringOrNil("279110b0-5ab6-11ec-afb8-e311840c5826"),
// 					UserID:        11,
// 					ReferenceType: queuecall.ReferenceTypeCall,
// 					ReferenceID:   uuid.FromStringOrNil("2690cc32-5ab6-11ec-b445-27ad1e0a543a"),
// 					Source:        cmaddress.Address{},
// 					TagIDs:        []uuid.UUID{},
// 					TMCreate:      "2020-04-18T03:22:17.994000",
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			ctx := context.Background()

// 			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
// 			for _, u := range tt.data {
// 				if err := h.QueuecallCreate(ctx, u); err != nil {
// 					t.Errorf("Wrong match. expect: ok, got: %v", err)
// 				}
// 			}

// 			res, err := h.QueuecallGetsByReferenceID(ctx, tt.callID)
// 			if err != nil {
// 				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
// 			}

// 			if reflect.DeepEqual(tt.expectRes, res) == false {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
// 			}
// 		})
// 	}
// }

// func TestQueuecallDelete(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockCache := cachehandler.NewMockCacheHandler(mc)

// 	tests := []struct {
// 		name string

// 		id uuid.UUID

// 		data *queuecall.Queuecall

// 		expectRes *queuecall.Queuecall
// 	}{
// 		{
// 			"test normal",

// 			uuid.FromStringOrNil("240779f6-5ab7-11ec-8993-a74ac488bded"),

// 			&queuecall.Queuecall{
// 				ID:     uuid.FromStringOrNil("240779f6-5ab7-11ec-8993-a74ac488bded"),
// 				UserID: 1,
// 			},

// 			&queuecall.Queuecall{
// 				ID:     uuid.FromStringOrNil("240779f6-5ab7-11ec-8993-a74ac488bded"),
// 				UserID: 1,
// 				Source: cmaddress.Address{},
// 				TagIDs: []uuid.UUID{},
// 				Status: queuecall.StatusDone,
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			ctx := context.Background()

// 			h := NewHandler(dbTest, mockCache)

// 			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
// 			mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
// 			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			err := h.QueuecallDelete(ctx, tt.id, queuecall.StatusDone)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			res, err := h.QueuecallGet(ctx, tt.id)
// 			if err != nil {
// 				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
// 			}
// 			if res.TMDelete == "" {
// 				t.Errorf("Wrong match. expect: not empty, got: empty")
// 			}

// 			res.TMCreate = ""
// 			tt.expectRes.TMUpdate = res.TMUpdate
// 			tt.expectRes.TMDelete = res.TMDelete
// 			if reflect.DeepEqual(tt.expectRes, res) == false {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }

// func TestQueuecallSetServiceAgentID(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockCache := cachehandler.NewMockCacheHandler(mc)

// 	tests := []struct {
// 		name string

// 		id             uuid.UUID
// 		serviceAgentID uuid.UUID

// 		data *queuecall.Queuecall

// 		expectRes *queuecall.Queuecall
// 	}{
// 		{
// 			"normal",

// 			uuid.FromStringOrNil("7f82cb36-5ab8-11ec-9c95-5bb7be87064f"),
// 			uuid.FromStringOrNil("85b89f08-5ab8-11ec-94ea-5bed0069b7e9"),

// 			&queuecall.Queuecall{
// 				ID:     uuid.FromStringOrNil("7f82cb36-5ab8-11ec-9c95-5bb7be87064f"),
// 				UserID: 1,
// 			},

// 			&queuecall.Queuecall{
// 				ID:             uuid.FromStringOrNil("7f82cb36-5ab8-11ec-9c95-5bb7be87064f"),
// 				UserID:         1,
// 				Status:         queuecall.StatusService,
// 				Source:         cmaddress.Address{},
// 				TagIDs:         []uuid.UUID{},
// 				ServiceAgentID: uuid.FromStringOrNil("85b89f08-5ab8-11ec-94ea-5bed0069b7e9"),
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			ctx := context.Background()

// 			h := NewHandler(dbTest, mockCache)

// 			mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
// 			mockCache.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
// 			if err := h.QueuecallCreate(ctx, tt.data); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			err := h.QueuecallSetServiceAgentID(ctx, tt.id, tt.serviceAgentID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			res, err := h.QueuecallGet(ctx, tt.id)
// 			if err != nil {
// 				t.Errorf("Wrong match.\nexpect: ok\ngot: %v\n", err)
// 			}
// 			if res.TMService == "" || res.TMUpdate == "" {
// 				t.Errorf("Wrong match. expect: not empty, got: empty")
// 			}

// 			tt.expectRes.TMUpdate = res.TMUpdate
// 			tt.expectRes.TMService = res.TMService
// 			if reflect.DeepEqual(tt.expectRes, res) == false {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }
