package listenhandler

import (
	"errors"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/pushring"
	"monorepo/bin-call-manager/pkg/pushringhandler"
)

func Test_processRequest_pushRings(t *testing.T) {
	var (
		customerID   = uuid.FromStringOrNil("6b000000-0000-4000-8000-000000000001")
		confbridgeID = uuid.FromStringOrNil("6b000000-0000-4000-8000-000000000002")
		masterCallID = uuid.FromStringOrNil("6b000000-0000-4000-8000-000000000003")
		flowID       = uuid.FromStringOrNil("6b000000-0000-4000-8000-000000000004")
		sessionID    = uuid.FromStringOrNil("6b000000-0000-4000-8000-000000000005")
	)
	destination := commonaddress.Address{Type: commonaddress.TypeExtension, Target: "6b000000-0000-4000-8000-0000000000e1", TargetName: "2000"}
	source := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100"}

	tests := []struct {
		name    string
		request *sock.Request

		expectStart  bool
		responseTmp  *pushring.StartResult
		responseErr  error
		expectExpire bool
		expectClose  bool

		expectStatus int
		expectData   string
	}{
		{
			name: "start",
			request: &sock.Request{
				URI:      "/v1/push-rings",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"6b000000-0000-4000-8000-000000000001","destination":{"type":"extension","target":"6b000000-0000-4000-8000-0000000000e1","target_name":"2000"},"confbridge_id":"6b000000-0000-4000-8000-000000000002","master_call_id":"6b000000-0000-4000-8000-000000000003","flow_id":"6b000000-0000-4000-8000-000000000004","source":{"type":"tel","target":"+15550100"}}`),
			},
			expectStart:  true,
			responseTmp:  &pushring.StartResult{Started: true, SessionID: sessionID, SIPRegistered: true},
			expectStatus: 200,
			expectData:   `{"started":true,"session_id":"6b000000-0000-4000-8000-000000000005","sip_registered":true}`,
		},
		{
			name: "start without a session",
			request: &sock.Request{
				URI:      "/v1/push-rings",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"6b000000-0000-4000-8000-000000000001","destination":{"type":"extension","target":"6b000000-0000-4000-8000-0000000000e1","target_name":"2000"},"confbridge_id":"6b000000-0000-4000-8000-000000000002","master_call_id":"6b000000-0000-4000-8000-000000000003","flow_id":"6b000000-0000-4000-8000-000000000004","source":{"type":"tel","target":"+15550100"}}`),
			},
			expectStart:  true,
			responseTmp:  &pushring.StartResult{},
			expectStatus: 200,
			expectData:   `{"started":false,"session_id":"00000000-0000-0000-0000-000000000000"}`,
		},
		{
			name: "start with a broken body",
			request: &sock.Request{
				URI:    "/v1/push-rings",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{`),
			},
			expectStatus: 400,
		},
		{
			name: "expire",
			request: &sock.Request{
				URI:    "/v1/push-rings/6b000000-0000-4000-8000-000000000005/expire",
				Method: sock.RequestMethodPost,
			},
			expectExpire: true,
			expectStatus: 200,
		},
		{
			name: "expire failure is reported so the request is not taken as done",
			request: &sock.Request{
				URI:    "/v1/push-rings/6b000000-0000-4000-8000-000000000005/expire",
				Method: sock.RequestMethodPost,
			},
			expectExpire: true,
			responseErr:  errors.New("cache down"),
			expectStatus: 500,
		},
		{
			name: "close",
			request: &sock.Request{
				URI:    "/v1/push-rings/6b000000-0000-4000-8000-000000000005/close",
				Method: sock.RequestMethodPost,
			},
			expectClose:  true,
			expectStatus: 200,
		},
		{
			name: "close failure",
			request: &sock.Request{
				URI:    "/v1/push-rings/6b000000-0000-4000-8000-000000000005/close",
				Method: sock.RequestMethodPost,
			},
			expectClose:  true,
			responseErr:  errors.New("cache down"),
			expectStatus: 500,
		},
		{
			name: "an unknown method is not routed",
			request: &sock.Request{
				URI:    "/v1/push-rings/6b000000-0000-4000-8000-000000000005/expire",
				Method: sock.RequestMethodGet,
			},
			expectStatus: 404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockPush := pushringhandler.NewMockPushRingHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				pushRingHandler: mockPush,
			}

			if tt.expectStart {
				mockPush.EXPECT().Start(gomock.Any(), customerID, destination, confbridgeID, masterCallID, flowID, source).Return(tt.responseTmp, tt.responseErr)
			}
			if tt.expectExpire {
				mockPush.EXPECT().Expire(gomock.Any(), sessionID).Return(tt.responseErr)
			}
			if tt.expectClose {
				mockPush.EXPECT().Close(gomock.Any(), sessionID).Return(tt.responseErr)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: nil, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
			if tt.expectData != "" && string(res.Data) != tt.expectData {
				t.Errorf("Wrong data.\nexpect: %s\ngot: %s", tt.expectData, res.Data)
			}
		})
	}
}
