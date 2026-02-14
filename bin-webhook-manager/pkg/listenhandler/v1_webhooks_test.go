package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-webhook-manager/models/webhook"
	"monorepo/bin-webhook-manager/pkg/webhookhandler"
)

func Test_processV1WebhooksPost(t *testing.T) {

	tests := []struct {
		name string

		request    *sock.Request
		customerID uuid.UUID
		dataType   webhook.DataType
		data       json.RawMessage

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/webhooks",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"2791e4a8-8253-11ec-811a-07cc4bcfbdaa","data_type":"application/json","data":{"type":"call_created","data":{"master_call_id":"00000000-0000-0000-0000-000000000000","direction":"outgoing","hangup_by":"","user_id":1,"id":"cdbe20b4-4f49-4a27-96d5-4dda0ec2134b","action":{"id":"00000000-0000-0000-0000-000000000001","tm_execute":"","type":""},"flow_id":"883f40c7-7f16-48fe-a404-772d9b038808","type":"flow","source":{"target":"+821028286521","name":"","type":"tel"},"tm_update":"","tm_ringing":"","data":null,"recording_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":[],"hangup_reason":"","webhook_uri":"https://endxhr87aa0bkge.m.pipedream.net","tm_progressing":"","conf_id":"00000000-0000-0000-0000-000000000000","tm_hangup":"","asterisk_id":"","status":"dialing","recording_ids":[],"destination":{"name":"","target":"+821021656521","type":"tel"},"channel_id":"8edc9ed7-2ec0-4027-a7b9-3d6eced18afd","tm_create":"2021-03-13T18:18:02.489462Z"}}}`),
			},
			uuid.FromStringOrNil("2791e4a8-8253-11ec-811a-07cc4bcfbdaa"),
			"application/json",
			[]byte(`{"type":"call_created","data":{"master_call_id":"00000000-0000-0000-0000-000000000000","direction":"outgoing","hangup_by":"","user_id":1,"id":"cdbe20b4-4f49-4a27-96d5-4dda0ec2134b","action":{"id":"00000000-0000-0000-0000-000000000001","tm_execute":"","type":""},"flow_id":"883f40c7-7f16-48fe-a404-772d9b038808","type":"flow","source":{"target":"+821028286521","name":"","type":"tel"},"tm_update":"","tm_ringing":"","data":null,"recording_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":[],"hangup_reason":"","webhook_uri":"https://endxhr87aa0bkge.m.pipedream.net","tm_progressing":"","conf_id":"00000000-0000-0000-0000-000000000000","tm_hangup":"","asterisk_id":"","status":"dialing","recording_ids":[],"destination":{"name":"","target":"+821021656521","type":"tel"},"channel_id":"8edc9ed7-2ec0-4027-a7b9-3d6eced18afd","tm_create":"2021-03-13T18:18:02.489462Z"}}`),

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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockWeb := webhookhandler.NewMockWebhookHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				whHandler:   mockWeb,
			}

			mockWeb.EXPECT().SendWebhookToCustomer(gomock.Any(), tt.customerID, tt.dataType, tt.data).Return(nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1WebhookDestinationsPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		customerID uuid.UUID
		uri        string
		method     webhook.MethodType
		dataType   webhook.DataType
		data       json.RawMessage

		expectRes *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/webhook_destinations",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","uri":"test.com","method":"POST","data_type":"application/json","data":"test webhook."}`),
			},

			customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			uri:        "test.com",
			method:     webhook.MethodTypePOST,
			dataType:   "application/json",
			data:       []byte(`"test webhook."`),

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockWeb := webhookhandler.NewMockWebhookHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				whHandler:   mockWeb,
			}

			mockWeb.EXPECT().SendWebhookToURI(gomock.Any(), tt.customerID, tt.uri, tt.method, tt.dataType, tt.data).Return(nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1WebhooksPostInvalidURI(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		expectRes *sock.Response
	}{
		{
			"invalid_uri_too_short",
			&sock.Request{
				URI:      "/v1",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{}`),
			},
			&sock.Response{
				StatusCode: 400,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockWeb := webhookhandler.NewMockWebhookHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				whHandler:   mockWeb,
			}

			ctx := context.Background()
			res, err := h.processV1WebhooksPost(ctx, tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectRes.StatusCode {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes.StatusCode, res.StatusCode)
			}
		})
	}
}

