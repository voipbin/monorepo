package listenhandler

import (
	"encoding/json"
	"testing"

	"go.uber.org/mock/gomock"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/voip-rtpengine-proxy/pkg/ngclient"
)

func TestProcessCommandPost_MissingCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{ngClient: ngclient.NewMockNGClient(ctrl)}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"call-id":"abc123"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestProcessCommandPost_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNG := ngclient.NewMockNGClient(ctrl)
	mockNG.EXPECT().Send(gomock.Any()).Return(map[string]interface{}{
		"result":  "ok",
		"call-id": "abc123",
	}, nil)

	h := &listenHandler{ngClient: mockNG}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"command":"query","call-id":"abc123"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result["result"] != "ok" {
		t.Errorf("expected result ok, got %v", result["result"])
	}
}
