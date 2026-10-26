package listenhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// Test_processV1AIsIDPut_EngineKeyOmittedVsEmpty pins that the PUT body keeps "omitted" and ""
// apart on the way to the AI handler: omitted is nil (keep), "" is a pointer to "" (delete).
func Test_processV1AIsIDPut_EngineKeyOmittedVsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		expect *string
	}{
		{"omitted", `{"name":"n"}`, nil},
		{"null", `{"name":"n","engine_key":null}`, nil},
		{"empty", `{"name":"n","engine_key":""}`, strPtr("")},
		{"value", `{"name":"n","engine_key":"dummy-key"}`, strPtr("dummy-key")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			h := &listenHandler{sockHandler: mockSock, aiHandler: mockAI}

			id := uuid.FromStringOrNil("de99e522-a770-11ed-a0ab-5b39ee2db203")
			req := &sock.Request{URI: "/v1/ais/" + id.String(), Method: sock.RequestMethodPut, Data: []byte(tt.body)}

			mockAI.EXPECT().Update(gomock.Any(), id, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ uuid.UUID, _, _ string, _ ai.Type, _ ai.EngineModel, _ map[string]any, key *string, _ string, _ *string,
					_ uuid.UUID, _ string, _ ai.TTSType, _ string, _ ai.STTType, _ string, _ []tool.ToolName, _ *ai.VADConfig, _, _ bool) (*ai.AI, error) {
					if (key == nil) != (tt.expect == nil) || (key != nil && *key != *tt.expect) {
						t.Errorf("engine key. expect: %v, got: %v", tt.expect, key)
					}
					return &ai.AI{}, nil
				})

			res, err := h.processRequest(req)
			if err != nil || res.StatusCode != 200 {
				t.Fatalf("unexpected result. err: %v, res: %+v", err, res)
			}
		})
	}
}
