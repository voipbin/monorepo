package actionhandler

import (
	"context"
	"encoding/json"
	reflect "reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

func Test_generateFlowActions(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// mockDB := dbhandler.NewMockDBHandler(mc)
	// mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &actionHandler{
		// db:            mockDB,
		// notifyHandler: mockNotify,
		// actionHandler: mockAction,
	}

	tests := []struct {
		name string

		actions   []action.Action
		expectRes []action.Action
	}{
		{
			"normal",
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
					Option: []byte(`{"target_index":1}`),
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
					Option: []byte(`{"target_index":1}`),
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
					Option: []byte(`{"forward_index": 1, "target_indexes":{"1": 0}}`),
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
					Option: []byte(`{"forward_index":1,"target_indexes":{"1":0},"target_ids":{}}`),
				},
			},
		},
		{
			"branch has many index",
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world1"}`),
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world2"}`),
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world3"}`),
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world4"}`),
				},
				{
					Type:   action.TypeBranch,
					Option: []byte(`{"forward_index": 1, "target_indexes":{"1": 0,"2": 1,"3": 2}}`),
				},
			},
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world1"}`),
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world2"}`),
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world3"}`),
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world4"}`),
				},
				{
					Type:   action.TypeBranch,
					Option: []byte(`{"forward_index":1,"target_indexes":{"1": 0,"2": 1,"3": 2},"target_ids":{}}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// mockAction.EXPECT().ValidateActions(tt.actions).Return(nil)
			res, err := h.GenerateFlowActions(ctx, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			for i, a := range res {
				tt.expectRes[i].ID = a.ID

				// type goto
				if a.Type == action.TypeGoto {
					var option action.OptionGoto
					if err := json.Unmarshal(tt.expectRes[i].Option, &option); err != nil {
						t.Errorf("Wrong match. expect: ok. err: %v", err)
					}

					var resOpt action.OptionGoto
					if err := json.Unmarshal(a.Option, &resOpt); err != nil {
						t.Errorf("Wrong match. expect: ok. err: %v", err)
					}

					option.TargetID = resOpt.TargetID
					tmp, err := json.Marshal(option)
					if err != nil {
						t.Errorf("Wrong match. expect: ok. err: %v", err)
					}

					tt.expectRes[i].Option = tmp
				}

				// type branch
				if a.Type == action.TypeBranch {
					var option action.OptionBranch
					if err := json.Unmarshal(tt.expectRes[i].Option, &option); err != nil {
						t.Errorf("Wrong match. expect: ok. err: %v", err)
					}

					var resOpt action.OptionBranch
					if err := json.Unmarshal(a.Option, &resOpt); err != nil {
						t.Errorf("Wrong match. expect: ok. err: %v", err)
					}

					option.DefaultID = resOpt.DefaultID

					for j, targetID := range resOpt.TargetIDs {
						option.TargetIDs[j] = targetID
					}

					tmp, err := json.Marshal(option)
					if err != nil {
						t.Errorf("Wrong match. expect: ok. err: %v", err)
					}

					tt.expectRes[i].Option = tmp
				}

			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
