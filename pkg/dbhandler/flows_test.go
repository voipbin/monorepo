package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/cachehandler"
)

func TestFlowCreate(t *testing.T) {

	tests := []struct {
		name       string
		flow       *flow.Flow
		expectFlow *flow.Flow
	}{
		{
			"have no actions",
			&flow.Flow{
				ID:       uuid.FromStringOrNil("2386221a-88e6-11ea-adeb-5f7b70fc89ff"),
				Name:     "test flow name",
				Detail:   "test flow detail",
				TMCreate: "2020-04-18 03:22:17.995000",
			},
			&flow.Flow{
				ID:       uuid.FromStringOrNil("2386221a-88e6-11ea-adeb-5f7b70fc89ff"),
				Name:     "test flow name",
				Detail:   "test flow detail",
				Persist:  true,
				TMCreate: "2020-04-18 03:22:17.995000",
			},
		},
		{
			"have 1 action echo without option",
			&flow.Flow{
				ID:     uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				Name:   "test flow name",
				Detail: "test flow detail",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("9613a4e8-88e5-11ea-beeb-e7a27ea4b0f7"),
						Type: action.TypeEcho,
					},
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&flow.Flow{
				ID:      uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				Name:    "test flow name",
				Detail:  "test flow detail",
				Persist: true,
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("9613a4e8-88e5-11ea-beeb-e7a27ea4b0f7"),
						Type: action.TypeEcho,
					},
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
		{
			"have 1 action echo with option",
			&flow.Flow{
				ID:     uuid.FromStringOrNil("72c4b8fa-88e6-11ea-a9cd-7bc36ee781ab"),
				Name:   "test flow name",
				Detail: "test flow detail",
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("7c911cfc-88e6-11ea-972e-cf8263196185"),
						Type:   action.TypeEcho,
						Option: []byte(`{"duration":180}`),
					},
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&flow.Flow{
				ID:      uuid.FromStringOrNil("72c4b8fa-88e6-11ea-a9cd-7bc36ee781ab"),
				Name:    "test flow name",
				Detail:  "test flow detail",
				Persist: true,
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("7c911cfc-88e6-11ea-972e-cf8263196185"),
						Type:   action.TypeEcho,
						Option: []byte(`{"duration":180}`),
					},
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			if err := h.FlowCreate(ctx, tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowGet(ctx, tt.flow.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			res, err := h.FlowGet(ctx, tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created flow. flow: %v", res)

			tt.expectFlow.TMCreate = res.TMCreate
			if reflect.DeepEqual(tt.expectFlow, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectFlow, res)
			}
		})
	}
}

func TestFlowGets(t *testing.T) {

	tests := []struct {
		name       string
		customerID uuid.UUID
		limit      uint64
		flows      []flow.Flow
		expectFlow []*flow.Flow
	}{
		{
			"have no actions",
			uuid.FromStringOrNil("9610650e-7f46-11ec-bef4-9f1afed0c6ef"),
			10,
			[]flow.Flow{
				{
					ID:         uuid.FromStringOrNil("837117d8-0c31-11eb-9f9e-6b4ac01a7e66"),
					CustomerID: uuid.FromStringOrNil("9610650e-7f46-11ec-bef4-9f1afed0c6ef"),
					Name:       "test1",
					Persist:    true,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("845e04f8-0c31-11eb-a8cf-6f8836b86b2b"),
					CustomerID: uuid.FromStringOrNil("9610650e-7f46-11ec-bef4-9f1afed0c6ef"),
					Name:       "test2",
					Persist:    true,
					TMDelete:   DefaultTimeStamp,
				},
			},
			[]*flow.Flow{
				{
					ID:         uuid.FromStringOrNil("845e04f8-0c31-11eb-a8cf-6f8836b86b2b"),
					CustomerID: uuid.FromStringOrNil("9610650e-7f46-11ec-bef4-9f1afed0c6ef"),
					Name:       "test2",
					Persist:    true,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("837117d8-0c31-11eb-9f9e-6b4ac01a7e66"),
					CustomerID: uuid.FromStringOrNil("9610650e-7f46-11ec-bef4-9f1afed0c6ef"),
					Name:       "test1",
					Persist:    true,
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			for _, flow := range tt.flows {
				mockCache.EXPECT().FlowSet(ctx, gomock.Any())
				if err := h.FlowCreate(ctx, &flow); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			flows, err := h.FlowGetsByCustomerID(ctx, tt.customerID, GetCurTime(), tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, flow := range flows {
				flow.TMCreate = ""
			}

			if reflect.DeepEqual(flows, tt.expectFlow) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectFlow, flows)
			}
		})
	}
}

func TestFlowGetsByType(t *testing.T) {

	tests := []struct {
		name       string
		customerID uuid.UUID
		flowType   flow.Type
		limit      uint64
		flows      []flow.Flow
		expectFlow []*flow.Flow
	}{
		{
			"normal",
			uuid.FromStringOrNil("b6563a82-7f46-11ec-98f8-8f45a152e25a"),
			flow.TypeFlow,
			10,
			[]flow.Flow{
				{
					ID:         uuid.FromStringOrNil("4f351e4c-6c0c-11ec-aeb7-63ef13f21b04"),
					CustomerID: uuid.FromStringOrNil("b6563a82-7f46-11ec-98f8-8f45a152e25a"),
					Type:       flow.TypeFlow,
					Name:       "test1",
					Persist:    true,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("4fb2c612-6c0c-11ec-af63-832d2d72863f"),
					CustomerID: uuid.FromStringOrNil("b6563a82-7f46-11ec-98f8-8f45a152e25a"),
					Type:       flow.TypeFlow,
					Name:       "test2",
					Persist:    true,
					TMDelete:   DefaultTimeStamp,
				},
			},
			[]*flow.Flow{
				{
					ID:         uuid.FromStringOrNil("4fb2c612-6c0c-11ec-af63-832d2d72863f"),
					CustomerID: uuid.FromStringOrNil("b6563a82-7f46-11ec-98f8-8f45a152e25a"),
					Type:       flow.TypeFlow,
					Name:       "test2",
					Persist:    true,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("4f351e4c-6c0c-11ec-aeb7-63ef13f21b04"),
					CustomerID: uuid.FromStringOrNil("b6563a82-7f46-11ec-98f8-8f45a152e25a"),
					Type:       flow.TypeFlow,
					Name:       "test1",
					Persist:    true,
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			for _, flow := range tt.flows {
				mockCache.EXPECT().FlowSet(ctx, gomock.Any())
				if err := h.FlowCreate(ctx, &flow); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			flows, err := h.FlowGetsByType(ctx, tt.customerID, tt.flowType, GetCurTime(), tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, flow := range flows {
				flow.TMCreate = ""
			}

			if reflect.DeepEqual(flows, tt.expectFlow) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectFlow, flows)
			}
		})
	}
}

func TestFlowUpdate(t *testing.T) {

	tests := []struct {
		name string
		flow *flow.Flow

		flowName  string
		detail    string
		actions   []action.Action
		expectRes *flow.Flow
	}{
		{
			"test normal",
			&flow.Flow{
				ID: uuid.FromStringOrNil("8d2abdc6-6760-11eb-b328-f76a25eb9e38"),
			},

			"test name",
			"test detail",
			[]action.Action{
				{
					ID:   uuid.FromStringOrNil("a915c10c-6760-11eb-86c1-530dc1cd7cc9"),
					Type: action.TypeAnswer,
				},
			},

			&flow.Flow{
				ID:      uuid.FromStringOrNil("8d2abdc6-6760-11eb-b328-f76a25eb9e38"),
				Name:    "test name",
				Detail:  "test detail",
				Persist: true,
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("a915c10c-6760-11eb-86c1-530dc1cd7cc9"),
						Type: action.TypeAnswer,
					},
				},
			},
		},
		{
			"2 actions",
			&flow.Flow{
				ID: uuid.FromStringOrNil("c19618de-6761-11eb-90f0-eb3bb8690b31"),
			},

			"test name",
			"test detail",
			[]action.Action{
				{
					ID:   uuid.FromStringOrNil("c642ab68-6761-11eb-942e-4fa4f2851c63"),
					Type: action.TypeAnswer,
				},
				{
					ID:   uuid.FromStringOrNil("d158cc12-6761-11eb-b60e-23b7402d1c55"),
					Type: action.TypeEcho,
				},
			},

			&flow.Flow{
				ID:      uuid.FromStringOrNil("c19618de-6761-11eb-90f0-eb3bb8690b31"),
				Name:    "test name",
				Detail:  "test detail",
				Persist: true,
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("c642ab68-6761-11eb-942e-4fa4f2851c63"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("d158cc12-6761-11eb-b60e-23b7402d1c55"),
						Type: action.TypeEcho,
					},
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

			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			if err := h.FlowCreate(context.Background(), tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			if err := h.FlowUpdate(context.Background(), tt.flow.ID, tt.flowName, tt.detail, tt.actions); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowGet(ctx, tt.flow.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			res, err := h.FlowGet(context.Background(), tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowDelete(t *testing.T) {

	tests := []struct {
		name string
		flow *flow.Flow
	}{
		{
			"normal deletion",
			&flow.Flow{
				ID:         uuid.FromStringOrNil("9f59d11a-67c1-11eb-9cf4-1b8a94365c22"),
				CustomerID: uuid.FromStringOrNil("cf304d36-7f46-11ec-9455-93fccf7c0fdf"),
				Name:       "test flow name",
				Detail:     "test flow detail",
				TMCreate:   "2020-04-18T03:22:17.995000",
				TMDelete:   DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().FlowSet(gomock.Any(), gomock.Any())
			if err := h.FlowCreate(context.Background(), tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowDel(gomock.Any(), tt.flow.ID)
			if err := h.FlowDelete(context.Background(), tt.flow.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(nil, fmt.Errorf("error"))
			mockCache.EXPECT().FlowSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.FlowGet(context.Background(), tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == DefaultTimeStamp {
				t.Errorf("Wrong match. expect: any other, got: %s", res.TMDelete)
			}

		})
	}
}

func TestFlowUpdateActions(t *testing.T) {

	tests := []struct {
		name string
		flow *flow.Flow

		actions   []action.Action
		expectRes *flow.Flow
	}{
		{
			"test normal",
			&flow.Flow{
				ID:      uuid.FromStringOrNil("585b7a74-18a0-48ac-b4c5-1ba5ddea87ae"),
				Name:    "test name",
				Detail:  "test detail",
				Persist: true,
			},

			[]action.Action{
				{
					ID:   uuid.FromStringOrNil("330047cb-6259-4eb9-aa08-548bf6d82e79"),
					Type: action.TypeAnswer,
				},
			},

			&flow.Flow{
				ID:      uuid.FromStringOrNil("585b7a74-18a0-48ac-b4c5-1ba5ddea87ae"),
				Name:    "test name",
				Detail:  "test detail",
				Persist: true,
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("330047cb-6259-4eb9-aa08-548bf6d82e79"),
						Type: action.TypeAnswer,
					},
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

			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			if err := h.FlowCreate(context.Background(), tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			if err := h.FlowUpdateActions(context.Background(), tt.flow.ID, tt.actions); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowGet(ctx, tt.flow.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			res, err := h.FlowGet(context.Background(), tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
