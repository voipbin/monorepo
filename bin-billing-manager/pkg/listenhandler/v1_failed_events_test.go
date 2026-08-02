package listenhandler

import (
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/pkg/failedeventhandler"
)

func Test_processV1FailedEventsRetryPost(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseRetried   int
		responseSucceeded int
		responseExhausted int
		responseErr       error

		expectRes *sock.Response
	}

	tests := []test{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/failed_events/retry",
				Method: sock.RequestMethodPost,
			},

			responseRetried:   5,
			responseSucceeded: 3,
			responseExhausted: 1,
			responseErr:       nil,

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"retried":5,"succeeded":3,"exhausted":1}`),
			},
		},
		{
			name: "handler error maps to non-2xx",
			request: &sock.Request{
				URI:    "/v1/failed_events/retry",
				Method: sock.RequestMethodPost,
			},

			responseRetried:   0,
			responseSucceeded: 0,
			responseExhausted: 0,
			responseErr:       fmt.Errorf("could not query failed events"),

			expectRes: &sock.Response{
				StatusCode: 500,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockFailedEvent := failedeventhandler.NewMockFailedEventHandler(mc)

			h := &listenHandler{
				sockHandler:        mockSock,
				failedEventHandler: mockFailedEvent,
			}

			mockFailedEvent.EXPECT().RetryPending(gomock.Any()).Return(tt.responseRetried, tt.responseSucceeded, tt.responseExhausted, tt.responseErr)
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
