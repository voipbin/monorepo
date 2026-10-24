package requesthandler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// statusMode selects how a wrapper reports a non-2xx response.
type statusMode int

const (
	// statusSentinel means the wrapper passes the response to parseResponse, so a 404 maps to ErrNotFound.
	statusSentinel statusMode = iota
	// statusCustom means the wrapper checks the status code itself and returns an error naming the code.
	statusCustom
)

// rpcCase describes one RPC client wrapper call.
// Expected request values and the expected result are written by hand from the wire contract.
type rpcCase struct {
	name string

	// queue is the literal queue name the wrapper publishes to.
	queue string

	expectURI      string
	expectMethod   sock.RequestMethod
	expectDataType string
	// expectData is the literal JSON body. Empty means the request carries no body.
	expectData string

	// responseData is the literal JSON body of the successful response.
	responseData string

	// call invokes the wrapper under test.
	call func(h *requestHandler) (any, error)

	// expectRes is the expected result for the successful response.
	expectRes any

	statusMode statusMode
}

func (c *rpcCase) expectedRequest() *sock.Request {
	req := &sock.Request{
		URI:      c.expectURI,
		Method:   c.expectMethod,
		DataType: c.expectDataType,
	}
	if c.expectData != "" {
		req.Data = []byte(c.expectData)
	}
	return req
}

func (c *rpcCase) newHandler(t *testing.T) (*requestHandler, *sockhandler.MockSockHandler) {
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockSock := sockhandler.NewMockSockHandler(mc)
	return &requestHandler{sock: mockSock}, mockSock
}

// runRPCCases runs the success, transport error and non-2xx status subtests of each case.
func runRPCCases(t *testing.T, cases []rpcCase) {
	for i := range cases {
		tt := cases[i]

		t.Run(tt.name+"_success", func(t *testing.T) {
			h, mockSock := tt.newHandler(t)
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.queue, tt.expectedRequest()).Return(&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(tt.responseData),
			}, nil)

			res, err := tt.call(h)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %#v\ngot: %#v\n", tt.expectRes, res)
			}
		})

		t.Run(tt.name+"_transport_error", func(t *testing.T) {
			h, mockSock := tt.newHandler(t)
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.queue, tt.expectedRequest()).Return(nil, fmt.Errorf("connection refused"))

			res, err := tt.call(h)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if errors.Is(err, ErrNotFound) {
				t.Errorf("Wrong match. a transport error must not map to ErrNotFound: %v", err)
			}
			if !strings.Contains(err.Error(), "connection refused") {
				t.Errorf("Wrong match. the transport error is lost: %v", err)
			}
			assertZeroResult(t, res)
		})

		t.Run(tt.name+"_status_error", func(t *testing.T) {
			h, mockSock := tt.newHandler(t)
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.queue, tt.expectedRequest()).Return(&sock.Response{
				StatusCode: 404,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{}`),
			}, nil)

			res, err := tt.call(h)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}

			switch tt.statusMode {
			case statusCustom:
				if errors.Is(err, ErrNotFound) {
					t.Errorf("Wrong match. expect a wrapper specific error, got: %v", err)
				}
				if !strings.Contains(err.Error(), "404") {
					t.Errorf("Wrong match. the status code is missing in the error: %v", err)
				}
			default:
				if !errors.Is(err, ErrNotFound) {
					t.Errorf("Wrong match. expect: %v, got: %v", ErrNotFound, err)
				}
			}
			assertZeroResult(t, res)
		})
	}
}

// assertZeroResult checks that a failed call returns the zero value of the declared result type.
func assertZeroResult(t *testing.T, res any) {
	t.Helper()

	if res == nil {
		return
	}
	if !reflect.ValueOf(res).IsZero() {
		t.Errorf("Wrong match. expect: zero value, got: %#v", res)
	}
}

// runMarshalErrorCase checks that a wrapper fails before publishing when its argument can not be marshaled.
func runMarshalErrorCase(t *testing.T, name string, call func(h *requestHandler, ctx context.Context) (any, error)) {
	t.Run(name, func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		// no EXPECT: the request must never be published.
		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &requestHandler{sock: mockSock}

		res, err := call(h, context.Background())
		if err == nil {
			t.Fatalf("Wrong match. expect: error, got: nil")
		}
		if !strings.Contains(err.Error(), "json") {
			t.Errorf("Wrong match. expect a json marshal error, got: %v", err)
		}
		assertZeroResult(t, res)
	})
}
