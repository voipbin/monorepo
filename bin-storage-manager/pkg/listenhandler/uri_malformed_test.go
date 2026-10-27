package listenhandler

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
)

// Test_handlers_malformedURI checks that a request URI with an invalid percent escape is
// rejected with an error and no response, before any downstream handler is called. The
// downstream mocks have no expectations, so any call to them fails the test.
func Test_handlers_malformedURI(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		uri    string
		expect string // part of the expected error message
		call   func(h *listenHandler, m *sock.Request) (*sock.Response, error)
	}{
		{
			name:   "accounts list",
			uri:    "/%zz/v1/accounts?page_size=3",
			expect: "invalid URL escape",
			call: func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
				return h.v1AccountsGet(ctx, m)
			},
		},
		{
			name:   "accounts get",
			uri:    "/%zz/v1/accounts/" + testID,
			expect: "invalid URL escape",
			call: func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
				return h.v1AccountsIDGet(ctx, m)
			},
		},
		{
			name:   "files list",
			uri:    "/%zz/v1/files?page_size=3",
			expect: "invalid URL escape",
			call: func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
				return h.v1FilesGet(ctx, m)
			},
		},
		{
			name:   "files get",
			uri:    "/%zz/v1/files/" + testID,
			expect: "invalid URL escape",
			call: func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
				return h.v1FilesIDGet(ctx, m)
			},
		},
		{
			name:   "recording get",
			uri:    "/v1/recordings/%zz",
			expect: "could not unescape the recording id",
			call: func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
				return h.v1RecordingsIDGet(ctx, m)
			},
		},
		{
			name:   "recording peaks",
			uri:    "/v1/recordings/%zz/peaks",
			expect: "could not unescape the recording id",
			call: func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
				return h.v1RecordingsIDPeaksGet(ctx, m)
			},
		},
		{
			name:   "recording delete",
			uri:    "/v1/recordings/%zz",
			expect: "could not unescape the recording id",
			call: func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
				return h.v1RecordingsIDDelete(ctx, m)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _ := newTestListenHandler(mc)

			res, err := tt.call(h, &sock.Request{URI: tt.uri, Method: sock.RequestMethodGet, Data: []byte(`{}`)})
			if err == nil || !strings.Contains(err.Error(), tt.expect) {
				t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expect, err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil response, got: %v", res)
			}
		})
	}
}
