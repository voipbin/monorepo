package pipecatcallhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	aitool "monorepo/bin-ai-manager/models/tool"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"
)

func Test_runnerStartScript_errorPaths(t *testing.T) {
	aicallID := uuid.FromStringOrNil("44444444-4444-4444-4444-4444444444a1")
	assistanceID := uuid.FromStringOrNil("55555555-5555-5555-5555-5555555555a1")

	aiPC := &pipecatcall.Pipecatcall{
		Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("66666666-6666-6666-6666-6666666666a1")},
		ReferenceType: pipecatcall.ReferenceTypeAICall,
		ReferenceID:   aicallID,
	}

	t.Run("team lookup failure stops before the python runner", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockPython := NewMockPythonRunner(mc) // no Start expectation: any call fails the test
		h := &pipecatcallHandler{requestHandler: mockReq, pythonRunner: mockPython}

		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(&amaicall.AIcall{
			Identity:       commonidentity.Identity{ID: aicallID},
			AssistanceType: amaicall.AssistanceTypeTeam,
			AssistanceID:   assistanceID,
		}, nil)
		mockReq.EXPECT().AIV1TeamGet(gomock.Any(), assistanceID).Return(nil, fmt.Errorf("team lookup failed"))

		se := &pipecatcall.Session{Ctx: context.Background(), LLMRunnerType: "openai.gpt-5"}
		err := h.runnerStartScript(aiPC, se)
		if err == nil || !strings.Contains(err.Error(), "could not resolve team for python") || !strings.Contains(err.Error(), "team lookup failed") {
			t.Errorf("Wrong error: %v", err)
		}
	})

	t.Run("python runner start failure is wrapped", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockPython := NewMockPythonRunner(mc)
		mockTool := toolhandler.NewMockToolHandler(mc)
		h := &pipecatcallHandler{pythonRunner: mockPython, toolHandler: mockTool}

		callPC := &pipecatcall.Pipecatcall{
			Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("66666666-6666-6666-6666-6666666666a2")},
			ReferenceType: pipecatcall.ReferenceTypeCall,
		}
		mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return(nil)
		mockPython.EXPECT().Start(
			gomock.Any(), callPC.ID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Return(fmt.Errorf("runner unreachable"))

		se := &pipecatcall.Session{Ctx: context.Background(), LLMRunnerType: "openai.gpt-5"}
		err := h.runnerStartScript(callPC, se)
		if err == nil || !strings.Contains(err.Error(), "could not start python client") || !strings.Contains(err.Error(), "runner unreachable") {
			t.Errorf("Wrong error: %v", err)
		}
	})

	t.Run("an mcp tool named like a built-in tool is dropped", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockTool := toolhandler.NewMockToolHandler(mc)
		mockPython := NewMockPythonRunner(mc)
		h := &pipecatcallHandler{requestHandler: mockReq, toolHandler: mockTool, pythonRunner: mockPython}

		ai := &amai.AI{
			Identity:  commonidentity.Identity{ID: assistanceID},
			Type:      amai.TypeNormal,
			ToolNames: []aitool.ToolName{aitool.ToolNameAll},
		}
		mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(&amaicall.AIcall{
			Identity:       commonidentity.Identity{ID: aicallID},
			AssistanceType: amaicall.AssistanceTypeAI,
			AssistanceID:   assistanceID,
		}, nil)
		mockReq.EXPECT().AIV1AIGet(gomock.Any(), assistanceID).Return(ai, nil)
		mockTool.EXPECT().GetByNames(ai.Type, ai.ToolNames).Return([]aitool.Tool{{Name: "call_hangup"}})
		mockReq.EXPECT().AIV1AIcallToolList(gomock.Any(), aicallID).Return([]aitool.Tool{{Name: "call_hangup"}, {Name: "mcp_deadbeef_do_thing"}}, nil)

		var got []aitool.Tool
		mockPython.EXPECT().Start(
			gomock.Any(), aiPC.ID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(func(
			_ context.Context, _ uuid.UUID, _ string, _ string, _ []map[string]any, _ string, _ string, _ string, _ string, _ string,
			tools []aitool.Tool, _ *resolvedTeamData, _ *amai.VADConfig, _ bool, _ uuid.UUID, _ string, _ string,
		) error {
			got = tools
			return nil
		})

		se := &pipecatcall.Session{Ctx: context.Background(), LLMRunnerType: "openai.gpt-5"}
		if err := h.runnerStartScript(aiPC, se); err != nil {
			t.Fatalf("Wrong error: %v", err)
		}
		if len(got) != 2 || got[0].Name != "call_hangup" || got[1].Name != "mcp_deadbeef_do_thing" {
			t.Errorf("Wrong tools. expect: the built-in tool once and the mcp tool, got: %+v", got)
		}
	})
}

