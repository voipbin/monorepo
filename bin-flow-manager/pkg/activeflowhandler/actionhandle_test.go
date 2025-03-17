package activeflowhandler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cbchatbotcall "monorepo/bin-ai-manager/models/chatbotcall"

	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"

	conversationmedia "monorepo/bin-conversation-manager/models/media"
	conversationmessage "monorepo/bin-conversation-manager/models/message"

	ememail "monorepo/bin-email-manager/models/email"

	mmmessage "monorepo/bin-message-manager/models/message"

	commonservice "monorepo/bin-common-handler/models/service"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	wmwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/stackmaphandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

func Test_actionHandleConnect(t *testing.T) {

	tests := []struct {
		name string

		af *activeflow.Activeflow

		responseConfbridge         *cmconfbridge.Confbridge
		responseFlow               *flow.Flow
		responseCalls              []*cmcall.Call
		responseGroupcalls         []*cmgroupcall.Groupcall
		responseUUIDConfbridgeJoin uuid.UUID
		responseUUIDHangup         uuid.UUID
		responsePushStack          *stack.Stack
		responsePushStackID        uuid.UUID
		responsePushAction         *action.Action

		expectFlowCreateActions         []action.Action
		expectCallSource                *commonaddress.Address
		expectCallDestinations          []commonaddress.Address
		expectEarlyExecution            bool
		expectExecuteNextMasterOnHangup bool
		expectPushActions               []action.Action
		expectUpdateActiveflow          *activeflow.Activeflow
	}{
		{
			name: "single destination",

			af: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("8220d086-7f48-11ec-a1fd-a35a08ad282c"),
				},
				ReferenceID: uuid.FromStringOrNil("e1a258ca-0a98-11eb-8e3b-e7d2a18277fa"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
							},
						},
					},
				},
			},

			responseConfbridge: &cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("363b4ae8-0a9b-11eb-9d08-436d6934a451"),
			},
			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fa26f0ce-0a9b-11eb-8850-afda1bb6bc03"),
				},
			},
			responseCalls: []*cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1273f20c-b586-11ed-ab1c-03ffc97ffcfb"),
					},
				},
			},
			responseGroupcalls: []*cmgroupcall.Groupcall{},

			responseUUIDConfbridgeJoin: uuid.FromStringOrNil("b7181286-a256-11ed-bcab-8bfb6884800b"),
			responsePushStack: &stack.Stack{
				ID: uuid.FromStringOrNil("6ba8ba2c-d4bf-11ec-bb34-1f6a8e0bf102"),
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("7b764a6e-d4bf-11ec-8f93-279c9970f53e"),
						Type:   action.TypeConfbridgeJoin,
						Option: []byte(`{"confbridge_id":"363b4ae8-0a9b-11eb-9d08-436d6934a451"}`),
					},
				},
			},

			expectFlowCreateActions: []action.Action{
				{
					Type:   action.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"363b4ae8-0a9b-11eb-9d08-436d6934a451"}`),
				},
				{
					Type: action.TypeHangup,
				},
			},
			expectCallSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+123456789",
			},
			expectCallDestinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+987654321",
				},
			},
			expectEarlyExecution:            false,
			expectExecuteNextMasterOnHangup: true,
			expectPushActions: []action.Action{
				{
					ID:     uuid.FromStringOrNil("b7181286-a256-11ed-bcab-8bfb6884800b"),
					Type:   action.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"363b4ae8-0a9b-11eb-9d08-436d6934a451"}`),
				},
			},
			expectUpdateActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("8220d086-7f48-11ec-a1fd-a35a08ad282c"),
				},
				ReferenceID:     uuid.FromStringOrNil("e1a258ca-0a98-11eb-8e3b-e7d2a18277fa"),
				ForwardStackID:  uuid.FromStringOrNil("6ba8ba2c-d4bf-11ec-bb34-1f6a8e0bf102"),
				ForwardActionID: uuid.FromStringOrNil("7b764a6e-d4bf-11ec-8f93-279c9970f53e"),

				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}]}`),
							},
						},
					},
				},
			},
		},
		{
			name: "multiple destinations",

			af: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				},
				ReferenceID:    uuid.FromStringOrNil("cb4accf8-2710-11eb-8e49-e73409394bef"),
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("cbe12fa4-2710-11eb-8959-87391e4bbc77"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}]}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("cbe12fa4-2710-11eb-8959-87391e4bbc77"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}]}`),
							},
						},
					},
				},
			},

			responseConfbridge: &cmconfbridge.Confbridge{
				ID:         uuid.FromStringOrNil("cc131f96-2710-11eb-b3b2-1b43dc6ffa2f"),
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			},
			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cc480ff8-2710-11eb-8869-0fcf3d58fd6a"),
					CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				},
			},
			responseCalls: []*cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2f03e863-5cba-4661-84d0-972c1e860815"),
					},
				},
			},
			responseGroupcalls:         []*cmgroupcall.Groupcall{},
			responseUUIDConfbridgeJoin: uuid.FromStringOrNil("8b138d81-5d06-44d0-b7fb-36dea3a00ded"),
			responsePushStack: &stack.Stack{
				ID: uuid.FromStringOrNil("73af2dfc-d4c2-11ec-a692-9b1eafe93075"),
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("845b566c-d4c2-11ec-ba4e-f739bb357410"),
						Type:   action.TypeConfbridgeJoin,
						Option: []byte(`{"confbridge_id":"cc131f96-2710-11eb-b3b2-1b43dc6ffa2f"}`),
					},
				},
			},

			expectFlowCreateActions: []action.Action{
				{
					Type:   action.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"cc131f96-2710-11eb-b3b2-1b43dc6ffa2f"}`),
				},
				{
					Type: action.TypeHangup,
				},
			},
			expectCallSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+123456789",
			},
			expectCallDestinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+987654321",
				},
				{
					Type:   commonaddress.TypeTel,
					Target: "+9876543210",
				},
			},
			expectPushActions: []action.Action{
				{
					ID:     uuid.FromStringOrNil("8b138d81-5d06-44d0-b7fb-36dea3a00ded"),
					Type:   action.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"cc131f96-2710-11eb-b3b2-1b43dc6ffa2f"}`),
				},
			},
			expectUpdateActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				},
				ReferenceID:     uuid.FromStringOrNil("cb4accf8-2710-11eb-8e49-e73409394bef"),
				ForwardStackID:  uuid.FromStringOrNil("73af2dfc-d4c2-11ec-a692-9b1eafe93075"),
				ForwardActionID: uuid.FromStringOrNil("845b566c-d4c2-11ec-ba4e-f739bb357410"),
				CurrentStackID:  stack.IDMain,
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("cbe12fa4-2710-11eb-8959-87391e4bbc77"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}]}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("cbe12fa4-2710-11eb-8959-87391e4bbc77"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}]}`),
							},
						},
					},
				},
			},
		},
		{
			name: "multiple destinations with early media",

			af: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				},
				ReferenceID: uuid.FromStringOrNil("211a68fe-2712-11eb-ad71-97e2b1546a91"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("22311f94-2712-11eb-8550-0f0b066f8120"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}], "early_media": true}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("22311f94-2712-11eb-8550-0f0b066f8120"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}], "early_media": true}`),
							},
						},
					},
				},
			},

			responseConfbridge: &cmconfbridge.Confbridge{
				ID:         uuid.FromStringOrNil("2266e688-2712-11eb-aab4-eb00b0a3efbe"),
				CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
			},
			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("229ef410-2712-11eb-9dea-a737f7b6ef2b"),
					CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				},
			},
			responseCalls: []*cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("97d2b51e-b58b-11ed-b1e0-93fbb2f1280c"),
					},
				},
			},
			responseGroupcalls:         []*cmgroupcall.Groupcall{},
			responseUUIDConfbridgeJoin: uuid.FromStringOrNil("6f9adfc1-7d2e-49bc-b8b2-ca5123b013c3"),
			responsePushStack: &stack.Stack{
				ID: uuid.FromStringOrNil("d913dcf6-d4c2-11ec-902b-37f50ff7b4b4"),
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("d96a09aa-d4c2-11ec-bcea-0bce8dd7e065"),
						Type:   action.TypeConfbridgeJoin,
						Option: []byte(`{"confbridge_id":"2266e688-2712-11eb-aab4-eb00b0a3efbe"}`),
					},
				},
			},

			expectFlowCreateActions: []action.Action{
				{
					Type:   action.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"2266e688-2712-11eb-aab4-eb00b0a3efbe"}`),
				},
				{
					Type: action.TypeHangup,
				},
			},
			expectCallSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+123456789",
			},
			expectCallDestinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+987654321",
				},
				{
					Type:   commonaddress.TypeTel,
					Target: "+9876543210",
				},
			},
			expectEarlyExecution:            true,
			expectExecuteNextMasterOnHangup: false,
			expectPushActions: []action.Action{
				{
					ID:     uuid.FromStringOrNil("6f9adfc1-7d2e-49bc-b8b2-ca5123b013c3"),
					Type:   action.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"2266e688-2712-11eb-aab4-eb00b0a3efbe"}`),
				},
			},
			expectUpdateActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("a356975a-8055-11ec-9c11-37c0ba53de51"),
				},
				ReferenceID:     uuid.FromStringOrNil("211a68fe-2712-11eb-ad71-97e2b1546a91"),
				ForwardStackID:  uuid.FromStringOrNil("d913dcf6-d4c2-11ec-902b-37f50ff7b4b4"),
				ForwardActionID: uuid.FromStringOrNil("d96a09aa-d4c2-11ec-bcea-0bce8dd7e065"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("22311f94-2712-11eb-8550-0f0b066f8120"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}], "early_media": true}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("22311f94-2712-11eb-8550-0f0b066f8120"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}, {"type": "tel", "name": "", "target": "+9876543210"}], "early_media": true}`),
							},
						},
					},
				},
			},
		},
		{
			name: "single destination with relay reason",

			af: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("8220d086-7f48-11ec-a1fd-a35a08ad282c"),
				},
				ReferenceID: uuid.FromStringOrNil("0bd920ac-a253-11ed-b372-371e3ba29e82"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}],"relay_reason":true}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}],"relay_reason":true}`),
							},
						},
					},
				},
			},

			responseConfbridge: &cmconfbridge.Confbridge{
				ID:         uuid.FromStringOrNil("0c3bd774-a253-11ed-b6f4-2b405333577e"),
				CustomerID: uuid.FromStringOrNil("8220d086-7f48-11ec-a1fd-a35a08ad282c"),
			},
			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("fa26f0ce-0a9b-11eb-8850-afda1bb6bc03"),
					CustomerID: uuid.FromStringOrNil("8220d086-7f48-11ec-a1fd-a35a08ad282c"),
				},
			},
			responseCalls: []*cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("f7e05cb8-a253-11ed-9f37-0fef5e1b2aa9"),
					},
				},
			},
			responseGroupcalls:         []*cmgroupcall.Groupcall{},
			responseUUIDConfbridgeJoin: uuid.FromStringOrNil("222a9d00-a257-11ed-8e79-5309100e27e4"),
			responseUUIDHangup:         uuid.FromStringOrNil("2257c8e8-a257-11ed-b228-a38777d47451"),
			responsePushStack: &stack.Stack{
				ID: uuid.FromStringOrNil("6ba8ba2c-d4bf-11ec-bb34-1f6a8e0bf102"),
				Actions: []action.Action{
					{
						ID:     uuid.FromStringOrNil("7b764a6e-d4bf-11ec-8f93-279c9970f53e"),
						Type:   action.TypeConfbridgeJoin,
						Option: []byte(`{"confbridge_id":"d96a09aa-d4c2-11ec-bcea-0bce8dd7e065"}`),
					},
				},
			},

			expectCallSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+123456789",
			},
			expectCallDestinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+987654321",
				},
			},
			expectExecuteNextMasterOnHangup: true,
			expectFlowCreateActions: []action.Action{
				{
					Type:   action.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"0c3bd774-a253-11ed-b6f4-2b405333577e"}`),
				},
				{
					Type: action.TypeHangup,
				},
			},
			expectPushActions: []action.Action{
				{
					ID:     uuid.FromStringOrNil("222a9d00-a257-11ed-8e79-5309100e27e4"),
					Type:   action.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"0c3bd774-a253-11ed-b6f4-2b405333577e"}`),
				},
				{
					ID:     uuid.FromStringOrNil("2257c8e8-a257-11ed-b228-a38777d47451"),
					Type:   action.TypeHangup,
					Option: []byte(`{"reason":"","reference_id":"f7e05cb8-a253-11ed-9f37-0fef5e1b2aa9"}`),
				},
			},
			expectUpdateActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("8220d086-7f48-11ec-a1fd-a35a08ad282c"),
				},
				ReferenceID:     uuid.FromStringOrNil("0bd920ac-a253-11ed-b372-371e3ba29e82"),
				ForwardStackID:  uuid.FromStringOrNil("6ba8ba2c-d4bf-11ec-bb34-1f6a8e0bf102"),
				ForwardActionID: uuid.FromStringOrNil("7b764a6e-d4bf-11ec-8f93-279c9970f53e"),

				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
					Type:   action.TypeConnect,
					Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}],"relay_reason":true}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("f4a4a87e-0a98-11eb-8f96-cba83b8b3f76"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+123456789"}, "destinations": [{"type": "tel", "name": "", "target": "+987654321"}],"relay_reason":true}`),
							},
						},
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
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				utilHandler: mockUtil,
				db:          mockDB,
				reqHandler:  mockReq,

				actionHandler:   mockAction,
				stackmapHandler: mockStack,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, tt.af.Identity.CustomerID, cmconfbridge.TypeConnect).Return(tt.responseConfbridge, nil)
			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.af.Identity.CustomerID, flow.TypeFlow, gomock.Any(), gomock.Any(), tt.expectFlowCreateActions, false).Return(tt.responseFlow, nil)
			mockReq.EXPECT().CallV1CallsCreate(ctx, tt.responseFlow.CustomerID, tt.responseFlow.ID, tt.af.ReferenceID, tt.expectCallSource, tt.expectCallDestinations, tt.expectEarlyExecution, tt.expectExecuteNextMasterOnHangup).Return(tt.responseCalls, tt.responseGroupcalls, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDConfbridgeJoin)
			if tt.responseUUIDHangup != uuid.Nil {
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDHangup)
			}
			mockStack.EXPECT().PushStackByActions(tt.af.StackMap, uuid.Nil, tt.expectPushActions, tt.af.CurrentStackID, tt.af.CurrentAction.ID).Return(tt.responsePushStack, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectUpdateActiveflow).Return(nil)

			if err := h.actionHandleConnect(ctx, tt.af); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleGotoLoopContinue(t *testing.T) {

	tests := []struct {
		name string

		callID   uuid.UUID
		targetID uuid.UUID

		activeFlow       *activeflow.Activeflow
		updateActiveFlow *activeflow.Activeflow

		responseOrgStackID    uuid.UUID
		responseOrgAction     *action.Action
		responseTargetStackID uuid.UUID
		responseTargetAction  *action.Action
	}{
		{
			name:     "normal",
			callID:   uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
			targetID: uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
					Type:   action.TypeGoto,
					Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":3}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
								Type: action.TypeAnswer,
							},
							{
								ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
								Type:   action.TypeGoto,
								Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":3}`),
							},
						},
					},
				},
			},
			updateActiveFlow: &activeflow.Activeflow{
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
					Type:   action.TypeGoto,
					Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":3}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
								Type: action.TypeAnswer,
							},
							{
								ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
								Type:   action.TypeGoto,
								Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":3}`),
							},
						},
					},
				},
			},

			responseOrgStackID: stack.IDMain,
			responseOrgAction: &action.Action{
				ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
				Type:   action.TypeGoto,
				Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":3}`),
			},
			responseTargetStackID: stack.IDMain,
			responseTargetAction: &action.Action{
				ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
				Type: action.TypeAnswer,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				stackmapHandler: mockStack,
			}

			ctx := context.Background()

			mockStack.EXPECT().GetAction(tt.activeFlow.StackMap, tt.activeFlow.CurrentStackID, tt.activeFlow.CurrentAction.ID, false).Return(tt.responseOrgStackID, tt.responseOrgAction, nil)
			mockStack.EXPECT().GetAction(tt.activeFlow.StackMap, tt.activeFlow.CurrentStackID, tt.targetID, false).Return(tt.responseTargetStackID, tt.responseTargetAction, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.updateActiveFlow).Return(nil)

			if err := h.actionHandleGoto(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleGotoLoopOver(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		activeFlow *activeflow.Activeflow

		expectActiveFlow *activeflow.Activeflow
	}{
		{
			"normal",
			uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
			&activeflow.Activeflow{
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
					Type:   action.TypeGoto,
					Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":0}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
								Type: action.TypeAnswer,
							},
							{
								ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
								Type:   action.TypeGoto,
								Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":0}`),
							},
							{
								ID:   uuid.FromStringOrNil("c299daf0-984c-11ec-9288-0b50517b314d"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			&activeflow.Activeflow{
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
					Type:   action.TypeGoto,
					Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
								Type: action.TypeAnswer,
							},
							{
								ID:     uuid.FromStringOrNil("2d099c6e-55a3-11ec-85b0-db3612865f6e"),
								Type:   action.TypeGoto,
								Option: []byte(`{"target_id":"7dbc6998-410d-11ec-91b8-d722b27bb799","loop_count":0}`),
							},
							{
								ID:   uuid.FromStringOrNil("c299daf0-984c-11ec-9288-0b50517b314d"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()

			if err := h.actionHandleGoto(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleQueueJoin(t *testing.T) {
	tests := []struct {
		name string

		activeflow *activeflow.Activeflow

		responseService *commonservice.Service
		responseStack   *stack.Stack

		expectQueueID uuid.UUID
	}{
		{
			name: "normal",

			activeflow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("bf1f9cb4-6590-11ec-8502-ffcab16cf0d1"),
					Type:   action.TypeQueueJoin,
					Option: []byte(`{"queue_id": "bf45ea2c-6590-11ec-9a8c-ff92b7ef9aad"}`),
				},
				ReferenceID: uuid.FromStringOrNil("3de1fb7a-adfb-11ec-8765-9bb130635c87"),
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("bf1f9cb4-6590-11ec-8502-ffcab16cf0d1"),
								Type:   action.TypeQueueJoin,
								Option: []byte(`{"queue_id": "bf45ea2c-6590-11ec-9a8c-ff92b7ef9aad"}`),
							},
							{
								ID:   uuid.FromStringOrNil("cdd46f0e-6591-11ec-aff5-63bb1f2f2e5f"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},

			responseService: &commonservice.Service{
				ID: uuid.FromStringOrNil("af231114-ad02-11ed-a485-2bce52de1ce4"),
				PushActions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("f51e6456-f9e1-11ef-8b03-d728628eb40d"),
						Type: action.TypeAnswer,
					},
				},
			},
			responseStack: &stack.Stack{
				ID: uuid.FromStringOrNil("af231114-ad02-11ed-a485-2bce52de1ce4"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("f51e6456-f9e1-11ef-8b03-d728628eb40d"),
						Type: action.TypeAnswer,
					},
				},
			},

			expectQueueID: uuid.FromStringOrNil("bf45ea2c-6590-11ec-9a8c-ff92b7ef9aad"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				stackmapHandler: mockStack,
			}
			ctx := context.Background()

			mockReq.EXPECT().QueueV1ServiceTypeQueuecallStart(ctx, tt.expectQueueID, tt.activeflow.Identity.ID, qmqueuecall.ReferenceType(qmqueuecall.ReferenceTypeCall), tt.activeflow.ReferenceID).Return(tt.responseService, nil)

			// PushStack
			mockStack.EXPECT().PushStackByActions(tt.activeflow.StackMap, tt.responseService.ID, tt.responseService.PushActions, tt.activeflow.CurrentStackID, tt.activeflow.CurrentAction.ID).Return(tt.responseStack, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, gomock.Any()).Return(nil)

			mockReq.EXPECT().QueueV1QueuecallUpdateStatusWaiting(ctx, tt.responseService.ID).Return(&qmqueuecall.Queuecall{}, nil)

			if err := h.actionHandleQueueJoin(ctx, tt.activeflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleFetch(t *testing.T) {

	tests := []struct {
		name string

		activeflow *activeflow.Activeflow

		responseFetch []action.Action
		responseStack *stack.Stack

		expectUpdateActiveflow *activeflow.Activeflow
	}{
		{
			name: "normal",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("de10062c-d4df-11ec-bd42-a76fe4d96b2f"),
				},
				ReferenceID: uuid.FromStringOrNil("d79ad434-d4df-11ec-8edf-0f15868a6578"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("c1c76320-d4df-11ec-acdf-8304c3ca8c1f"),
					Type: action.TypeFetch,
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("f0b5605e-648e-11ec-b318-a7f267cc71fc"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("c1c76320-d4df-11ec-acdf-8304c3ca8c1f"),
								Type: action.TypeFetch,
							},
							{
								ID:   uuid.FromStringOrNil("ad108d6a-648e-11ec-a226-536bc1253066"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},

			responseFetch: []action.Action{
				{
					ID:   uuid.FromStringOrNil("0dc5e10c-d4e0-11ec-8dd0-a326b2d87c71"),
					Type: action.TypeAnswer,
				},
			},
			responseStack: &stack.Stack{
				ID: uuid.FromStringOrNil("5d18b072-d4e0-11ec-a4ab-1fcd7ec4f258"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("0dc5e10c-d4e0-11ec-8dd0-a326b2d87c71"),
						Type: action.TypeAnswer,
					},
				},
			},

			expectUpdateActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("de10062c-d4df-11ec-bd42-a76fe4d96b2f"),
				},
				ForwardStackID:  uuid.FromStringOrNil("5d18b072-d4e0-11ec-a4ab-1fcd7ec4f258"),
				ForwardActionID: uuid.FromStringOrNil("0dc5e10c-d4e0-11ec-8dd0-a326b2d87c71"),
				ReferenceID:     uuid.FromStringOrNil("d79ad434-d4df-11ec-8edf-0f15868a6578"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("c1c76320-d4df-11ec-acdf-8304c3ca8c1f"),
					Type: action.TypeFetch,
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("f0b5605e-648e-11ec-b318-a7f267cc71fc"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("c1c76320-d4df-11ec-acdf-8304c3ca8c1f"),
								Type: action.TypeFetch,
							},
							{
								ID:   uuid.FromStringOrNil("ad108d6a-648e-11ec-a226-536bc1253066"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				actionHandler:   mockAction,
				stackmapHandler: mockStack,
			}

			ctx := context.Background()

			mockAction.EXPECT().ActionFetchGet(&tt.activeflow.CurrentAction, tt.activeflow.Identity.ID, tt.activeflow.ReferenceID).Return(tt.responseFetch, nil)
			mockStack.EXPECT().PushStackByActions(tt.activeflow.StackMap, uuid.Nil, tt.responseFetch, tt.activeflow.CurrentStackID, tt.activeflow.CurrentAction.ID).Return(tt.responseStack, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectUpdateActiveflow).Return(nil)

			if err := h.actionHandleFetch(ctx, tt.activeflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleFetchFlow(t *testing.T) {

	tests := []struct {
		name string

		flowID     uuid.UUID
		activeflow *activeflow.Activeflow

		responseflow  *flow.Flow
		responseStack *stack.Stack

		expectUpdateActiveflow *activeflow.Activeflow
	}{
		{
			name: "normal",

			flowID: uuid.FromStringOrNil("a1d247b4-3cbf-11ec-8d08-970ce7001aaa"),
			activeflow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("ec99431a-3cbf-11ec-b530-b3c665dd8156"),
					Type:   action.TypeFetchFlow,
					Option: []byte(`{"flow_id": "a1d247b4-3cbf-11ec-8d08-970ce7001aaa"}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("f0b5605e-648e-11ec-b318-a7f267cc71fc"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("ec99431a-3cbf-11ec-b530-b3c665dd8156"),
								Type: action.TypeFetchFlow,
							},
							{
								ID:   uuid.FromStringOrNil("ad108d6a-648e-11ec-a226-536bc1253066"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},

			responseflow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1d247b4-3cbf-11ec-8d08-970ce7001aaa"),
				},
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("e2af181a-648e-11ec-878b-2bb6c0cebb3e"),
						Type: action.TypeAMD,
					},
				},
			},
			responseStack: &stack.Stack{
				ID: uuid.FromStringOrNil("ede4083a-d4e1-11ec-917f-7f730832f0d0"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("e2af181a-648e-11ec-878b-2bb6c0cebb3e"),
						Type: action.TypeAMD,
					},
				},
			},

			expectUpdateActiveflow: &activeflow.Activeflow{
				ForwardStackID:  uuid.FromStringOrNil("ede4083a-d4e1-11ec-917f-7f730832f0d0"),
				ForwardActionID: uuid.FromStringOrNil("e2af181a-648e-11ec-878b-2bb6c0cebb3e"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("ec99431a-3cbf-11ec-b530-b3c665dd8156"),
					Type:   action.TypeFetchFlow,
					Option: []byte(`{"flow_id": "a1d247b4-3cbf-11ec-8d08-970ce7001aaa"}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("f0b5605e-648e-11ec-b318-a7f267cc71fc"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("ec99431a-3cbf-11ec-b530-b3c665dd8156"),
								Type: action.TypeFetchFlow,
							},
							{
								ID:   uuid.FromStringOrNil("ad108d6a-648e-11ec-a226-536bc1253066"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},
		},
		{
			name: "replace flow has 2 actions",

			flowID: uuid.FromStringOrNil("36e14dae-648f-11ec-b947-6f91a363d29e"),
			activeflow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("36679982-648f-11ec-b604-63e47c25e1e7"),
					Type:   action.TypeFetchFlow,
					Option: []byte(`{"flow_id": "36e14dae-648f-11ec-b947-6f91a363d29e"}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("36900886-648f-11ec-88c7-5bc937041ab5"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("36679982-648f-11ec-b604-63e47c25e1e7"),
								Type: action.TypeFetchFlow,
							},
							{
								ID:   uuid.FromStringOrNil("36ba131a-648f-11ec-8a6b-830a37358fbe"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},

			responseflow: &flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("36e14dae-648f-11ec-b947-6f91a363d29e"),
				},
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("59b5a226-648f-11ec-a356-ff8a386afbb9"),
						Type: action.TypeAMD,
					},
					{
						ID:   uuid.FromStringOrNil("59e09512-648f-11ec-bcec-438ee13c4be1"),
						Type: action.TypeTalk,
					},
				},
			},
			responseStack: &stack.Stack{
				ID: uuid.FromStringOrNil("b0a90640-d4e2-11ec-ac01-878f8d902c0b"),
				Actions: []action.Action{
					{
						ID:   uuid.FromStringOrNil("59b5a226-648f-11ec-a356-ff8a386afbb9"),
						Type: action.TypeAMD,
					},
				},
			},

			expectUpdateActiveflow: &activeflow.Activeflow{
				ForwardStackID:  uuid.FromStringOrNil("b0a90640-d4e2-11ec-ac01-878f8d902c0b"),
				ForwardActionID: uuid.FromStringOrNil("59b5a226-648f-11ec-a356-ff8a386afbb9"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("36679982-648f-11ec-b604-63e47c25e1e7"),
					Type:   action.TypeFetchFlow,
					Option: []byte(`{"flow_id": "36e14dae-648f-11ec-b947-6f91a363d29e"}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("36900886-648f-11ec-88c7-5bc937041ab5"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("36679982-648f-11ec-b604-63e47c25e1e7"),
								Type: action.TypeFetchFlow,
							},
							{
								ID:   uuid.FromStringOrNil("36ba131a-648f-11ec-8a6b-830a37358fbe"),
								Type: action.TypeTalk,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				actionHandler:   mockAction,
				stackmapHandler: mockStack,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(tt.responseflow, nil)

			mockStack.EXPECT().PushStackByActions(tt.activeflow.StackMap, uuid.Nil, tt.responseflow.Actions, tt.activeflow.CurrentStackID, tt.activeflow.CurrentAction.ID).Return(tt.responseStack, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectUpdateActiveflow).Return(nil)

			if err := h.actionHandleFetchFlow(ctx, tt.activeflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleConferenceJoin(t *testing.T) {

	tests := []struct {
		name string

		activeflow *activeflow.Activeflow

		responseService *commonservice.Service
		responseStack   *stack.Stack

		expectConferenceID uuid.UUID
		expectActiveFlow   *activeflow.Activeflow
	}{
		{
			name: "normal",

			activeflow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
					Type:   action.TypeConferenceJoin,
					Option: []byte(`{"conference_id": "b7c84d66-410b-11ec-ab21-23726c7dc3b9"}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
								Type:   action.TypeConferenceJoin,
								Option: []byte(`{"conference_id": "b7c84d66-410b-11ec-ab21-23726c7dc3b9"}`),
							},
						},
					},
				},
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("c71e769a-155f-11ed-9bfb-9b3081a1b9f0"),
			},

			responseService: &commonservice.Service{
				ID:   uuid.FromStringOrNil("2b2f8fe8-ab74-11ed-a3a0-b7d673c42e64"),
				Type: commonservice.TypeConferencecall,
				PushActions: []action.Action{
					{
						ID: uuid.FromStringOrNil("3dba7cea-ab74-11ed-83b9-2b746a7d46a1"),
					},
				},
			},
			responseStack: &stack.Stack{
				ID: uuid.FromStringOrNil("2b2f8fe8-ab74-11ed-a3a0-b7d673c42e64"),
				Actions: []action.Action{
					{
						ID: uuid.FromStringOrNil("3dba7cea-ab74-11ed-83b9-2b746a7d46a1"),
					},
				},
			},

			expectConferenceID: uuid.FromStringOrNil("b7c84d66-410b-11ec-ab21-23726c7dc3b9"),
			expectActiveFlow: &activeflow.Activeflow{
				ForwardStackID:  uuid.FromStringOrNil("fd6d9b84-d4e3-11ec-a53b-879007c0bc0a"),
				ForwardActionID: uuid.FromStringOrNil("c74b311c-410c-11ec-84ac-1759f56d04b5"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
					Type:   action.TypeConferenceJoin,
					Option: []byte(`{"conference_id": "b7c84d66-410b-11ec-ab21-23726c7dc3b9"}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("7dbc6998-410d-11ec-91b8-d722b27bb799"),
								Type:   action.TypeConferenceJoin,
								Option: []byte(`{"conference_id": "b7c84d66-410b-11ec-ab21-23726c7dc3b9"}`),
							},
						},
					},
				},
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("c71e769a-155f-11ed-9bfb-9b3081a1b9f0"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				stackmapHandler: mockStack,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConferenceV1ServiceTypeConferencecallStart(ctx, tt.expectConferenceID, cfconferencecall.ReferenceTypeCall, tt.activeflow.ReferenceID).Return(tt.responseService, nil)

			// push stack
			mockStack.EXPECT().PushStackByActions(tt.activeflow.StackMap, tt.responseService.ID, tt.responseService.PushActions, tt.activeflow.CurrentStackID, tt.activeflow.CurrentAction.ID).Return(tt.responseStack, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, gomock.Any()).Return(nil)

			if err := h.actionHandleConferenceJoin(ctx, tt.activeflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleBranch(t *testing.T) {

	tests := []struct {
		name string

		activeflow     *activeflow.Activeflow
		targetActionID uuid.UUID

		expectVariables map[string]string

		responseVariable *variable.Variable
		responseStackID  uuid.UUID
		responseAction   *action.Action

		expectActiveFlow *activeflow.Activeflow
	}{
		{
			name: "normal",

			activeflow: &activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
					Type:   action.TypeBranch,
					Option: []byte(`{"variable":"voipbin.call.tmpdigits","default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
								Type:   action.TypeBranch,
								Option: []byte(`{"variable":"voipbin.call.tmpdigits","default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
							},
							{
								ID:   uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
			targetActionID: uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),

			expectVariables: map[string]string{
				"voipbin.call.tmpdigits": "",
			},

			responseVariable: &variable.Variable{
				Variables: map[string]string{
					"voipbin.call.tmpdigits": "1",
				},
			},
			responseStackID: stack.IDMain,
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),
				Type: action.TypeAnswer,
			},

			expectActiveFlow: &activeflow.Activeflow{
				ReferenceID:     uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
					Type:   action.TypeBranch,
					Option: []byte(`{"variable":"voipbin.call.tmpdigits","default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
								Type:   action.TypeBranch,
								Option: []byte(`{"variable":"voipbin.call.tmpdigits","default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
							},
							{
								ID:   uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
		},
		{
			name: "use default",

			activeflow: &activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
					Type:   action.TypeBranch,
					Option: []byte(`{"default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
								Type:   action.TypeBranch,
								Option: []byte(`{"default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
							},
							{
								ID:   uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
			targetActionID: uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),

			expectVariables: map[string]string{
				"voipbin.call.digits": "",
			},

			responseVariable: &variable.Variable{
				Variables: map[string]string{
					"voipbin.call.digits": "",
				},
			},
			responseStackID: stack.IDMain,
			responseAction: &action.Action{
				ID:   uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),
				Type: action.TypeAnswer,
			},

			expectActiveFlow: &activeflow.Activeflow{
				ReferenceID:     uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
					Type:   action.TypeBranch,
					Option: []byte(`{"default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
				},
				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4d174b14-91a3-11ec-861b-0f6aaeff6362"),
								Type:   action.TypeBranch,
								Option: []byte(`{"default_target_id":"59e4a526-91a3-11ec-83a3-7373495be152","target_ids":{"1":"623e8e48-91a4-11ec-aab0-d741c6c9423c"}}`),
							},
							{
								ID:   uuid.FromStringOrNil("59e4a526-91a3-11ec-83a3-7373495be152"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("623e8e48-91a4-11ec-aab0-d741c6c9423c"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				stackmapHandler: mockStack,
				variableHandler: mockVar,
			}

			ctx := context.Background()

			mockVar.EXPECT().Get(ctx, tt.activeflow.Identity.ID).Return(tt.responseVariable, nil)
			mockVar.EXPECT().SetVariable(ctx, tt.activeflow.Identity.ID, tt.expectVariables).Return(nil)
			mockStack.EXPECT().GetAction(tt.activeflow.StackMap, tt.activeflow.CurrentStackID, tt.targetActionID, false).Return(tt.responseStackID, tt.responseAction, nil)

			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectActiveFlow).Return(nil)

			if err := h.actionHandleBranch(ctx, tt.activeflow); err != nil {
				t.Errorf("Wrong match. exepct: ok, got: %v", err)
			}

		})
	}
}

func Test_actionHandleConditionCallDigits(t *testing.T) {

	tests := []struct {
		name string

		activeFlow *activeflow.Activeflow

		callID uuid.UUID

		responseDigits string
	}{
		{
			"length match",

			&activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("c2dbc228-92b4-11ec-8cc9-3358e0b8bbb5"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("c3434cae-92b4-11ec-aa8a-07d4fef0bef1"),
					Type:   action.TypeConditionCallDigits,
					Option: []byte(`{"length": 1}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("c3434cae-92b4-11ec-aa8a-07d4fef0bef1"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"length": 1}`),
							},
							{
								ID:   uuid.FromStringOrNil("c37492fa-92b4-11ec-94a0-1bfcaf781964"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			uuid.FromStringOrNil("c2dbc228-92b4-11ec-8cc9-3358e0b8bbb5"),

			"1",
		},
		{
			"key match",

			&activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("6ef04e44-92b5-11ec-a70a-1b80e125f020"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("6f553cd2-92b5-11ec-a9cc-070ec1a9c665"),
					Type:   action.TypeConditionCallDigits,
					Option: []byte(`{"key": "3"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("6f553cd2-92b5-11ec-a9cc-070ec1a9c665"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"key": "3"}`),
							},
							{
								ID:   uuid.FromStringOrNil("6f893f3c-92b5-11ec-9c9d-437fc938558b"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			uuid.FromStringOrNil("6ef04e44-92b5-11ec-a70a-1b80e125f020"),

			"123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGetDigits(ctx, tt.callID).Return(tt.responseDigits, nil)

			if err := h.actionHandleConditionCallDigits(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleConditionCallDigitsFail(t *testing.T) {

	tests := []struct {
		name string

		activeFlow *activeflow.Activeflow

		callID               uuid.UUID
		expectTargetActionID uuid.UUID

		responseDigits      string
		expectReqActiveFlow *activeflow.Activeflow

		responseAction *action.Action
	}{
		{
			"length fail",

			&activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("c2dbc228-92b4-11ec-8cc9-3358e0b8bbb5"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("c3434cae-92b4-11ec-aa8a-07d4fef0bef1"),
					Type:   action.TypeConditionCallDigits,
					Option: []byte(`{"length": 3, "false_target_id": "c37492fa-92b4-11ec-94a0-1bfcaf781964"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("c3434cae-92b4-11ec-aa8a-07d4fef0bef1"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"length": 3, "false_target_id": "c37492fa-92b4-11ec-94a0-1bfcaf781964"}`),
							},
							{
								ID:   uuid.FromStringOrNil("7dac84b8-d58e-11ec-8b33-b76f5cf8651a"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("c37492fa-92b4-11ec-94a0-1bfcaf781964"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			uuid.FromStringOrNil("c2dbc228-92b4-11ec-8cc9-3358e0b8bbb5"),
			uuid.FromStringOrNil("c37492fa-92b4-11ec-94a0-1bfcaf781964"),

			"1",
			&activeflow.Activeflow{
				ReferenceID:     uuid.FromStringOrNil("c2dbc228-92b4-11ec-8cc9-3358e0b8bbb5"),
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("c37492fa-92b4-11ec-94a0-1bfcaf781964"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("c3434cae-92b4-11ec-aa8a-07d4fef0bef1"),
					Type:   action.TypeConditionCallDigits,
					Option: []byte(`{"length": 3, "false_target_id": "c37492fa-92b4-11ec-94a0-1bfcaf781964"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("c3434cae-92b4-11ec-aa8a-07d4fef0bef1"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"length": 3, "false_target_id": "c37492fa-92b4-11ec-94a0-1bfcaf781964"}`),
							},
							{
								ID:   uuid.FromStringOrNil("7dac84b8-d58e-11ec-8b33-b76f5cf8651a"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("c37492fa-92b4-11ec-94a0-1bfcaf781964"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			&action.Action{
				ID:   uuid.FromStringOrNil("c37492fa-92b4-11ec-94a0-1bfcaf781964"),
				Type: action.TypeAnswer,
			},
		},
		{
			"key fail",

			&activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("6ef04e44-92b5-11ec-a70a-1b80e125f020"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("6f553cd2-92b5-11ec-a9cc-070ec1a9c665"),
					Type:   action.TypeConditionCallDigits,
					Option: []byte(`{"key": "5", "false_target_id": "6f893f3c-92b5-11ec-9c9d-437fc938558b"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("6f553cd2-92b5-11ec-a9cc-070ec1a9c665"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"key": "5", "false_target_id": "6f893f3c-92b5-11ec-9c9d-437fc938558b"}`),
							},
							{
								ID:   uuid.FromStringOrNil("a128f228-d58e-11ec-9a02-c70b96d93760"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("6f893f3c-92b5-11ec-9c9d-437fc938558b"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			uuid.FromStringOrNil("6ef04e44-92b5-11ec-a70a-1b80e125f020"),
			uuid.FromStringOrNil("6f893f3c-92b5-11ec-9c9d-437fc938558b"),

			"123",
			&activeflow.Activeflow{
				ReferenceID:     uuid.FromStringOrNil("6ef04e44-92b5-11ec-a70a-1b80e125f020"),
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("6f893f3c-92b5-11ec-9c9d-437fc938558b"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("6f553cd2-92b5-11ec-a9cc-070ec1a9c665"),
					Type:   action.TypeConditionCallDigits,
					Option: []byte(`{"key": "5", "false_target_id": "6f893f3c-92b5-11ec-9c9d-437fc938558b"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("6f553cd2-92b5-11ec-a9cc-070ec1a9c665"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"key": "5", "false_target_id": "6f893f3c-92b5-11ec-9c9d-437fc938558b"}`),
							},
							{
								ID:   uuid.FromStringOrNil("a128f228-d58e-11ec-9a02-c70b96d93760"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("6f893f3c-92b5-11ec-9c9d-437fc938558b"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
			&action.Action{
				ID:   uuid.FromStringOrNil("6f893f3c-92b5-11ec-9c9d-437fc938558b"),
				Type: action.TypeAnswer,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				stackmapHandler: mockStack,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGetDigits(ctx, tt.callID).Return(tt.responseDigits, nil)
			mockStack.EXPECT().GetAction(tt.activeFlow.StackMap, tt.activeFlow.CurrentStackID, tt.expectTargetActionID, false).Return(stack.IDMain, tt.responseAction, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectReqActiveFlow).Return(nil)

			if err := h.actionHandleConditionCallDigits(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleConditionCallStatus(t *testing.T) {

	tests := []struct {
		name string

		activeFlow *activeflow.Activeflow

		responseCall *cmcall.Call
	}{
		{
			"normal",

			&activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("4980416e-9832-11ec-b189-5f96149e7ed8"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("49e01864-9832-11ec-a2de-8f0b27470613"),
					Type:   action.TypeConditionCallStatus,
					Option: []byte(`{"status": "ringing", "false_target_id": "52c1da9e-9832-11ec-9fc6-1b6bb10dc345"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("49e01864-9832-11ec-a2de-8f0b27470613"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"status": "ringing", "false_target_id": "52c1da9e-9832-11ec-9fc6-1b6bb10dc345"}`),
							},
							{
								ID:   uuid.FromStringOrNil("5beb1950-9832-11ec-9c32-d7afb89fde90"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("52c1da9e-9832-11ec-9fc6-1b6bb10dc345"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4980416e-9832-11ec-b189-5f96149e7ed8"),
				},
				Status: cmcall.StatusRinging,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.activeFlow.ReferenceID).Return(tt.responseCall, nil)

			if err := h.actionHandleConditionCallStatus(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleConditionCallStatusFalse(t *testing.T) {

	tests := []struct {
		name string

		activeFlow *activeflow.Activeflow

		expectTargetID uuid.UUID
		responseAction *action.Action

		responseCall        *cmcall.Call
		expectReqActiveFlow *activeflow.Activeflow
	}{
		{
			"normal",
			&activeflow.Activeflow{
				ReferenceID: uuid.FromStringOrNil("2a497210-9833-11ec-9c8c-6f81d7341b91"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("2a9dcc20-9833-11ec-8c07-4f5b407e5cdd"),
					Type:   action.TypeConditionCallStatus,
					Option: []byte(`{"status": "ringing", "false_target_id": "2afd89c6-9833-11ec-8e96-37a807af7aa9"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("2a9dcc20-9833-11ec-8c07-4f5b407e5cdd"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"status": "ringing", "false_target_id": "2afd89c6-9833-11ec-8e96-37a807af7aa9"}`),
							},
							{
								ID:   uuid.FromStringOrNil("2ad1f3ce-9833-11ec-9625-13a4b9bd21a0"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("2afd89c6-9833-11ec-8e96-37a807af7aa9"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			uuid.FromStringOrNil("2afd89c6-9833-11ec-8e96-37a807af7aa9"),
			&action.Action{
				ID:   uuid.FromStringOrNil("2afd89c6-9833-11ec-8e96-37a807af7aa9"),
				Type: action.TypeAnswer,
			},

			&cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a497210-9833-11ec-9c8c-6f81d7341b91"),
				},
				Status: cmcall.StatusProgressing,
			},
			&activeflow.Activeflow{
				ReferenceID:     uuid.FromStringOrNil("2a497210-9833-11ec-9c8c-6f81d7341b91"),
				ForwardStackID:  stack.IDMain,
				ForwardActionID: uuid.FromStringOrNil("2afd89c6-9833-11ec-8e96-37a807af7aa9"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("2a9dcc20-9833-11ec-8c07-4f5b407e5cdd"),
					Type:   action.TypeConditionCallStatus,
					Option: []byte(`{"status": "ringing", "false_target_id": "2afd89c6-9833-11ec-8e96-37a807af7aa9"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("2a9dcc20-9833-11ec-8c07-4f5b407e5cdd"),
								Type:   action.TypeConditionCallDigits,
								Option: []byte(`{"status": "ringing", "false_target_id": "2afd89c6-9833-11ec-8e96-37a807af7aa9"}`),
							},
							{
								ID:   uuid.FromStringOrNil("2ad1f3ce-9833-11ec-9625-13a4b9bd21a0"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("2afd89c6-9833-11ec-8e96-37a807af7aa9"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				stackmapHandler: mockStack,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.activeFlow.ReferenceID).Return(tt.responseCall, nil)
			mockStack.EXPECT().GetAction(tt.activeFlow.StackMap, tt.activeFlow.CurrentStackID, tt.expectTargetID, false).Return(stack.IDMain, tt.responseAction, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.expectReqActiveFlow)

			if err := h.actionHandleConditionCallStatus(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleConditionDatetime_match(t *testing.T) {

	// test values
	minute := 0x01
	hour := 0x02
	day := 0x04
	month := 0x08
	weekdays := 0x10

	tests := []struct {
		name string

		conditions int
	}{
		{name: "minute", conditions: minute},
		{name: "hour", conditions: hour},
		{name: "day", conditions: day},
		{name: "month", conditions: month},
		{name: "weekdays", conditions: weekdays},

		{name: "min + hour", conditions: minute | hour},
		{name: "min + day", conditions: minute | day},
		{name: "min + month", conditions: minute | month},
		{name: "min + weekdays", conditions: minute | weekdays},

		{name: "hour + day", conditions: hour | day},
		{name: "hour + month", conditions: hour | month},
		{name: "hour + weekdays", conditions: hour | weekdays},

		{name: "day + month", conditions: day | month},
		{name: "day + weekdays", conditions: day | weekdays},

		{name: "month + weekdays", conditions: month | weekdays},

		{name: "min + hour + day", conditions: minute | hour | day},
		{name: "min + hour + month", conditions: minute | hour | month},
		{name: "min + hour + weekdays", conditions: minute | hour | weekdays},

		{name: "min + day + month", conditions: minute | day | month},
		{name: "min + day + weekdays", conditions: minute | day | weekdays},

		{name: "min + month + weekdays", conditions: minute | month | weekdays},

		{name: "hour + day + month", conditions: hour | day | month},
		{name: "hour + day + weekdays", conditions: hour | day | weekdays},

		{name: "day + month + weekdays", conditions: day | month | weekdays},

		{name: "min + hour + day + month", conditions: minute | hour | day | month},
		{name: "min + hour + day + weekdays", conditions: minute | hour | day | weekdays},

		{name: "hour + day + month + weekdays", conditions: minute | hour | day | month | weekdays},

		{name: "all", conditions: minute | hour | day | month | weekdays},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				stackmapHandler: mockStack,
			}

			af := &activeflow.Activeflow{
				ReferenceID:    uuid.FromStringOrNil("1bd34ec7-f306-4c4b-af0e-0f446f5561b1"),
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("177946a3-d1c3-4c5d-8468-739a1dd65d4c"),
					Type: action.TypeConditionDatetime,
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("177946a3-d1c3-4c5d-8468-739a1dd65d4c"),
								Type: action.TypeConditionDatetime,
							},
						},
					},
				},
			}

			ctx := context.Background()

			// get currnet time
			current := time.Now().UTC()

			// generate test options
			opt := &action.OptionConditionDatetime{
				Condition: action.OptionConditionCommonConditionLessEqual,
			}
			if tt.conditions&minute == minute {
				opt.Minute = current.Minute()
			}
			if tt.conditions&hour == hour {
				opt.Hour = current.Hour()
			}
			if tt.conditions&day == day {
				opt.Day = current.Day()
			}
			if tt.conditions&month == month {
				opt.Month = int(current.Month())
			}
			if tt.conditions&weekdays == weekdays {
				opt.Weekdays = []int{int(current.Weekday())}
			}

			tmp, _ := json.Marshal(opt)
			af.CurrentAction.Option = tmp

			if err := h.actionHandleConditionDatetime(ctx, af); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleConditionDatetime_unmatch(t *testing.T) {

	// test values
	minute := 0x01
	hour := 0x02
	day := 0x04
	month := 0x08
	weekdays := 0x10

	tests := []struct {
		name string

		conditions int
	}{
		{name: "minute", conditions: minute},
		{name: "hour", conditions: hour},
		{name: "day", conditions: day},
		{name: "month", conditions: month},
		{name: "weekdays", conditions: weekdays},

		{name: "min + hour", conditions: minute | hour},
		{name: "min + day", conditions: minute | day},
		{name: "min + month", conditions: minute | month},
		{name: "min + weekdays", conditions: minute | weekdays},

		{name: "hour + day", conditions: hour | day},
		{name: "hour + month", conditions: hour | month},
		{name: "hour + weekdays", conditions: hour | weekdays},

		{name: "day + month", conditions: day | month},
		{name: "day + weekdays", conditions: day | weekdays},

		{name: "month + weekdays", conditions: month | weekdays},

		{name: "min + hour + day", conditions: minute | hour | day},
		{name: "min + hour + month", conditions: minute | hour | month},
		{name: "min + hour + weekdays", conditions: minute | hour | weekdays},

		{name: "min + day + month", conditions: minute | day | month},
		{name: "min + day + weekdays", conditions: minute | day | weekdays},

		{name: "min + month + weekdays", conditions: minute | month | weekdays},

		{name: "hour + day + month", conditions: hour | day | month},
		{name: "hour + day + weekdays", conditions: hour | day | weekdays},

		{name: "day + month + weekdays", conditions: day | month | weekdays},

		{name: "min + hour + day + month", conditions: minute | hour | day | month},
		{name: "min + hour + day + weekdays", conditions: minute | hour | day | weekdays},

		{name: "hour + day + month + weekdays", conditions: minute | hour | day | month | weekdays},

		{name: "all", conditions: minute | hour | day | month | weekdays},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				stackmapHandler: mockStack,
			}

			af := &activeflow.Activeflow{
				ReferenceID:    uuid.FromStringOrNil("1bd34ec7-f306-4c4b-af0e-0f446f5561b1"),
				CurrentStackID: stack.IDMain,
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("177946a3-d1c3-4c5d-8468-739a1dd65d4c"),
					Type: action.TypeConditionDatetime,
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:   uuid.FromStringOrNil("177946a3-d1c3-4c5d-8468-739a1dd65d4c"),
								Type: action.TypeConditionDatetime,
							},
						},
					},
				},
			}

			ctx := context.Background()

			// get currnet time
			current := time.Now().UTC()

			// generate test options
			opt := &action.OptionConditionDatetime{
				Condition: action.OptionConditionCommonConditionGreater,
			}
			if tt.conditions&minute == minute {
				opt.Minute = current.Minute()
			}
			if tt.conditions&hour == hour {
				opt.Hour = current.Hour()
			}
			if tt.conditions&day == day {
				opt.Day = current.Day()
			}
			if tt.conditions&month == month {
				opt.Month = int(current.Month())
			}
			if tt.conditions&weekdays == weekdays {
				opt.Weekdays = []int{int(current.Weekday())}
			}

			tmp, _ := json.Marshal(opt)
			af.CurrentAction.Option = tmp

			mockStack.EXPECT().GetAction(af.StackMap, af.CurrentStackID, gomock.Any(), false).Return(stack.IDMain, &action.Action{}, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, af).Return(nil)

			if err := h.actionHandleConditionDatetime(ctx, af); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleConditionVariable_match(t *testing.T) {

	tests := []struct {
		name string

		activeFlow *activeflow.Activeflow
	}{
		{
			name: "string equal",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "==", "variable": "test value", "value_type": "string", "value_string": "test value"}`),
				},
			},
		},
		{
			name: "string not equal",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "!=", "variable": "test value 1", "value_type": "string", "value_string": "test value 2"}`),
				},
			},
		},
		{
			name: "string greater",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": ">", "variable": "test value 123", "value_type": "string", "value_string": "test value 111"}`),
				},
			},
		},
		{
			name: "string greater equal",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": ">=", "variable": "test value 123", "value_type": "string", "value_string": "test value 123"}`),
				},
			},
		},
		{
			name: "string less",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "<", "variable": "test value 111", "value_type": "string", "value_string": "test value 123"}`),
				},
			},
		},
		{
			name: "string less equal",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "<=", "variable": "test value 111", "value_type": "string", "value_string": "test value 111"}`),
				},
			},
		},
		{
			name: "number equal",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "==", "variable": "123", "value_type": "number", "value_number": 123}`),
				},
			},
		},
		{
			name: "number not equal",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "!=", "variable": "123", "value_type": "number", "value_number": 456}`),
				},
			},
		},
		{
			name: "number greater",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": ">", "variable": "123.1", "value_type": "number", "value_number": 111.1}`),
				},
			},
		},
		{
			name: "number greater equal",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": ">=", "variable": "123.1", "value_type": "number", "value_number": 123.1}`),
				},
			},
		},
		{
			name: "number less",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "<", "variable": "111.1", "value_type": "number", "value_number": 123.1}`),
				},
			},
		},
		{
			name: "number less equal",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "<=", "variable": "111.1", "value_type": "number", "value_number": 111.1}`),
				},
			},
		},
		{
			name: "length equal",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "==", "variable": "test length", "value_type": "length", "value_length": 11}`),
				},
			},
		},
		{
			name: "length not equal",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "!=", "variable": "test length", "value_type": "length", "value_length": 12}`),
				},
			},
		},
		{
			name: "length greater",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": ">", "variable": "test length", "value_type": "length", "value_length": 10}`),
				},
			},
		},
		{
			name: "length greater equal",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": ">=", "variable": "test length", "value_type": "length", "value_length": 11}`),
				},
			},
		},
		{
			name: "length less",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "<", "variable": "test length", "value_type": "length", "value_length": 12}`),
				},
			},
		},
		{
			name: "length less equal",

			activeFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "<=", "variable": "test length", "value_type": "length", "value_length": 11}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				variableHandler: mockVar,
			}

			ctx := context.Background()

			if err := h.actionHandleConditionVariable(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleConditionVariable_unmatch(t *testing.T) {

	tests := []struct {
		name string

		activeflow *activeflow.Activeflow
	}{
		{
			name: "string",

			activeflow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "==", "variable": "test value", "value_type": "string", "value_string": "test value unmatch"}`),
				},
			},
		},
		{
			name: "number",

			activeflow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "==", "variable": "123", "value_type": "number", "value_number": 123.1}`),
				},
			},
		},
		{
			name: "length",

			activeflow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					Type:   action.TypeConditionVariable,
					Option: []byte(`{"condition": "==", "variable": "12345", "value_type": "length", "value_length": 6}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				variableHandler: mockVar,
				stackmapHandler: mockStack,
			}

			ctx := context.Background()

			mockStack.EXPECT().GetAction(tt.activeflow.StackMap, tt.activeflow.CurrentStackID, gomock.Any(), false).Return(stack.IDMain, &action.Action{}, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.activeflow).Return(nil)

			if err := h.actionHandleConditionVariable(ctx, tt.activeflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleMessageSend(t *testing.T) {

	tests := []struct {
		name string

		activeFlow *activeflow.Activeflow

		expectCustomerID   uuid.UUID
		expectSource       *commonaddress.Address
		expectDestinations []commonaddress.Address
		expectText         string

		responseCall        *cmcall.Call
		expectReqActiveFlow *activeflow.Activeflow
	}{
		{
			name: "normal",

			activeFlow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9332ed50-dc86-11ec-ba14-8b96908d546b"),
					CustomerID: uuid.FromStringOrNil("184d60ac-a2cf-11ec-a800-fb524059f338"),
				},
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4dcd7b64-a2ce-11ec-8711-6f247c91aa5d"),
					Type:   action.TypeMessageSend,
					Option: []byte(`{"source": {"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"},{"type": "tel", "target": "+821100000003"}], "text": "hello world. test variable: ttteeesssttt"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4dcd7b64-a2ce-11ec-8711-6f247c91aa5d"),
								Type:   action.TypeMessageSend,
								Option: []byte(`{"source": {"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"},{"type": "tel", "target": "+821100000003"}], "text": "hello world. test variable: ttteeesssttt"}`),
							},
							{
								ID:   uuid.FromStringOrNil("9c06bcfa-a2ce-11ec-bcc6-5bc0b10cd014"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("9c311c5c-a2ce-11ec-b1a2-d735b06f36c8"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},

			expectCustomerID: uuid.FromStringOrNil("184d60ac-a2cf-11ec-a800-fb524059f338"),
			expectSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			expectDestinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000003",
				},
			},
			expectText: "hello world. test variable: ttteeesssttt",

			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4d496946-a2ce-11ec-a96e-eb9ac0dce8e7"),
				},
				Status: cmcall.StatusProgressing,
			},
			expectReqActiveFlow: &activeflow.Activeflow{
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4dcd7b64-a2ce-11ec-8711-6f247c91aa5d"),
					Type:   action.TypeMessageSend,
					Option: []byte(`{"source": {"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "text": "hello world"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4dcd7b64-a2ce-11ec-8711-6f247c91aa5d"),
								Type:   action.TypeMessageSend,
								Option: []byte(`{"source": {"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "text": "hello world"}`),
							},
							{
								ID:   uuid.FromStringOrNil("9c06bcfa-a2ce-11ec-bcc6-5bc0b10cd014"),
								Type: action.TypeAnswer,
							},
							{
								ID:   uuid.FromStringOrNil("9c311c5c-a2ce-11ec-b1a2-d735b06f36c8"),
								Type: action.TypeAnswer,
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				variableHandler: mockVar,
			}

			ctx := context.Background()

			mockReq.EXPECT().MessageV1MessageSend(ctx, uuid.Nil, tt.expectCustomerID, tt.expectSource, tt.expectDestinations, tt.expectText).Return(&mmmessage.Message{}, nil)

			if err := h.actionHandleMessageSend(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleTranscribeRecording(t *testing.T) {

	type test struct {
		name       string
		activeflow *activeflow.Activeflow

		callID     uuid.UUID
		customerID uuid.UUID
		language   string

		responseCall *cmcall.Call
	}

	tests := []test{
		{
			"normal",

			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("321089b0-8795-11ec-907f-0bae67409ef6"),
				},
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("66e928da-9b42-11eb-8da0-3783064961f6"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("673ed4d8-9b42-11eb-bb79-ff02c5650f35"),
					Type:   action.TypeTranscribeRecording,
					Option: []byte(`{"language":"en-US"}`),
				},
			},

			uuid.FromStringOrNil("66e928da-9b42-11eb-8da0-3783064961f6"),
			uuid.FromStringOrNil("321089b0-8795-11ec-907f-0bae67409ef6"),
			"en-US",

			&cmcall.Call{
				RecordingIDs: []uuid.UUID{
					uuid.FromStringOrNil("01e4c8a0-82a3-11ed-b30e-8f633f969f44"),
					uuid.FromStringOrNil("021e88e2-82a3-11ed-a3de-fba809f8b728"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.activeflow.ReferenceID).Return(tt.responseCall, nil)
			for _, recordingID := range tt.responseCall.RecordingIDs {
				mockReq.EXPECT().TranscribeV1TranscribeStart(ctx, tt.activeflow.Identity.CustomerID, tmtranscribe.ReferenceTypeRecording, recordingID, tt.language, tmtranscribe.DirectionBoth).Return(&tmtranscribe.Transcribe{}, nil)
			}

			if err := h.actionHandleTranscribeRecording(ctx, tt.activeflow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleTranscribeStart(t *testing.T) {

	type test struct {
		name       string
		activeFlow *activeflow.Activeflow

		customerID    uuid.UUID
		referenceID   uuid.UUID
		referenceType tmtranscribe.ReferenceType
		language      string

		response *tmtranscribe.Transcribe
	}

	tests := []test{
		{
			"normal",
			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("b4d3fb66-8795-11ec-997c-7f2786edbef2"),
				},
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("0737bd5c-0c08-11ec-9ba8-3bc700c21fd4"),
					Type:   action.TypeTranscribeStart,
					Option: []byte(`{"language":"en-US","webhook_uri":"http://test.com/webhook","webhook_method":"POST"}`),
				},
			},

			uuid.FromStringOrNil("b4d3fb66-8795-11ec-997c-7f2786edbef2"),
			uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
			tmtranscribe.ReferenceTypeCall,
			"en-US",

			&tmtranscribe.Transcribe{
				ID:            uuid.FromStringOrNil("e1e69720-0c08-11ec-9f5c-db1f63f63215"),
				ReferenceType: tmtranscribe.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
				HostID:        uuid.FromStringOrNil("f91b4f58-0c08-11ec-88fd-cfbbb1957a54"),
				Language:      "en-US",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}
			ctx := context.Background()

			mockReq.EXPECT().TranscribeV1TranscribeStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.language, tmtranscribe.DirectionBoth).Return(tt.response, nil)
			if err := h.actionHandleTranscribeStart(ctx, tt.activeFlow); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_actionHandleCall(t *testing.T) {

	tests := []struct {
		name string
		af   *activeflow.Activeflow

		source         *commonaddress.Address
		destinations   []commonaddress.Address
		flowID         uuid.UUID
		actions        []action.Action
		masterCallID   uuid.UUID
		earlyExecution bool

		responseFlow      *flow.Flow
		responseCall      []*cmcall.Call
		responseGroupcall []*cmgroupcall.Groupcall
	}{
		{
			"have all",
			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7fb0af78-a942-11ec-8c60-67384864d90a"),
					CustomerID: uuid.FromStringOrNil("4ea19a38-a941-11ec-b04d-bb69d70f3461"),
				},
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("4edb5840-a941-11ec-b674-93c2ef347891"),
					Type:   action.TypeCall,
					Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "flow_id": "15df43ee-a941-11ec-a903-2b7266f49e4b", "early_execution": true}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("4edb5840-a941-11ec-b674-93c2ef347891"),
								Type:   action.TypeCall,
								Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "flow_id": "15df43ee-a941-11ec-a903-2b7266f49e4b"}`),
							},
						},
					},
				},
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			uuid.FromStringOrNil("15df43ee-a941-11ec-a903-2b7266f49e4b"),
			[]action.Action{},
			uuid.Nil,
			true,

			&flow.Flow{},
			[]*cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a49dda82-a941-11ec-b5a9-9baf180541e9"),
					},
				},
			},
			[]*cmgroupcall.Groupcall{},
		},
		{
			"2 destinations with flow id",
			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("fe7b3838-a941-11ec-b3d6-5337e9635d88"),
					CustomerID: uuid.FromStringOrNil("fea0d30e-a941-11ec-a38a-478b19a5dfd2"),
				},
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("fec56ebc-a941-11ec-8e4c-9fafab93ddcc"),
					Type:   action.TypeCall,
					Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"},{"type": "tel", "target": "+821100000003"}], "flow_id": "feedab34-a941-11ec-a6a8-1bbdada16b4d"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("fec56ebc-a941-11ec-8e4c-9fafab93ddcc"),
								Type:   action.TypeCall,
								Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [,{"type": "tel", "target": "+821100000003"}], "flow_id": "feedab34-a941-11ec-a6a8-1bbdada16b4d"}`),
							},
						},
					},
				},
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000003",
				},
			},
			uuid.FromStringOrNil("feedab34-a941-11ec-a6a8-1bbdada16b4d"),
			[]action.Action{},
			uuid.Nil,
			false,

			&flow.Flow{},
			[]*cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ff16e1f2-a941-11ec-b2c1-c3aa4ce144a0"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ff477790-a941-11ec-8475-035d159c8a77"),
					},
				},
			},
			[]*cmgroupcall.Groupcall{},
		},
		{
			"single destination with actions",
			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("38a78fec-a943-11ec-b279-c7b264a9d36a"),
					CustomerID: uuid.FromStringOrNil("38d87102-a943-11ec-99aa-5fc910add207"),
				},
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("39063286-a943-11ec-b54c-d3be23fdf738"),
					Type:   action.TypeCall,
					Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "actions": [{"type": "answer"}, {"type": "talk", "option": {"text": "hello world"}}]}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("39063286-a943-11ec-b54c-d3be23fdf738"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "actions": [{"type": "answer"}, {"type": "talk", "option": {"text": "hello world"}}]}`),
							},
						},
					},
				},
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			uuid.Nil,
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text": "hello world"}`),
				},
			},
			uuid.Nil,
			false,

			&flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3936013c-a943-11ec-bdf1-af72361eecf4"),
				},
			},
			[]*cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("39691e00-a943-11ec-a69c-7f0e69becb70"),
					},
				},
			},
			[]*cmgroupcall.Groupcall{},
		},
		{
			"2 destinations with actions",
			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f3ddb5ca-a943-11ec-be88-ebdd9b1c7f1b"),
					CustomerID: uuid.FromStringOrNil("f40e389e-a943-11ec-83a7-7f90e0c22e96"),
				},
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("f43f85de-a943-11ec-9ba5-b3f019b002e7"),
					Type:   action.TypeCall,
					Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "actions": [{"type": "answer"}, {"type": "talk", "option": {"text": "hello world"}}]}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("f43f85de-a943-11ec-9ba5-b3f019b002e7"),
								Type:   action.TypeConnect,
								Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "actions": [{"type": "answer"}, {"type": "talk", "option": {"text": "hello world"}}]}`),
							},
						},
					},
				},
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			uuid.Nil,
			[]action.Action{
				{
					Type: action.TypeAnswer,
				},
				{
					Type:   action.TypeTalk,
					Option: []byte(`{"text": "hello world"}`),
				},
			},
			uuid.Nil,
			false,

			&flow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f46f3e96-a943-11ec-b4e1-2bfce6b84c2b"),
				},
			},
			[]*cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("f4a90298-a943-11ec-b31a-bf1f552ced44"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("f4dd1e0c-a943-11ec-9295-6f71727bd164"),
					},
				},
			},
			[]*cmgroupcall.Groupcall{},
		},
		{
			"single destination with flow id and chained",
			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ec55367a-a993-11ec-9eaf-3bd79ecebfdb"),
					CustomerID: uuid.FromStringOrNil("ec7e7e4a-a993-11ec-85a1-2f8c41cac00e"),
				},
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("3f0cd396-a994-11ec-95db-e73c30df842c"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("eca44350-a993-11ec-bb4d-cbe7cec73166"),
					Type:   action.TypeCall,
					Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "flow_id": "ecc964dc-a993-11ec-9c4c-13e5b3d40ea8", "chained": true}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("eca44350-a993-11ec-bb4d-cbe7cec73166"),
								Type:   action.TypeCall,
								Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "flow_id": "ecc964dc-a993-11ec-9c4c-13e5b3d40ea8", "chained": true}`),
							},
						},
					},
				},
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			uuid.FromStringOrNil("ecc964dc-a993-11ec-9c4c-13e5b3d40ea8"),
			[]action.Action{},
			uuid.FromStringOrNil("3f0cd396-a994-11ec-95db-e73c30df842c"),
			false,

			&flow.Flow{},
			[]*cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2aafe7e4-a994-11ec-8bae-338c6a067225"),
					},
				},
			},
			[]*cmgroupcall.Groupcall{},
		},
		{
			"single destination with flow id and chained but reference type is message",
			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("87a4f032-a996-11ec-b260-4b6f3f52e1c9"),
					CustomerID: uuid.FromStringOrNil("87ede80a-a996-11ec-9086-d77d045a5f03"),
				},
				ReferenceType: activeflow.ReferenceTypeMessage,
				ReferenceID:   uuid.FromStringOrNil("8819cf88-a996-11ec-bd8b-f3a7053103f1"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("eca44350-a993-11ec-bb4d-cbe7cec73166"),
					Type:   action.TypeCall,
					Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "flow_id": "88497954-a996-11ec-b194-a71b02fcb6a8", "chained": true}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("eca44350-a993-11ec-bb4d-cbe7cec73166"),
								Type:   action.TypeCall,
								Option: []byte(`{"source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "flow_id": "88497954-a996-11ec-b194-a71b02fcb6a8", "chained": true}`),
							},
						},
					},
				},
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			uuid.FromStringOrNil("88497954-a996-11ec-b194-a71b02fcb6a8"),
			[]action.Action{},
			uuid.Nil,
			false,

			&flow.Flow{},
			[]*cmcall.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8873be9e-a996-11ec-993b-438622bb78da"),
					},
				},
			},
			[]*cmgroupcall.Groupcall{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler:   mockAction,
				variableHandler: mockVar,
			}

			ctx := context.Background()

			flowID := tt.flowID
			if flowID == uuid.Nil {
				mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.af.Identity.CustomerID, flow.TypeFlow, "", "", tt.actions, false).Return(tt.responseFlow, nil)
				flowID = tt.responseFlow.ID
			}

			mockReq.EXPECT().CallV1CallsCreate(ctx, tt.af.Identity.CustomerID, flowID, tt.masterCallID, tt.source, tt.destinations, tt.earlyExecution, false).Return(tt.responseCall, tt.responseGroupcall, nil)

			if errCall := h.actionHandleCall(ctx, tt.af); errCall != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCall)
			}
		})
	}
}

