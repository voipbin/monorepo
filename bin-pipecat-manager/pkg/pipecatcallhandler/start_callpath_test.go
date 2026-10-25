package pipecatcallhandler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	aitool "monorepo/bin-ai-manager/models/tool"
	cmcall "monorepo/bin-call-manager/models/call"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// waitSessionCtxDone waits until the session context handed to a runner goroutine is cancelled.
func waitSessionCtxDone(t *testing.T, ctxCh <-chan context.Context) {
	t.Helper()

	select {
	case ctx := <-ctxCh:
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
			t.Fatalf("the session context was not cancelled")
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("the runner goroutine did not reach its last mock call")
	}
}

// expectCallRunnerGoroutine registers the calls made by the RunnerStart goroutine of a
// ReferenceTypeCall pipecatcall. The last call, the python runner start, reports the session
// context, so a test can wait for the goroutine before the controller finishes.
func expectCallRunnerGoroutine(mockTool *toolhandler.MockToolHandler, mockPython *MockPythonRunner) <-chan context.Context {
	ctxCh := make(chan context.Context, 1)

	mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockPython.EXPECT().Start(
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).DoAndReturn(func(
		ctx context.Context, _ uuid.UUID, _ string, _ string, _ []map[string]any, _ string, _ string, _ string, _ string, _ string,
		_ []aitool.Tool, _ *resolvedTeamData, _ *amai.VADConfig, _ bool, _ uuid.UUID, _ string, _ string,
	) error {
		ctxCh <- ctx
		return nil
	})

	return ctxCh
}

func Test_startReferenceTypeCall_sessionCreateFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	pcID := uuid.FromStringOrNil("a1b2c3d4-4001-4000-8000-000000000001")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-4001-4000-8000-000000000002")
	callID := uuid.FromStringOrNil("d4e5f6a7-4001-4000-8000-000000000003")

	// no other expectation: the external media start, the python runner and every other
	// collaborator are not reached after the session creation fails.
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &pipecatcallHandler{
		requestHandler:        mockReq,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	// a session with the same id already exists.
	existing := &pipecatcall.Session{Identity: commonidentity.Identity{ID: pcID}}
	h.mapPipecatcallSession[pcID] = existing

	pc := &pipecatcall.Pipecatcall{
		Identity:      commonidentity.Identity{ID: pcID},
		ReferenceType: pipecatcall.ReferenceTypeCall,
		ReferenceID:   referenceID,
		LLMType:       pipecatcall.LLMType("openai.gpt-5"),
	}
	mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).Return(&cmcall.Call{
		Identity: commonidentity.Identity{ID: callID},
	}, nil)

	err := h.startReferenceTypeCall(context.Background(), pc)
	if err == nil ||
		!strings.Contains(err.Error(), "could not create pipecatcall session") ||
		!strings.Contains(err.Error(), "session already exists") {
		t.Errorf("Wrong error: %v", err)
	}
	if got := h.mapPipecatcallSession[pcID]; got != existing {
		t.Errorf("the existing session must be left untouched")
	}
}

func Test_startReferenceTypeCall_orphanStopFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockWS := NewMockWebsocketHandler(mc)
	mockPython := NewMockPythonRunner(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)
	h := &pipecatcallHandler{
		requestHandler:        mockReq,
		websocketHandler:      mockWS,
		pythonRunner:          mockPython,
		toolHandler:           mockTool,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	pcID := uuid.FromStringOrNil("a1b2c3d4-4002-4000-8000-000000000001")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-4002-4000-8000-000000000002")
	callID := uuid.FromStringOrNil("d4e5f6a7-4002-4000-8000-000000000003")
	emID := uuid.FromStringOrNil("e5f6a7b8-4002-4000-8000-000000000004")
	mediaURI := "ws://media.example.invalid/ws/test"

	pc := &pipecatcall.Pipecatcall{
		Identity:      commonidentity.Identity{ID: pcID},
		ReferenceType: pipecatcall.ReferenceTypeCall,
		ReferenceID:   referenceID,
		LLMType:       pipecatcall.LLMType("openai.gpt-5"),
	}

	mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).Return(&cmcall.Call{Identity: commonidentity.Identity{ID: callID}}, nil)
	mockReq.EXPECT().CallV1ExternalMediaStart(
		gomock.Any(), pcID, cmexternalmedia.ReferenceTypeCall, callID, "INCOMING",
		defaultEncapsulation, defaultTransport, "", defaultConnectionType, defaultFormat,
		cmexternalmedia.DirectionIn, cmexternalmedia.DirectionOut,
	).Return(&cmexternalmedia.ExternalMedia{ID: emID, MediaURI: mediaURI}, nil)
	mockWS.EXPECT().DialContext(gomock.Any(), mediaURI, gomock.Any()).Return(nil, nil, fmt.Errorf("connection refused"))
	// the cleanup of the orphaned external media fails as well: the start error must still be the dial error.
	mockReq.EXPECT().CallV1ExternalMediaStop(gomock.Any(), emID).Return(nil, fmt.Errorf("stop failed"))
	ctxCh := expectCallRunnerGoroutine(mockTool, mockPython)

	err := h.startReferenceTypeCall(context.Background(), pc)
	if err == nil ||
		!strings.Contains(err.Error(), "asterisk setup failed") ||
		!strings.Contains(err.Error(), "could not connect to asterisk websocket") ||
		!strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "stop failed") {
		t.Errorf("Wrong error: %v", err)
	}
	if _, ok := h.mapPipecatcallSession[pcID]; ok {
		t.Errorf("the session must be deleted after the setup failure")
	}
	waitSessionCtxDone(t, ctxCh)
}

