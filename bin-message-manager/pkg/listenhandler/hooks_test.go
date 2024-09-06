package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/pkg/messagehandler"
)

func Test_processV1HooksPost(t *testing.T) {

	tests := []struct {
		name string

		uri  string
		data []byte

		responseSend *message.Message

		request   *sock.Request
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",

			"hook.voipbin.net/v1.0/messages/telnyx",
			[]byte(`{
  "data": {
    "event_type": "message.received",
    "id": "19539336-11ba-4792-abd8-26d4f8745c4c",
    "occurred_at": "2022-03-15T16:16:24.073+00:00",
    "payload": {
      "cc": [],
      "completed_at": null,
      "cost": null,
      "direction": "inbound",
      "encoding": "GSM-7",
      "errors": [],
      "from": {
        "carrier": "",
        "line_type": "",
        "phone_number": "+75973"
      },
      "id": "5d7f9c50-330a-4d7a-9ca8-4157d7a09047",
      "media": [],
      "messaging_profile_id": "40017f8e-49bd-4f16-9e3d-ef103f916228",
      "organization_id": "a506eae0-f72c-449c-bbe5-19ce35f82e0b",
      "parts": 1,
      "received_at": "2022-03-15T16:16:23.466+00:00",
      "record_type": "message",
      "sent_at": null,
      "subject": "",
      "tags": [],
      "text": "pchero21:\nTest message from skype.",
      "to": [
        {
          "carrier": "Telnyx",
          "line_type": "Wireless",
          "phone_number": "+15734531118",
          "status": "webhook_delivered"
        }
      ],
      "type": "SMS",
      "valid_until": null,
      "webhook_failover_url": null,
      "webhook_url": "https://en7evajwhmqbt.x.pipedream.net"
    },
    "record_type": "event"
  },
  "meta": {
    "attempt": 1,
    "delivered_to": "https://en7evajwhmqbt.x.pipedream.net"
  }
}
`),

			&message.Message{
				ID: uuid.FromStringOrNil("abed7ae4-a22b-11ec-8b95-efa78516ed55"),
			},

			&sock.Request{
				URI:      "/v1/hooks",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"received_uri":"hook.voipbin.net/v1.0/messages/telnyx","received_data":"ewogICJkYXRhIjogewogICAgImV2ZW50X3R5cGUiOiAibWVzc2FnZS5yZWNlaXZlZCIsCiAgICAiaWQiOiAiMTk1MzkzMzYtMTFiYS00NzkyLWFiZDgtMjZkNGY4NzQ1YzRjIiwKICAgICJvY2N1cnJlZF9hdCI6ICIyMDIyLTAzLTE1VDE2OjE2OjI0LjA3MyswMDowMCIsCiAgICAicGF5bG9hZCI6IHsKICAgICAgImNjIjogW10sCiAgICAgICJjb21wbGV0ZWRfYXQiOiBudWxsLAogICAgICAiY29zdCI6IG51bGwsCiAgICAgICJkaXJlY3Rpb24iOiAiaW5ib3VuZCIsCiAgICAgICJlbmNvZGluZyI6ICJHU00tNyIsCiAgICAgICJlcnJvcnMiOiBbXSwKICAgICAgImZyb20iOiB7CiAgICAgICAgImNhcnJpZXIiOiAiIiwKICAgICAgICAibGluZV90eXBlIjogIiIsCiAgICAgICAgInBob25lX251bWJlciI6ICIrNzU5NzMiCiAgICAgIH0sCiAgICAgICJpZCI6ICI1ZDdmOWM1MC0zMzBhLTRkN2EtOWNhOC00MTU3ZDdhMDkwNDciLAogICAgICAibWVkaWEiOiBbXSwKICAgICAgIm1lc3NhZ2luZ19wcm9maWxlX2lkIjogIjQwMDE3ZjhlLTQ5YmQtNGYxNi05ZTNkLWVmMTAzZjkxNjIyOCIsCiAgICAgICJvcmdhbml6YXRpb25faWQiOiAiYTUwNmVhZTAtZjcyYy00NDljLWJiZTUtMTljZTM1ZjgyZTBiIiwKICAgICAgInBhcnRzIjogMSwKICAgICAgInJlY2VpdmVkX2F0IjogIjIwMjItMDMtMTVUMTY6MTY6MjMuNDY2KzAwOjAwIiwKICAgICAgInJlY29yZF90eXBlIjogIm1lc3NhZ2UiLAogICAgICAic2VudF9hdCI6IG51bGwsCiAgICAgICJzdWJqZWN0IjogIiIsCiAgICAgICJ0YWdzIjogW10sCiAgICAgICJ0ZXh0IjogInBjaGVybzIxOlxuVGVzdCBtZXNzYWdlIGZyb20gc2t5cGUuIiwKICAgICAgInRvIjogWwogICAgICAgIHsKICAgICAgICAgICJjYXJyaWVyIjogIlRlbG55eCIsCiAgICAgICAgICAibGluZV90eXBlIjogIldpcmVsZXNzIiwKICAgICAgICAgICJwaG9uZV9udW1iZXIiOiAiKzE1NzM0NTMxMTE4IiwKICAgICAgICAgICJzdGF0dXMiOiAid2ViaG9va19kZWxpdmVyZWQiCiAgICAgICAgfQogICAgICBdLAogICAgICAidHlwZSI6ICJTTVMiLAogICAgICAidmFsaWRfdW50aWwiOiBudWxsLAogICAgICAid2ViaG9va19mYWlsb3Zlcl91cmwiOiBudWxsLAogICAgICAid2ViaG9va191cmwiOiAiaHR0cHM6Ly9lbjdldmFqd2htcWJ0LngucGlwZWRyZWFtLm5ldCIKICAgIH0sCiAgICAicmVjb3JkX3R5cGUiOiAiZXZlbnQiCiAgfSwKICAibWV0YSI6IHsKICAgICJhdHRlbXB0IjogMSwKICAgICJkZWxpdmVyZWRfdG8iOiAiaHR0cHM6Ly9lbjdldmFqd2htcWJ0LngucGlwZWRyZWFtLm5ldCIKICB9Cn0K"}`),
			},
			&rabbitmqhandler.Response{
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
			mockMessageHandler := messagehandler.NewMockMessageHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				messageHandler: mockMessageHandler,
			}

			mockMessageHandler.EXPECT().Hook(gomock.Any(), tt.uri, tt.data).Return(nil)

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
