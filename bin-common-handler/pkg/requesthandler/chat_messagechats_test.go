package requesthandler

import (
	"context"
	"reflect"
	"testing"

	chatmedia "monorepo/bin-chat-manager/models/media"
	chatmessagechat "monorepo/bin-chat-manager/models/messagechat"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_ChatV1MessagechatCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		chatID      uuid.UUID
		source      commonaddress.Address
		messageType chatmessagechat.Type
		text        string
		medias      []chatmedia.Media

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *chatmessagechat.Messagechat
	}{
		{
			"normal",

			uuid.FromStringOrNil("e0a01384-369e-11ed-94ec-3f100d0d2c9f"),
			uuid.FromStringOrNil("e0fbacbc-369e-11ed-9ba0-b3dcc584c182"),
			commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			chatmessagechat.TypeNormal,
			"test message",
			[]chatmedia.Media{},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d50945c4-3697-11ed-9ffb-570b42b0ddd4"}`),
			},

			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/messagechats",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"e0a01384-369e-11ed-94ec-3f100d0d2c9f","chat_id":"e0fbacbc-369e-11ed-9ba0-b3dcc584c182","source":{"type":"tel","target":"+821100000001","target_name":"","name":"","detail":""},"message_type":"normal","text":"test message","medias":[]}`),
			},
			&chatmessagechat.Messagechat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d50945c4-3697-11ed-9ffb-570b42b0ddd4"),
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

			res, err := reqHandler.ChatV1MessagechatCreate(
				ctx,
				tt.customerID,
				tt.chatID,
				tt.source,
				tt.messageType,
				tt.text,
				tt.medias,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong matchdfdsfd.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_ChatV1MessagechatGet(t *testing.T) {

	tests := []struct {
		name string

		messagechatID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *chatmessagechat.Messagechat
	}{
		{
			"normal",

			uuid.FromStringOrNil("9e6997aa-369f-11ed-9453-cb9c81406b8b"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9e6997aa-369f-11ed-9453-cb9c81406b8b"}`),
			},

			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/messagechats/9e6997aa-369f-11ed-9453-cb9c81406b8b",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&chatmessagechat.Messagechat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9e6997aa-369f-11ed-9453-cb9c81406b8b"),
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

			res, err := reqHandler.ChatV1MessagechatGet(ctx, tt.messagechatID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_ChatV1MessagechatGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		response *rabbitmqhandler.Response

		expectURL     string
		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  []chatmessagechat.Messagechat
	}{
		{
			"normal",

			"2020-09-20 03:23:20.995000",
			10,
			map[string]string{
				"chat_id": "fdf8ca74-369f-11ed-b48b-b728ad308b30",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"fe1ec10c-369f-11ed-aa7b-6f4631dff513"}]`),
			},

			"/v1/messagechats?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10",
			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/messagechats?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10&filter_chat_id=fdf8ca74-369f-11ed-b48b-b728ad308b30",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]chatmessagechat.Messagechat{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("fe1ec10c-369f-11ed-aa7b-6f4631dff513"),
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.ChatV1MessagechatGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_ChatV1MessagechatDelete(t *testing.T) {

	tests := []struct {
		name string

		messagechatID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *chatmessagechat.Messagechat
	}{
		{
			"normal",

			uuid.FromStringOrNil("2a3b1862-36a0-11ed-8f05-57a3e81bdcc9"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2a3b1862-36a0-11ed-8f05-57a3e81bdcc9"}`),
			},

			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/messagechats/2a3b1862-36a0-11ed-8f05-57a3e81bdcc9",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&chatmessagechat.Messagechat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a3b1862-36a0-11ed-8f05-57a3e81bdcc9"),
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

			res, err := reqHandler.ChatV1MessagechatDelete(ctx, tt.messagechatID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
