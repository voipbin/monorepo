package aihandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

const (
	// secretLogDirectHash is longer than 12 chars and unique after the first 12 chars,
	// so neither the raw hash nor the masked form of it can hide a leak.
	secretLogDirectHash = "direct.0123456789ab-SENTINEL-MUST-NOT-LEAK-9f3c"
	secretLogDirectFrag = "SENTINEL-MUST-NOT-LEAK"
)

// newSecretLogHook attaches a hook to the standard logger and enables the debug
// level, because the log calls under test are Debug and would otherwise be
// dropped, which would make the assertion pass vacuously.
func newSecretLogHook(t *testing.T) *logrustest.Hook {
	t.Helper()

	hook := logrustest.NewLocal(logrus.StandardLogger())
	prev := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() {
		logrus.SetLevel(prev)
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	})
	return hook
}

// assertNoSecretInLogs requires at least one collected entry whose message
// contains wantMsg, and that no field of any entry carries a secret, both when
// the field is JSON-marshaled (the production formatter) and when it is printed
// with %v.
func assertNoSecretInLogs(t *testing.T, hook *logrustest.Hook, wantMsg string, secrets ...string) {
	t.Helper()

	found := 0
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Message, wantMsg) {
			found++
		}
		b, err := json.Marshal(entry.Data)
		if err != nil {
			t.Fatalf("could not marshal the log fields: %v", err)
		}
		for _, secret := range secrets {
			if strings.Contains(entry.Message, secret) {
				t.Errorf("secret leaked into the log message: %q", entry.Message)
			}
			if strings.Contains(string(b), secret) {
				t.Errorf("secret leaked into the JSON log fields of %q: %s", entry.Message, b)
			}
			if strings.Contains(fmt.Sprintf("%v", entry.Data), secret) {
				t.Errorf("secret leaked into the %%v log fields of %q", entry.Message)
			}
		}
	}
	if found == 0 {
		t.Fatalf("no log entry with message %q was collected, the assertion would be vacuous", wantMsg)
	}
}

func Test_DirectHashLog_NoSecret_Regenerate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &aiHandler{
		utilHandler:   utilhandler.NewMockUtilHandler(mc),
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
	}
	ctx := context.Background()
	hook := newSecretLogHook(t)

	id := uuid.FromStringOrNil("b3b3b3b3-3333-3333-3333-333333333333")
	customerID := uuid.FromStringOrNil("c3c3c3c3-3333-3333-3333-333333333333")
	directID := uuid.FromStringOrNil("d3d3d3d3-3333-3333-3333-333333333333")
	responseDirect := &dmdirect.Direct{
		Identity:     commonidentity.Identity{ID: directID, CustomerID: customerID},
		ResourceType: dmdirect.ResourceTypeAI,
		ResourceID:   id,
		Hash:         secretLogDirectHash,
	}

	mockDB.EXPECT().AIGet(ctx, id).Return(&ai.AI{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, DirectID: directID}, nil)
	mockReq.EXPECT().DirectV1DirectRegenerate(ctx, directID).Return(responseDirect, nil)
	mockDB.EXPECT().AIUpdate(ctx, id, gomock.Any()).Return(nil)
	mockDB.EXPECT().AIGet(ctx, id).Return(&ai.AI{Identity: commonidentity.Identity{ID: id}}, nil)

	if _, err := h.DirectHashRegenerate(ctx, id); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	assertNoSecretInLogs(t, hook, "Direct hash regenerated", secretLogDirectHash, secretLogDirectFrag)
}

func Test_DirectHashLog_NoSecret_Create(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &aiHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()
	hook := newSecretLogHook(t)

	id := uuid.FromStringOrNil("b4b4b4b4-4444-4444-4444-444444444444")
	customerID := uuid.FromStringOrNil("c4c4c4c4-4444-4444-4444-444444444444")
	responseDirect := &dmdirect.Direct{
		Identity:     commonidentity.Identity{ID: uuid.FromStringOrNil("d4d4d4d4-4444-4444-4444-444444444444"), CustomerID: customerID},
		ResourceType: dmdirect.ResourceTypeAI,
		ResourceID:   id,
		Hash:         secretLogDirectHash,
	}
	responseAI := &ai.AI{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}}

	mockUtil.EXPECT().UUIDCreate().Return(id)
	mockReq.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeAI, id).Return(responseDirect, nil)
	mockDB.EXPECT().AICreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().AIGet(ctx, id).Return(responseAI, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, ai.EventTypeCreated, responseAI)

	if _, err := h.dbCreate(ctx, customerID, "n", "d", ai.TypeNormal, ai.EngineModelOpenaiGPT5Dot1, nil, "", uuid.Nil, "", ai.TTSTypeCartesia, "v", ai.STTTypeDeepgram, "", nil, nil, false, false, uuid.Nil); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	assertNoSecretInLogs(t, hook, "Created direct hash", secretLogDirectHash, secretLogDirectFrag)
}
