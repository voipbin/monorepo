package requesthandler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	amai "monorepo/bin-ai-manager/models/ai"
	amrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

// Test_AIV1AIUpdate_EngineKeyWire pins what reaches the wire: an omitted key is absent (ai-manager
// keeps the stored key), an empty key is sent as "" (delete), a value is sent as is.
func Test_AIV1AIUpdate_EngineKeyWire(t *testing.T) {
	tests := []struct {
		name         string
		key          *string
		expectInWire bool
		expectValue  string
	}{
		{"omitted", nil, false, ""},
		{"empty string", ptrStr(""), true, ""},
		{"value", ptrStr("dummy-key"), true, "dummy-key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := requestHandler{sock: mockSock}

			var wire []byte
			mockSock.EXPECT().RequestPublish(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ string, req *sock.Request) (*sock.Response, error) {
					wire = req.Data
					return &sock.Response{StatusCode: 200, DataType: "application/json", Data: []byte(`{}`)}, nil
				})

			if _, err := h.AIV1AIUpdate(context.Background(), uuid.Nil, "n", "d", amai.TypeNone, "openai.gpt-5", nil, tt.key, "", nil,
				uuid.Nil, "", "", "", "", "", nil, nil, false); err != nil {
				t.Fatalf("request failed: %v", err)
			}

			var raw map[string]json.RawMessage
			if err := json.Unmarshal(wire, &raw); err != nil {
				t.Fatalf("wire: %v", err)
			}
			if _, ok := raw["engine_key"]; ok != tt.expectInWire {
				t.Errorf("engine_key on the wire. expect: %v, got: %v (%s)", tt.expectInWire, ok, wire)
			}

			var d amrequest.V1DataAIsIDPut
			if err := json.Unmarshal(wire, &d); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if (d.EngineKey != nil) != tt.expectInWire || (d.EngineKey != nil && *d.EngineKey != tt.expectValue) {
				t.Errorf("decoded key. expect: %v %q, got: %v", tt.expectInWire, tt.expectValue, d.EngineKey)
			}
		})
	}
}