func Test_runnerWebsocketHandleAudio_errorPaths(t *testing.T) {
	newReadySession := func(ctx context.Context, conn *websocket.Conn) *pipecatcall.Session {
		ready := make(chan struct{})
		close(ready)
		return &pipecatcall.Session{Ctx: ctx, ConnAst: conn, ConnAstReady: ready}
	}

	t.Run("resample failure is returned and nothing is written", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockAudio := NewMockAudiosocketHandler(mc)
		mockWS := NewMockWebsocketHandler(mc) // no WriteMessage expectation
		h := &pipecatcallHandler{audiosocketHandler: mockAudio, websocketHandler: mockWS}

		data := []byte{0x01}
		mockAudio.EXPECT().GetDataSamples(8000, data).Return(nil, fmt.Errorf("resample failed"))

		err := h.runnerWebsocketHandleAudio(newReadySession(context.Background(), &websocket.Conn{}), 8000, 1, data)
		if err == nil || !strings.Contains(err.Error(), "could not resample audio data") || !strings.Contains(err.Error(), "resample failed") {
			t.Errorf("Wrong error: %v", err)
		}
	})

	t.Run("a session that ends before the connection is ready drops the audio", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockWS := NewMockWebsocketHandler(mc) // no WriteMessage expectation
		h := &pipecatcallHandler{websocketHandler: mockWS}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		se := &pipecatcall.Session{Ctx: ctx, ConnAstReady: make(chan struct{})}

		if err := h.runnerWebsocketHandleAudio(se, 16000, 1, []byte{0x01, 0x02}); err != nil {
			t.Errorf("Wrong error. expect: nil, got: %v", err)
		}
	})

	t.Run("write failure is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatcallHandler{websocketHandler: mockWS}

		conn := &websocket.Conn{}
		data := []byte{0x01, 0x02}
		mockWS.EXPECT().WriteMessage(conn, websocket.BinaryMessage, data).Return(fmt.Errorf("write failed"))

		err := h.runnerWebsocketHandleAudio(newReadySession(context.Background(), conn), 16000, 1, data)
		if err == nil || !strings.Contains(err.Error(), "could not write audio data to asterisk websocket") || !strings.Contains(err.Error(), "write failed") {
			t.Errorf("Wrong error: %v", err)
		}
	})
}

func Test_runLLMIntermediateFlush_contextCancelledWithBufferedTokens(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &pipecatcallHandler{notifyHandler: mockNotify}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	const tokenCount = 64
	se := &pipecatcall.Session{
		Identity:     commonidentity.Identity{ID: uuid.FromStringOrNil("a1b2c3d4-6301-4000-8000-000000000001")},
		Ctx:          ctx,
		LLMTokenChan: make(chan string, tokenCount),
		LLMStopChan:  make(chan struct{}),
		LLMDoneChan:  make(chan struct{}),
	}

	// the select may pick buffered tokens or the cancelled context first; with this many tokens it
	// reaches the cancelled context while tokens remain, and the final text holds all of them in order.
	var want strings.Builder
	for i := 0; i < tokenCount; i++ {
		token := fmt.Sprintf("t%d ", i)
		want.WriteString(token)
		se.LLMTokenChan <- token
	}

	var finalText string
	mockNotify.EXPECT().PublishEvent(gomock.Any(), message.EventTypeBotLLM, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, evt *message.Message) {
			finalText = evt.Text
		}).Times(1)

	h.runLLMIntermediateFlush(se, uuid.FromStringOrNil("a1b2c3d4-6301-4000-8000-000000000002"))

	select {
	case <-se.LLMDoneChan:
	default:
		t.Fatalf("the done channel must be closed when the flush returns")
	}
	if got := se.LLMStopReason.Load(); got != int32(StopReasonContextCancel) {
		t.Errorf("Wrong stop reason. expect: %d, got: %d", StopReasonContextCancel, got)
	}
	if finalText != want.String() {
		t.Errorf("Wrong final text.\nexpect: %q\ngot: %q", want.String(), finalText)
	}
}
