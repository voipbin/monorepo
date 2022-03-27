package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/cachehandler"
)

func Test_ActiveflowCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:    dbTest,
		cache: mockCache,
	}

	tests := []struct {
		name string
		af   *activeflow.Activeflow
	}{
		{
			"normal",
			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("2752907c-ace3-11ec-8fa5-b7c9abbb778b"),

				CustomerID: uuid.FromStringOrNil("27803e46-ace3-11ec-bad1-2fd1981d5580"),
				FlowID:     uuid.FromStringOrNil("27b0d6c8-ace3-11ec-a47f-7bce91046e73"),

				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("27dcddf4-ace3-11ec-90e8-63edc2ed04c7"),
				CurrentAction: action.Action{
					ID: action.IDEmpty,
				},
				ExecuteCount:    0,
				ForwardActionID: uuid.Nil,
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("672dc130-ace3-11ec-95a8-677bb46055a9"),
						Type: action.TypeAnswer,
					},
				},
				ExecutedActions: []action.Action{},
				TMCreate:        "2020-04-18 03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()

			mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any())
			if err := h.ActiveflowCreate(ctx, tt.af); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ActiveflowGet(gomock.Any(), tt.af.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.ActiveflowGet(ctx, tt.af.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created activeflow. activeflow: %v", res)
			t.Logf("Expect: %v", tt.af)

			if reflect.DeepEqual(tt.af, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.af, res)
			}
		})
	}
}

func Test_ActiveflowUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:    dbTest,
		cache: mockCache,
	}

	tests := []struct {
		name             string
		af               *activeflow.Activeflow
		updateActiveflow *activeflow.Activeflow
	}{
		{
			"normal",
			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("7b55d582-ace6-11ec-a6de-b7dda3562854"),

				CustomerID: uuid.FromStringOrNil("27803e46-ace3-11ec-bad1-2fd1981d5580"),
				FlowID:     uuid.FromStringOrNil("27b0d6c8-ace3-11ec-a47f-7bce91046e73"),

				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("27dcddf4-ace3-11ec-90e8-63edc2ed04c7"),
				CurrentAction: action.Action{
					ID: action.IDEmpty,
				},
				ExecuteCount:    0,
				ForwardActionID: uuid.Nil,
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("672dc130-ace3-11ec-95a8-677bb46055a9"),
						Type: action.TypeAnswer,
					},
				},
				ExecutedActions: []action.Action{},
				TMCreate:        "2020-04-18 03:22:17.995000",
			},
			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("7b55d582-ace6-11ec-a6de-b7dda3562854"),

				CustomerID: uuid.FromStringOrNil("27803e46-ace3-11ec-bad1-2fd1981d5580"),
				FlowID:     uuid.FromStringOrNil("27b0d6c8-ace3-11ec-a47f-7bce91046e73"),

				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("27dcddf4-ace3-11ec-90e8-63edc2ed04c7"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("791c55d4-ace6-11ec-9fd6-032afea377bc"),
					Type: action.TypeAnswer,
				},
				ExecuteCount:    1,
				ForwardActionID: uuid.FromStringOrNil("7b08f758-ace6-11ec-a13c-5b27004a9376"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("672dc130-ace3-11ec-95a8-677bb46055a9"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("7b2e5e1c-ace6-11ec-b3e3-c71fd637fde9"),
						Type: action.TypeAnswer,
					},
				},
				ExecutedActions: []action.Action{},
				TMCreate:        "2020-04-18 03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()

			mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any())
			if err := h.ActiveflowCreate(ctx, tt.af); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any())
			if err := h.ActiveflowUpdate(ctx, tt.updateActiveflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ActiveflowGet(gomock.Any(), tt.af.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.ActiveflowGet(ctx, tt.af.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			tt.updateActiveflow.TMUpdate = res.TMUpdate
			if reflect.DeepEqual(tt.updateActiveflow, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.af, res)
			}
		})
	}
}

func Test_ActiveflowDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:    dbTest,
		cache: mockCache,
	}

	tests := []struct {
		name string
		af   *activeflow.Activeflow
	}{
		{
			"normal",

			&activeflow.Activeflow{
				ID: uuid.FromStringOrNil("56d3aa20-aced-11ec-bcec-e3508fe4f7e1"),

				CustomerID: uuid.FromStringOrNil("27803e46-ace3-11ec-bad1-2fd1981d5580"),
				FlowID:     uuid.FromStringOrNil("27b0d6c8-ace3-11ec-a47f-7bce91046e73"),

				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("56fed452-aced-11ec-8281-73c867112f40"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("791c55d4-ace6-11ec-9fd6-032afea377bc"),
					Type: action.TypeAnswer,
				},
				ExecuteCount:    1,
				ForwardActionID: uuid.FromStringOrNil("7b08f758-ace6-11ec-a13c-5b27004a9376"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("672dc130-ace3-11ec-95a8-677bb46055a9"),
						Type: action.TypeAnswer,
					},
					{
						ID:   uuid.FromStringOrNil("7b2e5e1c-ace6-11ec-b3e3-c71fd637fde9"),
						Type: action.TypeAnswer,
					},
				},
				ExecutedActions: []action.Action{},
				TMCreate:        "2020-04-18 03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := context.Background()

			mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any())
			if err := h.ActiveflowCreate(ctx, tt.af); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any())
			if err := h.ActiveflowDelete(ctx, tt.af.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ActiveflowGet(gomock.Any(), tt.af.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any()).Return(nil)
			res, err := h.ActiveflowGet(ctx, tt.af.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == DefaultTimeStamp {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", DefaultTimeStamp, res)
			}
		})
	}
}

func Test_ActiveflowGetsByCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{
		db:    dbTest,
		cache: mockCache,
	}

	tests := []struct {
		name        string
		customerID  uuid.UUID
		limit       uint64
		activeflows []activeflow.Activeflow
		expectRes   []*activeflow.Activeflow
	}{
		{
			"have no actions",
			uuid.FromStringOrNil("4a40ebee-add1-11ec-8a67-9b84be9fbfb5"),
			10,
			[]activeflow.Activeflow{
				{
					ID:         uuid.FromStringOrNil("49c467e0-add1-11ec-b88b-87989662b8c0"),
					CustomerID: uuid.FromStringOrNil("4a40ebee-add1-11ec-8a67-9b84be9fbfb5"),
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("4a107676-add1-11ec-ad99-33457dadbc35"),
					CustomerID: uuid.FromStringOrNil("4a40ebee-add1-11ec-8a67-9b84be9fbfb5"),
					TMDelete:   DefaultTimeStamp,
				},
			},
			[]*activeflow.Activeflow{
				{
					ID:         uuid.FromStringOrNil("4a107676-add1-11ec-ad99-33457dadbc35"),
					CustomerID: uuid.FromStringOrNil("4a40ebee-add1-11ec-8a67-9b84be9fbfb5"),
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("49c467e0-add1-11ec-b88b-87989662b8c0"),
					CustomerID: uuid.FromStringOrNil("4a40ebee-add1-11ec-8a67-9b84be9fbfb5"),
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			for _, activeflow := range tt.activeflows {
				mockCache.EXPECT().ActiveflowSet(gomock.Any(), gomock.Any())
				if err := h.ActiveflowCreate(ctx, &activeflow); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			flows, err := h.ActiveflowGetsByCustomerID(ctx, tt.customerID, GetCurTime(), tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for _, flow := range flows {
				flow.TMCreate = ""
			}

			if reflect.DeepEqual(flows, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, flows)
			}
		})
	}
}
