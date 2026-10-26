package requesthandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"monorepo/bin-common-handler/models/sock"
)

// Test_PipecatV1Ping_ResponseInterpretation covers the response-body
// branches that Test_PipecatV1Ping (pipecat_ping_test.go) does not exercise:
// a non-200 status with a mismatched body, an empty-object body, a body
// that fails to parse, and a nil response. None of these are expected to
// turn into an error (see the IMPORTANT comment on PipecatV1Ping: any
// response at all, or none, means liveness; only the explicit host_id
// mismatch on a 200 does).
func Test_PipecatV1Ping_ResponseInterpretation(t *testing.T) {
	const hostID = "pod-a"

	tests := []struct {
		name string

		res *sock.Response

		expectErr    bool
		expectErrSub string
	}{
		{
			name:      "non-200 status with a mismatched body is not inspected",
			res:       &sock.Response{StatusCode: 404, Data: []byte(`{"host_id":"pod-b"}`)},
			expectErr: false,
		},
		{
			name:      "200 with an empty object body has no host_id to mismatch",
			res:       &sock.Response{StatusCode: 200, Data: []byte(`{}`)},
			expectErr: false,
		},
		{
			name:      "200 with an unparseable body is not an error",
			res:       &sock.Response{StatusCode: 200, Data: []byte(`not-json`)},
			expectErr: false,
		},
		{
			name:      "nil response is not an error",
			res:       nil,
			expectErr: false,
		},
		{
			name:      "200 with a matching host_id is not an error",
			res:       &sock.Response{StatusCode: 200, Data: []byte(`{"host_id":"pod-a"}`)},
			expectErr: false,
		},
		{
			name:         "200 with a mismatched host_id is an error naming the requested host",
			res:          &sock.Response{StatusCode: 200, Data: []byte(`{"host_id":"pod-b"}`)},
			expectErr:    true,
			expectErrSub: hostID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &errPathSock{res: tt.res}
			h := &requestHandler{sock: fs}

			err := h.PipecatV1Ping(context.Background(), hostID)
			if tt.expectErr && err == nil {
				t.Errorf("Wrong match. expect: an error, got: nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
			if tt.expectErrSub != "" && (err == nil || !strings.Contains(err.Error(), tt.expectErrSub)) {
				t.Errorf("Wrong match. expect: the error to mention %q, got: %v", tt.expectErrSub, err)
			}

			if !strings.HasSuffix(fs.lastQueue, "."+hostID) {
				t.Errorf("Wrong match. expect: the queue name to end with .%s, got: %s", hostID, fs.lastQueue)
			}
		})
	}
}

// Test_PipecatV1Ping_TransportErrorPropagates is covered by the generic
// Test_RPCWrappers_TransportErrorPropagates table; this file only adds the
// response-interpretation cases that table cannot express.
func Test_PipecatV1Ping_TransportErrorPropagates(t *testing.T) {
	sentinel := errors.New("broker unavailable")
	fs := &errPathSock{err: sentinel}
	h := &requestHandler{sock: fs}

	err := h.PipecatV1Ping(context.Background(), "pod-a")
	if !errors.Is(err, sentinel) {
		t.Errorf("Wrong match. expect: the transport error propagated, got: %v", err)
	}
}
