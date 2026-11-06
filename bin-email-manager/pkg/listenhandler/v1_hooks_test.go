package listenhandler

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-email-manager/pkg/emailhandler"
	reflect "reflect"
	"testing"

	gomock "go.uber.org/mock/gomock"
)

func Test_processV1HooksPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		expectReceivedURI  string
		expectReceivedData []byte
		expectRes          *sock.Response
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:      "/v1/hooks",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"received_uri":"https://hook.voipbin.net/v1.0/conversation/customers/a92e60ea-e85b-11ec-a173-0b1cf8c9d3e9/line","received_data":"eyJkZXN0aW5hdGlvbiI6IlUxMTI5ODIxNDExNmUzYWZiYWQ0MzJiNTc5NGE2ZDNhMCIsImV2ZW50cyI6W119"}`),
			},

			expectReceivedURI:  "https://hook.voipbin.net/v1.0/conversation/customers/a92e60ea-e85b-11ec-a173-0b1cf8c9d3e9/line",
			expectReceivedData: []byte(`{"destination":"U11298214116e3afbad432b5794a6d3a0","events":[]}`),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockEmail := emailhandler.NewMockEmailHandler(mc)

			h := &listenHandler{
				sockHandler:  mockSock,
				emailHandler: mockEmail,
			}

			mockEmail.EXPECT().Hook(gomock.Any(), tt.expectReceivedURI, tt.expectReceivedData).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

// Test_processV1HooksPost_noPayloadInLogs checks that a rejected hook request
// (a Hook error) and an unparsable one write nothing of the request body to the logs.
func Test_processV1HooksPost_noPayloadInLogs(t *testing.T) {
	// the body is "synthetic-secret-marker" in base64 inside the JSON data.
	const marker = "synthetic-secret-marker"
	const markerBase64 = "c3ludGhldGljLXNlY3JldC1tYXJrZXI="

	tests := []struct {
		name string
		data []byte

		hookErr    error
		expectCall bool
		expectCode int
	}{
		{"hook_error", []byte(`{"received_uri":"https://hook.example.com/v1.0/emails/mailgun","received_data":"` + markerBase64 + `"}`), fmt.Errorf("rejected"), true, 200},
		{"unparsable_data", []byte(`{"received_uri":"x","received_data":"` + marker + `"`), nil, false, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := logrustest.NewGlobal()
			oldHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
			logrus.StandardLogger().AddHook(hook)
			t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(oldHooks) })
			oldLevel := logrus.GetLevel()
			logrus.SetLevel(logrus.DebugLevel)
			t.Cleanup(func() { logrus.SetLevel(oldLevel) })

			mockEmail := emailhandler.NewMockEmailHandler(mc)
			h := &listenHandler{emailHandler: mockEmail}
			if tt.expectCall {
				mockEmail.EXPECT().Hook(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.hookErr)
			}

			res, err := h.processV1HooksPost(context.Background(), &sock.Request{URI: "/v1/hooks", Method: sock.RequestMethodPost, DataType: "application/json", Data: tt.data})
			if err != nil || res.StatusCode != tt.expectCode {
				t.Fatalf("Wrong match. expect: %d, got: %v, %v", tt.expectCode, res, err)
			}

			if len(hook.AllEntries()) == 0 {
				t.Fatalf("expected a log entry")
			}
			for _, e := range hook.AllEntries() {
				// the request is a struct whose Data is printed as decimal bytes, so check the field itself.
				if _, ok := e.Data["request"]; ok {
					t.Errorf("the log has the request field: %s", e.Message)
				}
				dump := fmt.Sprintf("%s %v", e.Message, e.Data)
				if strings.Contains(dump, marker) || strings.Contains(dump, markerBase64) {
					t.Errorf("the log contains the request body: %s", e.Message)
				}
			}
		})
	}
}
