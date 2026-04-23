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

func Test_Setup_UnknownCarrier(t *testing.T) {
	h := &providerHandler{}
	_, err := h.setupWithClient(context.Background(), "vonage", "name", "detail",
		telnyxclient.NewTelnyxClient("key"))
	if err == nil || err.Error() != "unsupported carrier: vonage" {
		t.Fatalf("expected unsupported carrier error, got %v", err)
	}
}

func Test_Setup_InvalidKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := telnyxclient.NewMockTelnyxClient(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(telnyxclient.ErrInvalidKey)

	h := &providerHandler{}
	_, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if !errors.Is(err, telnyxclient.ErrInvalidKey) {
		t.Fatalf("expected ErrInvalidKey, got %v", err)
	}
}

func Test_Setup_CreateConnectionFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := telnyxclient.NewMockTelnyxClient(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateCredentialConnection(gomock.Any(), "name").Return("", errors.New("telnyx down"))

	h := &providerHandler{}
	_, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_Setup_ProviderCreateFails_TriggersCleanup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := telnyxclient.NewMockTelnyxClient(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateCredentialConnection(gomock.Any(), "name").Return("conn-123", nil)
	mockClient.EXPECT().DeleteCredentialConnection(gomock.Any(), "conn-123").Return(nil)

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockDB.EXPECT().ProviderCreate(gomock.Any(), gomock.Any()).Return(errors.New("db error"))

	h := &providerHandler{db: mockDB}
	_, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func Test_Setup_ProviderCreateFails_CleanupAlsoFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := telnyxclient.NewMockTelnyxClient(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateCredentialConnection(gomock.Any(), "name").Return("conn-456", nil)
	mockClient.EXPECT().DeleteCredentialConnection(gomock.Any(), "conn-456").Return(errors.New("cleanup failed"))

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockDB.EXPECT().ProviderCreate(gomock.Any(), gomock.Any()).Return(errors.New("db error"))

	h := &providerHandler{db: mockDB}
	_, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err == nil {
		t.Fatal("expected error even when cleanup fails, got nil")
	}
}

func Test_Setup_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	created := &provider.Provider{Hostname: "sip.telnyx.com", Name: "name"}

	mockClient := telnyxclient.NewMockTelnyxClient(ctrl)
	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateCredentialConnection(gomock.Any(), "name").Return("conn-123", nil)

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockDB.EXPECT().ProviderCreate(gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().ProviderGet(gomock.Any(), gomock.Any()).Return(created, nil)

	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any())

	h := &providerHandler{db: mockDB, notifyHandler: mockNotify}
	res, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if res.Hostname != "sip.telnyx.com" {
		t.Fatalf("expected sip.telnyx.com, got %s", res.Hostname)
	}
}
