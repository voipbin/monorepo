package streaminghandler

import (
	"context"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// isolateAWSSharedConfig points the AWS SDK's shared config and credentials
// file lookups at files inside t.TempDir() so a developer's real ~/.aws
// configuration (or lack of it) cannot influence the test, and clears the
// profile-selection env vars the SDK also reads.
func isolateAWSSharedConfig(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("AWS_CONFIG_FILE", dir+"/config")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", dir+"/credentials")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_DEFAULT_PROFILE", "")
}

// Test_awsHandler_createClient_success verifies createClient succeeds with
// static credentials and the fixed region, without any network call.
func Test_awsHandler_createClient_success(t *testing.T) {
	isolateAWSSharedConfig(t)

	h := &awsHandler{accessKey: "test-access-key", secretKey: "test-secret-key"}

	client, err := h.createClient(context.Background())
	if err != nil {
		t.Fatalf("Wrong match. expected: nil, got: %v", err)
	}
	if client == nil {
		t.Fatal("Wrong match. expected: a client, got: nil")
	}
	if client.Options().Region != defaultAWSStreamingRegion {
		t.Errorf("Wrong region. expected: %s, got: %s", defaultAWSStreamingRegion, client.Options().Region)
	}
}

// Test_awsHandler_createClient_error verifies createClient's error wrapping
// when the SDK cannot resolve a non-existent AWS_PROFILE, without any
// network call.
func Test_awsHandler_createClient_error(t *testing.T) {
	isolateAWSSharedConfig(t)
	t.Setenv("AWS_PROFILE", "does-not-exist")

	h := &awsHandler{accessKey: "test-access-key", secretKey: "test-secret-key"}

	_, err := h.createClient(context.Background())
	if err == nil || !strings.Contains(err.Error(), "failed to create AWS") {
		t.Fatalf("Wrong error. expected to mention AWS config creation, got: %v", err)
	}
}

// Test_awsHandler_Init_success verifies Init wires createClient's result and
// the resolved voice ID into the returned *AWSConfig, and publishes the
// Initiated event.
func Test_awsHandler_Init_success(t *testing.T) {
	isolateAWSSharedConfig(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	h := &awsHandler{reqHandler: mockReq, notifyHandler: mockNotify, accessKey: "test-access-key", secretKey: "test-secret-key"}

	st := &streaming.Streaming{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000002")},
		VoiceID:  "explicit-voice",
	}

	res, err := h.Init(context.Background(), st)
	if err != nil {
		t.Fatalf("Wrong match. expected: nil, got: %v", err)
	}

	cf, ok := res.(*AWSConfig)
	if !ok {
		t.Fatalf("Wrong type. expected: *AWSConfig, got: %T", res)
	}
	if cf.Client == nil {
		t.Error("Wrong match. expected: a Client, got: nil")
	}
	if cf.VoiceID != "explicit-voice" {
		t.Errorf("Wrong voice id. expected: explicit-voice, got: %s", cf.VoiceID)
	}
	cf.Cancel()
}

// Test_awsHandler_Init_createClientError verifies Init wraps a createClient
// failure instead of returning a partially built *AWSConfig.
func Test_awsHandler_Init_createClientError(t *testing.T) {
	isolateAWSSharedConfig(t)
	t.Setenv("AWS_PROFILE", "does-not-exist")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &awsHandler{reqHandler: mockReq, notifyHandler: mockNotify, accessKey: "test-access-key", secretKey: "test-secret-key"}

	st := &streaming.Streaming{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("c0000000-0000-0000-0000-000000000003")},
		VoiceID:  "explicit-voice",
	}

	res, err := h.Init(context.Background(), st)
	if err == nil || !strings.Contains(err.Error(), "failed to create AWS Polly client") {
		t.Fatalf("Wrong error. expected to mention Polly client creation, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expected: nil, got: %v", res)
	}
}
