package requesthandler

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_RTPEngineV1CommandsSend(t *testing.T) {

	tests := []struct {
		name        string
		rtpengineID string
		command     map[string]interface{}

		expectQueue string
		expectData  []byte

		response      *sock.Response
		responseError error
		expectError   bool
	}{
		{
			"start recording",
			"10.164.0.12",
			map[string]interface{}{
				"call-id": "abc123@sip-server",
				"command": "start recording",
			},

			"rtpengine.10.164.0.12.request",
			[]byte(`{"call-id":"abc123@sip-server","command":"start recording"}`),

			&sock.Response{StatusCode: 200, Data: []byte(`{"result":"ok"}`)},
			nil,
			false,
		},
		{
			"different rtpengine instance",
			"10.164.0.99",
			map[string]interface{}{
				"call-id": "def456@sip-server",
				"command": "stop recording",
			},

			"rtpengine.10.164.0.99.request",
			[]byte(`{"call-id":"def456@sip-server","command":"stop recording"}`),

			&sock.Response{StatusCode: 200, Data: []byte(`{"result":"ok"}`)},
			nil,
			false,
		},
		{
			"rpc error",
			"10.164.0.12",
			map[string]interface{}{
				"call-id": "abc123@sip-server",
				"command": "query",
			},

			"rtpengine.10.164.0.12.request",
			[]byte(`{"call-id":"abc123@sip-server","command":"query"}`),

			nil,
			fmt.Errorf("connection refused"),
			true,
		},
		{
			"error response status code",
			"10.164.0.12",
			map[string]interface{}{
				"call-id": "abc123@sip-server",
				"command": "start recording",
			},

			"rtpengine.10.164.0.12.request",
			[]byte(`{"call-id":"abc123@sip-server","command":"start recording"}`),

			&sock.Response{StatusCode: 500, Data: []byte(`{"result":"error","error-reason":"Unknown call-id"}`)},
			nil,
			true,
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

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				&sock.Request{
					URI:      "/v1/commands",
					Method:   sock.RequestMethodPost,
					DataType: "application/json",
					Data:     tt.expectData,
				},
			).Return(tt.response, tt.responseError)

			res, err := reqHandler.RTPEngineV1CommandsSend(context.Background(), tt.rtpengineID, tt.command)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res["result"] != "ok" {
				t.Errorf("Wrong result. expect: ok, got: %v", res["result"])
			}
		})
	}
}
