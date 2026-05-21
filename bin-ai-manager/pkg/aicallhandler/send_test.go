package aicallhandler

import (
	"context"
	"fmt"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	"monorepo/bin-ai-manager/pkg/teamhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_SendReferenceTypeOthers(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall

		messageText string

		responseUUIDPipecatcallID uuid.UUID
		responseUpdatedAIcall    *aicall.AIcall
		responseTeam             *team.Team
		responseTeamErr          error
		responseAI               *ai.AI
		responseMessages         []*message.Message
		responsePipecatcall      *pmpipecatcall.Pipecatcall
		// for fallback case: UpdateCurrentMemberID response
		responseFallbackAIcall *aicall.AIcall

		// interrupt-previous-pipecatcall expectations.
		// expectInterruptGet=true means interruptPreviousPipecatcall calls
		// PipecatV1PipecatcallGet on the previous PipecatcallID. The Get response
		// is responseInterruptPipecatcall (HostID drives ping/terminate).
		// expectInterruptPing/Terminate gate the subsequent calls.
		expectInterruptGet         bool
		expectInterruptPing        bool
		expectInterruptTerminate   bool
		responseInterruptPipecatcall *pmpipecatcall.Pipecatcall
		responseInterruptPingErr   error

		expectTeamGet              bool
		expectAIGet                bool
		expectUpdateCurrentMember  bool
		expectLLMType              pmpipecatcall.LLMType
		expectPipecatcallMessages  []map[string]any
		expectCurrentMemberIDAfter uuid.UUID
		expectRes                  *message.Message

		// updatePipecatcallIDErr — when non-nil, the AIcallUpdate call inside
		// UpdatePipecatcallID returns this error. Subsequent steps
		// (team resolve, startPipecatcall, etc.) MUST NOT run.
		updatePipecatcallIDErr error

		expectErr          bool
		expectErrSubstring string
	}{
		{
			name: "non_team_aicall_uses_existing_engine_model",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000010"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000020"),
				AIEngineModel:  ai.EngineModel("openai.gpt-5"),
				ActiveflowID:   uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000030"),
				ReferenceType:  aicall.ReferenceTypeConversation,
				ReferenceID:    uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000040"),
				PipecatcallID:  uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000050"),
			},
			messageText: "hello from user",

			responseUUIDPipecatcallID: uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000060"),
			responseUpdatedAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000010"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000020"),
				AIEngineModel:  ai.EngineModel("openai.gpt-5"),
				ActiveflowID:   uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000030"),
				ReferenceType:  aicall.ReferenceTypeConversation,
				ReferenceID:    uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000040"),
				PipecatcallID:  uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000060"),
			},
			responseMessages:    []*message.Message{},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{},

			expectInterruptGet:       true,
			expectInterruptPing:      true,
			expectInterruptTerminate: true,
			responseInterruptPipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0000001-0000-0000-0000-000000000050"),
				},
				HostID: "host-a",
			},
			responseInterruptPingErr: nil,

			expectTeamGet:             false,
			expectAIGet:               false,
			expectUpdateCurrentMember: false,
			expectLLMType:             pmpipecatcall.LLMType("openai.gpt-5"),
			expectPipecatcallMessages: []map[string]any{},
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0000001-0000-0000-0000-0000000000f0"),
				},
			},
		},
		{
			name: "team_aicall_resolves_current_member",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000010"),
				},
				AssistanceType:  aicall.AssistanceTypeTeam,
				AssistanceID:    uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000020"),
				AIEngineModel:   ai.EngineModel("openai.gpt-5"), // stale model
				ActiveflowID:    uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000030"),
				ReferenceType:   aicall.ReferenceTypeConversation,
				ReferenceID:     uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000040"),
				PipecatcallID:   uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000050"),
				CurrentMemberID: uuid.FromStringOrNil("b0000001-0000-0000-0000-0000000000a0"), // current member
			},
			messageText: "hello from user",

			responseUUIDPipecatcallID: uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000060"),
			responseUpdatedAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000010"),
				},
				AssistanceType:  aicall.AssistanceTypeTeam,
				AssistanceID:    uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000020"),
				AIEngineModel:   ai.EngineModel("openai.gpt-5"), // still stale after UpdatePipecatcallID
				ActiveflowID:    uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000030"),
				ReferenceType:   aicall.ReferenceTypeConversation,
				ReferenceID:     uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000040"),
				PipecatcallID:   uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000060"),
				CurrentMemberID: uuid.FromStringOrNil("b0000001-0000-0000-0000-0000000000a0"),
			},
			responseTeam: &team.Team{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000020"),
				},
				StartMemberID: uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000090"),
				Members: []team.Member{
					{
						ID:   uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000090"), // start member
						AIID: uuid.FromStringOrNil("b0000001-0000-0000-0000-0000000000b0"),
					},
					{
						ID:   uuid.FromStringOrNil("b0000001-0000-0000-0000-0000000000a0"), // current member
						AIID: uuid.FromStringOrNil("b0000001-0000-0000-0000-0000000000c0"),
					},
				},
			},
			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b0000001-0000-0000-0000-0000000000c0"),
				},
				EngineModel: ai.EngineModel("grok.grok-3"), // resolved model
			},
			responseMessages:    []*message.Message{},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{},

			expectInterruptGet:       true,
			expectInterruptPing:      true,
			expectInterruptTerminate: true,
			responseInterruptPipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b0000001-0000-0000-0000-000000000050"),
				},
				HostID: "host-b",
			},
			responseInterruptPingErr: nil,

			expectTeamGet:              true,
			expectAIGet:                true,
			expectUpdateCurrentMember:  false, // no fallback needed
			expectLLMType:              pmpipecatcall.LLMType("grok.grok-3"),
			expectPipecatcallMessages:  []map[string]any{},
			expectCurrentMemberIDAfter: uuid.FromStringOrNil("b0000001-0000-0000-0000-0000000000a0"),
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b0000001-0000-0000-0000-0000000000f0"),
				},
			},
		},
		{
			name: "team_aicall_fallback_to_start_member",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000010"),
				},
				AssistanceType:  aicall.AssistanceTypeTeam,
				AssistanceID:    uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000020"),
				AIEngineModel:   ai.EngineModel("openai.gpt-5"), // stale
				ActiveflowID:    uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000030"),
				ReferenceType:   aicall.ReferenceTypeConversation,
				ReferenceID:     uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000040"),
				PipecatcallID:   uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000050"),
				CurrentMemberID: uuid.FromStringOrNil("c0000001-0000-0000-0000-0000000000ff"), // non-existent member
			},
			messageText: "hello from user",

			responseUUIDPipecatcallID: uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000060"),
			responseUpdatedAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000010"),
				},
				AssistanceType:  aicall.AssistanceTypeTeam,
				AssistanceID:    uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000020"),
				AIEngineModel:   ai.EngineModel("openai.gpt-5"),
				ActiveflowID:    uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000030"),
				ReferenceType:   aicall.ReferenceTypeConversation,
				ReferenceID:     uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000040"),
				PipecatcallID:   uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000060"),
				CurrentMemberID: uuid.FromStringOrNil("c0000001-0000-0000-0000-0000000000ff"),
			},
			responseTeam: &team.Team{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000020"),
				},
				StartMemberID: uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000090"),
				Members: []team.Member{
					{
						ID:   uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000090"), // start member
						AIID: uuid.FromStringOrNil("c0000001-0000-0000-0000-0000000000b0"),
					},
				},
			},
			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c0000001-0000-0000-0000-0000000000b0"),
				},
				EngineModel: ai.EngineModel("gemini.gemini-2.5-flash"), // start member's model
			},
			responseFallbackAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000010"),
				},
				CurrentMemberID: uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000090"),
			},
			responseMessages:    []*message.Message{},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{},

			expectInterruptGet:       true,
			expectInterruptPing:      true,
			expectInterruptTerminate: true,
			responseInterruptPipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000050"),
				},
				HostID: "host-c",
			},
			responseInterruptPingErr: nil,

			expectTeamGet:              true,
			expectAIGet:                true,
			expectUpdateCurrentMember:  true,
			expectLLMType:              pmpipecatcall.LLMType("gemini.gemini-2.5-flash"),
			expectPipecatcallMessages:  []map[string]any{},
			expectCurrentMemberIDAfter: uuid.FromStringOrNil("c0000001-0000-0000-0000-000000000090"),
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c0000001-0000-0000-0000-0000000000f0"),
				},
			},
		},
		{
			name: "team_fetch_fails_uses_fallback",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000010"),
				},
				AssistanceType:  aicall.AssistanceTypeTeam,
				AssistanceID:    uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000020"),
				AIEngineModel:   ai.EngineModel("openai.gpt-5"), // will be kept as-is
				ActiveflowID:    uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000030"),
				ReferenceType:   aicall.ReferenceTypeConversation,
				ReferenceID:     uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000040"),
				PipecatcallID:   uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000050"),
				CurrentMemberID: uuid.FromStringOrNil("d0000001-0000-0000-0000-0000000000a0"),
			},
			messageText: "hello from user",

			responseUUIDPipecatcallID: uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000060"),
			responseUpdatedAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000010"),
				},
				AssistanceType:  aicall.AssistanceTypeTeam,
				AssistanceID:    uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000020"),
				AIEngineModel:   ai.EngineModel("openai.gpt-5"),
				ActiveflowID:    uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000030"),
				ReferenceType:   aicall.ReferenceTypeConversation,
				ReferenceID:     uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000040"),
				PipecatcallID:   uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000060"),
				CurrentMemberID: uuid.FromStringOrNil("d0000001-0000-0000-0000-0000000000a0"),
			},
			responseTeamErr:     fmt.Errorf("team not found"),
			responseMessages:    []*message.Message{},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{},

			expectInterruptGet:       true,
			expectInterruptPing:      true,
			expectInterruptTerminate: true,
			responseInterruptPipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d0000001-0000-0000-0000-000000000050"),
				},
				HostID: "host-d",
			},
			responseInterruptPingErr: nil,

			expectTeamGet:             true,
			expectAIGet:               false,
			expectUpdateCurrentMember: false,
			expectLLMType:             pmpipecatcall.LLMType("openai.gpt-5"), // falls back to existing
			expectPipecatcallMessages: []map[string]any{},
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d0000001-0000-0000-0000-0000000000f0"),
				},
			},
		},
		{
			name: "dead_previous_pipecat_interrupt_skipped",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000010"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000020"),
				AIEngineModel:  ai.EngineModel("openai.gpt-5"),
				ActiveflowID:   uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000030"),
				ReferenceType:  aicall.ReferenceTypeConversation,
				ReferenceID:    uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000040"),
				PipecatcallID:  uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000050"),
			},
			messageText: "hello dead pod",

			responseUUIDPipecatcallID: uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000060"),
			responseUpdatedAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("71000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000010"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000020"),
				AIEngineModel:  ai.EngineModel("openai.gpt-5"),
				ActiveflowID:   uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000030"),
				ReferenceType:  aicall.ReferenceTypeConversation,
				ReferenceID:    uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000040"),
				PipecatcallID:  uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000060"),
			},
			responseMessages:    []*message.Message{},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{},

			// dead pod: Get succeeds, Ping fails, Terminate is NOT called
			expectInterruptGet:       true,
			expectInterruptPing:      true,
			expectInterruptTerminate: false,
			responseInterruptPipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000050"),
				},
				HostID: "host-e-dead",
			},
			responseInterruptPingErr: context.DeadlineExceeded,

			expectTeamGet:             false,
			expectAIGet:               false,
			expectUpdateCurrentMember: false,
			expectLLMType:             pmpipecatcall.LLMType("openai.gpt-5"),
			expectPipecatcallMessages: []map[string]any{},
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0000001-0000-0000-0000-0000000000f0"),
				},
			},
		},
		{
			// After a successful interrupt of the previous pipecat session and
			// a successful messageHandler.Create for the user message, the
			// UpdatePipecatcallID call (db.AIcallUpdate) fails. The function
			// MUST return a wrapped error and MUST NOT call startPipecatcall.
			name: "update_pipecatcall_id_fails_after_interrupt",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a8000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("a8000001-0000-0000-0000-000000000010"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("a8000001-0000-0000-0000-000000000020"),
				AIEngineModel:  ai.EngineModel("openai.gpt-5"),
				ActiveflowID:   uuid.FromStringOrNil("a8000001-0000-0000-0000-000000000030"),
				ReferenceType:  aicall.ReferenceTypeConversation,
				ReferenceID:    uuid.FromStringOrNil("a8000001-0000-0000-0000-000000000040"),
				PipecatcallID:  uuid.FromStringOrNil("a8000001-0000-0000-0000-000000000050"),
			},
			messageText: "hello before update fails",

			responseUUIDPipecatcallID: uuid.FromStringOrNil("a8000001-0000-0000-0000-000000000060"),
			// responseUpdatedAIcall is unused (UpdatePipecatcallID fails before
			// the AIcallGet step), but is required by other table-driven mock
			// expectations that reference it. Kept minimal.
			responseUpdatedAIcall: nil,

			// interrupt succeeds
			expectInterruptGet:       true,
			expectInterruptPing:      true,
			expectInterruptTerminate: true,
			responseInterruptPipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a8000001-0000-0000-0000-000000000050"),
				},
				HostID: "host-a8",
			},
			responseInterruptPingErr: nil,

			// AIcallUpdate fails inside UpdatePipecatcallID — no AIcallGet,
			// no team resolve, no startPipecatcall, no terminate-with-delay.
			updatePipecatcallIDErr: fmt.Errorf("update failed"),
			expectTeamGet:          false,
			expectAIGet:            false,
			expectUpdateCurrentMember: false,
			// expectRes is the message returned by messageHandler.Create — that
			// runs and succeeds; the test asserts the returned error wrap and
			// nil res from SendReferenceTypeOthers.
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a8000001-0000-0000-0000-0000000000f0"),
				},
			},
			expectErr:          true,
			expectErrSubstring: "could not update the pipecatcall id for existing aicall",
		},
		{
			name: "nil_previous_pipecatcall_id_skips_interrupt",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000010"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000020"),
				AIEngineModel:  ai.EngineModel("openai.gpt-5"),
				ActiveflowID:   uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000030"),
				ReferenceType:  aicall.ReferenceTypeConversation,
				ReferenceID:    uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000040"),
				PipecatcallID:  uuid.Nil, // no previous pipecat session
			},
			messageText: "first message",

			responseUUIDPipecatcallID: uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000060"),
			responseUpdatedAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000010"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000020"),
				AIEngineModel:  ai.EngineModel("openai.gpt-5"),
				ActiveflowID:   uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000030"),
				ReferenceType:  aicall.ReferenceTypeConversation,
				ReferenceID:    uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000040"),
				PipecatcallID:  uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000060"),
			},
			responseMessages:    []*message.Message{},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{},

			// nil previous PipecatcallID: NO interrupt mocks at all (early return inside helper)
			expectInterruptGet:       false,
			expectInterruptPing:      false,
			expectInterruptTerminate: false,

			expectTeamGet:             false,
			expectAIGet:               false,
			expectUpdateCurrentMember: false,
			expectLLMType:             pmpipecatcall.LLMType("openai.gpt-5"),
			expectPipecatcallMessages: []map[string]any{},
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f0000001-0000-0000-0000-0000000000f0"),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			mockTeam := teamhandler.NewMockTeamHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				teamHandler:    mockTeam,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			// 1. messageHandler.Create
			mockMessage.EXPECT().Create(
				ctx,
				uuid.Nil,
				tt.aicall.CustomerID,
				tt.aicall.ID,
				tt.aicall.ActiveflowID,
				message.DirectionOutgoing,
				message.RoleUser,
				tt.messageText,
				nil,
				"",
				gomock.Any(),
			).Return(tt.expectRes, nil)

			// 2. interruptPreviousPipecatcall: ping-gated termination of previous pipecat session.
			//    Skipped entirely when PipecatcallID is uuid.Nil.
			if tt.expectInterruptGet {
				mockReq.EXPECT().PipecatV1PipecatcallGet(gomock.Any(), tt.aicall.PipecatcallID).Return(tt.responseInterruptPipecatcall, nil)
			}
			if tt.expectInterruptPing {
				mockReq.EXPECT().PipecatV1Ping(gomock.Any(), tt.responseInterruptPipecatcall.HostID).Return(tt.responseInterruptPingErr)
			}
			if tt.expectInterruptTerminate {
				mockReq.EXPECT().PipecatV1PipecatcallTerminate(gomock.Any(), tt.responseInterruptPipecatcall.HostID, tt.aicall.PipecatcallID).Return(nil, nil)
			}

			// 3. utilHandler.UUIDCreate for new pipecatcallID
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)

			// 4. UpdatePipecatcallID (db.AIcallUpdate + db.AIcallGet)
			//    When updatePipecatcallIDErr is non-nil, AIcallUpdate fails and
			//    no subsequent expectations (AIcallGet, team resolve, list,
			//    pipecatStart, terminate-with-delay) should be set.
			if tt.updatePipecatcallIDErr != nil {
				mockDB.EXPECT().AIcallUpdate(ctx, tt.aicall.ID, gomock.Any()).Return(tt.updatePipecatcallIDErr)
			} else {
				mockDB.EXPECT().AIcallUpdate(ctx, tt.aicall.ID, gomock.Any()).Return(nil)
				mockDB.EXPECT().AIcallGet(ctx, tt.aicall.ID).Return(tt.responseUpdatedAIcall, nil)

				// 5. team resolution (conditional)
				//    resolveActiveAIIDFromAIcall (called before messageHandler.Create) also
				//    calls teamHandler.Get for team-type aicalls, so there are TWO Get calls
				//    for team cases: one for resolveActiveAIIDFromAIcall and one for
				//    resolveTeamMemberForSend.
				if tt.expectTeamGet {
					// first call: resolveActiveAIIDFromAIcall (before messageHandler.Create)
					mockTeam.EXPECT().Get(ctx, tt.aicall.AssistanceID).Return(tt.responseTeam, tt.responseTeamErr)
					// second call: resolveTeamMemberForSend (after UpdatePipecatcallID)
					mockTeam.EXPECT().Get(ctx, tt.aicall.AssistanceID).Return(tt.responseTeam, tt.responseTeamErr)
				}

				if tt.expectAIGet {
					mockAI.EXPECT().Get(ctx, tt.responseAI.ID).Return(tt.responseAI, nil)
				}

				if tt.expectUpdateCurrentMember {
					// UpdateCurrentMemberID calls AIcallUpdate + AIcallGet
					mockDB.EXPECT().AIcallUpdate(ctx, tt.aicall.ID, gomock.Any()).Return(nil)
					mockDB.EXPECT().AIcallGet(ctx, tt.aicall.ID).Return(tt.responseFallbackAIcall, nil)
				}

				// 6. startPipecatcall: messageHandler.List for getPipecatcallMessages
				mockMessage.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(tt.responseMessages, nil)

				// 7. PipecatV1PipecatcallStart with the expected LLM type
				mockReq.EXPECT().PipecatV1PipecatcallStart(
					ctx,
					tt.responseUpdatedAIcall.PipecatcallID,
					tt.responseUpdatedAIcall.CustomerID,
					tt.responseUpdatedAIcall.ActiveflowID,
					pmpipecatcall.ReferenceTypeAICall,
					tt.responseUpdatedAIcall.ID,
					tt.expectLLMType,
					tt.expectPipecatcallMessages,
					pmpipecatcall.STTTypeNone,
					tt.responseUpdatedAIcall.STTLanguage,
					pmpipecatcall.TTSTypeNone,
					"",
					"",
				).Return(tt.responsePipecatcall, nil)

				// 8. PipecatV1PipecatcallTerminateWithDelay
				mockReq.EXPECT().PipecatV1PipecatcallTerminateWithDelay(ctx, tt.responsePipecatcall.HostID, tt.responsePipecatcall.ID, defaultAITaskTimeout).Return(nil)
			}

			res, err := h.SendReferenceTypeOthers(ctx, tt.aicall, message.RoleUser, tt.messageText)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.expectErrSubstring != "" && !strings.Contains(err.Error(), tt.expectErrSubstring) {
					t.Errorf("expected error containing %q, got: %v", tt.expectErrSubstring, err)
				}
				if res != nil {
					t.Errorf("expected nil res on error, got: %v", res)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}

			// Verify CurrentMemberID was updated in-place on the updated aicall (for team cases with fallback)
			if tt.expectCurrentMemberIDAfter != uuid.Nil && tt.responseUpdatedAIcall.CurrentMemberID != tt.expectCurrentMemberIDAfter {
				t.Errorf("expected CurrentMemberID after: %v, got: %v", tt.expectCurrentMemberIDAfter, tt.responseUpdatedAIcall.CurrentMemberID)
			}
		})
	}
}

