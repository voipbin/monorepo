package agenthandler

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

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/dbhandler"

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
	agentID := uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111")
	customerID := uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111")
	directID := uuid.FromStringOrNil("d1d1d1d1-1111-1111-1111-111111111111")

	tests := []struct {
		name string

		responseAgent *agent.Agent
		wantMsg       string
		regenerate    bool
	}{
		{
			name: "regenerate path",
			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{ID: agentID, CustomerID: customerID},
				DirectID: directID,
			},
			wantMsg:    "Direct hash regenerated",
			regenerate: true,
		},
		{
			name: "create path",
			responseAgent: &agent.Agent{
				Identity: commonidentity.Identity{ID: agentID, CustomerID: customerID},
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
			h := &agentHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()
			d := newSecretLogDirect("agent", agentID)

			mockDB.EXPECT().AgentGet(ctx, agentID).Return(tt.responseAgent, nil)
			if tt.regenerate {
				mockReq.EXPECT().DirectV1DirectRegenerate(ctx, directID).Return(d, nil)
			} else {
				mockReq.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeAgent, agentID).Return(d, nil)
			}
			mockDB.EXPECT().AgentUpdate(ctx, agentID, gomock.Any()).Return(nil)
			mockDB.EXPECT().AgentGet(ctx, agentID).Return(&agent.Agent{Identity: commonidentity.Identity{ID: agentID}}, nil)

			if _, err := h.DirectHashRegenerate(ctx, agentID); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			assertNoSecretInLogs(t, hook, tt.wantMsg, secretLogDirectHash, secretLogDirectTail)
		})
	}
}

func Test_dbCreate_DoesNotLogDirectHash(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := newSecretLogHook(t)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &agentHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	customerID := uuid.FromStringOrNil("c1c1c1c1-1111-1111-1111-111111111111")
	agentID := uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111")
	created := &agent.Agent{Identity: commonidentity.Identity{ID: agentID, CustomerID: customerID}}

	mockUtil.EXPECT().HashGenerate("password", defaultPasswordHashCost).Return("hashed", nil)
	mockUtil.EXPECT().UUIDCreate().Return(agentID)
	mockReq.EXPECT().DirectV1DirectCreate(ctx, customerID, dmdirect.ResourceTypeAgent, agentID).Return(newSecretLogDirect("agent", agentID), nil)
	mockDB.EXPECT().AgentCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().AgentGet(ctx, agentID).Return(created, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, agent.EventTypeAgentCreated, created)

	if _, err := h.dbCreate(ctx, customerID, "user@example.com", "password", "name", "detail", agent.RingMethodRingAll, agent.PermissionNone, nil, nil); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	assertNoSecretInLogs(t, hook, "Created direct hash", secretLogDirectHash, secretLogDirectTail)
}
