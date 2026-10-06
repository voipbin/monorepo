package listenhandler

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-ai-manager/pkg/aihandler"
)

// The request body of an AI create/update carries engine_key. No log line from the
// request path may include it, even when the downstream handler fails.
func Test_processRequest_errorLogOmitsRequestBody(t *testing.T) {
	const dummy = "dummy-engine-key-not-real"

	tests := []struct {
		name    string
		request *sock.Request
		setup   func(m *aihandler.MockAIHandler)
	}{
		{
			name: "create fails",
			request: &sock.Request{
				URI:    "/v1/ais",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"customer_id":"58e7502c-a770-11ed-9b86-7fabe2dba847","name":"n","engine_model":"openrouter.vendor/model-a","engine_key":"` + dummy + `"}`),
			},
			setup: func(m *aihandler.MockAIHandler) {
				m.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("create failed"))
			},
		},
		{
			name: "update fails",
			request: &sock.Request{
				URI:    "/v1/ais/fa4d3b6a-f82f-11ed-9176-d32f5705e10c",
				Method: sock.RequestMethodPut,
				Data:   []byte(`{"name":"n","engine_model":"openrouter.vendor/model-a","engine_key":"` + dummy + `"}`),
			},
			setup: func(m *aihandler.MockAIHandler) {
				m.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("update failed"))
			},
		},
		{
			name: "unknown uri",
			request: &sock.Request{
				URI:    "/v1/unknown-resource",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"engine_key":"` + dummy + `"}`),
			},
			setup: func(m *aihandler.MockAIHandler) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			origOut, origLevel := logrus.StandardLogger().Out, logrus.GetLevel()
			logrus.SetOutput(&buf)
			logrus.SetLevel(logrus.TraceLevel)
			defer func() {
				logrus.SetOutput(origOut)
				logrus.SetLevel(origLevel)
			}()

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAI := aihandler.NewMockAIHandler(mc)
			tt.setup(mockAI)
			h := &listenHandler{
				sockHandler: sockhandler.NewMockSockHandler(mc),
				aiHandler:   mockAI,
			}

			if _, err := h.processRequest(tt.request); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			logs := buf.String()
			if strings.Contains(logs, dummy) {
				t.Errorf("the request body leaked into the log:\n%s", logs)
			}
			if !strings.Contains(logs, tt.request.URI) {
				t.Errorf("the log must keep the request uri:\n%s", logs)
			}
		})
	}
}
