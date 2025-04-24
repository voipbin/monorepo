package dbhandler

import (
	context "context"
	"fmt"
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/pkg/cachehandler"
)

func Test_QueueCreate(t *testing.T) {

	tests := []struct {
		name string

		queue *queue.Queue

		responseCurTime string
		expectRes       *queue.Queue
	}{
		{
			name: "normal",
			queue: &queue.Queue{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cba57fb6-59de-11ec-b230-5b6ab3380040"),
					CustomerID: uuid.FromStringOrNil("4fc7cef8-7f54-11ec-8e1f-6f6a91905190"),
				},
				Name:          "test name",
				Detail:        "test detail",
				RoutingMethod: "random",
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("e4368e4e-59de-11ec-badd-378688c95856"),
				},

				Execute: queue.ExecuteRun,

				WaitFlowID:     uuid.FromStringOrNil("4dfaf278-205d-11f0-8be0-d74aed2ef0bc"),
				WaitTimeout:    6000,
				ServiceTimeout: 60000,

				WaitQueuecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("4e43f680-205d-11f0-8d52-efe5c18633e3"),
				},
				ServiceQueuecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("4e687cbc-205d-11f0-9be2-c719e5802cf8"),
				},

				TotalIncomingCount:  0,
				TotalServicedCount:  0,
				TotalAbandonedCount: 0,
			},

			responseCurTime: "2023-02-15 03:22:17.994000",
			expectRes: &queue.Queue{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cba57fb6-59de-11ec-b230-5b6ab3380040"),
					CustomerID: uuid.FromStringOrNil("4fc7cef8-7f54-11ec-8e1f-6f6a91905190"),
				},
				Name:          "test name",
				Detail:        "test detail",
				RoutingMethod: "random",
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("e4368e4e-59de-11ec-badd-378688c95856"),
				},
				Execute:        queue.ExecuteRun,
				WaitFlowID:     uuid.FromStringOrNil("4dfaf278-205d-11f0-8be0-d74aed2ef0bc"),
				WaitTimeout:    6000,
				ServiceTimeout: 60000,
				WaitQueuecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("4e43f680-205d-11f0-8d52-efe5c18633e3"),
				},
				ServiceQueuecallIDs: []uuid.UUID{
					uuid.FromStringOrNil("4e687cbc-205d-11f0-9be2-c719e5802cf8"),
				},
				TotalIncomingCount:  0,
				TotalServicedCount:  0,
				TotalAbandonedCount: 0,
				TMCreate:            "2023-02-15 03:22:17.994000",
				TMUpdate:            DefaultTimeStamp,
				TMDelete:            DefaultTimeStamp,
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
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueueGet(gomock.Any(), tt.queue.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.QueueCreate(ctx, tt.queue); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueueGet(ctx, tt.queue.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueGets(t *testing.T) {
	type test struct {
		name string
		data []*queue.Queue

		size    uint64
		token   string
		filters map[string]string

		responseCurtime string
		expectRes       []*queue.Queue
	}

	tests := []test{
		{
			name: "normal",
			data: []*queue.Queue{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
						CustomerID: uuid.FromStringOrNil("68079af2-7f54-11ec-99c2-53bfcf885867"),
					},
					Name:                "test name 1",
					Detail:              "test detail 1",
					TagIDs:              []uuid.UUID{},
					WaitQueuecallIDs:    []uuid.UUID{},
					ServiceQueuecallIDs: []uuid.UUID{},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
						CustomerID: uuid.FromStringOrNil("68079af2-7f54-11ec-99c2-53bfcf885867"),
					},
					Name:                "test name 2",
					Detail:              "test detail 2",
					TagIDs:              []uuid.UUID{},
					WaitQueuecallIDs:    []uuid.UUID{},
					ServiceQueuecallIDs: []uuid.UUID{},
				},
			},

			size:  2,
			token: "2021-04-18T03:22:17.994000",
			filters: map[string]string{
				"customer_id": "68079af2-7f54-11ec-99c2-53bfcf885867",
				"deleted":     "false",
			},

			responseCurtime: "2020-04-18T03:22:17.995000",
			expectRes: []*queue.Queue{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
						CustomerID: uuid.FromStringOrNil("68079af2-7f54-11ec-99c2-53bfcf885867"),
					},
					Name:                "test name 1",
					Detail:              "test detail 1",
					TagIDs:              []uuid.UUID{},
					WaitQueuecallIDs:    []uuid.UUID{},
					ServiceQueuecallIDs: []uuid.UUID{},
					TMCreate:            "2020-04-18T03:22:17.995000",
					TMUpdate:            DefaultTimeStamp,
					TMDelete:            DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
						CustomerID: uuid.FromStringOrNil("68079af2-7f54-11ec-99c2-53bfcf885867"),
					},
					Name:                "test name 2",
					Detail:              "test detail 2",
					TagIDs:              []uuid.UUID{},
					WaitQueuecallIDs:    []uuid.UUID{},
					ServiceQueuecallIDs: []uuid.UUID{},
					TMCreate:            "2020-04-18T03:22:17.995000",
					TMUpdate:            DefaultTimeStamp,
					TMDelete:            DefaultTimeStamp,
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

			for _, u := range tt.data {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurtime)
				mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any())
				if err := h.QueueCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.QueueGets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_QueueDelete(t *testing.T) {
	tests := []struct {
		name string
		data *queue.Queue

		responseCurTime string
		expectRes       *queue.Queue
	}{
		{
			name: "normal",
			data: &queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0f86bb8-53a7-11ec-a123-c70052e998aa"),
				},
			},

			responseCurTime: "2023-02-18 03:22:17.995000",
			expectRes: &queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0f86bb8-53a7-11ec-a123-c70052e998aa"),
				},
				TagIDs:              []uuid.UUID{},
				WaitQueuecallIDs:    []uuid.UUID{},
				ServiceQueuecallIDs: []uuid.UUID{},
				TMCreate:            "2023-02-18 03:22:17.995000",
				TMUpdate:            "2023-02-18 03:22:17.995000",
				TMDelete:            "2023-02-18 03:22:17.995000",
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
			mockCache.EXPECT().QueueSet(ctx, gomock.Any())
			if err := h.QueueCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().QueueSet(ctx, gomock.Any())
			if err := h.QueueDelete(ctx, tt.data.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().QueueGet(ctx, tt.data.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().QueueSet(ctx, gomock.Any())
			res, err := h.QueueGet(ctx, tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. AgentGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueSetBasicInfo(t *testing.T) {
	tests := []struct {
		name string

		data *queue.Queue

		id             uuid.UUID
		queueName      string
		detail         string
		routingMethod  queue.RoutingMethod
		tagIDs         []uuid.UUID
		waitFlowID     uuid.UUID
		waitTimeout    int
		serviceTimeout int

		responseCurTime string
		expectRes       *queue.Queue
	}{
		{
			name: "normal",

			data: &queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5ddf2884-5a73-11ec-af95-43b28c48368b"),
				},
			},

			id:            uuid.FromStringOrNil("5ddf2884-5a73-11ec-af95-43b28c48368b"),
			queueName:     "new name",
			detail:        "new detail",
			routingMethod: queue.RoutingMethodRandom,
			tagIDs: []uuid.UUID{
				uuid.FromStringOrNil("ae89cfb2-4a79-11ee-bb42-afff4d0fb8b0"),
				uuid.FromStringOrNil("af7d6078-4a79-11ee-91d3-cfebfe71419e"),
			},
			waitFlowID:     uuid.FromStringOrNil("4e8d94de-205d-11f0-90b8-471a5134c23e"),
			waitTimeout:    60000,
			serviceTimeout: 6000000,

			responseCurTime: "2020-04-18T03:22:17.995000",
			expectRes: &queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5ddf2884-5a73-11ec-af95-43b28c48368b"),
				},
				Name:          "new name",
				Detail:        "new detail",
				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("ae89cfb2-4a79-11ee-bb42-afff4d0fb8b0"),
					uuid.FromStringOrNil("af7d6078-4a79-11ee-91d3-cfebfe71419e"),
				},
				WaitFlowID:          uuid.FromStringOrNil("4e8d94de-205d-11f0-90b8-471a5134c23e"),
				WaitTimeout:         60000,
				ServiceTimeout:      6000000,
				WaitQueuecallIDs:    []uuid.UUID{},
				ServiceQueuecallIDs: []uuid.UUID{},
				TMCreate:            "2020-04-18T03:22:17.995000",
				TMUpdate:            "2020-04-18T03:22:17.995000",
				TMDelete:            DefaultTimeStamp,
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
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any())
			if err := h.QueueCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any())
			if err := h.QueueSetBasicInfo(
				ctx,
				tt.id,
				tt.queueName,
				tt.detail,
				tt.routingMethod,
				tt.tagIDs,
				tt.waitFlowID,
				tt.waitTimeout,
				tt.serviceTimeout,
			); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().QueueGet(gomock.Any(), tt.data.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any())
			res, err := h.QueueGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. AgentGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueSetRoutingMethod(t *testing.T) {
	tests := []struct {
		name string

		data *queue.Queue

		id            uuid.UUID
		routingMethod queue.RoutingMethod

		responseCurTime string
		expectRes       *queue.Queue
	}{
		{
			name: "test normal",

			data: &queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5e2a6740-5a73-11ec-83a2-07ef5e2c1687"),
				},
			},
			id:            uuid.FromStringOrNil("5e2a6740-5a73-11ec-83a2-07ef5e2c1687"),
			routingMethod: queue.RoutingMethodRandom,

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5e2a6740-5a73-11ec-83a2-07ef5e2c1687"),
				},
				RoutingMethod:       queue.RoutingMethodRandom,
				TagIDs:              []uuid.UUID{},
				WaitQueuecallIDs:    []uuid.UUID{},
				ServiceQueuecallIDs: []uuid.UUID{},
				TMCreate:            "2020-04-18 03:22:17.995000",
				TMUpdate:            "2020-04-18 03:22:17.995000",
				TMDelete:            DefaultTimeStamp,
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
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any())
			if err := h.QueueCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any())
			if err := h.QueueSetRoutingMethod(ctx, tt.id, tt.routingMethod); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().QueueGet(gomock.Any(), tt.data.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any())
			res, err := h.QueueGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. AgentGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_QueueSetTagIDs(t *testing.T) {

	tests := []struct {
		name string

		data *queue.Queue

		id     uuid.UUID
		tagIDs []uuid.UUID

		responseCurTime string
		expectRes       *queue.Queue
	}{
		{
			name: "test normal",

			data: &queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5e4a3b7e-5a73-11ec-81e9-a79e401158f0"),
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			id: uuid.FromStringOrNil("5e4a3b7e-5a73-11ec-81e9-a79e401158f0"),
			tagIDs: []uuid.UUID{
				uuid.FromStringOrNil("21fcd3d4-5a73-11ec-a185-935d2e1f0846"),
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectRes: &queue.Queue{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5e4a3b7e-5a73-11ec-81e9-a79e401158f0"),
				},
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("21fcd3d4-5a73-11ec-a185-935d2e1f0846"),
				},
				WaitQueuecallIDs:    []uuid.UUID{},
				ServiceQueuecallIDs: []uuid.UUID{},
				TMCreate:            "2020-04-18 03:22:17.995000",
				TMUpdate:            "2020-04-18 03:22:17.995000",
				TMDelete:            DefaultTimeStamp,
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
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any())
			if err := h.QueueCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any())
			if err := h.QueueSetTagIDs(ctx, tt.id, tt.tagIDs); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().QueueGet(gomock.Any(), tt.data.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any())
			res, err := h.QueueGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. AgentGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// func Test_QueueSetWaitActions(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		data *queue.Queue

// 		id             uuid.UUID
// 		waitTimeout    int
// 		serviceTimeout int

// 		responseCurTime string
// 		expectRes       *queue.Queue
// 	}{
// 		{
// 			name: "normal",

// 			data: &queue.Queue{
// 				Identity: commonidentity.Identity{
// 					ID: uuid.FromStringOrNil("5ef4f122-5a73-11ec-8a63-3f0918c21af8"),
// 				},
// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 			id:             uuid.FromStringOrNil("5ef4f122-5a73-11ec-8a63-3f0918c21af8"),
// 			waitTimeout:    60000,
// 			serviceTimeout: 600000,

// 			responseCurTime: "2020-04-18 03:22:17.995000",
// 			expectRes: &queue.Queue{
// 				Identity: commonidentity.Identity{
// 					ID: uuid.FromStringOrNil("5ef4f122-5a73-11ec-8a63-3f0918c21af8"),
// 				},
// 				TagIDs:         []uuid.UUID{},
// 				WaitTimeout:    60000,
// 				ServiceTimeout: 600000,

// 				WaitQueuecallIDs:    []uuid.UUID{},
// 				ServiceQueuecallIDs: []uuid.UUID{},
// 				TMCreate:            "2020-04-18 03:22:17.995000",
// 				TMUpdate:            "2020-04-18 03:22:17.995000",
// 				TMDelete:            DefaultTimeStamp,
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockCache := cachehandler.NewMockCacheHandler(mc)
// 			h := handler{
// 				utilHandler: mockUtil,
// 				db:          dbTest,
// 				cache:       mockCache,
// 			}
// 			ctx := context.Background()

// 			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
// 			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any())
// 			if err := h.QueueCreate(ctx, tt.data); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
// 			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any())
// 			if err := h.QueueSetWaitActionsAndTimeouts(ctx, tt.id, tt.waitActions, tt.waitTimeout, tt.serviceTimeout); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().QueueGet(gomock.Any(), tt.data.ID).Return(nil, fmt.Errorf(""))
// 			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any())
// 			res, err := h.QueueGet(ctx, tt.id)
// 			if err != nil {
// 				t.Errorf("Wrong match. AgentGet expect: ok, got: %v", err)
// 			}

// 			if reflect.DeepEqual(tt.expectRes, res) == false {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }
