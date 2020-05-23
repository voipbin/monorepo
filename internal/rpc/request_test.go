package rpc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"gitlab.com/voipbin/voip/asterisk-proxy/pkg/rabbitmq"
	// "gitlab.com/voipbin/voip/asterisk-proxy/internal/rabbitmq"
)

func TestInitiate(t *testing.T) {
	Initiate("localhost:9099", "test:test")

	if ariAddr != "localhost:9099" {
		t.Errorf("Wrong match. expect: localhost:9099, got: %s", ariAddr)
	}

	if ariAccount != "test:test" {
		t.Errorf("Wrong match. expect: test:test, got: %s", ariAccount)
	}
}

func TestParseNormal(t *testing.T) {
	req := `{
		"uri": "/channels/create",
		"method": "POST",
		"data": "{ \"endpoint\": \"PJSIP/pchero-voip/sip:test@127.0.0.1\", \"app\": \"voipbin_test\", \"appArgs\": \"\", \"channelId\": \"03102122-7895-11ea-9721-63dc96d187a1\", \"callerId\": \"test\" }",
		"data_type": "application/json"
	}`

	res, err := parse([]byte(req))
	if err != nil {
		t.Errorf("Could not parse the message. err: %v", err)
	}

	if res.URI != "/channels/create" {
		t.Errorf("Could not parse the URL.")
	}
	if res.Method != "POST" {
		t.Errorf("Expected POST, but got %s", res.Method)
	}
	if res.DataType != "application/json" {
		t.Errorf("Expected application/json, but got %s", res.DataType)
	}
	if res.Data != "{ \"endpoint\": \"PJSIP/pchero-voip/sip:test@127.0.0.1\", \"app\": \"voipbin_test\", \"appArgs\": \"\", \"channelId\": \"03102122-7895-11ea-9721-63dc96d187a1\", \"callerId\": \"test\" }" {
		t.Errorf("Expected whole string, but got %s", res.Data)
	}
}

func TestParseError(t *testing.T) {
	type test struct {
		name string
		data string
	}

	tests := []test{
		{
			name: "wrong url type",
			data: `{"uri": 123}`,
		},
		{
			name: "wrong method type",
			data: `{"method": 12345}`,
		},
		{
			name: "wrong data_type type",
			data: `{"data_type": 12345}`,
		},
		{
			name: "wrong data type",
			data: `{"data": 12345}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parse([]byte(tt.data))
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func TestSendRequest(t *testing.T) {

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
	Initiate(url, "asterisk:asterisk")

	m := &rabbitmq.Request{
		URI:      "/channels?endpoint=pjsip/test@sippuas&app=test",
		Method:   "POST",
		DataType: "application/json",
		Data:     "{\"endpoint\": \"pjsip/test@sippuas\", \"app\": \"test\"}",
	}

	status, res, err := sendRequest(m)
	if err != nil {
		t.Errorf("Expected ok, but got error. err: %v", err)
	}
	if status != 200 {
		t.Errorf("Expected 200, but got %d", status)
	}
	t.Logf("status: %d, res: %s", status, res)
}

func TestRequestHandler(t *testing.T) {
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
	Initiate(url, "asterisk:asterisk")

	// m := `{
	// 	"url": "/channels/create",
	// 	"method": "POST",
	// 	"data": "{ \"endpoint\": \"PJSIP/pchero-voip/sip:test@127.0.0.1\", \"app\": \"voipbin_test\", \"appArgs\": \"\", \"channelId\": \"03102122-7895-11ea-9721-63dc96d187a1\", \"callerId\": \"test\" }",
	// 	"data_type": "application/json"
	// }`

	request := &rabbitmq.Request{
		URI:      "/channels/create",
		Method:   "POST",
		Data:     "{ \"endpoint\": \"PJSIP/pchero-voip/sip:test@127.0.0.1\", \"app\": \"voipbin_test\", \"appArgs\": \"\", \"channelId\": \"03102122-7895-11ea-9721-63dc96d187a1\", \"callerId\": \"test\" }",
		DataType: "application/json",
	}

	_, err := RequestHandler(request)
	if err != nil {
		t.Errorf("Wront match. expect: ok, got: %v", err)
	}
}
