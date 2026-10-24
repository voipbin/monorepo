package listenhandler

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/voip-asterisk-proxy/pkg/servicehandler"
)

// hostPortOf returns host:port of the test server.
func hostPortOf(t *testing.T, ts *httptest.Server) string {
	t.Helper()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("could not parse the server url. err: %v", err)
	}

	return u.Host
}

type capturedARIRequest struct {
	method      string
	uri         string
	body        string
	auth        string
	contentType string
	// contentTypes is the number of Content-Type header lines the server saw.
	contentTypes int
}

func Test_ariSendRequestToAsterisk_forwards_request(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		responseStatus int
		responseBody   string

		expectCaptured capturedARIRequest
	}{
		{
			name: "post with content type",

			request: &sock.Request{
				URI:      "/ari/channels?app=test",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"endpoint":"pjsip/test"}`),
			},

			responseStatus: http.StatusOK,
			responseBody:   `{"id":"ch-1"}`,

			expectCaptured: capturedARIRequest{
				method:       "POST",
				uri:          "/ari/channels?app=test",
				body:         `{"endpoint":"pjsip/test"}`,
				auth:         "Basic " + base64.StdEncoding.EncodeToString([]byte("fake-user:fake-pass")),
				contentType:  "application/json",
				contentTypes: 1,
			},
		},
		{
			name: "get without data type keeps the content type empty and passes the status through",

			request: &sock.Request{
				URI:    "/ari/channels/ch-404",
				Method: sock.RequestMethodGet,
			},

			responseStatus: http.StatusNotFound,
			responseBody:   `{"message":"Channel not found"}`,

			expectCaptured: capturedARIRequest{
				method:       "GET",
				uri:          "/ari/channels/ch-404",
				body:         "",
				auth:         "Basic " + base64.StdEncoding.EncodeToString([]byte("fake-user:fake-pass")),
				contentType:  "",
				contentTypes: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured capturedARIRequest
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				captured = capturedARIRequest{
					method:       r.Method,
					uri:          r.URL.RequestURI(),
					body:         string(b),
					auth:         r.Header.Get("Authorization"),
					contentType:  r.Header.Get("Content-Type"),
					contentTypes: len(r.Header.Values("Content-Type")),
				}
				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer ts.Close()

			h := &listenHandler{
				ariAddr:    hostPortOf(t, ts),
				ariAccount: "fake-user:fake-pass",
			}

			res, err := h.listenHandlerARI(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != tt.responseStatus || res.DataType != "application/json" || string(res.Data) != tt.responseBody {
				t.Errorf("Wrong response. status: %d, type: %s, data: %s", res.StatusCode, res.DataType, res.Data)
			}
			if captured != tt.expectCaptured {
				t.Errorf("Wrong forwarded request.\nexpect: %v\ngot: %v", tt.expectCaptured, captured)
			}
		})
	}
}

func Test_ariSendRequestToAsterisk_invalid_method(t *testing.T) {
	h := &listenHandler{ariAddr: "127.0.0.1:1", ariAccount: "fake-user:fake-pass"}

	status, res, err := h.ariSendRequestToAsterisk(&sock.Request{
		URI:    "/ari/channels",
		Method: sock.RequestMethod("BAD METHOD"),
	})
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}
	if status != 0 || res != nil {
		t.Errorf("Wrong match. expect: 0 and nil, got: %d, %v", status, res)
	}
}

func Test_listenHandlerARI_unreachable_server(t *testing.T) {
	// the address of a server that was just closed refuses connections.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := hostPortOf(t, ts)
	ts.Close()

	h := &listenHandler{ariAddr: addr, ariAccount: "fake-user:fake-pass"}

	res, err := h.listenHandlerARI(&sock.Request{
		URI:    "/ari/channels",
		Method: sock.RequestMethodGet,
	})
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
}

func Test_processRequest_error_mapping(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		serviceErr error
		expectCall bool

		expectStatus int
	}{
		{
			name: "unknown uri is not found",

			request: &sock.Request{
				URI:    "/unknown/path",
				Method: sock.RequestMethodGet,
			},

			expectStatus: 404,
		},
		{
			name: "proxy route with a wrong method is not found",

			request: &sock.Request{
				URI:    "/proxy/recording_file_move",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"filenames":["a.wav"]}`),
			},

			expectStatus: 404,
		},
		{
			name: "proxy route with a malformed body is a bad request",

			request: &sock.Request{
				URI:    "/proxy/recording_file_move",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"filenames":`),
			},

			expectStatus: 400,
		},
		{
			name: "service failure is an internal error",

			request: &sock.Request{
				URI:    "/proxy/recording_file_move",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"filenames":["a.wav"]}`),
			},

			serviceErr: io.ErrUnexpectedEOF,
			expectCall: true,

			expectStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockService := servicehandler.NewMockServiceHandler(mc)
			h := &listenHandler{serviceHandler: mockService}

			if tt.expectCall {
				mockService.EXPECT().RecordingFileMove(gomock.Any(), []string{"a.wav"}).Return(tt.serviceErr)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
			if len(res.Data) != 0 {
				t.Errorf("Wrong match. expect: empty data, got: %s", res.Data)
			}
		})
	}
}

func Test_NewListenHandler_wires_ari_settings(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	h := NewListenHandler(
		sockhandler.NewMockSockHandler(mc),
		"perm.queue",
		"vol.queue",
		hostPortOf(t, ts),
		"fake-user:fake-pass",
		nil,
		servicehandler.NewMockServiceHandler(mc),
	)

	lh, ok := h.(*listenHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", h)
	}

	res, err := lh.processRequest(&sock.Request{URI: "/ari/applications", Method: sock.RequestMethodGet})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("Wrong match. expect: 204, got: %d", res.StatusCode)
	}
	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Errorf("Wrong match. expect: basic auth header, got: %q", gotAuth)
	}
	if want := "Basic " + base64.StdEncoding.EncodeToString([]byte("fake-user:fake-pass")); gotAuth != want {
		t.Errorf("Wrong auth header.\nexpect: %s\ngot: %s", want, gotAuth)
	}
}
