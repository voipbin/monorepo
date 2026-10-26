package streaminghandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"go.uber.org/mock/gomock"
)

// Test_NewAWSHandler_setsFields verifies the constructor wires its
// dependencies and credentials into the returned handler.
func Test_NewAWSHandler_setsFields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	res := NewAWSHandler(mockReq, mockNotify, "access-key", "secret-key")

	h, ok := res.(*awsHandler)
	if !ok {
		t.Fatalf("Wrong type. expected: *awsHandler, got: %T", res)
	}
	if h.reqHandler != mockReq || h.notifyHandler != mockNotify {
		t.Error("Wrong match. expected: the injected dependencies to be stored")
	}
	if h.accessKey != "access-key" || h.secretKey != "secret-key" {
		t.Errorf("Wrong match. expected: access-key/secret-key, got: %s/%s", h.accessKey, h.secretKey)
	}
}

// Test_NewGCPHandler_setsFields verifies the constructor wires its
// dependencies into the returned handler.
func Test_NewGCPHandler_setsFields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	res := NewGCPHandler(mockReq, mockNotify)

	h, ok := res.(*gcpHandler)
	if !ok {
		t.Fatalf("Wrong type. expected: *gcpHandler, got: %T", res)
	}
	if h.reqHandler != mockReq || h.notifyHandler != mockNotify {
		t.Error("Wrong match. expected: the injected dependencies to be stored")
	}
}

// Test_NewElevenlabsHandler_setsFields verifies the constructor wires its
// dependencies and api key into the returned handler.
func Test_NewElevenlabsHandler_setsFields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	res := NewElevenlabsHandler(mockReq, mockNotify, "api-key")

	h, ok := res.(*elevenlabsHandler)
	if !ok {
		t.Fatalf("Wrong type. expected: *elevenlabsHandler, got: %T", res)
	}
	if h.reqHandler != mockReq || h.notifyHandler != mockNotify {
		t.Error("Wrong match. expected: the injected dependencies to be stored")
	}
	if h.apiKey != "api-key" {
		t.Errorf("Wrong match. expected: api-key, got: %s", h.apiKey)
	}
}

// Test_NewStreamingHandler_setsFields verifies the top-level constructor
// wires podID and initializes mapStreaming to a non-nil empty map, and that
// it builds the three vendor sub-handlers.
func Test_NewStreamingHandler_setsFields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	res := NewStreamingHandler(mockReq, mockNotify, "pod-1", "eleven-key", "aws-access", "aws-secret")

	h, ok := res.(*streamingHandler)
	if !ok {
		t.Fatalf("Wrong type. expected: *streamingHandler, got: %T", res)
	}
	if h.podID != "pod-1" {
		t.Errorf("Wrong match. expected: pod-1, got: %s", h.podID)
	}
	if h.mapStreaming == nil {
		t.Error("Wrong match. expected: a non-nil mapStreaming, got: nil")
	}
	if len(h.mapStreaming) != 0 {
		t.Errorf("Wrong match. expected: an empty mapStreaming, got: %d entries", len(h.mapStreaming))
	}
	if h.elevenlabsHandler == nil || h.gcpHandler == nil || h.awsHandler == nil {
		t.Error("Wrong match. expected: all three vendor handlers to be set")
	}
}
