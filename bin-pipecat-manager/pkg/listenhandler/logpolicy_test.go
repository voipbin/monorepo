package listenhandler

import (
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-pipecat-manager/internal/logtest"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"
)

const logPolicySentinel = "SENTINEL-CONVERSATION-TEXT-9f3c"

// The lines of a rejected or unknown request must not carry the request data. A pipecatcall request
// carries the llm messages and a message request carries the text to say.
func Test_logPolicy_requestData(t *testing.T) {
	tests := []struct {
		name string
		req  *sock.Request
	}{
		{
			name: "pipecatcall post with data that cannot be read",
			req: &sock.Request{
				URI:      "/v1/pipecatcalls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id": "` + logPolicySentinel + `"`),
			},
		},
		{
			name: "message post with data that cannot be read",
			req: &sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"text": "` + logPolicySentinel + `"`),
			},
		},
		{
			name: "unknown uri",
			req: &sock.Request{
				URI:      "/v1/unknown",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"text": "` + logPolicySentinel + `"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			logs := logtest.New(t)

			h := &listenHandler{pipecatcallHandler: pipecatcallhandler.NewMockPipecatcallHandler(mc)}
			if _, err := h.processRequest(tt.req); err != nil {
				t.Fatalf("processRequest: %v", err)
			}

			logs.AssertAbsent(t, logPolicySentinel)
			logs.AssertField(t, "data_len")
		})
	}
}
