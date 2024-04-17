package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/cachehandler"
)

func Test_FlowCreate(t *testing.T) {

	tests := []struct {
		name string
		flow *flow.Flow

		expectRes *flow.Flow
	}{
		{
			"have no actions",
			&flow.Flow{
				ID:     uuid.FromStringOrNil("2386221a-88e6-11ea-adeb-5f7b70fc89ff"),
				Name:   "test flow name",
				Detail: "test flow detail",
			},
			&flow.Flow{
				ID:       uuid.FromStringOrNil("2386221a-88e6-11ea-adeb-5f7b70fc89ff"),
				Name:     "test flow name",
				Detail:   "test flow detail",
				Persist:  true,
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
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
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
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
				TMUpdate: DefaultTimeStamp,
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
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
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

			tt.expectRes.TMCreate = res.TMCreate
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowGets(t *testing.T) {

	tests := []struct {
		name  string
		flows []flow.Flow

		size    uint64
		filters map[string]string

		expectRes []*flow.Flow
	}{
		{
			"normal",
			[]flow.Flow{
				{
					ID:         uuid.FromStringOrNil("3449b114-eccb-11ee-bac0-9b1dbae9fdf2"),
					CustomerID: uuid.FromStringOrNil("34c78666-eccb-11ee-bd07-7b7ad4965e58"),
					Name:       "test1",
					Persist:    true,
				},
				{
					ID:         uuid.FromStringOrNil("349c7cfa-eccb-11ee-87cc-6b61ba525e13"),
					CustomerID: uuid.FromStringOrNil("34c78666-eccb-11ee-bd07-7b7ad4965e58"),
					Name:       "test2",
					Persist:    true,
				},
			},

			10,
			map[string]string{
				"customer_id": "34c78666-eccb-11ee-bd07-7b7ad4965e58",
				"deleted":     "false",
			},

			[]*flow.Flow{
				{
					ID:         uuid.FromStringOrNil("349c7cfa-eccb-11ee-87cc-6b61ba525e13"),
					CustomerID: uuid.FromStringOrNil("34c78666-eccb-11ee-bd07-7b7ad4965e58"),
					Name:       "test2",
					Persist:    true,
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("3449b114-eccb-11ee-bac0-9b1dbae9fdf2"),
					CustomerID: uuid.FromStringOrNil("34c78666-eccb-11ee-bd07-7b7ad4965e58"),
					Name:       "test1",
					Persist:    true,
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
		{
			"has filter type",
			[]flow.Flow{
				{
					ID:         uuid.FromStringOrNil("54bee342-eccb-11ee-acb8-1358b69975c0"),
					CustomerID: uuid.FromStringOrNil("54e61d5e-eccb-11ee-8af8-639740efc157"),
					Type:       flow.TypeFlow,
					Name:       "test filter type",
					Persist:    true,
				},
			},

			10,
			map[string]string{
				"customer_id": "54e61d5e-eccb-11ee-8af8-639740efc157",
				"deleted":     "false",
				"type":        string(flow.TypeFlow),
			},

			[]*flow.Flow{
				{
					ID:         uuid.FromStringOrNil("54bee342-eccb-11ee-acb8-1358b69975c0"),
					CustomerID: uuid.FromStringOrNil("54e61d5e-eccb-11ee-8af8-639740efc157"),
					Type:       flow.TypeFlow,
					Name:       "test filter type",
					Persist:    true,
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
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
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			for _, flow := range tt.flows {
				mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
				mockCache.EXPECT().FlowSet(ctx, gomock.Any())
				if err := h.FlowCreate(ctx, &flow); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.FlowGets(ctx, utilhandler.TimeGetCurTime(), tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, f := range res {
				f.TMCreate = ""
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowUpdate(t *testing.T) {

	tests := []struct {
		name string
		flow *flow.Flow

		flowName string
		detail   string
		actions  []action.Action

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

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			if err := h.FlowCreate(context.Background(), tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
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
			res.TMDelete = ""
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
			"normal",
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
			ctx := context.Background()

			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			if err := h.FlowCreate(ctx, tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowDel(ctx, tt.flow.ID)
			if err := h.FlowDelete(ctx, tt.flow.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowGet(ctx, tt.flow.ID).Return(nil, fmt.Errorf("error"))
			mockCache.EXPECT().FlowSet(ctx, gomock.Any()).Return(nil)
			res, err := h.FlowGet(ctx, tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == DefaultTimeStamp {
				t.Errorf("Wrong match. expect: any other, got: %s", res.TMDelete)
			}

		})
	}
}

func Test_FlowUpdateActions(t *testing.T) {

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

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			if err := h.FlowCreate(ctx, tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			if err := h.FlowUpdateActions(ctx, tt.flow.ID, tt.actions); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowGet(ctx, tt.flow.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			res, err := h.FlowGet(ctx, tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			res.TMUpdate = ""
			res.TMDelete = ""
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
