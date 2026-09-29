package pipecatcallhandler

import (
	"context"
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	aitool "monorepo/bin-ai-manager/models/tool"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/testutil"
	gomock "go.uber.org/mock/gomock"
)

// Test_runnerStartScript_mcpToolList covers design step 6 / D13 (fail-open):
// runnerStartScript calls the new AIV1AIcallToolList callback RPC to
// supplement its own built-in tool resolution with the AIcall's MCP-derived
// tools. On success, the MCP tools are appended to the built-in list. On RPC
// failure, the session falls back to built-ins only (never fails closed) and
// increments metricsMcpToolListFallbackTotal.
func Test_runnerStartScript_mcpToolList(t *testing.T) {
	aicallID := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")
	assistanceID := uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")

	pc := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("66666666-6666-6666-6666-666666666666"),
		},
		ReferenceType: pipecatcall.ReferenceTypeAICall,
		ReferenceID:   aicallID,
	}
	ac := &amaicall.AIcall{
		Identity: commonidentity.Identity{
			ID: aicallID,
		},
		AssistanceType: amaicall.AssistanceTypeAI,
		AssistanceID:   assistanceID,
	}
	ai := &amai.AI{
		Identity: commonidentity.Identity{
			ID: assistanceID,
		},
		Type:      amai.TypeNormal,
		ToolNames: []aitool.ToolName{aitool.ToolNameAll},
		RagID:     uuid.Nil,
	}
	t.Run("callback success appends mcp tools", func(t *testing.T) {
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

		se := &pipecatcall.Session{Ctx: context.Background()}

		builtins := []aitool.Tool{{Name: "call_hangup"}}
		mcpTools := []aitool.Tool{{Name: "mcp_deadbeef_do_thing"}}

		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil)
		mockReq.EXPECT().AIV1AIGet(gomock.Any(), assistanceID).Return(ai, nil)
		mockTool.EXPECT().GetByNames(ai.Type, ai.ToolNames).Return(builtins)
		mockReq.EXPECT().AIV1AIcallToolList(gomock.Any(), aicallID).Return(mcpTools, nil)

		mockPython.EXPECT().Start(
			gomock.Any(), pc.ID, gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(func(
			_ context.Context, _ uuid.UUID, _ string, _ string, _ any,
			_ string, _ string, _ string, _ string, _ string,
			tools []aitool.Tool, _ any, _ any, _ bool,
		) error {
			if len(tools) != 2 {
				t.Fatalf("expected 2 tools (1 builtin + 1 mcp), got %d: %+v", len(tools), tools)
			}
			foundBuiltin, foundMcp := false, false
			for _, tl := range tools {
				if tl.Name == "call_hangup" {
					foundBuiltin = true
				}
				if tl.Name == "mcp_deadbeef_do_thing" {
					foundMcp = true
				}
			}
			if !foundBuiltin || !foundMcp {
				t.Fatalf("expected both builtin and mcp tool present, got %+v", tools)
			}
			return nil
		})

		if err := h.runnerStartScript(pc, se); err != nil {
			t.Fatalf("runnerStartScript returned unexpected error: %v", err)
		}
	})

	t.Run("callback failure falls back to builtins only", func(t *testing.T) {
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

		se := &pipecatcall.Session{Ctx: context.Background()}

		builtins := []aitool.Tool{{Name: "call_hangup"}}

		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(ac, nil)
		mockReq.EXPECT().AIV1AIGet(gomock.Any(), assistanceID).Return(ai, nil)
		mockTool.EXPECT().GetByNames(ai.Type, ai.ToolNames).Return(builtins)
		mockReq.EXPECT().AIV1AIcallToolList(gomock.Any(), aicallID).Return(nil, errors.New("rpc boom"))

		mockPython.EXPECT().Start(
			gomock.Any(), pc.ID, gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(func(
			_ context.Context, _ uuid.UUID, _ string, _ string, _ any,
			_ string, _ string, _ string, _ string, _ string,
			tools []aitool.Tool, _ any, _ any, _ bool,
		) error {
			if len(tools) != 1 || tools[0].Name != "call_hangup" {
				t.Fatalf("expected builtins-only fallback, got %+v", tools)
			}
			return nil
		})

		before := testutil.ToFloat64(metricsMcpToolListFallbackTotal)

		if err := h.runnerStartScript(pc, se); err != nil {
			t.Fatalf("runnerStartScript returned unexpected error: %v", err)
		}

		after := testutil.ToFloat64(metricsMcpToolListFallbackTotal)
		if after != before+1 {
			t.Errorf("expected metricsMcpToolListFallbackTotal to increment by 1, before=%v after=%v", before, after)
		}
	})
}
