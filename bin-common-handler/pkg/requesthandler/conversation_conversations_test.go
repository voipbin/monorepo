package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_ConversationV1ConversationsGet(t *testing.T) {

	type test struct {
		name string

		conversationID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *cvconversation.Conversation
	}

	tests := []test{
		{
			name: "normal",

			conversationID: uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),

			expectQueue: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/conversations/72179880-ec5f-11ec-920e-c77279756b6d",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"72179880-ec5f-11ec-920e-c77279756b6d"}`),
			},
			expectRes: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
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

			res, err := reqHandler.ConversationV1ConversationGet(ctx, tt.conversationID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ConversationV1ConversationGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[cvconversation.Field]any

		response *sock.Response

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		expectRes     []cvconversation.Conversation
	}{
		{
			name: "normal",

			pageToken: "2021-03-02 03:23:20.995000",
			pageSize:  10,
			filters: map[cvconversation.Field]any{
				cvconversation.FieldDeleted:    false,
				cvconversation.FieldCustomerID: uuid.FromStringOrNil("84b4b554-21ef-11f0-a5bb-e33bf5a5a345"),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"30071608-7e43-11ec-b04a-bb4270e3e223"},{"id":"5ca81a9a-7e43-11ec-b271-5b65823bfdd3"}]`),
			},

			expectURL:    "/v1/conversations?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10",
			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/conversations?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"84b4b554-21ef-11f0-a5bb-e33bf5a5a345","deleted":false}`),
			},
			expectRes: []cvconversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("30071608-7e43-11ec-b04a-bb4270e3e223"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5ca81a9a-7e43-11ec-b271-5b65823bfdd3"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConversationV1ConversationGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationV1ConversationCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID       uuid.UUID
		conversationName string
		detail           string
		conversationType cvconversation.Type
		dialogID         string
		self             address.Address
		peer             address.Address

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cvconversation.Conversation
	}{
		{
			name: "normal",

			customerID:       uuid.FromStringOrNil("8c9e3e90-1acc-11f0-8112-a7bddc5a51fd"),
			conversationName: "test name",
			detail:           "test detail",
			conversationType: cvconversation.TypeLine,
			dialogID:         "80031872-1acc-11f0-b6a7-436484610c22",
			self: address.Address{
				Type:       address.TypeLine,
				Target:     "",
				TargetName: "me",
			},
			peer: address.Address{
				Type:   address.TypeLine,
				Target: "8c790a08-1acc-11f0-b34c-ffae95d1d395",
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8cb9a9aa-1acc-11f0-b8cb-9f14cc6836d6"}`),
			},

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/conversations",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"8c9e3e90-1acc-11f0-8112-a7bddc5a51fd","name":"test name","detail":"test detail","type":"line","dialog_id":"80031872-1acc-11f0-b6a7-436484610c22","self":{"type":"line","target_name":"me"},"peer":{"type":"line","target":"8c790a08-1acc-11f0-b34c-ffae95d1d395"}}`),
			},
			expectRes: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8cb9a9aa-1acc-11f0-b8cb-9f14cc6836d6"),
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

			res, err := reqHandler.ConversationV1ConversationCreate(
				ctx,
				tt.customerID,
				tt.conversationName,
				tt.detail,
				tt.conversationType,
				tt.dialogID,
				tt.self,
				tt.peer,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationV1ConversationUpdate(t *testing.T) {

	tests := []struct {
		name string

		conversationID uuid.UUID
		fields         map[cvconversation.Field]any

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *cvconversation.Conversation
	}{
		{
			name: "normal",

			conversationID: uuid.FromStringOrNil("1397bde6-007a-11ee-903f-4b1fc025c9a9"),
			fields: map[cvconversation.Field]any{
				cvconversation.FieldName:   "test name",
				cvconversation.FieldDetail: "test detail",
			},

			expectTarget: "bin-manager.conversation-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/conversations/1397bde6-007a-11ee-903f-4b1fc025c9a9",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"detail":"test detail","name":"test name"}`),
			},
			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1397bde6-007a-11ee-903f-4b1fc025c9a9"}`),
			},

			expectRes: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1397bde6-007a-11ee-903f-4b1fc025c9a9"),
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

			res, err := reqHandler.ConversationV1ConversationUpdate(ctx, tt.conversationID, tt.fields)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
