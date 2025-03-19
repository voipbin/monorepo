package requesthandler

import (
	"context"
	"reflect"
	"testing"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_FlowV1ActiveflowCreate(t *testing.T) {

	tests := []struct {
		name string

		activeflowID  uuid.UUID
		customerID    uuid.UUID
		referenceType fmactiveflow.ReferenceType
		referenceID   uuid.UUID
		flowID        uuid.UUID

		response *sock.Response

		expectedQueue   string
		expectedRequest *sock.Request
		expectedRes     *fmactiveflow.Activeflow
	}{
		{
			name: "all",

			activeflowID:  uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c"),
			customerID:    uuid.FromStringOrNil("d1f87c4a-049b-11f0-8861-1b914bb9707d"),
			referenceType: fmactiveflow.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
			flowID:        uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"aa847807-6cc4-4713-9dec-53a42840e74c","flow_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50","reference_type":"call","reference_id":"447e712e-82d8-11eb-8900-7b97c080ddd8","customer_id":"f42b33e2-7f4d-11ec-8c86-ebf558a4306c","current_action":{"id":"00000000-0000-0000-0000-000000000001","type":""},"actions":[],"tm_create":"","tm_update":"","tm_delete":""}`),
			},

			expectedQueue: "bin-manager.flow-manager.request",
			expectedRequest: &sock.Request{
				URI:      "/v1/activeflows",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"aa847807-6cc4-4713-9dec-53a42840e74c","customer_id":"d1f87c4a-049b-11f0-8861-1b914bb9707d","flow_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50","reference_type":"call","reference_id":"447e712e-82d8-11eb-8900-7b97c080ddd8"}`),
			},
			expectedRes: &fmactiveflow.Activeflow{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c"),
					CustomerID: uuid.FromStringOrNil("f42b33e2-7f4d-11ec-8c86-ebf558a4306c"),
				},
				ReferenceType: fmactiveflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
				FlowID:        uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),
				CurrentAction: fmaction.Action{
					ID: fmaction.IDStart,
				},
			},
		},
		{
			name: "empty id",

			activeflowID:  uuid.Nil,
			customerID:    uuid.Nil,
			referenceType: fmactiveflow.ReferenceTypeMessage,
			referenceID:   uuid.FromStringOrNil("a8d145b8-a7b5-11ec-ac30-6b8228b173eb"),
			flowID:        uuid.FromStringOrNil("a929cd00-a7b5-11ec-a2bd-d375b3bee397"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"00000000-0000-0000-0000-000000000000","flow_id":"a929cd00-a7b5-11ec-a2bd-d375b3bee397","reference_type":"message","reference_id":"a8d145b8-a7b5-11ec-ac30-6b8228b173eb","customer_id":"f42b33e2-7f4d-11ec-8c86-ebf558a4306c","current_action":{"id":"00000000-0000-0000-0000-000000000001","type":""},"actions":[],"tm_create":"","tm_update":"","tm_delete":""}`),
			},

			expectedQueue: "bin-manager.flow-manager.request",
			expectedRequest: &sock.Request{
				URI:      "/v1/activeflows",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"00000000-0000-0000-0000-000000000000","customer_id":"00000000-0000-0000-0000-000000000000","flow_id":"a929cd00-a7b5-11ec-a2bd-d375b3bee397","reference_type":"message","reference_id":"a8d145b8-a7b5-11ec-ac30-6b8228b173eb"}`),
			},
			expectedRes: &fmactiveflow.Activeflow{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("00000000-0000-0000-0000-000000000000"),
					CustomerID: uuid.FromStringOrNil("f42b33e2-7f4d-11ec-8c86-ebf558a4306c"),
				},
				ReferenceType: fmactiveflow.ReferenceTypeMessage,
				ReferenceID:   uuid.FromStringOrNil("a8d145b8-a7b5-11ec-ac30-6b8228b173eb"),
				FlowID:        uuid.FromStringOrNil("a929cd00-a7b5-11ec-a2bd-d375b3bee397"),
				CurrentAction: fmaction.Action{
					ID: fmaction.IDStart,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectedQueue, tt.expectedRequest).Return(tt.response, nil)
			res, err := reqHandler.FlowV1ActiveflowCreate(ctx, tt.activeflowID, tt.customerID, tt.flowID, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}

		})
	}
}