func Test_actionHandleVariableSet(t *testing.T) {

	tests := []struct {
		name         string
		activeflowID uuid.UUID
		af           *activeflow.Activeflow

		expectVariables map[string]string
	}{
		{
			"single destination with flow id",
			uuid.FromStringOrNil("a65dd1f8-ce47-11ec-bc53-ff630bb4b69b"),
			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a65dd1f8-ce47-11ec-bc53-ff630bb4b69b"),
					CustomerID: uuid.FromStringOrNil("4ea19a38-a941-11ec-b04d-bb69d70f3461"),
				},
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("a6896cc8-ce47-11ec-8fff-1f2ab0d61b07"),
					Type:   action.TypeVariableSet,
					Option: []byte(`{"key": "key 1","value":"value 1"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("a6896cc8-ce47-11ec-8fff-1f2ab0d61b07"),
								Type:   action.TypeVariableSet,
								Option: []byte(`{"key": "key 1","value":"value 1"}`),
							},
						},
					},
				},
			},

			map[string]string{
				"key 1": "value 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockVariable := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler:   mockAction,
				variableHandler: mockVariable,
			}

			ctx := context.Background()

			mockVariable.EXPECT().SetVariable(ctx, tt.af.Identity.ID, tt.expectVariables).Return(nil)

			if errCall := h.actionHandleVariableSet(ctx, tt.af); errCall != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errCall)
			}
		})
	}
}

func Test_actionHandleWebhookSend(t *testing.T) {

	tests := []struct {
		name string

		af *activeflow.Activeflow

		expectSync     bool
		expectURI      string
		expectMethod   wmwebhook.MethodType
		expectDataType wmwebhook.DataType
		expectData     []byte
	}{
		{
			"normal",

			&activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("284a82d4-d9eb-11ec-aa89-3fb4df202ec8"),
					CustomerID: uuid.FromStringOrNil("4ea19a38-a941-11ec-b04d-bb69d70f3461"),
				},
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("63574506-d9eb-11ec-a261-67821a8699b0"),
					Type:   action.TypeWebhookSend,
					Option: []byte(`{"sync":false,"uri":"test.com","method":"POST","data_type":"application/json","data":"test message."}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("63574506-d9eb-11ec-a261-67821a8699b0"),
								Type:   action.TypeWebhookSend,
								Option: []byte(`{"sync":false,"uri":"test.com","method":"POST","data_type":"application/json","data":"test message."}`),
							},
						},
					},
				},
			},

			true,
			"test.com",
			wmwebhook.MethodTypePOST,
			wmwebhook.DataTypeJSON,
			[]byte(`test message.`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockVariable := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler:   mockAction,
				variableHandler: mockVariable,
			}

			ctx := context.Background()

			mockReq.EXPECT().WebhookV1WebhookSendToDestination(ctx, tt.af.Identity.CustomerID, tt.expectURI, tt.expectMethod, tt.expectDataType, tt.expectData).Return(nil)

			if errCall := h.actionHandleWebhookSend(ctx, tt.af); errCall != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v", errCall)
			}

			time.Sleep(500 * time.Millisecond)
		})
	}
}

