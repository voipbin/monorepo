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

const testSipGatewayFQDNForPSTN = "pstn.voipbin.net:5060"

func newTestProviderHandler(ctrl *gomock.Controller) (*providerHandler, *telnyxclient.MockTelnyxClient, *dbhandler.MockDBHandler, *notifyhandler.MockNotifyHandler) {
	mockClient := telnyxclient.NewMockTelnyxClient(ctrl)
	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	h := &providerHandler{db: mockDB, notifyHandler: mockNotify, sipGatewayFQDNForPSTN: testSipGatewayFQDNForPSTN}
	return h, mockClient, mockDB, mockNotify
}

func Test_Setup_UnknownCarrier(t *testing.T) {
	h := &providerHandler{sipGatewayFQDNForPSTN: testSipGatewayFQDNForPSTN}
	_, err := h.setupWithClient(context.Background(), "vonage", "name", "detail",
		telnyxclient.NewTelnyxClient("key"))
	if err == nil || err.Error() != "unsupported carrier: vonage" {
		t.Fatalf("expected unsupported carrier error, got %v", err)
	}
}

func Test_Setup_NoFQDNConfigured(t *testing.T) {
	h := &providerHandler{sipGatewayFQDNForPSTN: ""}
	_, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail",
		telnyxclient.NewTelnyxClient("key"))
	if err == nil {
		t.Fatal("expected error for empty sipGatewayFQDNForPSTN, got nil")
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

func Test_Setup_CreateFQDNConnectionFails_CleansUpProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h, mockClient, _, _ := newTestProviderHandler(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateOutboundVoiceProfile(gomock.Any(), "name").Return("profile-123", nil)
	mockClient.EXPECT().CreateFQDNConnection(gomock.Any(), "name", "profile-123", gomock.Any(), gomock.Any()).Return("", errors.New("telnyx down"))
	mockClient.EXPECT().DeleteOutboundVoiceProfile(gomock.Any(), "profile-123").Return(nil)

	_, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_Setup_RegisterFQDNFails_CleansUpConnectionAndProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h, mockClient, _, _ := newTestProviderHandler(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateOutboundVoiceProfile(gomock.Any(), "name").Return("profile-123", nil)
	mockClient.EXPECT().CreateFQDNConnection(gomock.Any(), "name", "profile-123", gomock.Any(), gomock.Any()).Return("conn-456", nil)
	mockClient.EXPECT().RegisterFQDN(gomock.Any(), "conn-456", "pstn.voipbin.net", 5060).Return("", errors.New("fqdn conflict"))
	mockClient.EXPECT().DeleteFQDNConnection(gomock.Any(), "conn-456").Return(nil)
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
	mockClient.EXPECT().CreateFQDNConnection(gomock.Any(), "name", "profile-123", gomock.Any(), gomock.Any()).Return("conn-456", nil)
	mockClient.EXPECT().RegisterFQDN(gomock.Any(), "conn-456", "pstn.voipbin.net", 5060).Return("fqdn-789", nil)
	mockDB.EXPECT().ProviderCreate(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
	mockClient.EXPECT().DeleteFQDN(gomock.Any(), "fqdn-789").Return(nil)
	mockClient.EXPECT().DeleteFQDNConnection(gomock.Any(), "conn-456").Return(nil)
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
	mockClient.EXPECT().CreateFQDNConnection(gomock.Any(), "name", "profile-123", gomock.Any(), gomock.Any()).Return("conn-456", nil)
	mockClient.EXPECT().RegisterFQDN(gomock.Any(), "conn-456", "pstn.voipbin.net", 5060).Return("fqdn-789", nil)
	mockDB.EXPECT().ProviderCreate(gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().ProviderGet(gomock.Any(), gomock.Any()).Return(created, nil)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any())
	mockDB.EXPECT().ProviderUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().ProviderGet(gomock.Any(), gomock.Any()).Return(created, nil)

	res, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if res.Hostname != "sip.telnyx.com" {
		t.Fatalf("expected sip.telnyx.com, got %s", res.Hostname)
	}
}

// Test_Setup_MetadataUpdateFails_StillSucceeds verifies that a failure to persist
// Telnyx resource IDs as metadata is non-fatal: the setup still returns the
// pre-update provider record without issuing a second ProviderGet.
func Test_Setup_MetadataUpdateFails_StillSucceeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	created := &provider.Provider{Hostname: "sip.telnyx.com", Name: "name"}

	h, mockClient, mockDB, mockNotify := newTestProviderHandler(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateOutboundVoiceProfile(gomock.Any(), "name").Return("profile-123", nil)
	mockClient.EXPECT().CreateFQDNConnection(gomock.Any(), "name", "profile-123", gomock.Any(), gomock.Any()).Return("conn-456", nil)
	mockClient.EXPECT().RegisterFQDN(gomock.Any(), "conn-456", "pstn.voipbin.net", 5060).Return("fqdn-789", nil)
	mockDB.EXPECT().ProviderCreate(gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().ProviderGet(gomock.Any(), gomock.Any()).Return(created, nil)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any())
	// Metadata update fails; no follow-up ProviderGet should be issued.
	mockDB.EXPECT().ProviderUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("db error"))

	res, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil provider, got nil")
	}
	if res.Hostname != "sip.telnyx.com" {
		t.Fatalf("expected sip.telnyx.com, got %s", res.Hostname)
	}
}

// Test_Setup_MetadataRefetchFails_StillSucceeds verifies that a failure to
// re-fetch the provider after a successful metadata update is non-fatal:
// setup still returns the pre-update provider record.
func Test_Setup_MetadataRefetchFails_StillSucceeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	created := &provider.Provider{Hostname: "sip.telnyx.com", Name: "name"}

	h, mockClient, mockDB, mockNotify := newTestProviderHandler(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateOutboundVoiceProfile(gomock.Any(), "name").Return("profile-123", nil)
	mockClient.EXPECT().CreateFQDNConnection(gomock.Any(), "name", "profile-123", gomock.Any(), gomock.Any()).Return("conn-456", nil)
	mockClient.EXPECT().RegisterFQDN(gomock.Any(), "conn-456", "pstn.voipbin.net", 5060).Return("fqdn-789", nil)
	mockDB.EXPECT().ProviderCreate(gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().ProviderGet(gomock.Any(), gomock.Any()).Return(created, nil)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any())
	mockDB.EXPECT().ProviderUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	// Re-fetch fails — setup should return the pre-update record anyway.
	mockDB.EXPECT().ProviderGet(gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))

	res, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil provider, got nil")
	}
	if res.Hostname != "sip.telnyx.com" {
		t.Fatalf("expected sip.telnyx.com, got %s", res.Hostname)
	}
}