// newAIcallStartEnv builds a handler and the AIcall lookups of a pipecatcall whose ai call has
// the given reference type. The synchronous lookups of startReferenceTypeAIcall are expected
// exactly once. The RunnerStart goroutine looks the ai call up again: that second lookup fails,
// ends the goroutine and reports the session context, so the test can wait for it.
func newAIcallStartEnv(mc *gomock.Controller, referenceType amaicall.ReferenceType) (*pipecatcallHandler, *requesthandler.MockRequestHandler, *pipecatcall.Pipecatcall, uuid.UUID, <-chan context.Context) {
	pcID := uuid.FromStringOrNil("a1b2c3d4-4003-4000-8000-000000000001")
	aicallID := uuid.FromStringOrNil("b2c3d4e5-4003-4000-8000-000000000002")
	aiID := uuid.FromStringOrNil("c3d4e5f6-4003-4000-8000-000000000003")
	callID := uuid.FromStringOrNil("d4e5f6a7-4003-4000-8000-000000000004")

	ctxCh := make(chan context.Context, 1)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &pipecatcallHandler{
		requestHandler:        mockReq,
		websocketHandler:      NewMockWebsocketHandler(mc),
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	pc := &pipecatcall.Pipecatcall{
		Identity:      commonidentity.Identity{ID: pcID},
		ReferenceType: pipecatcall.ReferenceTypeAICall,
		ReferenceID:   aicallID,
		LLMType:       pipecatcall.LLMType("openai.gpt-5"),
	}

	gomock.InOrder(
		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(&amaicall.AIcall{
			Identity:       commonidentity.Identity{ID: aicallID},
			ReferenceType:  referenceType,
			ReferenceID:    callID,
			AssistanceType: amaicall.AssistanceTypeAI,
			AssistanceID:   aiID,
		}, nil),
		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).DoAndReturn(
			func(ctx context.Context, _ uuid.UUID) (*amaicall.AIcall, error) {
				select {
				case ctxCh <- ctx:
				default:
				}
				return nil, fmt.Errorf("runner lookup failed")
			},
		).AnyTimes(),
	)
	mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{
		Identity:    commonidentity.Identity{ID: aiID},
		Type:        amai.TypeNormal,
		EngineModel: amai.EngineModel("openai.gpt-5"),
		EngineKey:   "synthetic-key",
	}, nil)

	return h, mockReq, pc, callID, ctxCh
}

func Test_startReferenceTypeAIcall_callBranchSetupFailures(t *testing.T) {
	tests := []struct {
		name string

		expectMedia func(mockReq *requesthandler.MockRequestHandler, h *pipecatcallHandler, pcID uuid.UUID, callID uuid.UUID)

		expectErrContains    []string
		expectErrNotContains []string
	}{
		{
			name: "external media start fails",
			expectMedia: func(mockReq *requesthandler.MockRequestHandler, _ *pipecatcallHandler, pcID uuid.UUID, callID uuid.UUID) {
				mockReq.EXPECT().CallV1ExternalMediaStart(
					gomock.Any(), pcID, cmexternalmedia.ReferenceTypeCall, callID, "INCOMING",
					defaultEncapsulation, defaultTransport, "", defaultConnectionType, defaultFormat,
					cmexternalmedia.DirectionIn, cmexternalmedia.DirectionOut,
				).Return(nil, fmt.Errorf("media refused"))
			},
			expectErrContains: []string{"asterisk setup failed", "could not create external media", "media refused"},
		},
		{
			name: "websocket connection fails and the orphan cleanup fails too",
			expectMedia: func(mockReq *requesthandler.MockRequestHandler, h *pipecatcallHandler, pcID uuid.UUID, callID uuid.UUID) {
				emID := uuid.FromStringOrNil("e5f6a7b8-4003-4000-8000-000000000005")
				mediaURI := "ws://media.example.invalid/ws/test"
				mockReq.EXPECT().CallV1ExternalMediaStart(
					gomock.Any(), pcID, cmexternalmedia.ReferenceTypeCall, callID, "INCOMING",
					defaultEncapsulation, defaultTransport, "", defaultConnectionType, defaultFormat,
					cmexternalmedia.DirectionIn, cmexternalmedia.DirectionOut,
				).Return(&cmexternalmedia.ExternalMedia{ID: emID, MediaURI: mediaURI}, nil)
				h.websocketHandler.(*MockWebsocketHandler).EXPECT().DialContext(gomock.Any(), mediaURI, gomock.Any()).
					Return(nil, nil, fmt.Errorf("connection refused"))
				mockReq.EXPECT().CallV1ExternalMediaStop(gomock.Any(), emID).Return(nil, fmt.Errorf("stop failed"))
			},
			expectErrContains:    []string{"asterisk setup failed", "could not connect to asterisk websocket", "connection refused"},
			expectErrNotContains: []string{"stop failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockReq, pc, callID, ctxCh := newAIcallStartEnv(mc, amaicall.ReferenceTypeCall)
			tt.expectMedia(mockReq, h, pc.ID, callID)

			err := h.startReferenceTypeAIcall(context.Background(), pc)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			for _, expect := range tt.expectErrContains {
				if !strings.Contains(err.Error(), expect) {
					t.Errorf("Wrong error. expect contains: %q, got: %v", expect, err)
				}
			}
			for _, notExpect := range tt.expectErrNotContains {
				if strings.Contains(err.Error(), notExpect) {
					t.Errorf("Wrong error. must not contain: %q, got: %v", notExpect, err)
				}
			}
			if _, ok := h.mapPipecatcallSession[pc.ID]; ok {
				t.Errorf("the session must be deleted after the setup failure")
			}
			waitSessionCtxDone(t, ctxCh)
		})
	}
}
