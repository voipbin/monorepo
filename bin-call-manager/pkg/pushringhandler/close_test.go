package pushringhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	pmnotification "monorepo/bin-push-manager/models/notification"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/pushring"
)

func Test_Expire(t *testing.T) {
	ctx := context.Background()

	t.Run("no winner: every path of the session ends and the master call ends with no answer", func(t *testing.T) {
		e, sid := sessionEnv(t)
		_ = e.h.Joined(ctx, cbOf(), tMasterCall)
		ticketCall(t, e, tDeviceA, tTicketFlowA, tTicketCallA)
		// a leg holds the lease at the end of the window
		_ = e.h.JoinGate(ctx, cbOf(), tLegCallA)

		if err := e.h.Expire(ctx, sid); err != nil {
			t.Fatalf("expire: %v", err)
		}
		e.h.wg.Wait()

		if s := e.session(sid); s.State != pushring.StateClosed {
			t.Errorf("the session must be closed: %s", s.State)
		}

		hang := e.hangupIDs()
		if hang[tMasterCall] != call.HangupReasonNoanswer {
			t.Errorf("the master call must end with no answer: %v", hang)
		}
		for _, id := range []uuid.UUID{tLegCallA, tLegCallB, tTicketCallA} {
			if hang[id] != call.HangupReasonNormal {
				t.Errorf("the call %s must be hung up: %v", id, hang)
			}
		}
		if _, ok := hang[tOtherCall]; ok {
			t.Errorf("a call of another flow must not be touched")
		}

		cancels := e.sendsOf(pmnotification.KindCallCancelled)
		if len(cancels) != 1 || cancels[0].deviceID != uuid.Nil {
			t.Errorf("every device must get a cancel notification: %+v", cancels)
		}
		if len(e.terminated) != 1 || e.terminated[0] != tConfbridge {
			t.Errorf("the confbridge must be terminated: %v", e.terminated)
		}
		for _, k := range e.mr.Keys() {
			if len(k) > len("call:pushticket:") && k[:len("call:pushticket:")] == "call:pushticket:" {
				t.Errorf("the tickets must be deleted: %s", k)
			}
		}
	})

	t.Run("a leg that holds the lease is cut at the window end even though it is joining", func(t *testing.T) {
		e, sid := sessionEnv(t)
		_ = e.h.JoinGate(ctx, cbOf(), tLegCallA)

		e.now = e.now.Add(pushring.Window)
		if err := e.h.Expire(ctx, sid); err != nil {
			t.Fatalf("expire: %v", err)
		}
		e.h.wg.Wait()

		if err := e.h.Joined(ctx, cbOf(), tLegCallA); err == nil {
			t.Errorf("the join after the window end must be refused")
		}
	})

	t.Run("a winner: only the keys are cleaned", func(t *testing.T) {
		e, sid := sessionEnv(t)
		_ = e.h.JoinGate(ctx, cbOf(), tLegCallA)
		_ = e.h.Joined(ctx, cbOf(), tLegCallA)
		e.h.wg.Wait()
		before := len(e.hangupIDs())
		beforeCancels := len(e.sendsOf(pmnotification.KindCallCancelled))

		if err := e.h.Expire(ctx, sid); err != nil {
			t.Fatalf("expire: %v", err)
		}
		e.h.wg.Wait()

		if len(e.hangupIDs()) != before {
			t.Errorf("nothing may be hung up with a winner: %v", e.hangupIDs())
		}
		if len(e.sendsOf(pmnotification.KindCallCancelled)) != beforeCancels {
			t.Errorf("no further cancel notification")
		}
		if len(e.terminated) != 0 {
			t.Errorf("the confbridge of a connected call must stay: %v", e.terminated)
		}
	})

	t.Run("the second expire request and an unknown session do nothing", func(t *testing.T) {
		e, sid := sessionEnv(t)

		if err := e.h.Expire(ctx, sid); err != nil {
			t.Fatalf("first: %v", err)
		}
		e.h.wg.Wait()
		count := len(e.sendsOf(pmnotification.KindCallCancelled))
		hangs := len(e.hangupIDs())

		if err := e.h.Expire(ctx, sid); err != nil {
			t.Fatalf("second: %v", err)
		}
		if err := e.h.Expire(ctx, uuid.Must(uuid.NewV4())); err != nil {
			t.Fatalf("unknown: %v", err)
		}
		e.h.wg.Wait()

		if len(e.sendsOf(pmnotification.KindCallCancelled)) != count || len(e.hangupIDs()) != hangs {
			t.Errorf("the close must run once")
		}
	})

	t.Run("a cache failure is reported", func(t *testing.T) {
		e, sid := sessionEnv(t)
		e.mr.Close()

		if err := e.h.Expire(ctx, sid); err == nil {
			t.Errorf("a cache failure must be an error, so the request is not taken as done")
		}
	})
}

func Test_Close(t *testing.T) {
	ctx := context.Background()

	e, sid := sessionEnv(t)
	if err := e.h.Close(ctx, sid); err != nil {
		t.Fatalf("close: %v", err)
	}
	e.h.wg.Wait()

	hang := e.hangupIDs()
	if _, ok := hang[tMasterCall]; ok {
		t.Errorf("the close request must not hang up the master call: %v", hang)
	}
	if hang[tLegCallA] != call.HangupReasonNormal {
		t.Errorf("the SIP legs must be hung up: %v", hang)
	}
	if s := e.session(sid); s.State != pushring.StateClosed {
		t.Errorf("the session must be closed: %s", s.State)
	}
}

// The hangup function is injected after the construction. Without it a hangup is an error, not a panic.
func Test_hangupCall_notSet(t *testing.T) {
	e := newEnv(t)
	e.h.hangup = nil

	if err := e.h.hangupCall(context.Background(), tMasterCall, call.HangupReasonNormal); err == nil {
		t.Errorf("expected an error")
	}
}
