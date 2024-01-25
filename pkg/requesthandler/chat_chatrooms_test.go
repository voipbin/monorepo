package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	chatchatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_ChatV1ChatroomGet(t *testing.T) {

	tests := []struct {
		name string

		chatroomID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *chatchatroom.Chatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("b76a30e8-3695-11ed-b331-7f4de361058c"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b76a30e8-3695-11ed-b331-7f4de361058c"}`),
			},

			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/chatrooms/b76a30e8-3695-11ed-b331-7f4de361058c",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&chatchatroom.Chatroom{
				ID: uuid.FromStringOrNil("b76a30e8-3695-11ed-b331-7f4de361058c"),
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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

func Test_ChatV1ChatroomGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  []chatchatroom.Chatroom
	}{
		{
			"normal",

			"2020-09-20 03:23:20.995000",
			10,
			map[string]string{
				"owner_id": "19de5bc8-3696-11ed-b9b7-3f54f6f0297b",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"1a6c8efc-3696-11ed-b8f6-f7375910007f"}]`),
			},

			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      fmt.Sprintf("/v1/chatrooms?page_token=%s&page_size=10&filter_owner_id=19de5bc8-3696-11ed-b9b7-3f54f6f0297b", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]chatchatroom.Chatroom{
				{
					ID: uuid.FromStringOrNil("1a6c8efc-3696-11ed-b8f6-f7375910007f"),
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ChatV1ChatroomGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
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

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *chatchatroom.Chatroom
	}{
		{
			"normal",

			uuid.FromStringOrNil("a4113036-3696-11ed-9c58-839c310695a8"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a4113036-3696-11ed-9c58-839c310695a8"}`),
			},

			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/chatrooms/a4113036-3696-11ed-9c58-839c310695a8",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&chatchatroom.Chatroom{
				ID: uuid.FromStringOrNil("a4113036-3696-11ed-9c58-839c310695a8"),
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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
