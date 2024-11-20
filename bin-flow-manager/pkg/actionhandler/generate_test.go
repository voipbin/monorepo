package actionhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
)

func Test_generateFlowActions(t *testing.T) {

	tests := []struct {
		name string

		actions   []action.Action
		expectRes []action.Action
	}{
		{
			"normal",
			[]action.Action{
				{
					ID:   uuid.FromStringOrNil("1a17219e-984c-11ec-8ae0-8fa990fecf22"),
					Type: action.TypeAnswer,
				},
			},
			[]action.Action{
				{
					ID:   uuid.FromStringOrNil("1a17219e-984c-11ec-8ae0-8fa990fecf22"),
					Type: action.TypeAnswer,
				},
			},
		},
		{
			"has no action id",
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
			},
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
			},
		},
		{
			"action has goto",
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world"}`),
				},
				{
					Type:   action.TypeGoto,
					Option: []byte(`{"target_id":"4dfdd5e0-984a-11ec-ae86-efa09978823e"}`),
				},
			},
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world"}`),
				},
				{
					Type:   action.TypeGoto,
					Option: []byte(`{"target_id":"4dfdd5e0-984a-11ec-ae86-efa09978823e"}`),
				},
			},
		},
		{
			"action has branch",
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world"}`),
				},
				{
					Type:   action.TypeBranch,
					Option: []byte(`{"default_target_id": "962de9f4-984a-11ec-a6b5-bba220315f29", "target_ids":{"1": "85f8a600-984a-11ec-b59a-dbe5b0c51dec"}}`),
				},
			},
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world"}`),
				},
				{
					Type:   action.TypeBranch,
					Option: []byte(`{"default_target_id": "962de9f4-984a-11ec-a6b5-bba220315f29", "target_ids":{"1": "85f8a600-984a-11ec-b59a-dbe5b0c51dec"}}`),
				},
			},
		},
		{
			"branch has many ids",
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type:   action.TypeBranch,
					Option: []byte(`{"default_target_id":"ea4f362c-984b-11ec-9bf3-976297bf44b8","target_ids":{"1": "f12edd8a-984b-11ec-8d44-0fadbb919954", "2": "f151020c-984b-11ec-ac5d-238f01404820", "3": "f17129b0-984b-11ec-9174-3b062faf6b35"}}`),
				},
			},
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type:   action.TypeBranch,
					Option: []byte(`{"default_target_id":"ea4f362c-984b-11ec-9bf3-976297bf44b8","target_ids":{"1": "f12edd8a-984b-11ec-8d44-0fadbb919954", "2": "f151020c-984b-11ec-ac5d-238f01404820", "3": "f17129b0-984b-11ec-9174-3b062faf6b35"}}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &actionHandler{}

			ctx := context.Background()

			res, err := h.GenerateFlowActions(ctx, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for i, a := range res {
				if tt.expectRes[i].ID == uuid.Nil {
					tt.expectRes[i].ID = a.ID
				}
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
