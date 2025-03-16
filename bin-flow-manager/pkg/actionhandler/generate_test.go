package actionhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-flow-manager/models/action"
)

func Test_generateFlowActions(t *testing.T) {

	tests := []struct {
		name string

		actions []action.Action

		responseUUIDs []uuid.UUID

		expectRes []action.Action
	}{
		{
			name: "normal",
			actions: []action.Action{
				{
					ID:   uuid.FromStringOrNil("1a17219e-984c-11ec-8ae0-8fa990fecf22"),
					Type: action.TypeAnswer,
				},
			},

			expectRes: []action.Action{
				{
					ID:   uuid.FromStringOrNil("1a17219e-984c-11ec-8ae0-8fa990fecf22"),
					Type: action.TypeAnswer,
				},
			},
		},
		{
			name: "has no action id",
			actions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
			},

			responseUUIDs: []uuid.UUID{
				uuid.FromStringOrNil("db154082-0294-11f0-ba22-5f46fb5d102b"),
			},
			expectRes: []action.Action{
				{
					ID:   uuid.FromStringOrNil("db154082-0294-11f0-ba22-5f46fb5d102b"),
					Type: action.TypeAnswer,
				},
			},
		},
		{
			name: "action has goto",
			actions: []action.Action{
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

			responseUUIDs: []uuid.UUID{
				uuid.FromStringOrNil("db55c184-0294-11f0-be55-73428bb2aa7a"),
				uuid.FromStringOrNil("db7edc04-0294-11f0-b5b6-6ffa57e31a38"),
				uuid.FromStringOrNil("db9f162c-0294-11f0-863c-a312ed2977ed"),
			},
			expectRes: []action.Action{
				{
					ID:   uuid.FromStringOrNil("db55c184-0294-11f0-be55-73428bb2aa7a"),
					Type: action.TypeAnswer,
				},
				{
					ID:     uuid.FromStringOrNil("db7edc04-0294-11f0-b5b6-6ffa57e31a38"),
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world"}`),
				},
				{
					ID:     uuid.FromStringOrNil("db9f162c-0294-11f0-863c-a312ed2977ed"),
					Type:   action.TypeGoto,
					Option: []byte(`{"target_id":"4dfdd5e0-984a-11ec-ae86-efa09978823e"}`),
				},
			},
		},
		{
			name: "action has branch",
			actions: []action.Action{
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

			responseUUIDs: []uuid.UUID{
				uuid.FromStringOrNil("dbc8ac4e-0294-11f0-b559-3f80a8ea5c17"),
				uuid.FromStringOrNil("dbf1a126-0294-11f0-92d9-a725d6d1f4e3"),
				uuid.FromStringOrNil("dc1f8320-0294-11f0-a64d-bbae5d48f057"),
			},
			expectRes: []action.Action{
				{
					ID:   uuid.FromStringOrNil("dbc8ac4e-0294-11f0-b559-3f80a8ea5c17"),
					Type: action.TypeAnswer,
				},
				{
					ID:     uuid.FromStringOrNil("dbf1a126-0294-11f0-92d9-a725d6d1f4e3"),
					Type:   action.TypeTalk,
					Option: []byte(`{"text":"hello world"}`),
				},
				{
					ID:     uuid.FromStringOrNil("dc1f8320-0294-11f0-a64d-bbae5d48f057"),
					Type:   action.TypeBranch,
					Option: []byte(`{"default_target_id": "962de9f4-984a-11ec-a6b5-bba220315f29", "target_ids":{"1": "85f8a600-984a-11ec-b59a-dbe5b0c51dec"}}`),
				},
			},
		},
		{
			name: "branch has many ids",
			actions: []action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type:   action.TypeBranch,
					Option: []byte(`{"default_target_id":"ea4f362c-984b-11ec-9bf3-976297bf44b8","target_ids":{"1": "f12edd8a-984b-11ec-8d44-0fadbb919954", "2": "f151020c-984b-11ec-ac5d-238f01404820", "3": "f17129b0-984b-11ec-9174-3b062faf6b35"}}`),
				},
			},

			responseUUIDs: []uuid.UUID{
				uuid.FromStringOrNil("dc4830ea-0294-11f0-909d-e3e67beab8a5"),
				uuid.FromStringOrNil("dc712c02-0294-11f0-8adc-733b154c6b73"),
			},
			expectRes: []action.Action{
				{
					ID:   uuid.FromStringOrNil("dc4830ea-0294-11f0-909d-e3e67beab8a5"),
					Type: action.TypeAnswer,
				},
				{
					ID:     uuid.FromStringOrNil("dc712c02-0294-11f0-8adc-733b154c6b73"),
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

			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &actionHandler{
				utilHandler: mockUtil,
			}

			ctx := context.Background()

			for i := range len(tt.responseUUIDs) {
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDs[i])
			}

			res, err := h.GenerateFlowActions(ctx, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