func Test_Setup_InvalidFQDNFormat_CleansUpConnectionAndProfile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := &providerHandler{
		db:                    dbhandler.NewMockDBHandler(ctrl),
		notifyHandler:         notifyhandler.NewMockNotifyHandler(ctrl),
		sipGatewayFQDNForPSTN: "pstn.voipbin.net", // missing :port
	}
	mockClient := telnyxclient.NewMockTelnyxClient(ctrl)

	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateOutboundVoiceProfile(gomock.Any(), "name").Return("profile-123", nil)
	mockClient.EXPECT().CreateFQDNConnection(gomock.Any(), "name", "profile-123", gomock.Any(), gomock.Any()).Return("conn-456", nil)
	mockClient.EXPECT().DeleteFQDNConnection(gomock.Any(), "conn-456").Return(nil)
	mockClient.EXPECT().DeleteOutboundVoiceProfile(gomock.Any(), "profile-123").Return(nil)

	_, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_GenerateCredentialSecret_AlphanumericOnly(t *testing.T) {
	secret, err := generateCredentialSecret(32)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if len(secret) != 32 {
		t.Fatalf("expected length 32, got %d", len(secret))
	}
	for _, c := range secret {
		isAlnum := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !isAlnum {
			t.Fatalf("expected alphanumeric only, got %q in %q", c, secret)
		}
	}
}

func Test_SanitizeCredentialUserName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"simple", "simple"},
		{"my-provider-name", "myprovidername"},
		{"my provider name", "myprovidername"},
		{"Provider_123!", "Provider123"},
	}
	for _, c := range cases {
		got := sanitizeCredentialUserName(c.in)
		if got != c.want {
			t.Errorf("sanitizeCredentialUserName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func Test_BuildCredentialUserName_NeverExceedsTelnyxLimit(t *testing.T) {
	cases := []string{
		"short",
		"a-very-long-provider-name-that-goes-on-and-on-and-on-and-on-and-on",
		"",
		"!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!",
	}
	for _, name := range cases {
		got := buildCredentialUserName(name)
		if len(got) > telnyxUserNameMaxLen {
			t.Errorf("buildCredentialUserName(%q) = %q (len %d), exceeds telnyx limit %d", name, got, len(got), telnyxUserNameMaxLen)
		}
		if got[:len(credentialUserNamePrefix)] != credentialUserNamePrefix {
			t.Errorf("buildCredentialUserName(%q) = %q, expected prefix %q", name, got, credentialUserNamePrefix)
		}
	}
}
