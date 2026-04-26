package requesthandler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_PipecatV1Ping(t *testing.T) {

	tests := []struct {
		name string

		hostID string

		expectTarget  string
		expectRequest *sock.Request

		response    *sock.Response
		responseErr error

		expectErr       bool
		expectErrSubstr string
	}{
		{
			name: "alive pod returns matching host_id",

			hostID: "10.4.2.18",

			expectTarget: fmt.Sprintf("%s.%s", commonoutline.QueueNamePipecatRequest, "10.4.2.18"),
			expectRequest: &sock.Request{
				URI:      "/v1/ping",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
				Data:     nil,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"host_id":"10.4.2.18","timestamp":"2026-04-26T12:00:00Z"}`),
			},
			responseErr: nil,

			expectErr: false,
		},
		{
			name: "host_id echo mismatch",

			hostID: "10.4.2.18",

			expectTarget: fmt.Sprintf("%s.%s", commonoutline.QueueNamePipecatRequest, "10.4.2.18"),
			expectRequest: &sock.Request{
				URI:      "/v1/ping",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
				Data:     nil,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"host_id":"10.4.2.99","timestamp":"2026-04-26T12:00:00Z"}`),
			},
			responseErr: nil,

			expectErr:       true,
			expectErrSubstr: "host_id mismatch",
		},
		{
			name: "old-pod 404 with empty body treated as alive",

			hostID: "10.4.2.18",

			expectTarget: fmt.Sprintf("%s.%s", commonoutline.QueueNamePipecatRequest, "10.4.2.18"),
			expectRequest: &sock.Request{
				URI:      "/v1/ping",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
				Data:     nil,
			},

			response: &sock.Response{
				StatusCode: 404,
			},
			responseErr: nil,

			expectErr: false,
		},
		{
			name: "200 with empty body treated as alive",

			hostID: "10.4.2.18",

			expectTarget: fmt.Sprintf("%s.%s", commonoutline.QueueNamePipecatRequest, "10.4.2.18"),
			expectRequest: &sock.Request{
				URI:      "/v1/ping",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
				Data:     nil,
			},

			response: &sock.Response{
				StatusCode: 200,
			},
			responseErr: nil,

			expectErr: false,
		},
		{
			name: "timeout from sendRequest propagates",

			hostID: "10.4.2.18",

			expectTarget: fmt.Sprintf("%s.%s", commonoutline.QueueNamePipecatRequest, "10.4.2.18"),
			expectRequest: &sock.Request{
				URI:      "/v1/ping",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
				Data:     nil,
			},

			response:    nil,
			responseErr: context.DeadlineExceeded,

			expectErr:       true,
			expectErrSubstr: "deadline exceeded",
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
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, tt.responseErr)

			err := reqHandler.PipecatV1Ping(ctx, tt.hostID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
					return
				}
				if tt.expectErrSubstr != "" && !strings.Contains(err.Error(), tt.expectErrSubstr) {
					t.Errorf("Wrong match. expect error containing %q, got: %v", tt.expectErrSubstr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}
		})
	}
}
