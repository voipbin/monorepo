package activeflowhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/actionhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

func TestActiveFlowHandleActionConnect(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockAction := actionhandler.NewMockActionHandler(mc)

	h := &activeflowHandler{
		db:         mockDB,
		reqHandler: mockReq,

		actionHandler: mockAction,
	}

	type test struct {
		name   string
		callID uuid.UUID
		af     *activeflow.ActiveFlow

		cf           *cfconference.Conference
		responseFlow *flow.Flow
		source       *cmaddress.Address
		destinations []cmaddress.Address
		unchained    bool

		expectReqFlowActions []action.Action
	}

	tests := []test{
		{
			"single destination",
			uuid.FromStringOrNil("e1a258ca-0a98-11eb-8e3b-e7d2a18277fa"),
			&activeflow.ActiveFlow{
				CustomerID: uuid.FromStringOrNil("8220d086-7f48-11ec-a1fd-a35a08ad282c"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
				},
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
						Type:   action.TypeConnect,
						Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
					},
				},
			},
			&cfconference.Conference{
				ID:         uuid.FromStringOrNil("363b4ae8-0a9b-11eb-9d08-436d6934a451"),
				CustomerID: uuid.FromStringOrNil("8220d086-7f48-11ec-a1fd-a35a08ad282c"),
			},
			&flow.Flow{
				ID:         uuid.FromStringOrNil("fa26f0ce-0a9b-11eb-8850-afda1bb6bc03"),
				CustomerID: uuid.FromStringOrNil("8220d086-7f48-11ec-a1fd-a35a08ad282c"),
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+123456789",
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+987654321",
				},
			},
			false,

			[]action.Action{
				{
					Type:   action.TypeConferenceJoin,
					Option: []byte(`{"conference_id":"363b4ae8-0a9b-11eb-9d08-436d6934a451"}`),
				},
			},
		},
		{
			"multiple destinations",
			uuid.FromStringOrNil("cb4accf8-2710-11eb-8e49-e73409394bef"),
			&activeflow.ActiveFlow{
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("cbe12fa4-2710-11eb-8959-87391e4bbc77"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}]}`),
				},
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("cbe12fa4-2710-11eb-8959-87391e4bbc77"),
						Type:   action.TypeConnect,
						Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}]}`),
					},
				},
			},
			&cfconference.Conference{
				ID:         uuid.FromStringOrNil("cc131f96-2710-11eb-b3b2-1b43dc6ffa2f"),
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			},
			&flow.Flow{
				ID:         uuid.FromStringOrNil("cc480ff8-2710-11eb-8869-0fcf3d58fd6a"),
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+123456789",
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+987654321",
				},
				{
					Type:   cmaddress.TypeTel,
					Target: "+9876543210",
				},
			},
			false,

			[]action.Action{
				{
					Type:   action.TypeConferenceJoin,
					Option: []byte(`{"conference_id":"cc131f96-2710-11eb-b3b2-1b43dc6ffa2f"}`),
				},
			},
		},
		{
			"multiple unchained destinations",
			uuid.FromStringOrNil("211a68fe-2712-11eb-ad71-97e2b1546a91"),
			&activeflow.ActiveFlow{
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("22311f94-2712-11eb-8550-0f0b066f8120"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}], "unchained": true}`),
				},
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("22311f94-2712-11eb-8550-0f0b066f8120"),
						Type:   action.TypeConnect,
						Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}], "unchained": true}`),
					},
				},
			},
			&cfconference.Conference{
				ID:         uuid.FromStringOrNil("2266e688-2712-11eb-aab4-eb00b0a3efbe"),
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			},
			&flow.Flow{
				ID:         uuid.FromStringOrNil("229ef410-2712-11eb-9dea-a737f7b6ef2b"),
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			},
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+123456789",
			},
			[]cmaddress.Address{
				{
					Type:   cmaddress.TypeTel,
					Target: "+987654321",
				},
				{
					Type:   cmaddress.TypeTel,
					Target: "+9876543210",
				},
			},
			true,

			[]action.Action{
				{
					Type:   action.TypeConferenceJoin,
					Option: []byte(`{"conference_id":"2266e688-2712-11eb-aab4-eb00b0a3efbe"}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockReq.EXPECT().CFV1ConferenceCreate(ctx, tt.af.CustomerID, cfconference.TypeConnect, "", "", 86400, nil, nil, nil).Return(tt.cf, nil)
			mockReq.EXPECT().FMV1FlowCreate(ctx, tt.af.CustomerID, flow.TypeFlow, "", "", tt.expectReqFlowActions, false).Return(tt.responseFlow, nil)

			masterCallID := tt.callID
			if tt.unchained {
				masterCallID = uuid.Nil
			}

			mockReq.EXPECT().CMV1CallsCreate(ctx, tt.responseFlow.CustomerID, tt.responseFlow.ID, masterCallID, tt.source, tt.destinations).Return([]cmcall.Call{{ID: uuid.Nil}}, nil)
			mockDB.EXPECT().ActiveFlowSet(gomock.Any(), gomock.Any()).Return(nil)

			if err := h.actionHandleConnect(ctx, tt.callID, tt.af); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestGetActionsFromFlow(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &activeflowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	tests := []struct {
		name   string
		flowID uuid.UUID
		flow   *flow.Flow
		callID uuid.UUID
	}{
		{
			"normal",
			uuid.FromStringOrNil("9091d6aa-3cbe-11ec-9a9e-7f0d954e1f7a"),
			&flow.Flow{
				ID: uuid.FromStringOrNil("9091d6aa-3cbe-11ec-9a9e-7f0d954e1f7a"),
			},
			uuid.FromStringOrNil("549d358a-fbfc-11ea-a625-43073fda56b9"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			tmpFlow := &flow.Flow{
				CustomerID: tt.flow.CustomerID,
			}
			mockReq.EXPECT().FMV1FlowGet(ctx, tt.flowID).Return(tmpFlow, nil)

			_, err := h.getActionsFromFlow(ctx, tt.flowID, tt.flow.CustomerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
