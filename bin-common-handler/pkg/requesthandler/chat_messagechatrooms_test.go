package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_ChatV1MessagechatroomGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		response *sock.Response

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		expectResult  []chatmessagechatroom.Messagechatroom
	}{
		{
			"normal",

			"2020-09-20 03:23:20.995000",
			10,
			map[string]string{
				"chatroom_id": "15f4abea-369e-11ed-b888-1f2976c17434",
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"161eef40-369e-11ed-9e79-fb15c8cb465a"}]`),
			},

			"/v1/messagechatrooms?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10",
			"bin-manager.chat-manager.request",
			&sock.Request{
				URI:      fmt.Sprintf("/v1/messagechatrooms?page_token=%s&page_size=10&filter_chatroom_id=15f4abea-369e-11ed-b888-1f2976c17434", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]chatmessagechatroom.Messagechatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("161eef40-369e-11ed-9e79-fb15c8cb465a"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.ChatV1MessagechatroomGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_ChatV1MessagechatroomGet(t *testing.T) {

	tests := []struct {
		name string

		messagechatroomID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *chatmessagechatroom.Messagechatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("677d8888-369e-11ed-84b3-ef10b6d21710"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"677d8888-369e-11ed-84b3-ef10b6d21710"}`),
			},

			"bin-manager.chat-manager.request",
			&sock.Request{
				URI:      "/v1/messagechatrooms/677d8888-369e-11ed-84b3-ef10b6d21710",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&chatmessagechatroom.Messagechatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("677d8888-369e-11ed-84b3-ef10b6d21710"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ChatV1MessagechatroomGet(ctx, tt.messagechatroomID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_ChatV1MessagechatroomDelete(t *testing.T) {

	tests := []struct {
		name string

		messagechatroomID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *chatmessagechatroom.Messagechatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("919c4582-369e-11ed-8a8c-77506adf5ffe"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"919c4582-369e-11ed-8a8c-77506adf5ffe"}`),
			},

			"bin-manager.chat-manager.request",
			&sock.Request{
				URI:      "/v1/messagechatrooms/919c4582-369e-11ed-8a8c-77506adf5ffe",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&chatmessagechatroom.Messagechatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("919c4582-369e-11ed-8a8c-77506adf5ffe"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ChatV1MessagechatroomDelete(ctx, tt.messagechatroomID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
