package listenhandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"
)

func Test_Run(t *testing.T) {

	type test struct {
		name string

		queue             string
		responseQueueErr  error
		responseConsume   error
		expectConsumeCall bool

		expectErr bool
	}

	tests := []test{
		{
			name: "normal",

			queue:             "bin-manager.billing-manager.request.test",
			expectConsumeCall: true,
		},
		{
			name: "consume error is only logged",

			queue:             "bin-manager.billing-manager.request.test",
			responseConsume:   errors.New("consume failure"),
			expectConsumeCall: true,
		},
		{
			name: "queue create error is returned and nothing is consumed",

			queue:            "bin-manager.billing-manager.request.test",
			responseQueueErr: errors.New("queue failure"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
			}

			done := make(chan struct{})

			mockSock.EXPECT().QueueCreate(tt.queue, "normal").Return(tt.responseQueueErr)
			if tt.expectConsumeCall {
				// the callback is a func value and can not be compared, so it is only checked to be set.
				mockSock.EXPECT().ConsumeRPC(gomock.Any(), tt.queue, "billing-manager", false, false, false, 10, gomock.Any()).DoAndReturn(
					func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, cb sock.CbMsgRPC) error {
						defer close(done)
						if cb == nil {
							t.Errorf("Wrong match. expect: a consume callback, got: nil")
						}
						return tt.responseConsume
					},
				).Times(1)
			} else {
				mockSock.EXPECT().ConsumeRPC(gomock.Any(), tt.queue, "billing-manager", false, false, false, 10, gomock.Any()).Times(0)
			}

			err := h.Run(tt.queue, "")
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}

			if tt.expectConsumeCall {
				// the wait is only a fail-safe so that a regression fails instead of hanging.
				select {
				case <-done:
				case <-time.After(10 * time.Second):
					t.Errorf("Wrong match. expect: ConsumeRPC is called, got: no call")
				}
			}
		})
	}
}

func Test_processRequest_notFound(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request
	}

	tests := []test{
		{
			name: "unregistered uri",
			request: &sock.Request{
				URI:    "/v1/no_such_resource",
				Method: sock.RequestMethodGet,
			},
		},
		{
			name: "registered uri with an unsupported method",
			request: &sock.Request{
				URI:    "/v1/accounts/922907b6-0942-11ee-960e-f31d2cc10daa",
				Method: sock.RequestMethodDelete,
			},
		},
		{
			name: "top_up with get",
			request: &sock.Request{
				URI:    "/v1/accounts/top_up",
				Method: sock.RequestMethodGet,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no handler mock is set: any downstream call would fail the test.
			h := &listenHandler{
				sockHandler: sockhandler.NewMockSockHandler(mc),
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			expectRes := &sock.Response{StatusCode: 404}
			if !reflect.DeepEqual(res, expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
			}
		})
	}
}
