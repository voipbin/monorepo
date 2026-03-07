package listenhandler

import (
	"testing"

	"go.uber.org/mock/gomock"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/voip-rtpengine-proxy/pkg/ngclient"
)

func TestProcessRequest_UnknownURI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{ngClient: ngclient.NewMockNGClient(ctrl)}
	resp, err := h.processRequest(&sock.Request{
		URI:    "/v1/unknown",
		Method: sock.RequestMethodPost,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 for unknown URI, got %d", resp.StatusCode)
	}
}

func TestProcessRequest_WrongMethod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &listenHandler{ngClient: ngclient.NewMockNGClient(ctrl)}
	resp, err := h.processRequest(&sock.Request{
		URI:    "/v1/commands",
		Method: sock.RequestMethodGet,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404 for GET /v1/commands, got %d", resp.StatusCode)
	}
}
