package pipecatcall

import (
	"testing"

	"github.com/gorilla/websocket"
)

func Test_Session_SetConnAst(t *testing.T) {
	s := &Session{ConnAstReady: make(chan struct{})}

	select {
	case <-s.ConnAstReady:
		t.Fatalf("the ready channel must be open before the connection is set")
	default:
	}

	first := &websocket.Conn{}
	firstDone := make(chan struct{})
	s.SetConnAst(first, firstDone)

	if s.ConnAst != first || s.ConnAstDone != firstDone {
		t.Errorf("Wrong match. the first connection and done channel must be stored")
	}
	select {
	case <-s.ConnAstReady:
	default:
		t.Errorf("the ready channel must be closed after the connection is set")
	}

	// a second call is ignored and must not close the ready channel again.
	s.SetConnAst(&websocket.Conn{}, make(chan struct{}))
	if s.ConnAst != first || s.ConnAstDone != firstDone {
		t.Errorf("Wrong match. a second call must keep the first connection and done channel")
	}
}
