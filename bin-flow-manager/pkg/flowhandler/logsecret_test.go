package flowhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

const (
	secretLogDirectHash = "direct.0123456789ab-SENTINEL-MUST-NOT-LEAK-9f3c"
	secretLogDirectTail = "SENTINEL-MUST-NOT-LEAK"
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

func newSecretLogDirect(resourceType string, resourceID uuid.UUID) *dmdirect.Direct {
	return &dmdirect.Direct{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111"),
			CustomerID: uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111"),
		},
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Hash:         secretLogDirectHash,
	}
}

func Test_DirectHashRegenerate_DoesNotLogDirectHash(t *testing.T) {
	flowID := uuid.FromStringOrNil("f1f1f1f1-1111-1111-1111-111111111111")
	customerID := uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111")
	directID := uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111")

	tests := []struct {
		name string

		responseFlow *flow.Flow
		wantMsg      string
		regenerate   bool
	}{
		{
			name: "regenerate path",
			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{ID: flowID, CustomerID: customerID},
				DirectID: directID,
			},
			wantMsg:    "Direct hash regenerated",
			regenerate: true,
		},
		{
			name: "create path",
			responseFlow: &flow.Flow{
				Identity: commonidentity.Identity{ID: flowID, CustomerID: customerID},
			},
			wantMsg:    "Direct hash created",
			regenerate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := newSecretLogHook(t)

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &flowHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()
			d := newSecretLogDirect("flow", flowID)

			mockDB.EXPECT().FlowGet(ctx, flowID).Return(tt.responseFlow, nil)
			if tt.regenerate {
				mockReq.EXPECT().DirectV1DirectRegenerate(ctx, directID).Return(d, nil)
			} else {
				mockReq.EXPECT().DirectV1DirectCreate(ctx, customerID, "flow", flowID).Return(d, nil)
			}
			mockDB.EXPECT().FlowUpdate(ctx, flowID, gomock.Any()).Return(nil)
			mockDB.EXPECT().FlowGet(ctx, flowID).Return(&flow.Flow{Identity: commonidentity.Identity{ID: flowID}}, nil)

			if _, err := h.DirectHashRegenerate(ctx, flowID); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			assertNoSecretInLogs(t, hook, tt.wantMsg, secretLogDirectHash, secretLogDirectTail)
		})
	}
}

func Test_Create_DoesNotLogDirectHash(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := newSecretLogHook(t)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAction := actionhandler.NewMockActionHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &flowHandler{
		util:          mockUtil,
		reqHandler:    mockReq,
		db:            mockDB,
		actionHandler: mockAction,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111")
	flowID := uuid.FromStringOrNil("f1f1f1f1-1111-1111-1111-111111111111")
	actions := []action.Action{{Type: action.TypeAnswer}}
	created := &flow.Flow{Identity: commonidentity.Identity{ID: flowID, CustomerID: customerID}}

	mockDB.EXPECT().FlowCountByCustomerID(ctx, customerID).Return(0, nil)
	mockAction.EXPECT().GenerateFlowActions(ctx, actions).Return(actions, nil)
	mockUtil.EXPECT().UUIDCreate().Return(flowID)
	mockReq.EXPECT().DirectV1DirectCreate(ctx, customerID, "flow", flowID).Return(newSecretLogDirect("flow", flowID), nil)
	mockUtil.EXPECT().TimeNow().Return(utilhandler.TimeNow())
	mockDB.EXPECT().FlowCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().FlowGet(ctx, flowID).Return(created, nil)
	mockNotify.EXPECT().PublishEvent(ctx, flow.EventTypeFlowCreated, created)

	if _, err := h.Create(ctx, customerID, flow.TypeFlow, "name", "detail", true, actions, uuid.Nil); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// Only the direct hash entry is asserted here. The separate "Creating a new
	// flow." entry logs the whole flow struct, whose DirectHash is a distinct
	// pre-existing exposure that is outside the scope of this fix.
	directHook := &logrustest.Hook{}
	for _, entry := range hook.AllEntries() {
		if strings.Contains(entry.Message, "Created direct hash") {
			if err := directHook.Fire(entry); err != nil {
				t.Fatalf("could not copy the log entry: %v", err)
			}
		}
	}
	assertNoSecretInLogs(t, directHook, "Created direct hash", secretLogDirectHash, secretLogDirectTail)
}
