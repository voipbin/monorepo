package streaminghandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-tts-manager/internal/ttsconfig"

	"go.uber.org/mock/gomock"
)

// Test_NewAWSHandler_setsFields verifies the constructor wires its
// dependencies and credentials into the returned handler.
func Test_NewAWSHandler_setsFields(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	res := NewAWSHandler(mockReq, mockNotify, fixedAWSCreds("access-key", "secret-key"))

	h, ok := res.(*awsHandler)
	if !ok {
		t.Fatalf("Wrong type. expected: *awsHandler, got: %T", res)
	}
	if h.reqHandler != mockReq || h.notifyHandler != mockNotify {
		t.Error("Wrong match. expected: the injected dependencies to be stored")
	}
	if a, sk, err := h.creds(context.Background()); err != nil || a != "access-key" || sk != "secret-key" {
		t.Errorf("Wrong match. expected: access-key/secret-key, got: %s/%s %v", a, sk, err)
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

	res := NewElevenlabsHandler(mockReq, mockNotify, fixedKey("api-key"))

	h, ok := res.(*elevenlabsHandler)
	if !ok {
		t.Fatalf("Wrong type. expected: *elevenlabsHandler, got: %T", res)
	}
	if h.reqHandler != mockReq || h.notifyHandler != mockNotify {
		t.Error("Wrong match. expected: the injected dependencies to be stored")
	}
	if k, err := h.keyFunc(context.Background()); err != nil || k != "api-key" {
		t.Errorf("Wrong match. expected: api-key, got: %s %v", k, err)
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

	res := NewStreamingHandler(mockReq, mockNotify, "pod-1", fixedKey("eleven-key"), fixedAWSCreds("aws-access", "aws-secret"))

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

// fixedKey returns a key supplier that always hands out the given key.
func fixedKey(key string) ttsconfig.KeyFunc {
	return func(context.Context) (string, error) { return key, nil }
}

// fixedAWSCreds returns a credentials supplier that always hands out the given pair.
func fixedAWSCreds(access, secret string) ttsconfig.AWSCredentialsFunc {
	return func(context.Context) (string, string, error) { return access, secret, nil }
}
