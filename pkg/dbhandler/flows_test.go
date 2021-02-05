package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler/models/flow"
)

func TestFlowCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name       string
		flow       *flow.Flow
		expectFlow *flow.Flow
	}

	tests := []test{
		{
			"have no actions",
			&flow.Flow{
				ID:       uuid.FromStringOrNil("2386221a-88e6-11ea-adeb-5f7b70fc89ff"),
				Name:     "test flow name",
				Detail:   "test flow detail",
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&flow.Flow{
				ID:       uuid.FromStringOrNil("2386221a-88e6-11ea-adeb-5f7b70fc89ff"),
				Name:     "test flow name",
				Detail:   "test flow detail",
				TMCreate: "2020-04-18T03:22:17.995000",
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().FlowSet(gomock.Any(), gomock.Any())
			if err := h.FlowCreate(context.Background(), tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().FlowSet(gomock.Any(), gomock.Any())
			res, err := h.FlowGet(context.Background(), tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created flow. flow: %v", res)

			if reflect.DeepEqual(tt.expectFlow, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectFlow, res)
			}
		})
	}
}

func TestFlowGetsByUserID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name       string
		userID     uint64
		limit      uint64
		token      string
		flows      []flow.Flow
		expectFlow []*flow.Flow
	}

	tests := []test{
		{
			"have no actions",
			1,
			10,
			"2020-04-18T03:30:17.000000",
			[]flow.Flow{
				{
					ID:       uuid.FromStringOrNil("837117d8-0c31-11eb-9f9e-6b4ac01a7e66"),
					UserID:   1,
					Name:     "test1",
					Persist:  true,
					TMCreate: "2020-04-18T03:22:17.995000",
				},
				{
					ID:       uuid.FromStringOrNil("845e04f8-0c31-11eb-a8cf-6f8836b86b2b"),
					UserID:   1,
					Name:     "test2",
					Persist:  true,
					TMCreate: "2020-04-18T03:23:17.995000",
				},
			},
			[]*flow.Flow{
				{
					ID:       uuid.FromStringOrNil("845e04f8-0c31-11eb-a8cf-6f8836b86b2b"),
					UserID:   1,
					Name:     "test2",
					TMCreate: "2020-04-18T03:23:17.995000",
				},
				{
					ID:       uuid.FromStringOrNil("837117d8-0c31-11eb-9f9e-6b4ac01a7e66"),
					UserID:   1,
					Name:     "test1",
					TMCreate: "2020-04-18T03:22:17.995000",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)
			ctx := context.Background()

			for _, flow := range tt.flows {
				mockCache.EXPECT().FlowSet(gomock.Any(), gomock.Any())
				if err := h.FlowCreate(ctx, &flow); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			flows, err := h.FlowGetsByUserID(ctx, tt.userID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(flows, tt.expectFlow) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectFlow, flows)
			}
		})
	}
}

func TestFlowSetData(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name       string
		flow       *flow.Flow
		updateFlow *flow.Flow
		expectFlow *flow.Flow
	}

	tests := []test{
		{
			"test normal",
			&flow.Flow{
				ID: uuid.FromStringOrNil("8d2abdc6-6760-11eb-b328-f76a25eb9e38"),
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("8d2abdc6-6760-11eb-b328-f76a25eb9e38"),
				Name:   "test name",
				Detail: "test detail",
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("a915c10c-6760-11eb-86c1-530dc1cd7cc9"),
						Type: action.TypeAnswer,
					},
				},
			},
			&flow.Flow{
				ID:     uuid.FromStringOrNil("8d2abdc6-6760-11eb-b328-f76a25eb9e38"),
				Name:   "test name",
				Detail: "test detail",
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
			&flow.Flow{
				ID:     uuid.FromStringOrNil("c19618de-6761-11eb-90f0-eb3bb8690b31"),
				Name:   "test name",
				Detail: "test detail",
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
			&flow.Flow{
				ID:     uuid.FromStringOrNil("c19618de-6761-11eb-90f0-eb3bb8690b31"),
				Name:   "test name",
				Detail: "test detail",
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
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().FlowSet(gomock.Any(), gomock.Any())
			if err := h.FlowCreate(context.Background(), tt.flow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowSet(gomock.Any(), gomock.Any())
			if err := h.FlowUpdate(context.Background(), tt.updateFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FlowGet(gomock.Any(), tt.flow.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().FlowSet(gomock.Any(), gomock.Any())
			res, err := h.FlowGet(context.Background(), tt.flow.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectFlow, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectFlow, res)
			}
		})
	}
}

func TestFlowDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string
		flow *flow.Flow
	}

	tests := []test{
		{
			"normal deletion",
			&flow.Flow{
				ID:       uuid.FromStringOrNil("9f59d11a-67c1-11eb-9cf4-1b8a94365c22"),
				Name:     "test flow name",
				Detail:   "test flow detail",
				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			if _, err := h.FlowGet(context.Background(), tt.flow.ID); err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
			}
		})
	}
}
