package requesthandler

import (
	"context"
	cbmessage "monorepo/bin-ai-manager/models/message"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_AIV1MessageGetsByAIcallID(t *testing.T) {

	tests := []struct {
		name string

		aicallID  uuid.UUID
		pageToken string
		pageSize  uint64
		filters   map[cbmessage.Field]any

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []cbmessage.Message
	}{
		{
			name: "normal",

			aicallID:  uuid.FromStringOrNil("d43e25a6-f2ce-11ef-bd10-3b19aa3747d8"),
			pageToken: "2020-09-20T03:23:20.995000Z",
			pageSize:  10,
			filters: map[cbmessage.Field]any{
				cbmessage.FieldDeleted: false,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d49161bc-f2ce-11ef-8263-17e36f2d0922"},{"id":"d4c481d2-f2ce-11ef-b59d-e3cfadb6b877"}]`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/messages?page_token=2020-09-20T03%3A23%3A20.995000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"aicall_id":"d43e25a6-f2ce-11ef-bd10-3b19aa3747d8","deleted":false}`),
			},
			expectRes: []cbmessage.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("d49161bc-f2ce-11ef-8263-17e36f2d0922"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("d4c481d2-f2ce-11ef-b59d-e3cfadb6b877"),
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
			h := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.AIV1MessageGetsByAIcallID(ctx, tt.aicallID, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1MessageSend(t *testing.T) {

	tests := []struct {
		name string

		aicallID       uuid.UUID
		role           cbmessage.Role
		content        string
		runImmediately bool
		audioResponse  bool
		timeout        int

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cbmessage.Message
	}{
		{
			name: "normal",

			aicallID:       uuid.FromStringOrNil("5398cd60-f2cf-11ef-ac07-2b477cb8a829"),
			role:           cbmessage.RoleUser,
			content:        "test content",
			runImmediately: true,
			audioResponse:  true,
			timeout:        30000,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"53f4f37e-f2cf-11ef-8f9a-77d75650dfd8"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"aicall_id":"5398cd60-f2cf-11ef-ac07-2b477cb8a829","role":"user","content":"test content","run_immediately":true,"audio_response":true}`),
			},
			expectRes: &cbmessage.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("53f4f37e-f2cf-11ef-8f9a-77d75650dfd8"),
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

			cf, err := reqHandler.AIV1MessageSend(ctx, tt.aicallID, tt.role, tt.content, tt.runImmediately, tt.audioResponse, tt.timeout)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}

func Test_AIV1MessageGet(t *testing.T) {

	type test struct {
		name string

		messageID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *cbmessage.Message
	}

	tests := []test{
		{
			name:      "normal",
			messageID: uuid.FromStringOrNil("cf449b2e-f2cf-11ef-a7e4-7b75fad55f68"),

			expectQueue: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/messages/cf449b2e-f2cf-11ef-a7e4-7b75fad55f68",
				Method: sock.RequestMethodGet,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"cf449b2e-f2cf-11ef-a7e4-7b75fad55f68"}`),
			},
			expectRes: &cbmessage.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("cf449b2e-f2cf-11ef-a7e4-7b75fad55f68"),
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

			res, err := reqHandler.AIV1MessageGet(ctx, tt.messageID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1MessageDelete(t *testing.T) {

	tests := []struct {
		name string

		messageID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cbmessage.Message
	}{
		{
			name: "normal",

			messageID: uuid.FromStringOrNil("1780bb8e-f2d0-11ef-82fc-bf7a5e0a585d"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"1780bb8e-f2d0-11ef-82fc-bf7a5e0a585d"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/messages/1780bb8e-f2d0-11ef-82fc-bf7a5e0a585d",
				Method: sock.RequestMethodDelete,
			},
			expectRes: &cbmessage.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("1780bb8e-f2d0-11ef-82fc-bf7a5e0a585d"),
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

			res, err := reqHandler.AIV1MessageDelete(ctx, tt.messageID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
