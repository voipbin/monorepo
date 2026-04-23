package providerhandler

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/dbhandler"
	"monorepo/bin-route-manager/pkg/telnyxclient"
)

const testSipLBIP = "10.0.0.1"
const testSipLBPort = 5060

func newTestProviderHandler(ctrl *gomock.Controller) (*providerHandler, *telnyxclient.MockTelnyxClient, *dbhandler.MockDBHandler, *notifyhandler.MockNotifyHandler) {
	mockClient := telnyxclient.NewMockTelnyxClient(ctrl)
	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	h := &providerHandler{db: mockDB, notifyHandler: mockNotify, sipLBIP: testSipLBIP, sipLBPort: testSipLBPort}
	return h, mockClient, mockDB, mockNotify
}

func Test_Setup_UnknownCarrier(t *testing.T) {
	h := &providerHandler{sipLBIP: testSipLBIP, sipLBPort: testSipLBPort}
	_, err := h.setupWithClient(context.Background(), "vonage", "name", "detail",
		telnyxclient.NewTelnyxClient("key"))
	if err == nil || err.Error() != "unsupported carrier: vonage" {
		t.Fatalf("expected unsupported carrier error, got %v", err)
	}
}

func Test_Setup_InvalidKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h, mockClient, _, _ := newTestProviderHandler(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(telnyxclient.ErrInvalidKey)

	_, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if !errors.Is(err, telnyxclient.ErrInvalidKey) {
		t.Fatalf("expected ErrInvalidKey, got %v", err)
	}
}

func Test_Setup_CreateVoiceProfileFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h, mockClient, _, _ := newTestProviderHandler(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateOutboundVoiceProfile(gomock.Any(), "name").Return("", errors.New("telnyx down"))

	_, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_Setup_CreateIPConnectionFails_CleansUpProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h, mockClient, _, _ := newTestProviderHandler(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateOutboundVoiceProfile(gomock.Any(), "name").Return("profile-123", nil)
	mockClient.EXPECT().CreateIPConnection(gomock.Any(), "name", "profile-123").Return("", errors.New("telnyx down"))
	mockClient.EXPECT().DeleteOutboundVoiceProfile(gomock.Any(), "profile-123").Return(nil)

	_, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_Setup_RegisterIPFails_CleansUpConnectionAndProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h, mockClient, _, _ := newTestProviderHandler(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateOutboundVoiceProfile(gomock.Any(), "name").Return("profile-123", nil)
	mockClient.EXPECT().CreateIPConnection(gomock.Any(), "name", "profile-123").Return("conn-456", nil)
	mockClient.EXPECT().RegisterIP(gomock.Any(), "conn-456", testSipLBIP, testSipLBPort).Return("", errors.New("ip conflict"))
	mockClient.EXPECT().DeleteIPConnection(gomock.Any(), "conn-456").Return(nil)
	mockClient.EXPECT().DeleteOutboundVoiceProfile(gomock.Any(), "profile-123").Return(nil)

	_, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_Setup_ProviderCreateFails_CleansUpAll(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h, mockClient, mockDB, _ := newTestProviderHandler(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateOutboundVoiceProfile(gomock.Any(), "name").Return("profile-123", nil)
	mockClient.EXPECT().CreateIPConnection(gomock.Any(), "name", "profile-123").Return("conn-456", nil)
	mockClient.EXPECT().RegisterIP(gomock.Any(), "conn-456", testSipLBIP, testSipLBPort).Return("ip-789", nil)
	mockDB.EXPECT().ProviderCreate(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
	mockClient.EXPECT().DeleteIP(gomock.Any(), "ip-789").Return(nil)
	mockClient.EXPECT().DeleteIPConnection(gomock.Any(), "conn-456").Return(nil)
	mockClient.EXPECT().DeleteOutboundVoiceProfile(gomock.Any(), "profile-123").Return(nil)

	_, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_Setup_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	created := &provider.Provider{Hostname: "sip.telnyx.com", Name: "name"}

	h, mockClient, mockDB, mockNotify := newTestProviderHandler(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateOutboundVoiceProfile(gomock.Any(), "name").Return("profile-123", nil)
	mockClient.EXPECT().CreateIPConnection(gomock.Any(), "name", "profile-123").Return("conn-456", nil)
	mockClient.EXPECT().RegisterIP(gomock.Any(), "conn-456", testSipLBIP, testSipLBPort).Return("ip-789", nil)
	mockDB.EXPECT().ProviderCreate(gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().ProviderGet(gomock.Any(), gomock.Any()).Return(created, nil)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any())

	res, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if res.Hostname != "sip.telnyx.com" {
		t.Fatalf("expected sip.telnyx.com, got %s", res.Hostname)
	}
}
