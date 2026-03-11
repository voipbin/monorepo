package listenhandler

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/voip-rtpengine-proxy/pkg/ngclient"
	"monorepo/voip-rtpengine-proxy/pkg/processmanager"
)

func TestProcessCommandPost_MissingType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{
		ngClient: ngclient.NewMockNGClient(ctrl),
		procMgr:  processmanager.NewMockProcessManager(ctrl),
	}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"command":"query","call-id":"abc123"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for missing type, got %d", resp.StatusCode)
	}
}

func TestProcessCommandPost_UnknownType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{
		ngClient: ngclient.NewMockNGClient(ctrl),
		procMgr:  processmanager.NewMockProcessManager(ctrl),
	}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"unknown","command":"query","call-id":"abc123"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for unknown type, got %d", resp.StatusCode)
	}
}

func TestProcessCommandPost_EmptyType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{
		ngClient: ngclient.NewMockNGClient(ctrl),
		procMgr:  processmanager.NewMockProcessManager(ctrl),
	}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"","command":"query"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for empty type, got %d", resp.StatusCode)
	}
}

func TestProcessCommandPost_InvalidJSON(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{
		ngClient: ngclient.NewMockNGClient(ctrl),
		procMgr:  processmanager.NewMockProcessManager(ctrl),
	}
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

func TestProcessCommandPost_NonStringParameters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{
		ngClient: ngclient.NewMockNGClient(ctrl),
		procMgr:  processmanager.NewMockProcessManager(ctrl),
	}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"exec","id":"00000000-0000-0000-0000-000000000001","command":"tcpdump","parameters":[123]}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for non-string parameter, got %d", resp.StatusCode)
	}
}

func TestProcessNG_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNG := ngclient.NewMockNGClient(ctrl)
	mockNG.EXPECT().Send(gomock.Any()).Return(map[string]interface{}{
		"result":  "ok",
		"call-id": "abc123",
	}, nil)

	h := &listenHandler{
		ngClient: mockNG,
		procMgr:  processmanager.NewMockProcessManager(ctrl),
	}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"ng","command":"query","call-id":"abc123"}`),
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

func TestProcessNG_MissingCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{
		ngClient: ngclient.NewMockNGClient(ctrl),
		procMgr:  processmanager.NewMockProcessManager(ctrl),
	}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"ng","call-id":"abc123"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestProcessNG_EmptyCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{
		ngClient: ngclient.NewMockNGClient(ctrl),
		procMgr:  processmanager.NewMockProcessManager(ctrl),
	}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"ng","command":"","call-id":"abc123"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestProcessNG_NonStringCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{
		ngClient: ngclient.NewMockNGClient(ctrl),
		procMgr:  processmanager.NewMockProcessManager(ctrl),
	}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"ng","command":123,"call-id":"abc"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for non-string command, got %d", resp.StatusCode)
	}
}

func TestProcessNG_NGError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNG := ngclient.NewMockNGClient(ctrl)
	mockNG.EXPECT().Send(gomock.Any()).Return(nil, errors.New("connection refused"))

	h := &listenHandler{
		ngClient: mockNG,
		procMgr:  processmanager.NewMockProcessManager(ctrl),
	}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"ng","command":"query","call-id":"abc"}`),
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

func TestProcessExec_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcMgr := processmanager.NewMockProcessManager(ctrl)
	mockProcMgr.EXPECT().Exec("00000000-0000-0000-0000-000000000001", "tcpdump", []string{"udp port 30000 or udp port 30002"}).Return(nil)

	h := &listenHandler{procMgr: mockProcMgr}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"exec","id":"00000000-0000-0000-0000-000000000001","command":"tcpdump","parameters":["udp port 30000 or udp port 30002"]}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	var result map[string]string
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result["result"] != "ok" {
		t.Errorf("expected result=ok, got %q", result["result"])
	}
}

func TestProcessExec_MissingID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{procMgr: processmanager.NewMockProcessManager(ctrl)}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"exec","command":"tcpdump"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestProcessExec_MissingCommand(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{procMgr: processmanager.NewMockProcessManager(ctrl)}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"exec","id":"00000000-0000-0000-0000-000000000001"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestProcessExec_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcMgr := processmanager.NewMockProcessManager(ctrl)
	mockProcMgr.EXPECT().Exec("00000000-0000-0000-0000-000000000001", "tcpdump", []string{"udp port 30000"}).Return(fmt.Errorf("max concurrent captures reached"))

	h := &listenHandler{procMgr: mockProcMgr}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"exec","id":"00000000-0000-0000-0000-000000000001","command":"tcpdump","parameters":["udp port 30000"]}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestProcessExec_NonUUIDID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{procMgr: processmanager.NewMockProcessManager(ctrl)}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"exec","id":"not-a-uuid","command":"tcpdump"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for non-UUID id, got %d", resp.StatusCode)
	}
}

func TestProcessKill_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcMgr := processmanager.NewMockProcessManager(ctrl)
	mockProcMgr.EXPECT().Kill("00000000-0000-0000-0000-000000000001").Return("/tmp/00000000-0000-0000-0000-000000000001.pcap", nil)

	h := &listenHandler{procMgr: mockProcMgr}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"kill","id":"00000000-0000-0000-0000-000000000001"}`),
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
		t.Errorf("expected result=ok, got %v", result["result"])
	}
	if result["pcap_path"] != "/tmp/00000000-0000-0000-0000-000000000001.pcap" {
		t.Errorf("expected pcap_path, got %v", result["pcap_path"])
	}
}

func TestProcessKill_MissingID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{procMgr: processmanager.NewMockProcessManager(ctrl)}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"kill"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestProcessKill_NonUUIDID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{procMgr: processmanager.NewMockProcessManager(ctrl)}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"kill","id":"not-a-uuid"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for non-UUID id, got %d", resp.StatusCode)
	}
}

func TestProcessKill_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProcMgr := processmanager.NewMockProcessManager(ctrl)
	mockProcMgr.EXPECT().Kill("00000000-0000-0000-0000-000000000099").Return("", fmt.Errorf("no process with id"))

	h := &listenHandler{procMgr: mockProcMgr}
	req := &sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodPost,
		Data:   json.RawMessage(`{"type":"kill","id":"00000000-0000-0000-0000-000000000099"}`),
	}
	resp, err := h.processCommandPost(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}
