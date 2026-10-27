package pipecatcallhandler

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/internal/logtest"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"
)

func newSessionTestHandler(mockPython PythonRunner) *pipecatcallHandler {
	return &pipecatcallHandler{
		pythonRunner:          mockPython,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
		muTerminated:          sync.Mutex{},
		terminatedPublished:   make(map[uuid.UUID]struct{}),
	}
}

func Test_SessionCreate_withAsteriskConnection(t *testing.T) {
	h := newSessionTestHandler(nil)

	pc := &pipecatcall.Pipecatcall{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a1b2c3d4-6101-4000-8000-000000000001")}}
	conn := &websocket.Conn{}
	done := make(chan struct{})

	se, err := h.SessionCreate(pc, pc.ID, conn, done, "openai.gpt-5", "synthetic-key", nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if se.ConnAst != conn || se.ConnAstDone != done {
		t.Errorf("Wrong match. the connection and the done channel must be kept")
	}

	// a session that is created with a connection is ready at once.
	select {
	case <-se.ConnAstReady:
	default:
		t.Errorf("the ready channel must be closed for a session created with a connection")
	}
}

func Test_SessionStop_connectionNotReady(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockPython := NewMockPythonRunner(mc)
	h := newSessionTestHandler(mockPython)

	id := uuid.FromStringOrNil("a1b2c3d4-6102-4000-8000-000000000001")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// the connection was never handed over: the ready channel stays open, so it is not closed.
	conn := newLoopbackConn(t)
	se := &pipecatcall.Session{
		Identity:     commonidentity.Identity{ID: id},
		Ctx:          ctx,
		Cancel:       cancel,
		ConnAst:      conn,
		ConnAstReady: make(chan struct{}),
	}
	h.mapPipecatcallSession[id] = se
	mockPython.EXPECT().Stop(gomock.Any(), id).Return(nil)

	finished := make(chan struct{})
	go func() {
		defer close(finished)
		h.SessionStop(id)
	}()
	select {
	case <-finished:
	case <-time.After(startSuccessWait):
		t.Fatalf("SessionStop must not wait for a connection that is not ready")
	}

	if _, err := h.SessionGet(id); err == nil {
		t.Errorf("the session must be removed")
	}
	if ctx.Err() == nil {
		t.Errorf("the session context must be cancelled")
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte("x")); err != nil {
		t.Errorf("a connection that was never handed over must be left open, got: %v", err)
	}
}

func Test_SessionStop_connectionCloseFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	logs := logtest.New(t)
	mockPython := NewMockPythonRunner(mc)
	h := newSessionTestHandler(mockPython)

	id := uuid.FromStringOrNil("a1b2c3d4-6103-4000-8000-000000000001")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// the connection is already closed, so closing it again reports an error.
	conn := newLoopbackConn(t)
	_ = conn.Close()
	ready := make(chan struct{})
	close(ready)
	h.mapPipecatcallSession[id] = &pipecatcall.Session{
		Identity:     commonidentity.Identity{ID: id},
		Ctx:          ctx,
		Cancel:       cancel,
		ConnAst:      conn,
		ConnAstReady: ready,
	}
	mockPython.EXPECT().Stop(gomock.Any(), id).Return(nil)

	h.SessionStop(id)

	if _, err := h.SessionGet(id); err == nil {
		t.Errorf("the session must be removed even when the connection close fails")
	}
	if dump := logs.Dump(); !strings.Contains(dump, "Could not close the asterisk connection.") {
		t.Errorf("Wrong log. expect: the close error, got: %q", dump)
	}
}