func Test_processV1WebhooksPostInvalidJSON(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		expectRes *sock.Response
	}{
		{
			"invalid_json",
			&sock.Request{
				URI:      "/v1/webhooks",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`invalid json`),
			},
			&sock.Response{
				StatusCode: 400,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockWeb := webhookhandler.NewMockWebhookHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				whHandler:   mockWeb,
			}

			ctx := context.Background()
			res, err := h.processV1WebhooksPost(ctx, tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectRes.StatusCode {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes.StatusCode, res.StatusCode)
			}
		})
	}
}

func Test_processV1WebhooksPostWebhookError(t *testing.T) {
	tests := []struct {
		name       string
		request    *sock.Request
		customerID uuid.UUID
		dataType   webhook.DataType
		expectRes  *sock.Response
	}{
		{
			"webhook_error",
			&sock.Request{
				URI:      "/v1/webhooks",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","data_type":"application/json","data":{"test":"value"}}`),
			},
			uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
			"application/json",
			&sock.Response{
				StatusCode: 500,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockWeb := webhookhandler.NewMockWebhookHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				whHandler:   mockWeb,
			}

			ctx := context.Background()
			mockWeb.EXPECT().SendWebhookToCustomer(ctx, tt.customerID, tt.dataType, gomock.Any()).Return(fmt.Errorf("webhook error"))

			res, err := h.processV1WebhooksPost(ctx, tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectRes.StatusCode {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes.StatusCode, res.StatusCode)
			}
		})
	}
}

func Test_processV1WebhookDestinationsPostInvalidURI(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		expectRes *sock.Response
	}{
		{
			"invalid_uri_too_short",
			&sock.Request{
				URI:      "/v1",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{}`),
			},
			&sock.Response{
				StatusCode: 400,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockWeb := webhookhandler.NewMockWebhookHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				whHandler:   mockWeb,
			}

			ctx := context.Background()
			res, err := h.processV1WebhookDestinationsPost(ctx, tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectRes.StatusCode {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes.StatusCode, res.StatusCode)
			}
		})
	}
}

func Test_processV1WebhookDestinationsPostInvalidJSON(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		expectRes *sock.Response
	}{
		{
			"invalid_json",
			&sock.Request{
				URI:      "/v1/webhook_destinations",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`invalid json`),
			},
			&sock.Response{
				StatusCode: 400,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockWeb := webhookhandler.NewMockWebhookHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				whHandler:   mockWeb,
			}

			ctx := context.Background()
			res, err := h.processV1WebhookDestinationsPost(ctx, tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectRes.StatusCode {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes.StatusCode, res.StatusCode)
			}
		})
	}
}

func Test_processV1WebhookDestinationsPostWebhookError(t *testing.T) {
	tests := []struct {
		name       string
		request    *sock.Request
		customerID uuid.UUID
		uri        string
		method     webhook.MethodType
		dataType   webhook.DataType
		expectRes  *sock.Response
	}{
		{
			"webhook_error",
			&sock.Request{
				URI:      "/v1/webhook_destinations",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","uri":"test.com","method":"POST","data_type":"application/json","data":"test"}`),
			},
			uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
			"test.com",
			webhook.MethodTypePOST,
			"application/json",
			&sock.Response{
				StatusCode: 500,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockWeb := webhookhandler.NewMockWebhookHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				whHandler:   mockWeb,
			}

			ctx := context.Background()
			mockWeb.EXPECT().SendWebhookToURI(ctx, tt.customerID, tt.uri, tt.method, tt.dataType, gomock.Any()).Return(fmt.Errorf("webhook error"))

			res, err := h.processV1WebhookDestinationsPost(ctx, tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectRes.StatusCode {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes.StatusCode, res.StatusCode)
			}
		})
	}
}

func Test_processRequestNotFound(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		expectRes *sock.Response
	}{
		{
			"not_found",
			&sock.Request{
				URI:      "/v1/unknown",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{}`),
			},
			&sock.Response{
				StatusCode: 404,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockWeb := webhookhandler.NewMockWebhookHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				whHandler:   mockWeb,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.expectRes.StatusCode {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes.StatusCode, res.StatusCode)
			}
		})
	}
}

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockWeb := webhookhandler.NewMockWebhookHandler(mc)

	h := NewListenHandler(mockSock, mockWeb)
	if h == nil {
		t.Errorf("Wrong match. expect: handler, got: nil")
	}
}

func Test_simpleResponse(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"status_200", 200},
		{"status_400", 400},
		{"status_404", 404},
		{"status_500", 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := simpleResponse(tt.statusCode)
			if res.StatusCode != tt.statusCode {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.statusCode, res.StatusCode)
			}
		})
	}
}
