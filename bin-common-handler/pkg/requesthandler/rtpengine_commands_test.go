package requesthandler

import (
	"context"
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

		expectQueue  string
		expectURI    string
		expectMethod sock.RequestMethod

		response    *sock.Response
		expectError bool
	}{
		{
			"start recording",
			"10.164.0.12",
			map[string]interface{}{
				"command": "start recording",
				"call-id": "abc123@sip-server",
			},

			"rtpengine.10.164.0.12.request",
			"/v1/commands",
			sock.RequestMethodPost,

			&sock.Response{StatusCode: 200, Data: []byte(`{"result":"ok"}`)},
			false,
		},
		{
			"stop recording",
			"10.164.0.12",
			map[string]interface{}{
				"command": "stop recording",
				"call-id": "abc123@sip-server",
			},

			"rtpengine.10.164.0.12.request",
			"/v1/commands",
			sock.RequestMethodPost,

			&sock.Response{StatusCode: 200, Data: []byte(`{"result":"ok"}`)},
			false,
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
				gomock.Any(),
			).Return(tt.response, nil)

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
