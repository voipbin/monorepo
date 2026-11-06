package listenhandler

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/pkg/messagehandler"
)

// Test_processRequest_hooksNotLogged drives processRequest with a hook request that carries a media url and checks
// that no log line carries the body of the hook, whether the hook succeeds, fails, or the request is unreadable.
func Test_processRequest_hooksNotLogged(t *testing.T) {
	const secret = "secret-token-123"
	body := []byte(`{"data":{"payload":{"media":[{"url":"https://media.telnyx.com/example/` + secret + `/photo.jpg"}]}}}`)
	data, _ := json.Marshal(map[string]any{"received_uri": "hook.example.net/v1.0/messages/telnyx", "received_data": body})

	tests := []struct {
		name string

		request *sock.Request
		hookErr error
		callsHk bool
	}{
		{name: "hook succeeds", request: &sock.Request{URI: "/v1/hooks", Method: sock.RequestMethodPost, Data: data}, callsHk: true},
		{name: "hook fails", request: &sock.Request{URI: "/v1/hooks", Method: sock.RequestMethodPost, Data: data}, hookErr: errors.New("simulated"), callsHk: true},
		{name: "unreadable request", request: &sock.Request{URI: "/v1/hooks", Method: sock.RequestMethodPost, Data: append([]byte(`{"received_data": 5, "x":"`+secret), '"')}},
		{name: "unknown method on the hooks uri", request: &sock.Request{URI: "/v1/hooks", Method: sock.RequestMethodGet, Data: data}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := logrus.GetLevel()
			logrus.SetLevel(logrus.DebugLevel)
			defer logrus.SetLevel(old)
			logHook := logrustest.NewGlobal()
			defer logHook.Reset()

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockMessageHandler := messagehandler.NewMockMessageHandler(mc)
			h := &listenHandler{messageHandler: mockMessageHandler}
			if tt.callsHk {
				mockMessageHandler.EXPECT().Hook(gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.hookErr)
			}

			_, _ = h.processRequest(tt.request)

			if len(logHook.AllEntries()) == 0 {
				t.Fatalf("Wrong match. expect: log lines to check, got: none")
			}
			for _, e := range logHook.AllEntries() {
				line, _ := e.String()
				if strings.Contains(line, secret) || strings.Contains(line, "photo.jpg") {
					t.Errorf("Wrong match. a log line carries the hook body: %s", line)
				}
			}
		})
	}
}

func Test_loggableRequest(t *testing.T) {
	hook := &sock.Request{URI: "/v1/hooks", Method: sock.RequestMethodPost, Data: []byte("body")}
	other := &sock.Request{URI: "/v1/messages", Method: sock.RequestMethodPost, Data: []byte("body")}

	got := loggableRequest(hook)
	if got.Data != nil || got.URI != hook.URI || got.Method != hook.Method {
		t.Errorf("Wrong match. got: %v", got)
	}
	if string(hook.Data) != "body" {
		t.Errorf("Wrong match. the original request must not change")
	}
	if loggableRequest(other) != other {
		t.Errorf("Wrong match. a request of another uri is logged as it is")
	}
	if loggableRequest(nil) != nil {
		t.Errorf("Wrong match. nil stays nil")
	}
}
