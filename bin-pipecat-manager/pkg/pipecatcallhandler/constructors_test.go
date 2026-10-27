package pipecatcallhandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_NewPipecatcallHandler_wiresDependencies(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)

	res := NewPipecatcallHandler(mockReq, mockNotify, mockDB, mockTool, "host-a")
	h, ok := res.(*pipecatcallHandler)
	if !ok {
		t.Fatalf("Wrong type. expect: *pipecatcallHandler, got: %T", res)
	}

	if h.requestHandler != mockReq || h.notifyHandler != mockNotify || h.db != mockDB || h.toolHandler != mockTool {
		t.Errorf("Wrong match. the given collaborators must be kept")
	}
	if h.hostID != "host-a" {
		t.Errorf("Wrong host id. expect: host-a, got: %s", h.hostID)
	}
	if h.utilHandler == nil || h.pythonRunner == nil || h.audiosocketHandler == nil || h.websocketHandler == nil || h.pipecatframeHandler == nil {
		t.Errorf("Wrong match. the default collaborators must be created")
	}
	if h.mapPipecatcallSession == nil || h.terminatedPublished == nil {
		t.Errorf("Wrong match. the session and terminated maps must be created")
	}
	if _, errGet := h.SessionGet(uuid.Nil); errGet == nil {
		t.Errorf("Wrong match. a new handler must have no session")
	}
}

func Test_NewPipecatframeHandler_createsWebsocketHandler(t *testing.T) {
	h := NewPipecatframeHandler()
	if h == nil || h.websocketHandler == nil {
		t.Errorf("Wrong match. the frame handler must carry a websocket handler")
	}
}

func Test_constructors_returnConcreteHandlers(t *testing.T) {
	if _, ok := NewPythonRunner().(*pythonRunner); !ok {
		t.Errorf("Wrong type. expect: *pythonRunner")
	}
	if _, ok := NewAudiosocketHandler().(*audiosocketHandler); !ok {
		t.Errorf("Wrong type. expect: *audiosocketHandler")
	}
}
