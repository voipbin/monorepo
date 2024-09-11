package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
)

func Test_processV1HooksPost(t *testing.T) {

	tests := []struct {
		name string

		uri  string
		data []byte

		responseSend *message.Message

		request   *sock.Request
		expectRes *sock.Response
	}{
		{
			"normal",

			"https://hook.voipbin.net/v1.0/conversation/customers/a92e60ea-e85b-11ec-a173-0b1cf8c9d3e9/line",
			[]byte(`{
				"destination": "U11298214116e3afbad432b5794a6d3a0",
				"events": [
					{
						"type": "follow",
						"webhookEventId": "01G49KGV3YYCWA0CPZHP9AA6H9",
						"deliveryContext": {
							"isRedelivery": false
						},
						"timestamp": 1653884873348,
						"source": {
							"type": "user",
							"userId": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
						},
						"replyToken": "44b7e0b5fa034a58bfd75c9e256ad2ed",
						"mode": "active"
					}
				]
			}`),

			&message.Message{
				ID: uuid.FromStringOrNil("abed7ae4-a22b-11ec-8b95-efa78516ed55"),
			},

			&sock.Request{
				URI:      "/v1/hooks",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"received_uri":"https://hook.voipbin.net/v1.0/conversation/customers/a92e60ea-e85b-11ec-a173-0b1cf8c9d3e9/line","received_data":"ewoJCQkJImRlc3RpbmF0aW9uIjogIlUxMTI5ODIxNDExNmUzYWZiYWQ0MzJiNTc5NGE2ZDNhMCIsCgkJCQkiZXZlbnRzIjogWwoJCQkJCXsKCQkJCQkJInR5cGUiOiAiZm9sbG93IiwKCQkJCQkJIndlYmhvb2tFdmVudElkIjogIjAxRzQ5S0dWM1lZQ1dBMENQWkhQOUFBNkg5IiwKCQkJCQkJImRlbGl2ZXJ5Q29udGV4dCI6IHsKCQkJCQkJCSJpc1JlZGVsaXZlcnkiOiBmYWxzZQoJCQkJCQl9LAoJCQkJCQkidGltZXN0YW1wIjogMTY1Mzg4NDg3MzM0OCwKCQkJCQkJInNvdXJjZSI6IHsKCQkJCQkJCSJ0eXBlIjogInVzZXIiLAoJCQkJCQkJInVzZXJJZCI6ICJVZDg3MWJjYWY3YzNhZDEzZDJhMGIwZDc4YTQyYTI4N2YiCgkJCQkJCX0sCgkJCQkJCSJyZXBseVRva2VuIjogIjQ0YjdlMGI1ZmEwMzRhNThiZmQ3NWM5ZTI1NmFkMmVkIiwKCQkJCQkJIm1vZGUiOiAiYWN0aXZlIgoJCQkJCX0KCQkJCV0KCQkJfQ=="}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
		{
			"normal",

			"https://hook.voipbin.net/v1.0/conversation/customers/a92e60ea-e85b-11ec-a173-0b1cf8c9d3e9/line",
			[]byte(`{"destination":"U11298214116e3afbad432b5794a6d3a0","events":[]}`),

			&message.Message{
				ID: uuid.FromStringOrNil("abed7ae4-a22b-11ec-8b95-efa78516ed55"),
			},

			&sock.Request{
				URI:      "/v1/hooks",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"received_uri":"https://hook.voipbin.net/v1.0/conversation/customers/a92e60ea-e85b-11ec-a173-0b1cf8c9d3e9/line","received_data":"eyJkZXN0aW5hdGlvbiI6IlUxMTI5ODIxNDExNmUzYWZiYWQ0MzJiNTc5NGE2ZDNhMCIsImV2ZW50cyI6W119"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConversation := conversationhandler.NewMockConversationHandler(mc)

			h := &listenHandler{
				sockHandler:         mockSock,
				conversationHandler: mockConversation,
			}

			mockConversation.EXPECT().Hook(gomock.Any(), tt.uri, tt.data)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
