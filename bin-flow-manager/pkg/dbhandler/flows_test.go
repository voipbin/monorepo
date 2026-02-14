package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/cachehandler"
)

func Test_FlowCreate(t *testing.T) {

	responseCurTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string

		flow *flow.Flow

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

			expectedRes: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2386221a-88e6-11ea-adeb-5f7b70fc89ff"),
				},
				Name:     "test flow name",
				Detail:   "test flow detail",
				Persist:  true,
				TMCreate: &responseCurTime,
				TMUpdate: nil,
				TMDelete: nil,
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
				TMCreate: &responseCurTime,
				TMUpdate: nil,
				TMDelete: nil,
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
				TMCreate:         &responseCurTime,
				TMUpdate:         nil,
				TMDelete:         nil,
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

			mockUtil.EXPECT().TimeNow().Return(&responseCurTime)
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

	responseCurTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name  string
		flows []flow.Flow

		size    uint64
		filters map[flow.Field]any

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

			expectedRes: []*flow.Flow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("3449b114-eccb-11ee-bac0-9b1dbae9fdf2"),
						CustomerID: uuid.FromStringOrNil("34c78666-eccb-11ee-bd07-7b7ad4965e58"),
					},
					Name:     "test1",
					Persist:  true,
					TMCreate: &responseCurTime,
					TMUpdate: nil,
					TMDelete: nil,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("349c7cfa-eccb-11ee-87cc-6b61ba525e13"),
						CustomerID: uuid.FromStringOrNil("34c78666-eccb-11ee-bd07-7b7ad4965e58"),
					},
					Name:     "test2",
					Persist:  true,
					TMCreate: &responseCurTime,
					TMUpdate: nil,
					TMDelete: nil,
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

			expectedRes: []*flow.Flow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("54bee342-eccb-11ee-acb8-1358b69975c0"),
						CustomerID: uuid.FromStringOrNil("54e61d5e-eccb-11ee-8af8-639740efc157"),
					},
					Type:     flow.TypeFlow,
					Name:     "test filter type",
					Persist:  true,
					TMCreate: &responseCurTime,
					TMUpdate: nil,
					TMDelete: nil,
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
				mockUtil.EXPECT().TimeNow().Return(&responseCurTime)
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

			mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())
			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			if err := h.FlowCreate(context.Background(), tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())
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

			res.TMUpdate = nil
			res.TMCreate = nil
			res.TMDelete = nil
			if reflect.DeepEqual(tt.expectedRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_FlowDelete(t *testing.T) {

	responseCurTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name string
		flow *flow.Flow

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

			expectedRes: &flow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9f59d11a-67c1-11eb-9cf4-1b8a94365c22"),
					CustomerID: uuid.FromStringOrNil("cf304d36-7f46-11ec-9455-93fccf7c0fdf"),
				},
				Name:    "test flow name",
				Detail:  "test flow detail",
				Persist: true,

				TMCreate: &responseCurTime,
				TMUpdate: &responseCurTime,
				TMDelete: &responseCurTime,
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

			mockUtil.EXPECT().TimeNow().Return(&responseCurTime)
			mockCache.EXPECT().FlowSet(ctx, gomock.Any())
			if err := h.FlowCreate(ctx, tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(&responseCurTime)
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

func Test_FlowCountByCustomerID(t *testing.T) {
	responseCurTime := time.Date(2020, 4, 18, 3, 22, 17, 995000000, time.UTC)

	tests := []struct {
		name          string
		flowCount     int
		expectedCount int
	}{
		{
			name:          "no_flows",
			flowCount:     0,
			expectedCount: 0,
		},
		{
			name:          "single_flow",
			flowCount:     1,
			expectedCount: 1,
		},
		{
			name:          "multiple_flows",
			flowCount:     2,
			expectedCount: 2,
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

			// Use unique customer ID for each test
			customerID := uuid.Must(uuid.NewV4())

			// Create flows
			for i := 0; i < tt.flowCount; i++ {
				f := &flow.Flow{
					Identity: commonidentity.Identity{
						ID:         uuid.Must(uuid.NewV4()),
						CustomerID: customerID,
					},
					Name: fmt.Sprintf("test flow %d", i),
				}
				mockUtil.EXPECT().TimeNow().Return(&responseCurTime)
				mockCache.EXPECT().FlowSet(ctx, gomock.Any())
				if err := h.FlowCreate(ctx, f); err != nil {
					t.Errorf("Failed to create flow: %v", err)
					return
				}
			}

			// Get count
			count, err := h.FlowCountByCustomerID(ctx, customerID)
			if err != nil {
				t.Errorf("FlowCountByCustomerID() error = %v", err)
				return
			}

			if count != tt.expectedCount {
				t.Errorf("FlowCountByCustomerID() count = %v, expected %v", count, tt.expectedCount)
			}
		})
	}
}
