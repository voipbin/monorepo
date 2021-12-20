package dbhandler

import (
	context "context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/cachehandler"
)

func TestQueueCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		q         *queue.Queue
		expectRes *queue.Queue
	}{
		{
			"normal",
			&queue.Queue{
				ID:            uuid.FromStringOrNil("cba57fb6-59de-11ec-b230-5b6ab3380040"),
				UserID:        1,
				FlowID:        uuid.FromStringOrNil("538791ae-5c81-11ec-9cd9-4f0755b8aca6"),
				Name:          "test name",
				Detail:        "test detail",
				WebhookURI:    "test.com",
				WebhookMethod: "POST",
				RoutingMethod: "random",
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("e4368e4e-59de-11ec-badd-378688c95856"),
				},
				WaitActions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				WaitQueueCallIDs:    []uuid.UUID{},
				ServiceQueueCallIDs: []uuid.UUID{},
				TotalIncomingCount:  0,
				TotalServicedCount:  0,
				TotalAbandonedCount: 0,
				TotalWaitDuration:   0,
			},
			&queue.Queue{
				ID:            uuid.FromStringOrNil("cba57fb6-59de-11ec-b230-5b6ab3380040"),
				UserID:        1,
				FlowID:        uuid.FromStringOrNil("538791ae-5c81-11ec-9cd9-4f0755b8aca6"),
				Name:          "test name",
				Detail:        "test detail",
				WebhookURI:    "test.com",
				WebhookMethod: "POST",
				RoutingMethod: "random",
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("e4368e4e-59de-11ec-badd-378688c95856"),
				},
				WaitActions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				WaitQueueCallIDs:    []uuid.UUID{},
				ServiceQueueCallIDs: []uuid.UUID{},
				TotalIncomingCount:  0,
				TotalServicedCount:  0,
				TotalAbandonedCount: 0,
				TotalWaitDuration:   0,
			},
		},
		{
			"have QueueCallID",
			&queue.Queue{
				ID:            uuid.FromStringOrNil("731e523e-59e1-11ec-9156-abd8ba26f843"),
				UserID:        1,
				Name:          "test name",
				Detail:        "test detail",
				WebhookURI:    "test.com",
				WebhookMethod: "POST",
				RoutingMethod: "random",
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("7366bb8c-59e1-11ec-8f94-9bc5e34bb104"),
				},
				WaitActions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				WaitQueueCallIDs: []uuid.UUID{
					uuid.FromStringOrNil("73a46e28-59e1-11ec-8ec7-eb9d12d3dcb5"),
				},
				ServiceQueueCallIDs: []uuid.UUID{},
				TotalIncomingCount:  0,
				TotalServicedCount:  0,
				TotalAbandonedCount: 0,
				TotalWaitDuration:   0,
			},
			&queue.Queue{
				ID:            uuid.FromStringOrNil("731e523e-59e1-11ec-9156-abd8ba26f843"),
				UserID:        1,
				Name:          "test name",
				Detail:        "test detail",
				WebhookURI:    "test.com",
				WebhookMethod: "POST",
				RoutingMethod: "random",
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("7366bb8c-59e1-11ec-8f94-9bc5e34bb104"),
				},
				WaitActions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				WaitQueueCallIDs: []uuid.UUID{
					uuid.FromStringOrNil("73a46e28-59e1-11ec-8ec7-eb9d12d3dcb5"),
				},
				ServiceQueueCallIDs: []uuid.UUID{},
				TotalIncomingCount:  0,
				TotalServicedCount:  0,
				TotalAbandonedCount: 0,
				TotalWaitDuration:   0,
			},
		},
		{
			"have wait timeout, service timeout",
			&queue.Queue{
				ID:            uuid.FromStringOrNil("2c4c233c-5f67-11ec-8eea-bbf4408ec1d8"),
				UserID:        1,
				Name:          "test name",
				Detail:        "test detail",
				WebhookURI:    "test.com",
				WebhookMethod: "POST",
				RoutingMethod: "random",
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("7366bb8c-59e1-11ec-8f94-9bc5e34bb104"),
				},
				WaitActions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				WaitQueueCallIDs: []uuid.UUID{
					uuid.FromStringOrNil("73a46e28-59e1-11ec-8ec7-eb9d12d3dcb5"),
				},
				WaitTimeout:         6000,
				ServiceQueueCallIDs: []uuid.UUID{},
				ServiceTimeout:      60000,

				TotalIncomingCount:  0,
				TotalServicedCount:  0,
				TotalAbandonedCount: 0,
				TotalWaitDuration:   0,
			},
			&queue.Queue{
				ID:            uuid.FromStringOrNil("2c4c233c-5f67-11ec-8eea-bbf4408ec1d8"),
				UserID:        1,
				Name:          "test name",
				Detail:        "test detail",
				WebhookURI:    "test.com",
				WebhookMethod: "POST",
				RoutingMethod: "random",
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("7366bb8c-59e1-11ec-8f94-9bc5e34bb104"),
				},
				WaitActions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				WaitQueueCallIDs: []uuid.UUID{
					uuid.FromStringOrNil("73a46e28-59e1-11ec-8ec7-eb9d12d3dcb5"),
				},
				WaitTimeout:         6000,
				ServiceQueueCallIDs: []uuid.UUID{},
				ServiceTimeout:      60000,

				TotalIncomingCount:  0,
				TotalServicedCount:  0,
				TotalAbandonedCount: 0,
				TotalWaitDuration:   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().QueueGet(gomock.Any(), tt.q.ID).Return(nil, fmt.Errorf("")).AnyTimes()

			if err := h.QueueCreate(ctx, tt.q); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueueGet(ctx, tt.q.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestQueueGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	type test struct {
		name   string
		userID uint64

		data []*queue.Queue

		size      uint64
		token     string
		expectRes []*queue.Queue
	}

	tests := []test{
		{
			"normal",
			11,
			[]*queue.Queue{
				{
					ID:                  uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
					UserID:              11,
					Name:                "test name 1",
					Detail:              "test detail 1",
					TagIDs:              []uuid.UUID{},
					WaitActions:         []fmaction.Action{},
					WaitQueueCallIDs:    []uuid.UUID{},
					ServiceQueueCallIDs: []uuid.UUID{},
					TMCreate:            "2020-04-18T03:22:17.995000",
				},
				{
					ID:                  uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
					UserID:              11,
					Name:                "test name 2",
					Detail:              "test detail 2",
					TagIDs:              []uuid.UUID{},
					WaitActions:         []fmaction.Action{},
					WaitQueueCallIDs:    []uuid.UUID{},
					ServiceQueueCallIDs: []uuid.UUID{},
					TMCreate:            "2020-04-18T03:22:17.994000",
				},
			},
			2,
			"2021-04-18T03:22:17.994000",
			[]*queue.Queue{
				{
					ID:                  uuid.FromStringOrNil("779a3f74-4b42-11ec-881e-2f7238a54efd"),
					UserID:              11,
					Name:                "test name 1",
					Detail:              "test detail 1",
					TagIDs:              []uuid.UUID{},
					WaitActions:         []fmaction.Action{},
					WaitQueueCallIDs:    []uuid.UUID{},
					ServiceQueueCallIDs: []uuid.UUID{},
					TMCreate:            "2020-04-18T03:22:17.995000",
				},
				{
					ID:                  uuid.FromStringOrNil("a2cae478-4b42-11ec-afb2-3f23cd119aa6"),
					UserID:              11,
					Name:                "test name 2",
					Detail:              "test detail 2",
					TagIDs:              []uuid.UUID{},
					WaitActions:         []fmaction.Action{},
					WaitQueueCallIDs:    []uuid.UUID{},
					ServiceQueueCallIDs: []uuid.UUID{},
					TMCreate:            "2020-04-18T03:22:17.994000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any()).AnyTimes()
			for _, u := range tt.data {
				if err := h.QueueCreate(ctx, u); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.QueueGets(ctx, tt.userID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. UserGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func TestQueueDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name      string
		data      *queue.Queue
		expectRes *queue.Queue
	}{
		{
			"test normal",
			&queue.Queue{
				ID:       uuid.FromStringOrNil("e0f86bb8-53a7-11ec-a123-c70052e998aa"),
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&queue.Queue{
				ID:                  uuid.FromStringOrNil("e0f86bb8-53a7-11ec-a123-c70052e998aa"),
				TagIDs:              []uuid.UUID{},
				WaitActions:         []fmaction.Action{},
				WaitQueueCallIDs:    []uuid.UUID{},
				ServiceQueueCallIDs: []uuid.UUID{},
				TMCreate:            "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().QueueGet(gomock.Any(), tt.data.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any()).AnyTimes()

			if err := h.QueueCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.QueueDelete(ctx, tt.data.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueueGet(context.Background(), tt.data.ID)
			if err != nil {
				t.Errorf("Wrong match. AgentGet expect: ok, got: %v", err)
			}

			if res.TMDelete == "" {
				t.Error("Wrong match. expect: not empty, got: empty.")
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			tt.expectRes.TMDelete = res.TMDelete
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestQueueSetBasicInfo(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		data *queue.Queue

		id            uuid.UUID
		queueName     string
		detail        string
		webhookURI    string
		webhookMethod string

		expectRes *queue.Queue
	}{
		{
			"test normal",

			&queue.Queue{
				ID:       uuid.FromStringOrNil("5ddf2884-5a73-11ec-af95-43b28c48368b"),
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			uuid.FromStringOrNil("5ddf2884-5a73-11ec-af95-43b28c48368b"),
			"new name",
			"new detail",
			"test.com",
			"POST",
			&queue.Queue{
				ID:                  uuid.FromStringOrNil("5ddf2884-5a73-11ec-af95-43b28c48368b"),
				Name:                "new name",
				Detail:              "new detail",
				WebhookURI:          "test.com",
				WebhookMethod:       "POST",
				TagIDs:              []uuid.UUID{},
				WaitActions:         []fmaction.Action{},
				WaitQueueCallIDs:    []uuid.UUID{},
				ServiceQueueCallIDs: []uuid.UUID{},
				TMCreate:            "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().QueueGet(gomock.Any(), tt.data.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any()).AnyTimes()

			if err := h.QueueCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.QueueSetBasicInfo(ctx, tt.id, tt.queueName, tt.detail, tt.webhookURI, tt.webhookMethod); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueueGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. AgentGet expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestQueueSetRoutingMethod(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		data *queue.Queue

		id            uuid.UUID
		routingMethod queue.RoutingMethod

		expectRes *queue.Queue
	}{
		{
			"test normal",

			&queue.Queue{
				ID:       uuid.FromStringOrNil("5e2a6740-5a73-11ec-83a2-07ef5e2c1687"),
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			uuid.FromStringOrNil("5e2a6740-5a73-11ec-83a2-07ef5e2c1687"),
			queue.RoutingMethodRandom,
			&queue.Queue{
				ID:                  uuid.FromStringOrNil("5e2a6740-5a73-11ec-83a2-07ef5e2c1687"),
				RoutingMethod:       queue.RoutingMethodRandom,
				TagIDs:              []uuid.UUID{},
				WaitActions:         []fmaction.Action{},
				WaitQueueCallIDs:    []uuid.UUID{},
				ServiceQueueCallIDs: []uuid.UUID{},
				TMCreate:            "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().QueueGet(gomock.Any(), tt.data.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any()).AnyTimes()

			if err := h.QueueCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.QueueSetRoutingMethod(ctx, tt.id, tt.routingMethod); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueueGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. AgentGet expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestQueueSetTagIDs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		data *queue.Queue

		id     uuid.UUID
		tagIDs []uuid.UUID

		expectRes *queue.Queue
	}{
		{
			"test normal",

			&queue.Queue{
				ID:       uuid.FromStringOrNil("5e4a3b7e-5a73-11ec-81e9-a79e401158f0"),
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			uuid.FromStringOrNil("5e4a3b7e-5a73-11ec-81e9-a79e401158f0"),
			[]uuid.UUID{
				uuid.FromStringOrNil("21fcd3d4-5a73-11ec-a185-935d2e1f0846"),
			},
			&queue.Queue{
				ID: uuid.FromStringOrNil("5e4a3b7e-5a73-11ec-81e9-a79e401158f0"),
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("21fcd3d4-5a73-11ec-a185-935d2e1f0846"),
				},
				WaitActions:         []fmaction.Action{},
				WaitQueueCallIDs:    []uuid.UUID{},
				ServiceQueueCallIDs: []uuid.UUID{},
				TMCreate:            "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().QueueGet(gomock.Any(), tt.data.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any()).AnyTimes()

			if err := h.QueueCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.QueueSetTagIDs(ctx, tt.id, tt.tagIDs); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueueGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. AgentGet expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestQueueSetWaitActions(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		data *queue.Queue

		id             uuid.UUID
		waitActions    []fmaction.Action
		waitTimeout    int
		serviceTimeout int

		expectRes *queue.Queue
	}{
		{
			"test normal",

			&queue.Queue{
				ID:       uuid.FromStringOrNil("5ef4f122-5a73-11ec-8a63-3f0918c21af8"),
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			uuid.FromStringOrNil("5ef4f122-5a73-11ec-8a63-3f0918c21af8"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			60000,
			600000,

			&queue.Queue{
				ID:     uuid.FromStringOrNil("5ef4f122-5a73-11ec-8a63-3f0918c21af8"),
				TagIDs: []uuid.UUID{},
				WaitActions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				WaitTimeout:    60000,
				ServiceTimeout: 600000,

				WaitQueueCallIDs:    []uuid.UUID{},
				ServiceQueueCallIDs: []uuid.UUID{},
				TMCreate:            "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().QueueGet(gomock.Any(), tt.data.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().QueueSet(gomock.Any(), gomock.Any()).AnyTimes()

			if err := h.QueueCreate(ctx, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.QueueSetWaitActionsAndTimeouts(ctx, tt.id, tt.waitActions, tt.waitTimeout, tt.serviceTimeout); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.QueueGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. AgentGet expect: ok, got: %v", err)
			}

			tt.expectRes.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
