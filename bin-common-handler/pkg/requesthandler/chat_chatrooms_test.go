package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	chatchatroom "monorepo/bin-chat-manager/models/chatroom"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_ChatV1ChatroomGet(t *testing.T) {

	tests := []struct {
		name string

		chatroomID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *chatchatroom.Chatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("b76a30e8-3695-11ed-b331-7f4de361058c"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b76a30e8-3695-11ed-b331-7f4de361058c"}`),
			},

			"bin-manager.chat-manager.request",
			&sock.Request{
				URI:      "/v1/chatrooms/b76a30e8-3695-11ed-b331-7f4de361058c",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&chatchatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b76a30e8-3695-11ed-b331-7f4de361058c"),
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

			res, err := reqHandler.ChatV1ChatroomGet(ctx, tt.chatroomID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_ChatV1ChatroomList(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[chatchatroom.Field]any

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  []chatchatroom.Chatroom
	}{
		{
			"normal",

			"2020-09-20 03:23:20.995000",
			10,
			map[chatchatroom.Field]any{
				chatchatroom.FieldOwnerID: uuid.FromStringOrNil("19de5bc8-3696-11ed-b9b7-3f54f6f0297b"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"1a6c8efc-3696-11ed-b8f6-f7375910007f"}]`),
			},

			"bin-manager.chat-manager.request",
			&sock.Request{
				URI:      fmt.Sprintf("/v1/chatrooms?page_token=%s&page_size=10", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"owner_id":"19de5bc8-3696-11ed-b9b7-3f54f6f0297b"}`),
			},
			[]chatchatroom.Chatroom{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1a6c8efc-3696-11ed-b8f6-f7375910007f"),
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

			res, err := reqHandler.ChatV1ChatroomList(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_ChatV1ChatroomDelete(t *testing.T) {

	tests := []struct {
		name string

		chatroomID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *chatchatroom.Chatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("a4113036-3696-11ed-9c58-839c310695a8"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a4113036-3696-11ed-9c58-839c310695a8"}`),
			},

			"bin-manager.chat-manager.request",
			&sock.Request{
				URI:      "/v1/chatrooms/a4113036-3696-11ed-9c58-839c310695a8",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&chatchatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4113036-3696-11ed-9c58-839c310695a8"),
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

			res, err := reqHandler.ChatV1ChatroomDelete(ctx, tt.chatroomID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_ChatV1ChatroomUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		chatID       uuid.UUID
		updateName   string
		updateDetail string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *chatchatroom.Chatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("800b6dae-bc60-11ee-94fb-23e2e1876984"),
			"update name",
			"update detail",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"800b6dae-bc60-11ee-94fb-23e2e1876984"}`),
			},

			"bin-manager.chat-manager.request",
			&sock.Request{
				URI:      "/v1/chatrooms/800b6dae-bc60-11ee-94fb-23e2e1876984",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"update name","detail":"update detail"}`),
			},
			&chatchatroom.Chatroom{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("800b6dae-bc60-11ee-94fb-23e2e1876984"),
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

			res, err := reqHandler.ChatV1ChatroomUpdateBasicInfo(ctx, tt.chatID, tt.updateName, tt.updateDetail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
