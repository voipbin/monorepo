package pipecatcallhandler

import (
	"context"
	"testing"

	amaicall "monorepo/bin-ai-manager/models/aicall"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/testutil"
	gomock "go.uber.org/mock/gomock"
)

// Test_runnerStartScript_toolResolveFallback verifies the fail-CLOSED policy
// (docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md
// §2.4, reversing the prior fail-open VOIP-1234 §6 v4 decision): when
// resolveAIFromAIcall fails for a non-team AICall-backed session,
// runnerStartScript now falls back to an EMPTY tool list (never GetByNames /
// GetAll) AND still increments metricsToolResolveFallbackTotal so the
// fallback stays observable.
func Test_runnerStartScript_toolResolveFallback(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)
	mockPython := NewMockPythonRunner(mc)

	h := &pipecatcallHandler{
		requestHandler: mockReq,
		toolHandler:    mockTool,
		pythonRunner:   mockPython,
	}

	aicallID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	assistanceID := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	pc := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
		},
		ReferenceType: pipecatcall.ReferenceTypeAICall,
		ReferenceID:   aicallID,
	}
	se := &pipecatcall.Session{
		Ctx: context.Background(),
	}

	ac := &amaicall.AIcall{
		Identity: commonidentity.Identity{
			ID: aicallID,
		},
		AssistanceType: amaicall.AssistanceTypeAI,
		AssistanceID:   assistanceID,
	}

	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil)
	// resolveAIFromAIcall (AssistanceTypeAI path) calls AIV1AIGet; force it to
	// fail so runnerStartScript takes the fallback branch under test.
	mockReq.EXPECT().AIV1AIGet(gomock.Any(), assistanceID).Return(nil, errors.New("boom"))

	// Fail-closed: no GetByNames/GetAll call at all -- runnerStartScript must
	// use an empty tool list directly rather than resolving any tool set.

	mockPython.EXPECT().Start(
		gomock.Any(), pc.ID, gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).Return(nil)

	before := testutil.ToFloat64(metricsToolResolveFallbackTotal)

	if err := h.runnerStartScript(pc, se); err != nil {
		t.Fatalf("runnerStartScript returned unexpected error: %v", err)
	}

	after := testutil.ToFloat64(metricsToolResolveFallbackTotal)
	if after != before+1 {
		t.Errorf("expected metricsToolResolveFallbackTotal to increment by 1, before=%v after=%v", before, after)
	}
}
