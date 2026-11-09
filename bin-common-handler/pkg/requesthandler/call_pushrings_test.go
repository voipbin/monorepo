package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cmpushring "monorepo/bin-call-manager/models/pushring"
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_CallV1PushRingStart(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		destination  commonaddress.Address
		confbridgeID uuid.UUID
		masterCallID uuid.UUID
		flowID       uuid.UUID
		source       commonaddress.Address

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *cmpushring.StartResult
	}{
		{
			name: "started",

			customerID:   uuid.FromStringOrNil("b3c5f1c2-1001-11f0-a001-000000000001"),
			destination:  commonaddress.Address{Type: commonaddress.TypeExtension, Target: "b3c5f1c2-1001-11f0-a001-000000000002", TargetName: "2000"},
			confbridgeID: uuid.FromStringOrNil("b3c5f1c2-1001-11f0-a001-000000000003"),
			masterCallID: uuid.FromStringOrNil("b3c5f1c2-1001-11f0-a001-000000000004"),
			flowID:       uuid.FromStringOrNil("b3c5f1c2-1001-11f0-a001-000000000005"),
			source:       commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100"},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"started":true,"session_id":"b3c5f1c2-1001-11f0-a001-000000000006","sip_registered":true}`),
			},

			expectRequest: &sock.Request{
				URI:      "/v1/push-rings",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"b3c5f1c2-1001-11f0-a001-000000000001","destination":{"type":"extension","target":"b3c5f1c2-1001-11f0-a001-000000000002","target_name":"2000"},"confbridge_id":"b3c5f1c2-1001-11f0-a001-000000000003","master_call_id":"b3c5f1c2-1001-11f0-a001-000000000004","flow_id":"b3c5f1c2-1001-11f0-a001-000000000005","source":{"type":"tel","target":"+15550100"}}`),
			},
			expectRes: &cmpushring.StartResult{
				Started:       true,
				SessionID:     uuid.FromStringOrNil("b3c5f1c2-1001-11f0-a001-000000000006"),
				SIPRegistered: true,
			},
		},
		{
			name: "not started",

			customerID:   uuid.FromStringOrNil("b3c5f1c2-1001-11f0-a001-000000000001"),
			destination:  commonaddress.Address{Type: commonaddress.TypeExtension, Target: "b3c5f1c2-1001-11f0-a001-000000000002"},
			confbridgeID: uuid.FromStringOrNil("b3c5f1c2-1001-11f0-a001-000000000003"),
			masterCallID: uuid.FromStringOrNil("b3c5f1c2-1001-11f0-a001-000000000004"),
			flowID:       uuid.FromStringOrNil("b3c5f1c2-1001-11f0-a001-000000000005"),
			source:       commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100"},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"started":false}`),
			},

			expectRequest: &sock.Request{
				URI:      "/v1/push-rings",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"b3c5f1c2-1001-11f0-a001-000000000001","destination":{"type":"extension","target":"b3c5f1c2-1001-11f0-a001-000000000002"},"confbridge_id":"b3c5f1c2-1001-11f0-a001-000000000003","master_call_id":"b3c5f1c2-1001-11f0-a001-000000000004","flow_id":"b3c5f1c2-1001-11f0-a001-000000000005","source":{"type":"tel","target":"+15550100"}}`),
			},
			expectRes: &cmpushring.StartResult{},
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1PushRingStart(context.Background(), tt.customerID, tt.destination, tt.confbridgeID, tt.masterCallID, tt.flowID, tt.source)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1PushRingStart_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.call-manager.request", gomock.Any()).Return(&sock.Response{StatusCode: 404}, nil)

	res, err := reqHandler.CallV1PushRingStart(context.Background(), uuid.Nil, commonaddress.Address{}, uuid.Nil, uuid.Nil, uuid.Nil, commonaddress.Address{})
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
	}
}

func Test_CallV1PushRingExpire(t *testing.T) {

	tests := []struct {
		name string

		sessionID uuid.UUID
		delay     int

		expectRequest *sock.Request
	}{
		{
			name: "delayed",

			sessionID: uuid.FromStringOrNil("b3c5f1c2-1001-11f0-a001-000000000006"),
			delay:     60000,

			expectRequest: &sock.Request{
				URI:      "/v1/push-rings/b3c5f1c2-1001-11f0-a001-000000000006/expire",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
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

			mockSock.EXPECT().RequestPublishWithDelay("bin-manager.call-manager.request", tt.expectRequest, tt.delay).Return(nil)

			if err := reqHandler.CallV1PushRingExpire(context.Background(), tt.sessionID, tt.delay); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_CallV1PushRingClose(t *testing.T) {

	tests := []struct {
		name string

		sessionID uuid.UUID

		response *sock.Response

		expectRequest *sock.Request
	}{
		{
			name: "normal",

			sessionID: uuid.FromStringOrNil("b3c5f1c2-1001-11f0-a001-000000000006"),

			response: &sock.Response{
				StatusCode: 200,
			},

			expectRequest: &sock.Request{
				URI:      "/v1/push-rings/b3c5f1c2-1001-11f0-a001-000000000006/close",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.CallV1PushRingClose(context.Background(), tt.sessionID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
