package listenhandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_processV1ChannelsIDGet_cov(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		expectGet     bool
		responseChan  *channel.Channel
		responseError error

		expectStatus int
		expectErr    bool
	}{
		{
			name:         "normal",
			request:      &sock.Request{URI: "/v1/channels/ch-1", Method: sock.RequestMethodGet},
			expectGet:    true,
			responseChan: &channel.Channel{ID: "ch-1"},
			expectStatus: 200,
		},
		{
			name:         "wrong uri",
			request:      &sock.Request{URI: "/v1/channels", Method: sock.RequestMethodGet},
			expectStatus: 400,
			expectErr:    true,
		},
		{
			name:          "handler returns not found",
			request:       &sock.Request{URI: "/v1/channels/ch-1", Method: sock.RequestMethodGet},
			expectGet:     true,
			responseError: dbhandler.ErrNotFound,
			expectStatus:  404,
		},
		{
			name:          "handler returns generic error",
			request:       &sock.Request{URI: "/v1/channels/ch-1", Method: sock.RequestMethodGet},
			expectGet:     true,
			responseError: errors.New("boom"),
			expectStatus:  500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockChannel := channelhandler.NewMockChannelHandler(mc)
			h := &listenHandler{channelHandler: mockChannel}

			if tt.expectGet {
				mockChannel.EXPECT().Get(gomock.Any(), "ch-1").Return(tt.responseChan, tt.responseError)
			}

			res, err := h.processV1ChannelsIDGet(context.Background(), tt.request)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong error. expect err: %v, got: %v", tt.expectErr, err)
			}
			if res == nil || res.StatusCode != tt.expectStatus {
				t.Fatalf("Wrong status. expect: %d, got: %v", tt.expectStatus, res)
			}
			if tt.expectStatus == 200 {
				if res.DataType != "application/json" {
					t.Errorf("Wrong data type. got: %s", res.DataType)
				}
				var got channel.Channel
				if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil {
					t.Fatalf("Could not unmarshal the response data. err: %v", errUnmarshal)
				}
				if got.ID != tt.responseChan.ID {
					t.Errorf("Wrong channel id. expect: %s, got: %s", tt.responseChan.ID, got.ID)
				}
			}
		})
	}
}

func Test_processV1ChannelsIDHealthPost_error_cov(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
	}{
		{
			name:    "wrong uri",
			request: &sock.Request{URI: "/v1/channels/ch-1", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
		},
		{
			name:    "unmarshal failure",
			request: &sock.Request{URI: "/v1/channels/ch-1/health-check", Method: sock.RequestMethodPost, Data: []byte(`{invalid`)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &listenHandler{channelHandler: channelhandler.NewMockChannelHandler(mc)}

			if _, err := h.processV1ChannelsIDHealthPost(context.Background(), tt.request); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_processV1ChannelsIDHealthPost_retryCount_cov(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		expectRetry int
	}{
		{name: "retry count 3", data: `{"retry_count":3}`, expectRetry: 3},
		{name: "retry count omitted", data: `{}`, expectRetry: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockChannel := channelhandler.NewMockChannelHandler(mc)
			h := &listenHandler{channelHandler: mockChannel}

			mockChannel.EXPECT().HealthCheck(gomock.Any(), "ch-1", tt.expectRetry).Times(1)

			req := &sock.Request{URI: "/v1/channels/ch-1/health-check", Method: sock.RequestMethodPost, Data: []byte(tt.data)}
			res, err := h.processV1ChannelsIDHealthPost(context.Background(), req)
			if err != nil || res != nil {
				t.Errorf("Wrong match. expect: (nil, nil), got: (%v, %v)", res, err)
			}
		})
	}
}