func Test_FlowV1ActiveflowGetNextAction(t *testing.T) {

	tests := []struct {
		name string

		activeflowID    uuid.UUID
		currentActionID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *fmaction.Action
	}{
		{
			"normal",

			uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
			uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),

			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:      "/v1/activeflows/447e712e-82d8-11eb-8900-7b97c080ddd8/next",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"current_action_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"}`),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"e52c5766-57c7-11ec-836b-333ce17a1ce6","type":"answer"}`),
			},
			&fmaction.Action{
				ID:   uuid.FromStringOrNil("e52c5766-57c7-11ec-836b-333ce17a1ce6"),
				Type: fmaction.TypeAnswer,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowV1ActiveflowGetNextAction(context.Background(), tt.activeflowID, tt.currentActionID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_FlowV1ActiveflowUpdateForwardActionID(t *testing.T) {

	tests := []struct {
		name string

		activeflowID    uuid.UUID
		forwardActionID uuid.UUID
		forwardNow      bool

		expectQueue   string
		expectRequest *sock.Request

		response *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
			uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),
			true,

			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:      "/v1/activeflows/447e712e-82d8-11eb-8900-7b97c080ddd8/forward_action_id",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"forward_action_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50","forward_now":true}`),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
			},
		},
		{
			"forward now false",

			uuid.FromStringOrNil("447e712e-82d8-11eb-8900-7b97c080ddd8"),
			uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"),
			false,

			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:      "/v1/activeflows/447e712e-82d8-11eb-8900-7b97c080ddd8/forward_action_id",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"forward_action_id":"44ebbd2e-82d8-11eb-8a4e-f7957fea9f50","forward_now":false}`),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.FlowV1ActiveflowUpdateForwardActionID(context.Background(), tt.activeflowID, tt.forwardActionID, tt.forwardNow); err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

		})
	}
}

func Test_FlowV1ActiveflowExecute(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("fde4653a-a7b5-11ec-a7ae-83d2f5255ec0"),

			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:      "/v1/activeflows/fde4653a-a7b5-11ec-a7ae-83d2f5255ec0/execute",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.FlowV1ActiveflowExecute(context.Background(), tt.activeflowID); err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

		})
	}
}