func Test_actionHandleConversationSend(t *testing.T) {

	tests := []struct {
		name string

		af         *activeflow.Activeflow
		optionText string

		expectSync           bool
		expectURI            string
		expectMethod         wmwebhook.MethodType
		expectDataType       wmwebhook.DataType
		expectConversationID uuid.UUID
		expectText           string
	}{
		{
			name: "normal",

			af: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5c82ef66-f474-11ec-b5da-07a1796b759d"),
					CustomerID: uuid.FromStringOrNil("4ea19a38-a941-11ec-b04d-bb69d70f3461"),
				},
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("5ce765cc-f474-11ec-b2e9-eb0259d7bacf"),
					Type:   action.TypeConversationSend,
					Option: []byte(`{"conversation_id":"7e5116e2-f477-11ec-9c08-b343a05abaee","text":"test message.","sync":true}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("5ce765cc-f474-11ec-b2e9-eb0259d7bacf"),
								Type:   action.TypeConversationSend,
								Option: []byte(`{"conversation_id":"7e5116e2-f477-11ec-9c08-b343a05abaee","text":"test message.","sync":true}`),
							},
						},
					},
				},
			},
			optionText: "test message.",

			expectSync:           true,
			expectURI:            "test.com",
			expectMethod:         wmwebhook.MethodTypePOST,
			expectDataType:       wmwebhook.DataTypeJSON,
			expectConversationID: uuid.FromStringOrNil("7e5116e2-f477-11ec-9c08-b343a05abaee"),
			expectText:           "test message.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockVariable := variablehandler.NewMockVariableHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler:   mockAction,
				variableHandler: mockVariable,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConversationV1MessageSend(ctx, tt.expectConversationID, tt.expectText, []conversationmedia.Media{}).Return(&conversationmessage.Message{}, nil)

			if errCall := h.actionHandleConversationSend(ctx, tt.af); errCall != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v", errCall)
			}

			time.Sleep(500 * time.Millisecond)
		})
	}
}

func Test_actionHandleChatbotTalk(t *testing.T) {

	tests := []struct {
		name string

		activeflow *activeflow.Activeflow

		responseService *commonservice.Service
		responseStack   *stack.Stack

		expectChatbotID     uuid.UUID
		expectActiveflowID  uuid.UUID
		expectReferenceType cbchatbotcall.ReferenceType
		expectReferenceID   uuid.UUID
		expectGender        cbchatbotcall.Gender
		expectLanguage      string
	}{
		{
			name: "normal",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ba68f5ae-a8f5-11ed-8a90-27dd6442f0e6"),
					CustomerID: uuid.FromStringOrNil("baba6a92-a8f5-11ed-926f-fb93cea60103"),
				},
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("bb41c82a-a8f5-11ed-a9ce-b7bbefea1a83"),
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("baea2278-a8f5-11ed-bac8-cf57f2d8de20"),
					Type:   action.TypeChatbotTalk,
					Option: []byte(`{"chatbot_id":"bb17f504-a8f5-11ed-a974-2f810c03cbf8","gender":"female","language":"en-US"}`),
				},

				StackMap: map[uuid.UUID]*stack.Stack{
					stack.IDMain: {
						ID: stack.IDMain,
						Actions: []action.Action{
							{
								ID:     uuid.FromStringOrNil("baea2278-a8f5-11ed-bac8-cf57f2d8de20"),
								Type:   action.TypeChatbotTalk,
								Option: []byte(`{"chatbot_id":"bb17f504-a8f5-11ed-a974-2f810c03cbf8","gender":"female","language":"en-US"}`),
							},
						},
					},
				},
			},

			responseService: &commonservice.Service{
				ID:   uuid.FromStringOrNil("bb68f67a-a8f5-11ed-9a2f-63b973d60f8c"),
				Type: commonservice.TypeChatbotcall,
				PushActions: []action.Action{
					{
						ID: uuid.FromStringOrNil("bb9239cc-a8f5-11ed-b21f-f7e43c6b6a60"),
					},
				},
			},
			responseStack: &stack.Stack{
				ID: uuid.FromStringOrNil("bb68f67a-a8f5-11ed-9a2f-63b973d60f8c"),
				Actions: []action.Action{
					{
						ID: uuid.FromStringOrNil("bb9239cc-a8f5-11ed-b21f-f7e43c6b6a60"),
					},
				},
			},

			expectChatbotID:     uuid.FromStringOrNil("bb17f504-a8f5-11ed-a974-2f810c03cbf8"),
			expectActiveflowID:  uuid.FromStringOrNil("ba68f5ae-a8f5-11ed-8a90-27dd6442f0e6"),
			expectReferenceType: cbchatbotcall.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("bb41c82a-a8f5-11ed-a9ce-b7bbefea1a83"),
			expectGender:        cbchatbotcall.GenderFemale,
			expectLanguage:      "en-US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockVariable := variablehandler.NewMockVariableHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler:   mockAction,
				variableHandler: mockVariable,
				stackmapHandler: mockStack,
			}

			ctx := context.Background()

			mockReq.EXPECT().ChatbotV1ServiceTypeChabotcallStart(ctx, tt.expectChatbotID, tt.expectActiveflowID, tt.expectReferenceType, tt.expectReferenceID, tt.expectGender, tt.expectLanguage, 3000).Return(tt.responseService, nil)

			// push stack
			mockStack.EXPECT().PushStackByActions(tt.activeflow.StackMap, tt.responseService.ID, tt.responseService.PushActions, tt.activeflow.CurrentStackID, tt.activeflow.CurrentAction.ID).Return(tt.responseStack, nil)
			mockDB.EXPECT().ActiveflowUpdate(ctx, gomock.Any()).Return(nil)

			if errCall := h.actionHandleChatbotTalk(ctx, tt.activeflow); errCall != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v", errCall)
			}

			time.Sleep(500 * time.Millisecond)
		})
	}
}

func Test_actionHandleEmailSend(t *testing.T) {

	tests := []struct {
		name string

		activeflow *activeflow.Activeflow

		responseEmail *ememail.Email

		expectDestinations []commonaddress.Address
		expectSubject      string
		expectContent      string
		expectAttachments  []ememail.Attachment
	}{
		{
			name: "normal",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e480deea-00f2-11f0-ba53-93fa896cde1f"),
					CustomerID: uuid.FromStringOrNil("e4b325e4-00f2-11f0-8cc2-c36f5776f5b2"),
				},
				CurrentAction: action.Action{
					ID:     uuid.FromStringOrNil("e4dc3eca-00f2-11f0-8a64-2b73140e49f0"),
					Type:   action.TypeChatbotTalk,
					Option: []byte(`{"destinations":[{"type":"email","target":"test@voipbin.net","target_name":"test name"}],"subject":"test subject","content":"test content","attachments":[{"reference_type":"recording","reference_id":"e50a99e6-00f2-11f0-957d-d36fc32d6b0d"}]}`),
				},
			},

			responseEmail: &ememail.Email{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e5380818-00f2-11f0-bb6f-cfdafff6be53"),
				},
			},

			expectDestinations: []commonaddress.Address{
				{
					Type:       commonaddress.TypeEmail,
					Target:     "test@voipbin.net",
					TargetName: "test name",
				},
			},
			expectSubject: "test subject",
			expectContent: "test content",
			expectAttachments: []ememail.Attachment{
				{
					ReferenceType: ememail.AttachmentReferenceTypeRecording,
					ReferenceID:   uuid.FromStringOrNil("e50a99e6-00f2-11f0-957d-d36fc32d6b0d"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAction := actionhandler.NewMockActionHandler(mc)
			mockVariable := variablehandler.NewMockVariableHandler(mc)
			mockStack := stackmaphandler.NewMockStackmapHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,

				actionHandler:   mockAction,
				variableHandler: mockVariable,
				stackmapHandler: mockStack,
			}
			ctx := context.Background()

			mockReq.EXPECT().EmailV1EmailSend(ctx, tt.activeflow.Identity.CustomerID, tt.activeflow.Identity.ID, tt.expectDestinations, tt.expectSubject, tt.expectContent, tt.expectAttachments).Return(tt.responseEmail, nil)

			if errCall := h.actionHandleEmailSend(ctx, tt.activeflow); errCall != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v", errCall)
			}
		})
	}
}
