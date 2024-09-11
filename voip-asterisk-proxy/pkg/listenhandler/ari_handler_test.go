package listenhandler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func TestARISendRequestToAsterisk(t *testing.T) {

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
		fmt.Fprintln(w, res)
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

	type test struct {
		name    string
		message *sock.Request
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:      "/channels?endpoint=pjsip/test@sippuas&app=test",
				Method:   "POST",
				DataType: "application/json",
				Data:     []byte("{\"endpoint\": \"pjsip/test@sippuas\", \"app\": \"test\"}"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

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