func Test_FlowV1ActiveflowDelete(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *fmactiveflow.Activeflow
	}{
		{
			"normal",

			uuid.FromStringOrNil("2f4bd474-ade1-11ec-9aca-83684de0c293"),

			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:      "/v1/activeflows/2f4bd474-ade1-11ec-9aca-83684de0c293",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"2f4bd474-ade1-11ec-9aca-83684de0c293"}`),
			},
			&fmactiveflow.Activeflow{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("2f4bd474-ade1-11ec-9aca-83684de0c293"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowV1ActiveflowDelete(context.Background(), tt.activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_FlowV1ActiveflowStop(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *fmactiveflow.Activeflow
	}{
		{
			"normal",

			uuid.FromStringOrNil("297ddcce-ca6c-11ed-8fe2-0740927aae87"),

			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:    "/v1/activeflows/297ddcce-ca6c-11ed-8fe2-0740927aae87/stop",
				Method: sock.RequestMethodPost,
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"297ddcce-ca6c-11ed-8fe2-0740927aae87"}`),
			},
			&fmactiveflow.Activeflow{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("297ddcce-ca6c-11ed-8fe2-0740927aae87"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowV1ActiveflowStop(ctx, tt.activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_FlowV1ActiveflowGet(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *fmactiveflow.Activeflow
	}{
		{
			"normal",

			uuid.FromStringOrNil("f7b11e28-ca6f-11ed-9ee2-2f18a39aac42"),

			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:    "/v1/activeflows/f7b11e28-ca6f-11ed-9ee2-2f18a39aac42",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f7b11e28-ca6f-11ed-9ee2-2f18a39aac42"}`),
			},
			&fmactiveflow.Activeflow{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("f7b11e28-ca6f-11ed-9ee2-2f18a39aac42"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowV1ActiveflowGet(ctx, tt.activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowV1ActiveflowGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     []fmactiveflow.Activeflow
	}{
		{
			"normal",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"/v1/activeflows?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:    "/v1/activeflows?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"559ede26-ca70-11ed-abba-e395946aa2e9"}]`),
			},
			[]fmactiveflow.Activeflow{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("559ede26-ca70-11ed-abba-e395946aa2e9"),
					},
				},
			},
		},
		{
			"2 items",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"/v1/activeflows?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.flow-manager.request",
			&sock.Request{
				URI:    "/v1/activeflows?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"56005566-ca70-11ed-81cf-6f43d9813fe9"},{"id":"5634bf5e-ca70-11ed-8158-03aa1817578a"}]`),
			},
			[]fmactiveflow.Activeflow{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("56005566-ca70-11ed-81cf-6f43d9813fe9"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("5634bf5e-ca70-11ed-8158-03aa1817578a"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			reqHandler := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowV1ActiveflowGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowV1ActiveflowAddActions(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID
		actions      []fmaction.Action

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *fmactiveflow.Activeflow
	}{
		{
			name: "normal",

			activeflowID: uuid.FromStringOrNil("6a5bb518-03f9-11f0-b284-bb23d250808c"),
			actions: []fmaction.Action{
				{
					ID: uuid.FromStringOrNil("6ac3888c-03f9-11f0-9cbc-0f40d578c119"),
				},
				{
					ID: uuid.FromStringOrNil("6b03b7c2-03f9-11f0-af7d-dbc31d2927cc"),
				},
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6a5bb518-03f9-11f0-b284-bb23d250808c"}`),
			},

			expectTarget: "bin-manager.flow-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/activeflows/6a5bb518-03f9-11f0-b284-bb23d250808c/add_actions",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"actions":[{"id":"6ac3888c-03f9-11f0-9cbc-0f40d578c119","next_id":"00000000-0000-0000-0000-000000000000","type":""},{"id":"6b03b7c2-03f9-11f0-af7d-dbc31d2927cc","next_id":"00000000-0000-0000-0000-000000000000","type":""}]}`),
			},
			expectRes: &fmactiveflow.Activeflow{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("6a5bb518-03f9-11f0-b284-bb23d250808c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowV1ActiveflowAddActions(ctx, tt.activeflowID, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowV1ActiveflowPushActions(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID
		actions      []fmaction.Action

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *fmactiveflow.Activeflow
	}{
		{
			name: "normal",

			activeflowID: uuid.FromStringOrNil("0dd10f12-fb1a-11ed-a3e2-dbe67cf6376c"),
			actions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0dd10f12-fb1a-11ed-a3e2-dbe67cf6376c"}`),
			},

			expectTarget: "bin-manager.flow-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/activeflows/0dd10f12-fb1a-11ed-a3e2-dbe67cf6376c/push_actions",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}]}`),
			},
			expectRes: &fmactiveflow.Activeflow{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("0dd10f12-fb1a-11ed-a3e2-dbe67cf6376c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.FlowV1ActiveflowPushActions(ctx, tt.activeflowID, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowV1ActiveflowServiceStop(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID
		serviceID    uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *fmactiveflow.Activeflow
	}{
		{
			name: "normal",

			activeflowID: uuid.FromStringOrNil("cad183f0-f9ea-11ef-84f4-63c4cb4776ac"),
			serviceID:    uuid.FromStringOrNil("cb1b9472-f9ea-11ef-9c09-4f2e84854f03"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},

			expectTarget: "bin-manager.flow-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/activeflows/cad183f0-f9ea-11ef-84f4-63c4cb4776ac/service_stop",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"service_id":"cb1b9472-f9ea-11ef-9c09-4f2e84854f03"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if errStop := reqHandler.FlowV1ActiveflowServiceStop(ctx, tt.activeflowID, tt.serviceID); errStop != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errStop)
			}
		})
	}
}
