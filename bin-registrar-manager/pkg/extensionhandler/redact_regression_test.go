package extensionhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	bmaccount "monorepo/bin-billing-manager/models/account"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/common"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

// assertNoLeak fails the test if any captured log entry (message or any
// field, formatted the same way logrus would emit it) contains needle.
// This is a regression guard for VOIP-1287: several rounds of review found
// plaintext SIP passwords / direct-hashes leaking into logrus fields even
// after the transport-layer (pkg/listenhandler) redaction was in place, so
// this asserts the business layer never emits the secret at all, rather
// than trusting each call site was updated correctly by inspection.
func assertNoLeak(t *testing.T, entries []*logrus.Entry, needle string) {
	t.Helper()

	for _, entry := range entries {
		if strings.Contains(entry.Message, needle) {
			t.Errorf("Secret leaked into log message: %q", entry.Message)
		}
		for k, v := range entry.Data {
			if strings.Contains(fmt.Sprintf("%v", v), needle) {
				t.Errorf("Secret leaked into log field %q: %v", k, v)
			}
		}
	}
}

func Test_Create_neverLogsPlaintextPassword(t *testing.T) {
	hook := logrustest.NewLocal(logrus.StandardLogger())
	defer func() { logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) }()

	const plaintextPassword = "S3cr3tSipPassw0rd-must-not-leak"

	customerID := uuid.FromStringOrNil("0040713e-7fed-11ec-954b-ff6d17e2a264")
	extID := uuid.FromStringOrNil("b2fce137-6ece-4259-8480-473b6c1f2dee")
	directID := uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &extensionHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		dbAst:         mockDBAst,
		dbBin:         mockDBBin,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	defer common.ResetBaseDomainNamesForTest()
	if errSet := common.SetBaseDomainNames("registrar.voipbin.net", "trunk.voipbin.net"); errSet != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errSet)
	}

	responseDirect := &dmdirect.Direct{
		Identity:     commonidentity.Identity{ID: directID, CustomerID: customerID},
		ResourceType: "extension",
		ResourceID:   extID,
		Hash:         "abc123def456-must-not-leak",
	}
	responseExtension := &extension.Extension{
		Identity:   commonidentity.Identity{ID: extID},
		Extension:  "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4",
		Realm:      "0040713e-7fed-11ec-954b-ff6d17e2a264.registrar.voipbin.net",
		Username:   "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4",
		Password:   plaintextPassword,
		DirectID:   directID,
		DirectHash: responseDirect.Hash,
	}

	mockReq.EXPECT().BillingV1AccountIsValidResourceLimitByCustomerID(ctx, customerID, bmaccount.ResourceTypeExtension).Return(true, nil)
	mockDBAst.EXPECT().AstAORCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstAuthCreate(ctx, gomock.Any()).Return(nil)
	mockDBAst.EXPECT().AstEndpointCreate(ctx, gomock.Any()).Return(nil)
	mockUtil.EXPECT().UUIDCreate().Return(extID)
	mockReq.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeExtension, extID).Return(responseDirect, nil)
	mockDBBin.EXPECT().ExtensionCreate(ctx, gomock.Any()).Return(nil)
	mockDBBin.EXPECT().ExtensionGet(ctx, extID).Return(responseExtension, nil)
	mockDBBin.EXPECT().SIPAuthCreate(ctx, gomock.Any()).Return(nil)
	mockNotify.EXPECT().PublishEvent(ctx, extension.EventTypeExtensionCreated, responseExtension)

	if _, err := h.Create(ctx, customerID, "test name", "test detail", "ce4f2a40-6ec1-11eb-a84c-2bb788ac26e4", plaintextPassword); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	assertNoLeak(t, hook.AllEntries(), plaintextPassword)
}

func Test_Update_neverLogsPlaintextPassword(t *testing.T) {
	hook := logrustest.NewLocal(logrus.StandardLogger())
	defer func() { logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) }()

	const plaintextPassword = "an0ther-S3cr3t-must-not-leak"

	id := uuid.FromStringOrNil("66f6b86c-6f44-11eb-ab55-934942c23f91")
	fields := map[extension.Field]any{
		extension.FieldName:     "update name",
		extension.FieldPassword: plaintextPassword,
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBAst := dbhandler.NewMockDBHandler(mc)
	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &extensionHandler{
		dbAst:         mockDBAst,
		dbBin:         mockDBBin,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	before := &extension.Extension{
		Identity: commonidentity.Identity{ID: id},
		AuthID:   "66f6b86c-6f44-11eb-ab55-934942c23f91@test.registrar.voipbin.net",
	}
	after := &extension.Extension{
		Identity: commonidentity.Identity{ID: id},
		AuthID:   before.AuthID,
		Password: plaintextPassword,
	}

	mockDBBin.EXPECT().ExtensionGet(gomock.Any(), id).Return(before, nil)
	mockDBAst.EXPECT().AstAuthUpdate(gomock.Any(), gomock.Any()).Return(nil)
	mockDBBin.EXPECT().ExtensionUpdate(gomock.Any(), id, fields).Return(nil)
	mockDBBin.EXPECT().ExtensionGet(gomock.Any(), id).Return(after, nil)
	mockDBBin.EXPECT().SIPAuthUpdate(ctx, id, gomock.Any()).Return(nil)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), extension.EventTypeExtensionUpdated, after)

	if _, err := h.Update(ctx, id, fields); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	assertNoLeak(t, hook.AllEntries(), plaintextPassword)
}
