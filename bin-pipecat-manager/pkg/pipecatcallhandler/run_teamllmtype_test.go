package pipecatcallhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	amteam "monorepo/bin-ai-manager/models/team"
	aitool "monorepo/bin-ai-manager/models/tool"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	gomock "go.uber.org/mock/gomock"
)

func Test_resolveTeamForPython_llmType(t *testing.T) {
	teamID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("cccccccc-0000-0000-0000-000000000009")
	member1ID := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000001")
	member2ID := uuid.FromStringOrNil("bbbbbbbb-0000-0000-0000-000000000002")
	ai1ID := uuid.FromStringOrNil("dddddddd-0000-0000-0000-000000000001")
	ai2ID := uuid.FromStringOrNil("dddddddd-0000-0000-0000-000000000002")

	const secretKey = "SECRET-CUSTOMER-KEY-123"

	tests := []struct {
		name string

		model1 amai.EngineModel // current member
		model2 amai.EngineModel // non-current member
		key1   string           // stored engine key override for member 1 (default secretKey)
		key2   string           // stored engine key override for member 2 (default secretKey, "keep-empty" = "")

		expect1LLMType string
		expect1Key     string
		expect2LLMType string
		expect2Key     string
		expectErrorLog []uuid.UUID // member ids that must be named in an error log
	}{
		{
			name:   "openrouter member gets slug type, blank key, customer model id",
			model1: "anthropic.claude-haiku-4.5", model2: "openai.gpt-5",
			expect1LLMType: "platform_openrouter.anthropic/claude-haiku-4.5", expect1Key: "",
			expect2LLMType: "openai.gpt-5", expect2Key: secretKey,
		},
		{
			name:   "direct member is unchanged and passthrough keeps the key",
			model1: "openai.gpt-5", model2: "openai.gpt-4o",
			expect1LLMType: "openai.gpt-5", expect1Key: secretKey,
			expect2LLMType: "openai.gpt-4o", expect2Key: secretKey,
		},
		{
			name:   "custom member keeps its model id and gets the trimmed customer key",
			model1: "openrouter.vendor/model-a", model2: "openai.gpt-5",
			key1:           "  dummy-typed-key  ",
			expect1LLMType: "openrouter.vendor/model-a", expect1Key: "dummy-typed-key",
			expect2LLMType: "openai.gpt-5", expect2Key: secretKey,
		},
		{
			name:   "custom member with an empty key is rejected with an operator error log",
			model1: "openai.gpt-5", model2: "openrouter.vendor/model-a",
			key2:           "keep-empty",
			expect1LLMType: "openai.gpt-5", expect1Key: secretKey,
			expect2LLMType: "", expect2Key: "",
			expectErrorLog: []uuid.UUID{member2ID},
		},
		{
			name:   "mixed team of direct, openrouter catalog and custom members",
			model1: "anthropic.claude-haiku-4.5", model2: "openrouter.vendor/model-b",
			expect1LLMType: "platform_openrouter.anthropic/claude-haiku-4.5", expect1Key: "",
			expect2LLMType: "openrouter.vendor/model-b", expect2Key: secretKey,
		},
		{
			name:   "rejected non-current member gets empty llm_type and an operator error log",
			model1: "openai.gpt-5", model2: "openrouter.openrouter/auto",
			expect1LLMType: "openai.gpt-5", expect1Key: secretKey,
			expect2LLMType: "", expect2Key: "",
			expectErrorLog: []uuid.UUID{member2ID},
		},
		{
			name:   "rejected current member gets empty llm_type",
			model1: "anthropic.claude-opus-4", model2: "meta.llama-4-maverick",
			expect1LLMType: "", expect1Key: "",
			expect2LLMType: "platform_openrouter.meta-llama/llama-4-maverick", expect2Key: "",
			expectErrorLog: []uuid.UUID{member1ID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTool := toolhandler.NewMockToolHandler(mc)
			h := &pipecatcallHandler{requestHandler: mockReq, toolHandler: mockTool}

			hook := logtest.NewGlobal()
			defer hook.Reset()
			prev := logrus.GetLevel()
			logrus.SetLevel(logrus.DebugLevel)
			defer logrus.SetLevel(prev)

			mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(&amteam.Team{
				Identity:      commonidentity.Identity{ID: teamID, CustomerID: customerID},
				StartMemberID: member1ID,
				Members: []amteam.Member{
					{ID: member1ID, Name: "one", AIID: ai1ID},
					{ID: member2ID, Name: "two", AIID: ai2ID},
				},
			}, nil)
			storedKey1, storedKey2 := secretKey, secretKey
			if tt.key1 != "" {
				storedKey1 = tt.key1
			}
			if tt.key2 == "keep-empty" {
				storedKey2 = ""
			} else if tt.key2 != "" {
				storedKey2 = tt.key2
			}
			mockReq.EXPECT().AIV1AIGet(gomock.Any(), ai1ID).Return(&amai.AI{
				Identity: commonidentity.Identity{ID: ai1ID, CustomerID: customerID}, EngineModel: tt.model1, EngineKey: storedKey1,
			}, nil)
			mockReq.EXPECT().AIV1AIGet(gomock.Any(), ai2ID).Return(&amai.AI{
				Identity: commonidentity.Identity{ID: ai2ID, CustomerID: customerID}, EngineModel: tt.model2, EngineKey: storedKey2,
			}, nil)
			mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return([]aitool.Tool{}).AnyTimes()

			res, err := h.resolveTeamForPython(context.Background(), &amaicall.AIcall{
				AssistanceType: amaicall.AssistanceTypeTeam,
				AssistanceID:   teamID,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			m1, m2 := res.Members[0].AI, res.Members[1].AI
			if m1.LLMType != tt.expect1LLMType || m1.EngineKey != tt.expect1Key {
				t.Errorf("member1 got (%q, %q), want (%q, %q)", m1.LLMType, m1.EngineKey, tt.expect1LLMType, tt.expect1Key)
			}
			if m2.LLMType != tt.expect2LLMType || m2.EngineKey != tt.expect2Key {
				t.Errorf("member2 got (%q, %q), want (%q, %q)", m2.LLMType, m2.EngineKey, tt.expect2LLMType, tt.expect2Key)
			}
			// EngineModel stays the customer-facing id (feeds member_switched); never the slug.
			if m1.EngineModel != string(tt.model1) || m2.EngineModel != string(tt.model2) {
				t.Errorf("engine_model must stay the customer id. got (%q, %q)", m1.EngineModel, m2.EngineModel)
			}
			if strings.Contains(m1.EngineModel+m2.EngineModel, "platform_openrouter") {
				t.Errorf("slug leaked into engine_model")
			}

			var errorLogs []*logrus.Entry
			for _, e := range hook.AllEntries() {
				if strings.Contains(e.Message, secretKey) {
					t.Errorf("log leaks the engine key: %s", e.Message)
				}
				for _, v := range e.Data {
					if strings.Contains(fmt.Sprint(v), secretKey) {
						t.Errorf("log field leaks the engine key")
					}
				}
				if e.Level == logrus.ErrorLevel {
					errorLogs = append(errorLogs, e)
				}
			}
			if len(tt.expectErrorLog) == 0 && len(errorLogs) != 0 {
				t.Errorf("unexpected error logs: %v", errorLogs[0].Message)
			}
			for _, memberID := range tt.expectErrorLog {
				found := false
				for _, e := range errorLogs {
					all := e.Message + fmt.Sprint(e.Data)
					if strings.Contains(all, memberID.String()) && strings.Contains(all, customerID.String()) {
						found = true
					}
				}
				if !found {
					t.Errorf("no error log naming member %s and customer %s", memberID, customerID)
				}
			}
		})
	}
}

// llm_type has no omitempty: Python must see an explicit empty value for a rejected member.
func Test_resolvedAIData_llmTypeAlwaysSerialized(t *testing.T) {
	b, err := json.Marshal(resolvedAIData{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"llm_type":""`) {
		t.Errorf("llm_type must be serialized even when empty, got: %s", b)
	}
}
