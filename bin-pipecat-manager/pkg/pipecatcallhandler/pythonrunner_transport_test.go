package pipecatcallhandler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
)

// failingBody is a response body whose read fails.
type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("body broken") }
func (failingBody) Close() error             { return nil }

// useTestHTTPClient replaces the package level client for one test and restores it at the end.
// Tests that use it must not run in parallel.
func useTestHTTPClient(t *testing.T, fn func(r *http.Request) (*http.Response, error)) {
	t.Helper()

	orig := httpClient
	httpClient = &http.Client{Transport: roundTripFunc(fn)}
	t.Cleanup(func() { httpClient = orig })
}

func Test_pythonRunner_transportErrors(t *testing.T) {
	id := uuid.FromStringOrNil("a1b2c3d4-5001-4000-8000-000000000001")

	start := func() error {
		return (&pythonRunner{}).Start(context.Background(), id, "openai.gpt-5", "", nil, "", "", "", "", "", nil, nil, nil, false, uuid.Nil, "", "")
	}
	stop := func() error {
		return (&pythonRunner{}).Stop(context.Background(), id)
	}

	tests := []struct {
		name string

		call      func() error
		transport func(r *http.Request) (*http.Response, error)

		expectErrContains string
	}{
		{
			name: "start request cannot be sent",
			call: start,
			transport: func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("dial refused")
			},
			expectErrContains: "could not send request to python runner",
		},
		{
			name: "start response body cannot be read",
			call: start,
			transport: func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: failingBody{}, Header: http.Header{}, Request: r}, nil
			},
			expectErrContains: "could not read response body from python runner",
		},
		{
			name: "stop request cannot be sent",
			call: stop,
			transport: func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("dial refused")
			},
			expectErrContains: "could not send stop request to python runner",
		},
		{
			name: "stop response body cannot be read",
			call: stop,
			transport: func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: failingBody{}, Header: http.Header{}, Request: r}, nil
			},
			expectErrContains: "could not read stop response body from python runner",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useTestHTTPClient(t, tt.transport)

			err := tt.call()
			if err == nil || !strings.Contains(err.Error(), tt.expectErrContains) {
				t.Errorf("Wrong error. expect contains: %q, got: %v", tt.expectErrContains, err)
			}
		})
	}
}

func Test_pythonRunner_Stop_success(t *testing.T) {
	id := uuid.FromStringOrNil("a1b2c3d4-5002-4000-8000-000000000001")

	var gotMethod, gotPath, gotQuery string
	useTestHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.RawQuery
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok")), Header: http.Header{}, Request: r}, nil
	})

	if err := (&pythonRunner{}).Stop(context.Background(), id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/stop" || gotQuery != "id=a1b2c3d4-5002-4000-8000-000000000001" {
		t.Errorf("Wrong request. got: %s %s?%s", gotMethod, gotPath, gotQuery)
	}
}
