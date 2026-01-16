package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	ammessage "monorepo/bin-ai-manager/models/message"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_AIV1AIcallStart(t *testing.T) {

	tests := []struct {
		name string

		activeflowID  uuid.UUID
		aiID          uuid.UUID
		referenceType amaicall.ReferenceType
		referenceID   uuid.UUID
		gender        amaicall.Gender
		language      string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amaicall.AIcall
	}{
		{
			name: "normal",

			activeflowID:  uuid.FromStringOrNil("eb23a6b0-0cc3-11f0-8150-0f33dc4cfdc4"),
			aiID:          uuid.FromStringOrNil("e8604e8a-ef52-11ef-88be-43d681e412f7"),
			referenceType: amaicall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e8c3a34a-ef52-11ef-b4d1-93c7d17c08e9"),
			gender:        amaicall.GenderFemale,
			language:      "en-US",

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e8ec8062-ef52-11ef-8fe9-27921b0be03c"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/aicalls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"activeflow_id":"eb23a6b0-0cc3-11f0-8150-0f33dc4cfdc4","ai_id":"e8604e8a-ef52-11ef-88be-43d681e412f7","reference_type":"call","reference_id":"e8c3a34a-ef52-11ef-b4d1-93c7d17c08e9","gender":"female","language":"en-US"}`),
			},
			expectRes: &amaicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e8ec8062-ef52-11ef-8fe9-27921b0be03c"),
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

			cf, err := reqHandler.AIV1AIcallStart(ctx, tt.activeflowID, tt.aiID, tt.referenceType, tt.referenceID, tt.gender, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}

func Test_AIV1AIcallList(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[amaicall.Field]any

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []amaicall.AIcall
	}{
		{
			name: "normal",

			pageToken: "2020-09-20 03:23:20.995000",
			pageSize:  10,
			filters: map[amaicall.Field]any{
				amaicall.FieldDeleted:    false,
				amaicall.FieldCustomerID: uuid.FromStringOrNil("ccf7720e-4838-4f97-bb61-3021e14c185a"),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c3ac26c7-567c-4230-aaf8-d19b6fde4d6c"},{"id":"eb36875a-0d7a-4a8f-92a9-7551f4f29fd6"}]`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      fmt.Sprintf("/v1/aicalls?page_token=%s&page_size=10", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"ccf7720e-4838-4f97-bb61-3021e14c185a","deleted":false}`),
			},
			expectRes: []amaicall.AIcall{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("c3ac26c7-567c-4230-aaf8-d19b6fde4d6c"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("eb36875a-0d7a-4a8f-92a9-7551f4f29fd6"),
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
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AIV1AIcallList(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIcallGet(t *testing.T) {

	type test struct {
		name string

		aicallID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *amaicall.AIcall
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("d3937170-ee3b-40d0-8b81-4261e5bb5ba4"),

			string(outline.QueueNameAIRequest),
			&sock.Request{
				URI:    "/v1/aicalls/d3937170-ee3b-40d0-8b81-4261e5bb5ba4",
				Method: sock.RequestMethodGet,
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"d3937170-ee3b-40d0-8b81-4261e5bb5ba4"}`),
			},
			&amaicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("d3937170-ee3b-40d0-8b81-4261e5bb5ba4"),
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

			res, err := reqHandler.AIV1AIcallGet(ctx, tt.aicallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIcallDelete(t *testing.T) {

	tests := []struct {
		name string

		aicallID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amaicall.AIcall
	}{
		{
			name: "normal",

			aicallID: uuid.FromStringOrNil("6078c492-25e6-4f31-baa0-2fef98379db7"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"6078c492-25e6-4f31-baa0-2fef98379db7"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/aicalls/6078c492-25e6-4f31-baa0-2fef98379db7",
				Method: sock.RequestMethodDelete,
			},
			expectRes: &amaicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("6078c492-25e6-4f31-baa0-2fef98379db7"),
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

			res, err := reqHandler.AIV1AIcallDelete(ctx, tt.aicallID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIcallTerminate(t *testing.T) {

	tests := []struct {
		name string

		aicallID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amaicall.AIcall
	}{
		{
			name: "normal",

			aicallID: uuid.FromStringOrNil("a0a4ef26-9199-11f0-97c6-870fb70436ad"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"a0a4ef26-9199-11f0-97c6-870fb70436ad"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/aicalls/a0a4ef26-9199-11f0-97c6-870fb70436ad/terminate",
				Method: sock.RequestMethodPost,
			},
			expectRes: &amaicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a0a4ef26-9199-11f0-97c6-870fb70436ad"),
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

			res, err := reqHandler.AIV1AIcallTerminate(ctx, tt.aicallID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIcallTerminateWithDelay(t *testing.T) {

	tests := []struct {
		name string

		aicallID uuid.UUID
		delay    int

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			name: "normal",

			aicallID: uuid.FromStringOrNil("05cdc0e8-d953-11f0-a77c-73b75aa0e129"),
			delay:    300000,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/aicalls/05cdc0e8-d953-11f0-a77c-73b75aa0e129/terminate",
				Method: sock.RequestMethodPost,
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

			mockSock.EXPECT().RequestPublishWithDelay(tt.expectTarget, tt.expectRequest, tt.delay).Return(nil)

			err := reqHandler.AIV1AIcallTerminateWithDelay(ctx, tt.aicallID, tt.delay)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}
		})
	}
}

func Test_AIV1AIcallToolExecute(t *testing.T) {

	tests := []struct {
		name string

		aicallID uuid.UUID
		toolID   string
		toolType ammessage.ToolType
		function *ammessage.FunctionCall

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     map[string]any
	}{
		{
			name: "normal",

			aicallID: uuid.FromStringOrNil("780281fa-bbec-11f0-a56b-fb82bf5a05ef"),
			toolID:   "77d4c710-bbec-11f0-826e-2f6827c0d353",
			toolType: ammessage.ToolTypeFunction,
			function: &ammessage.FunctionCall{
				Name:      ammessage.FunctionCallNameConnectCall,
				Arguments: `{"source":{"type":"tel","target":"+123456789"},"destinations":[{"type":"tel","target":"+111111"},{"type":"tel","target":"+22222"}]}`,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"result":"success", "message": ""}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/aicalls/780281fa-bbec-11f0-a56b-fb82bf5a05ef/tool_execute",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"77d4c710-bbec-11f0-826e-2f6827c0d353","type":"function","function":{"name":"connect_call","arguments":"{\"source\":{\"type\":\"tel\",\"target\":\"+123456789\"},\"destinations\":[{\"type\":\"tel\",\"target\":\"+111111\"},{\"type\":\"tel\",\"target\":\"+22222\"}]}"}}`),
			},
			expectRes: map[string]any{
				"result":  "success",
				"message": "",
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

			res, err := reqHandler.AIV1AIcallToolExecute(ctx, tt.aicallID, tt.toolID, tt.toolType, tt.function)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
