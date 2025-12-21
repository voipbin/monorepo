package listenhandler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_ariSendRequestToAsterisk(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		response string

		expectedRes *sock.Response

		message *sock.Request
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:    "/ari/channels?endpoint=pjsip/test@sippuas&app=test",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{\"endpoint\": \"pjsip/test@sippuas\", \"app\": \"test\"}`),
			},

			response: `{"id": "1543782963.26"}`,

			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id": "1543782963.26"}`),
			},

			message: &sock.Request{
				URI:      "/channels?endpoint=pjsip/test@sippuas&app=test",
				Method:   "POST",
				DataType: "application/json",
				Data:     []byte("{\"endpoint\": \"pjsip/test@sippuas\", \"app\": \"test\"}"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockRabbit := sockhandler.NewMockSockHandler(mc)

			// setup dummy server
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = fmt.Fprintf(w, "%s", tt.response)
			}))
			defer ts.Close()

			u, _ := url.Parse(ts.URL)
			url := fmt.Sprintf("%s:%s", u.Hostname(), u.Port())

			h := listenHandler{
				sockHandler:                       mockRabbit,
				rabbitQueueListenRequestPermanent: "",

				ariAddr:    url,
				ariAccount: "asterisk:asterisk",

				amiSock: nil,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_tmpariSendRequestToAsterisk(t *testing.T) {

	tests := []struct {
		name    string
		message *sock.Request
	}{
		{
			name: "normal",
			message: &sock.Request{
				URI:      "/channels?endpoint=pjsip/test@sippuas&app=test",
				Method:   "POST",
				DataType: "application/json",
				Data:     []byte("{\"endpoint\": \"pjsip/test@sippuas\", \"app\": \"test\"}"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockRabbit := sockhandler.NewMockSockHandler(mc)

			// setup dummy server
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				res := `
				{
					"id": "1543782963.26",
					"name": "PJSIP/sippuas-0000000d",
					"state": "Down",
					"caller": {
						"name": "",
						"number": ""
					},
					"connected": {
						"name": "",
						"number": ""
					},
					"accountcode": "",
					"dialplan": {
						"context": "sippuas",
						"exten": "s",
						"priority": 1
					},
					"creationtime": "2018-12-02T21:36:03.239+0100",
					"language": "en"
				}
				`
				_, _ = fmt.Fprintln(w, res)
			}))
			defer ts.Close()

			u, _ := url.Parse(ts.URL)
			url := fmt.Sprintf("%s:%s", u.Hostname(), u.Port())

			h := listenHandler{
				sockHandler:                       mockRabbit,
				rabbitQueueListenRequestPermanent: "",

				ariAddr:    url,
				ariAccount: "asterisk:asterisk",

				amiSock: nil,
			}

			status, res, err := h.ariSendRequestToAsterisk(tt.message)
			if err != nil {
				t.Errorf("Expected ok, but got error. err: %v", err)
			}
			if status != 200 {
				t.Errorf("Expected 200, but got %d", status)
			}
			t.Logf("status: %d, res: %s", status, res)
		})
	}
}
