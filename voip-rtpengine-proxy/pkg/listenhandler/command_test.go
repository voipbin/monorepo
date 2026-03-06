package listenhandler

import (
	"encoding/json"
	"errors"
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

func TestProcessCommandPost_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{ngClient: ngclient.NewMockNGClient(ctrl)}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{not valid json`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestProcessCommandPost_NonStringCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{ngClient: ngclient.NewMockNGClient(ctrl)}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"command":123,"call-id":"abc"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for non-string command, got %d", resp.StatusCode)
	}
}

func TestProcessCommandPost_EmptyCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{ngClient: ngclient.NewMockNGClient(ctrl)}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"command":"","call-id":"abc"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestProcessCommandPost_NGError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNG := ngclient.NewMockNGClient(ctrl)
	mockNG.EXPECT().Send(gomock.Any()).Return(nil, errors.New("connection refused"))

	h := &listenHandler{ngClient: mockNG}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"command":"query","call-id":"abc"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
	var result map[string]string
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if result["result"] != "error" {
		t.Errorf("expected result=error, got %q", result["result"])
	}
	if result["error-reason"] == "" {
		t.Error("expected non-empty error-reason")
	}
}
