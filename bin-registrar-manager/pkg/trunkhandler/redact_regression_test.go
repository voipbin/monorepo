package trunkhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

// assertNoLeak fails the test if any captured log entry (message or any
// field, formatted the same way logrus would emit it) contains needle. See
// pkg/extensionhandler/redact_regression_test.go for the rationale: this
// guards against the same class of regression VOIP-1287 review rounds
// found in this package's Update handler.
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

func Test_Update_neverLogsPlaintextPassword(t *testing.T) {
	hook := logrustest.NewLocal(logrus.StandardLogger())
	defer func() { logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) }()

	const plaintextPassword = "trunk-S3cr3t-must-not-leak"

	id := uuid.FromStringOrNil("80a7dd20-5229-11ee-bf8c-a3fb6b428056")
	fields := map[trunk.Field]any{
		trunk.FieldName:     "update name",
		trunk.FieldPassword: plaintextPassword,
	}
	responseTrunk := &trunk.Trunk{
		Identity: commonidentity.Identity{ID: id},
		Password: plaintextPassword,
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBBin := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &trunkHandler{
		db:            mockDBBin,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	mockDBBin.EXPECT().TrunkUpdate(ctx, id, fields).Return(nil)
	mockDBBin.EXPECT().TrunkGet(ctx, id).Return(responseTrunk, nil)
	mockDBBin.EXPECT().SIPAuthUpdate(ctx, id, gomock.Any()).Return(nil)
	mockNotify.EXPECT().PublishEvent(ctx, trunk.EventTypeTrunkUpdated, responseTrunk)

	if _, err := h.Update(ctx, id, fields); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	assertNoLeak(t, hook.AllEntries(), plaintextPassword)
}
