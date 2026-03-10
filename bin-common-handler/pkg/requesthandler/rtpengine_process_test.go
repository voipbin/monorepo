package requesthandler

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_RTPEngineV1ProcessSend(t *testing.T) {

	tests := []struct {
		name        string
		rtpengineID string
		data        interface{}

		expectQueue string
		expectData  []byte

		response      *sock.Response
		responseError error
		expectError   bool
	}{
		{
			"exec tcpdump",
			"10.164.0.12",
			map[string]interface{}{
				"type":       "exec",
				"id":         "180fc79d-af5b-4dfe-a5bc-2e64faa38662",
				"command":    "tcpdump",
				"parameters": []string{"udp port 30000 or udp port 30002"},
			},

			"rtpengine.10.164.0.12.request",
			[]byte(`{"command":"tcpdump","id":"180fc79d-af5b-4dfe-a5bc-2e64faa38662","parameters":["udp port 30000 or udp port 30002"],"type":"exec"}`),

			&sock.Response{StatusCode: 200, Data: []byte(`{"result":"ok"}`)},
			nil,
			false,
		},
		{
			"kill tcpdump",
			"10.164.0.12",
			map[string]interface{}{
				"type": "kill",
				"id":   "180fc79d-af5b-4dfe-a5bc-2e64faa38662",
			},

			"rtpengine.10.164.0.12.request",
			[]byte(`{"id":"180fc79d-af5b-4dfe-a5bc-2e64faa38662","type":"kill"}`),

			&sock.Response{StatusCode: 200, Data: []byte(`{"result":"ok"}`)},
			nil,
			false,
		},
		{
			"rpc error",
			"10.164.0.99",
			map[string]interface{}{
				"type": "kill",
				"id":   "abc",
			},

			"rtpengine.10.164.0.99.request",
			[]byte(`{"id":"abc","type":"kill"}`),

			nil,
			fmt.Errorf("connection refused"),
			true,
		},
		{
			"error response status code",
			"10.164.0.12",
			map[string]interface{}{
				"type": "kill",
				"id":   "abc",
			},

			"rtpengine.10.164.0.12.request",
			[]byte(`{"id":"abc","type":"kill"}`),

			&sock.Response{StatusCode: 500, Data: []byte(`{"error":"internal"}`)},
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
					URI:      "/v1/process",
					Method:   sock.RequestMethodPost,
					DataType: "application/json",
					Data:     tt.expectData,
				},
			).Return(tt.response, tt.responseError)

			err := reqHandler.RTPEngineV1ProcessSend(context.Background(), tt.rtpengineID, tt.data)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
