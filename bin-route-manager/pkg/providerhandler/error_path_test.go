package providerhandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/dbhandler"
	"monorepo/bin-route-manager/pkg/telnyxclient"
)

const testPSTNGatewayHost = "gateway.example.com"

func newErrorPathHandler(mc *gomock.Controller) (*providerHandler, *dbhandler.MockDBHandler) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	return &providerHandler{db: mockDB, notifyHandler: mockNotify}, mockDB
}

func assertVoipbinError(t *testing.T, err error, status cerrors.Status, reason string) {
	t.Helper()

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("Wrong match. expect: *VoipbinError, got: %T %v", err, err)
	}
	if ve.Status != status || ve.Reason != reason || ve.Domain != string(commonoutline.ServiceNameRouteManager) {
		t.Errorf("Wrong match. got: status %v, reason %v, domain %v", ve.Status, ve.Reason, ve.Domain)
	}
}

func Test_Get_ErrNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB := newErrorPathHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000001")

	mockDB.EXPECT().ProviderGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Get(ctx, id)
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
	assertVoipbinError(t, err, cerrors.StatusNotFound, "PROVIDER_NOT_FOUND")
	if !stderrors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("Wrong match. expect: the cause ErrNotFound to stay reachable")
	}
}

func Test_Update_PreFetchNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB := newErrorPathHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000002")
	name := "new name"

	// only the pre-fetch happens: there is no ProviderUpdate expectation
	mockDB.EXPECT().ProviderGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Update(ctx, id, nil, nil, nil, nil, nil, &name, nil, nil)
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
	assertVoipbinError(t, err, cerrors.StatusNotFound, "PROVIDER_NOT_FOUND")
}

func Test_Create_InvalidCodecs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no database expectation: nothing may be written for an invalid codecs value
	h, _ := newErrorPathHandler(mc)

	res, err := h.Create(context.Background(), provider.TypeSIP, "sip.example.com", "", "", nil, "name", "detail", "ulaw,,alaw")
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
	assertVoipbinError(t, err, cerrors.StatusInvalidArgument, "INVALID_CODECS")
	if !strings.Contains(err.Error(), "double comma") {
		t.Errorf("Wrong match. expect: the validation cause in the message, got: %v", err)
	}
}

func Test_Update_InvalidCodecs(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB := newErrorPathHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("d1000000-0000-0000-0000-000000000003")
	codecs := "ulaw(8000)"

	// the current provider is read, then the update is rejected before any write
	mockDB.EXPECT().ProviderGet(ctx, id).Return(&provider.Provider{ID: id}, nil)

	res, err := h.Update(ctx, id, nil, nil, nil, nil, nil, nil, nil, &codecs)
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
	assertVoipbinError(t, err, cerrors.StatusInvalidArgument, "INVALID_CODECS")
}

func Test_Setup_PublicWrapper_UnsupportedCarrier(t *testing.T) {
	// a non-telnyx carrier returns before any Telnyx call is made
	h := &providerHandler{sipGatewayFQDNForPSTN: testPSTNGatewayHost + ":5060"}

	res, err := h.Setup(context.Background(), "vonage", "name", "detail", "api-key-placeholder")
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
	if err == nil || err.Error() != "unsupported carrier: vonage" {
		t.Errorf("Wrong match. expect: unsupported carrier: vonage, got: %v", err)
	}
}

func Test_Setup_InvalidPort_CleansUpConnectionAndProfile(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &providerHandler{
		db:                    dbhandler.NewMockDBHandler(mc),
		notifyHandler:         notifyhandler.NewMockNotifyHandler(mc),
		sipGatewayFQDNForPSTN: testPSTNGatewayHost + ":abc", // the host and port split succeeds, the port is not a number
	}
	mockClient := telnyxclient.NewMockTelnyxClient(mc)

	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateOutboundVoiceProfile(gomock.Any(), "name").Return("profile-123", nil)
	mockClient.EXPECT().CreateFQDNConnection(gomock.Any(), "name", "profile-123", gomock.Any(), gomock.Any()).Return("conn-456", nil)
	// no RegisterFQDN and no DeleteFQDN: the FQDN was never registered
	mockClient.EXPECT().DeleteFQDNConnection(gomock.Any(), "conn-456").Return(nil).Times(1)
	mockClient.EXPECT().DeleteOutboundVoiceProfile(gomock.Any(), "profile-123").Return(nil).Times(1)

	res, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
	if err == nil || !strings.Contains(err.Error(), "invalid pstn sip gateway port") {
		t.Fatalf("Wrong match. expect: invalid port error, got: %v", err)
	}
	if stderrors.Unwrap(err) == nil {
		t.Errorf("Wrong match. expect: the conversion error to be wrapped, got: %v", err)
	}
}

func Test_Setup_ProviderCreateFails_CleanupFailuresKeepOriginalError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockClient, mockDB, _ := newTestProviderHandler(mc)
	dbErr := fmt.Errorf("db error")

	mockClient.EXPECT().ValidateKey(gomock.Any()).Return(nil)
	mockClient.EXPECT().CreateOutboundVoiceProfile(gomock.Any(), "name").Return("profile-123", nil)
	mockClient.EXPECT().CreateFQDNConnection(gomock.Any(), "name", "profile-123", gomock.Any(), gomock.Any()).Return("conn-456", nil)
	mockClient.EXPECT().RegisterFQDN(gomock.Any(), "conn-456", "pstn.voipbin.net", 5060).Return("fqdn-789", nil)
	mockDB.EXPECT().ProviderCreate(gomock.Any(), gomock.Any()).Return(dbErr)

	// every cleanup step fails and the remaining ones must still run
	gomock.InOrder(
		mockClient.EXPECT().DeleteFQDN(gomock.Any(), "fqdn-789").Return(fmt.Errorf("delete fqdn failed")).Times(1),
		mockClient.EXPECT().DeleteFQDNConnection(gomock.Any(), "conn-456").Return(fmt.Errorf("delete connection failed")).Times(1),
		mockClient.EXPECT().DeleteOutboundVoiceProfile(gomock.Any(), "profile-123").Return(fmt.Errorf("delete profile failed")).Times(1),
	)

	res, err := h.setupWithClient(context.Background(), "telnyx", "name", "detail", mockClient)
	if res != nil {
		t.Errorf("Wrong match. expect: nil response, got: %v", res)
	}
	if err == nil || !strings.HasPrefix(err.Error(), "provider create failed: ") {
		t.Fatalf("Wrong match. expect: provider create failed prefix, got: %v", err)
	}
	if !stderrors.Is(err, dbErr) {
		t.Errorf("Wrong match. expect: the original db error, got: %v", err)
	}
}