func Test_SendReferenceTypeCall(t *testing.T) {
	tests := []struct {
		name string

		aicall      *aicall.AIcall
		messageText string

		responsePipecatcall *pmpipecatcall.Pipecatcall
		responseMessage     *message.Message

		pingErr error

		expectPingHostID    string
		expectMessageSend   bool
		expectErr           bool
		expectErrSubstring  string
		expectRes           *message.Message
	}{
		{
			name: "alive pod sends message",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000010"),
				},
				ActiveflowID:  uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000020"),
				ReferenceType: aicall.ReferenceTypeCall,
				PipecatcallID: uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000050"),
			},
			messageText: "hello pipecat",

			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0000001-0000-0000-0000-000000000050"),
				},
				HostID: "10.4.2.18",
			},
			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0000001-0000-0000-0000-0000000000f0"),
				},
			},

			pingErr: nil,

			expectPingHostID:  "10.4.2.18",
			expectMessageSend: true,
			expectErr:         false,
			expectRes: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e0000001-0000-0000-0000-0000000000f0"),
				},
			},
		},
		{
			name: "dead pod (ping deadline) returns error and skips MessageSend",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000010"),
				},
				ActiveflowID:  uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000020"),
				ReferenceType: aicall.ReferenceTypeCall,
				PipecatcallID: uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000050"),
			},
			messageText: "hello dead pod",

			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f0000001-0000-0000-0000-000000000050"),
				},
				HostID: "10.4.2.99",
			},
			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f0000001-0000-0000-0000-0000000000f0"),
				},
			},

			pingErr: context.DeadlineExceeded,

			expectPingHostID:   "10.4.2.99",
			expectMessageSend:  false,
			expectErr:          true,
			expectErrSubstring: "no longer reachable",
			expectRes:          nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			mockTeam := teamhandler.NewMockTeamHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				teamHandler:    mockTeam,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			// 1. PipecatV1PipecatcallGet
			mockReq.EXPECT().PipecatV1PipecatcallGet(ctx, tt.aicall.PipecatcallID).Return(tt.responsePipecatcall, nil)

			// 2. PipecatV1Ping preflight (always runs after Get, before Create)
			mockReq.EXPECT().PipecatV1Ping(gomock.Any(), tt.expectPingHostID).Return(tt.pingErr)

			// 3. messageHandler.Create — only called if ping succeeded; gating
			//    ping before Create avoids orphaning a user-message row when
			//    the pipecat pod is unreachable.
			if tt.expectMessageSend {
				mockMessage.EXPECT().Create(
					ctx,
					uuid.Nil,
					tt.aicall.CustomerID,
					tt.aicall.ID,
					tt.aicall.ActiveflowID,
					message.DirectionOutgoing,
					message.RoleUser,
					tt.messageText,
					nil,
					"",
					gomock.Any(),
				).Return(tt.responseMessage, nil)

				// 4. PipecatV1MessageSend — only if ping succeeded
				mockReq.EXPECT().PipecatV1MessageSend(
					ctx,
					tt.responsePipecatcall.HostID,
					tt.responsePipecatcall.ID,
					tt.responseMessage.ID.String(),
					tt.messageText,
					true,
					true,
				).Return(nil, nil)
			}

			res, err := h.SendReferenceTypeCall(ctx, tt.aicall, message.RoleUser, tt.messageText, true, true)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.expectErrSubstring != "" && !strings.Contains(err.Error(), tt.expectErrSubstring) {
					t.Errorf("expected error containing %q, got: %v", tt.expectErrSubstring, err)
				}
				if res != nil {
					t.Errorf("expected nil res on error, got: %v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
