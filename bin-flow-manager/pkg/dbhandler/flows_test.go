package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/cachehandler"
)

func Test_FlowCreate(t *testing.T) {

	tests := []struct {
		name string

		flow *flow.Flow

		responseCurTime string

		expectedRes *flow.Flow
	}{
		{
			name: "have no actions",

			flow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2386221a-88e6-11ea-adeb-5f7b70fc89ff"),
				},
				Name:   "test flow name",
				Detail: "test flow detail",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectedRes: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2386221a-88e6-11ea-adeb-5f7b70fc89ff"),
				},
				Name:     "test flow name",
				Detail:   "test flow detail",
				Persist:  true,
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: commondatabasehandler.DefaultTimeStamp,
				TMDelete: commondatabasehandler.DefaultTimeStamp,
			},
		},
		{
			name: "have 1 action echo without option",

			flow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				},
				Name:   "test flow name",
				Detail: "test flow detail",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("9613a4e8-88e5-11ea-beeb-e7a27ea4b0f7"),
						Type: action.TypeEcho,
					},
				},
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectedRes: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				},
				Name:    "test flow name",
				Detail:  "test flow detail",
				Persist: true,
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("9613a4e8-88e5-11ea-beeb-e7a27ea4b0f7"),
						Type: action.TypeEcho,
					},
				},
				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: commondatabasehandler.DefaultTimeStamp,
				TMDelete: commondatabasehandler.DefaultTimeStamp,
			},
		},
		{
			name: "have all",

			flow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72c4b8fa-88e6-11ea-a9cd-7bc36ee781ab"),
				},
				Type: flow.TypeFlow,

				Name:   "test flow name",
				Detail: "test flow detail",

				Persist: true,

				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("7c911cfc-88e6-11ea-972e-cf8263196185"),
						Type: action.TypeEcho,
						Option: map[string]any{
							"duration": 180,
						},
					},
				},
				OnCompleteFlowID: uuid.FromStringOrNil("a7d3d97e-ce16-11f0-809a-ff2f69e4c16a"),
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectedRes: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72c4b8fa-88e6-11ea-a9cd-7bc36ee781ab"),
				},
				Type: flow.TypeFlow,

				Name:    "test flow name",
				Detail:  "test flow detail",
				Persist: true,
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("7c911cfc-88e6-11ea-972e-cf8263196185"),
						Type: action.TypeEcho,
						Option: map[string]any{
							"duration": float64(180),
						},
					},
				},
				OnCompleteFlowID: uuid.FromStringOrNil("a7d3d97e-ce16-11f0-809a-ff2f69e4c16a"),
				TMCreate:         "2020-04-18 03:22:17.995000",
				TMUpdate:         commondatabasehandler.DefaultTimeStamp,
				TMDelete:         commondatabasehandler.DefaultTimeStamp,
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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

			if reflect.DeepEqual(tt.expectedRes, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_FlowList(t *testing.T) {

	tests := []struct {
		name  string
		flows []flow.Flow

		size    uint64
		filters map[flow.Field]any

		responseCurTime string

		expectedRes []*flow.Flow
	}{
		{
			name: "normal",
			flows: []flow.Flow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("3449b114-eccb-11ee-bac0-9b1dbae9fdf2"),
						CustomerID: uuid.FromStringOrNil("34c78666-eccb-11ee-bd07-7b7ad4965e58"),
					},
					Name:    "test1",
					Persist: true,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("349c7cfa-eccb-11ee-87cc-6b61ba525e13"),
						CustomerID: uuid.FromStringOrNil("34c78666-eccb-11ee-bd07-7b7ad4965e58"),
					},
					Name:    "test2",
					Persist: true,
				},
			},

			size: 10,
			filters: map[flow.Field]any{
				flow.FieldCustomerID: uuid.FromStringOrNil("34c78666-eccb-11ee-bd07-7b7ad4965e58"),
				flow.FieldDeleted:    false,
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectedRes: []*flow.Flow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("3449b114-eccb-11ee-bac0-9b1dbae9fdf2"),
						CustomerID: uuid.FromStringOrNil("34c78666-eccb-11ee-bd07-7b7ad4965e58"),
					},
					Name:     "test1",
					Persist:  true,
					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: commondatabasehandler.DefaultTimeStamp,
					TMDelete: commondatabasehandler.DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("349c7cfa-eccb-11ee-87cc-6b61ba525e13"),
						CustomerID: uuid.FromStringOrNil("34c78666-eccb-11ee-bd07-7b7ad4965e58"),
					},
					Name:     "test2",
					Persist:  true,
					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: commondatabasehandler.DefaultTimeStamp,
					TMDelete: commondatabasehandler.DefaultTimeStamp,
				},
			},
		},
		{
			name: "has filter type",
			flows: []flow.Flow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("54bee342-eccb-11ee-acb8-1358b69975c0"),
						CustomerID: uuid.FromStringOrNil("54e61d5e-eccb-11ee-8af8-639740efc157"),
					},
					Type:    flow.TypeFlow,
					Name:    "test filter type",
					Persist: true,
				},
			},

			size: 10,
			filters: map[flow.Field]any{
				flow.FieldCustomerID: uuid.FromStringOrNil("54e61d5e-eccb-11ee-8af8-639740efc157"),
				flow.FieldDeleted:    false,
				flow.FieldType:       flow.TypeFlow,
			},

			responseCurTime: "2020-04-18 03:22:17.995000",
			expectedRes: []*flow.Flow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("54bee342-eccb-11ee-acb8-1358b69975c0"),
						CustomerID: uuid.FromStringOrNil("54e61d5e-eccb-11ee-8af8-639740efc157"),
					},
					Type:     flow.TypeFlow,
					Name:     "test filter type",
					Persist:  true,
					TMCreate: "2020-04-18 03:22:17.995000",
					TMUpdate: commondatabasehandler.DefaultTimeStamp,
					TMDelete: commondatabasehandler.DefaultTimeStamp,
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
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().FlowSet(ctx, gomock.Any())
				if err := h.FlowCreate(ctx, &flow); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.FlowList(ctx, utilhandler.TimeGetCurTime(), tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_FlowUpdate(t *testing.T) {

	tests := []struct {
		name string
		flow *flow.Flow

		id     uuid.UUID
		fields map[flow.Field]any

		responseCurTime string

		expectedRes *flow.Flow
	}{
		{
			name: "test normal",
			flow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8d2abdc6-6760-11eb-b328-f76a25eb9e38"),
				},
			},

			id: uuid.FromStringOrNil("8d2abdc6-6760-11eb-b328-f76a25eb9e38"),
			fields: map[flow.Field]any{
				flow.FieldName:   "test name",
				flow.FieldDetail: "test detail",
				flow.FieldActions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("a915c10c-6760-11eb-86c1-530dc1cd7cc9"),
						Type: action.TypeAnswer,
					},
				},
				flow.FieldOnCompleteFlowID: uuid.FromStringOrNil("9d1d6638-ce18-11f0-957f-1f0f3f0158b1"),
			},

			expectedRes: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8d2abdc6-6760-11eb-b328-f76a25eb9e38"),
				},
				Name:    "test name",
				Detail:  "test detail",
				Persist: true,
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("a915c10c-6760-11eb-86c1-530dc1cd7cc9"),
						Type: action.TypeAnswer,
					},
				},
				OnCompleteFlowID: uuid.FromStringOrNil("9d1d6638-ce18-11f0-957f-1f0f3f0158b1"),
			},
		},
		{
			name: "2 actions",
			flow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c19618de-6761-11eb-90f0-eb3bb8690b31"),
				},
			},

			id: uuid.FromStringOrNil("c19618de-6761-11eb-90f0-eb3bb8690b31"),
			fields: map[flow.Field]any{
				flow.FieldName:   "test name",
				flow.FieldDetail: "test detail",
				flow.FieldActions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("c642ab68-6761-11eb-942e-4fa4f2851c63"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("d158cc12-6761-11eb-b60e-23b7402d1c55"),
						Type: action.TypeEcho,
					},
				},
				flow.FieldOnCompleteFlowID: uuid.FromStringOrNil("9d468a7c-ce18-11f0-8afe-5bae9c781587"),
			},

			expectedRes: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c19618de-6761-11eb-90f0-eb3bb8690b31"),
				},
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
				OnCompleteFlowID: uuid.FromStringOrNil("9d468a7c-ce18-11f0-8afe-5bae9c781587"),
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
			if err := h.FlowUpdate(context.Background(), tt.id, tt.fields); err != nil {
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
			if reflect.DeepEqual(tt.expectedRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_FlowDelete(t *testing.T) {

	tests := []struct {
		name string
		flow *flow.Flow

		responseCurTime string

		expectedRes *flow.Flow
	}{
		{
			name: "normal",
			flow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9f59d11a-67c1-11eb-9cf4-1b8a94365c22"),
					CustomerID: uuid.FromStringOrNil("cf304d36-7f46-11ec-9455-93fccf7c0fdf"),
				},
				Name:   "test flow name",
				Detail: "test flow detail",
			},

			responseCurTime: "2020-04-18 03:22:17.995000",

			expectedRes: &flow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9f59d11a-67c1-11eb-9cf4-1b8a94365c22"),
					CustomerID: uuid.FromStringOrNil("cf304d36-7f46-11ec-9455-93fccf7c0fdf"),
				},
				Name:    "test flow name",
				Detail:  "test flow detail",
				Persist: true,

				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:17.995000",
				TMDelete: "2020-04-18 03:22:17.995000",
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

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			if err := h.FlowCreate(ctx, tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			if err := h.FlowDelete(ctx, tt.flow.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowGet(ctx, tt.flow.ID).Return(nil, fmt.Errorf("error"))
			mockCache.EXPECT().FlowSet(ctx, gomock.Any()).Return(nil)
			res, err := h.FlowGet(ctx, tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectedRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}
