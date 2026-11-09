package pushringhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"
	pmnotification "monorepo/bin-push-manager/models/notification"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/models/pushring"
)

var (
	tMasterFlow  = uuid.FromStringOrNil("e1000000-0000-4000-8000-0000000000f1")
	tOtherFlow   = uuid.FromStringOrNil("e1000000-0000-4000-8000-0000000000f2")
	tOtherCall   = uuid.FromStringOrNil("e1000000-0000-4000-8000-0000000000c1")
	tTicketFlowA = uuid.FromStringOrNil("e1000000-0000-4000-8000-0000000000f3")
	tTicketFlowB = uuid.FromStringOrNil("e1000000-0000-4000-8000-0000000000f4")
	tTicketCallA = uuid.FromStringOrNil("e1000000-0000-4000-8000-0000000000c2")
	tTicketCallB = uuid.FromStringOrNil("e1000000-0000-4000-8000-0000000000c3")
)

// sessionEnv starts a session and registers the calls of the scenario: the master call (the caller), two SIP legs of the
// session flow, and a call that runs another flow (an add-participant leg).
func sessionEnv(t *testing.T) (*env, uuid.UUID) {
	e := newEnv(t)

	res := e.start()
	if !res.Started {
		t.Fatalf("the session did not start: %+v", res)
	}

	master := e.addCall(tMasterCall, tMasterFlow)
	master.MasterCallID = uuid.Nil
	e.addCall(tLegCallA, tFlowID)
	e.addCall(tLegCallB, tFlowID)
	e.addCall(tOtherCall, tOtherFlow)
	e.groupcalls = []*groupcall.Groupcall{
		{
			Identity:     commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: tCustomerID},
			MasterCallID: tMasterCall,
			FlowID:       tFlowID,
			CallIDs:      []uuid.UUID{tLegCallA, tLegCallB},
		},
	}

	return e, res.SessionID
}

func cbOf() *confbridge.Confbridge {
	return &confbridge.Confbridge{
		Identity: commonidentity.Identity{ID: tConfbridge, CustomerID: tCustomerID},
		Type:     confbridge.TypeConnect,
		Flags:    []confbridge.Flag{confbridge.FlagPushRing},
	}
}

// ticketCall claims the ticket of the device, binds the flow and registers the call the way the incoming call path does.
func ticketCall(t *testing.T, e *env, deviceID, flowID, callID uuid.UUID) *pushring.TicketClaim {
	t.Helper()

	claim, err := e.h.TicketClaim(context.Background(), e.ticketOf(deviceID))
	if err != nil || claim == nil {
		t.Fatalf("claim: %v, %v", claim, err)
	}
	if errBind := e.h.TicketBind(context.Background(), claim, flowID); errBind != nil {
		t.Fatalf("bind: %v", errBind)
	}
	e.addCall(callID, flowID).MasterCallID = uuid.Nil

	return claim
}

func Test_JoinGate(t *testing.T) {
	ctx := context.Background()

	t.Run("the caller and a call of another flow pass without a lease", func(t *testing.T) {
		e, sid := sessionEnv(t)

		for _, id := range []uuid.UUID{tMasterCall, tOtherCall} {
			if err := e.h.JoinGate(ctx, cbOf(), id); err != nil {
				t.Errorf("call %s must pass: %v", id, err)
			}
		}
		if s := e.session(sid); s.LeaseOwner != uuid.Nil {
			t.Errorf("No lease may be taken: %v", s.LeaseOwner)
		}
		if len(e.hangupIDs()) != 0 {
			t.Errorf("Nothing may be hung up")
		}
	})

	t.Run("one leg takes the lease and the second is hung up", func(t *testing.T) {
		e, sid := sessionEnv(t)

		if err := e.h.JoinGate(ctx, cbOf(), tLegCallA); err != nil {
			t.Fatalf("the first leg must pass: %v", err)
		}
		if s := e.session(sid); s.LeaseOwner != tLegCallA {
			t.Errorf("Wrong lease owner: %v", s.LeaseOwner)
		}

		if err := e.h.JoinGate(ctx, cbOf(), tLegCallB); !errors.Is(err, ErrRefused) {
			t.Fatalf("the second leg must be refused: %v", err)
		}
		if got := e.hangupIDs(); len(got) != 1 || got[tLegCallB] != call.HangupReasonNormal {
			t.Errorf("The refused leg must be hung up: %v", got)
		}
	})

	t.Run("a released lease can be taken again", func(t *testing.T) {
		e, _ := sessionEnv(t)

		_ = e.h.JoinGate(ctx, cbOf(), tLegCallA)
		e.h.Release(ctx, tLegCallA)
		if err := e.h.JoinGate(ctx, cbOf(), tLegCallB); err != nil {
			t.Errorf("the second leg must pass after the release: %v", err)
		}
	})

	t.Run("an expired lease can be taken by another call", func(t *testing.T) {
		e, _ := sessionEnv(t)

		_ = e.h.JoinGate(ctx, cbOf(), tLegCallA)
		e.now = e.now.Add(pushring.LeaseTTL + time.Second)
		if err := e.h.JoinGate(ctx, cbOf(), tLegCallB); err != nil {
			t.Errorf("the second leg must pass after the lease time: %v", err)
		}
	})

	t.Run("no lease after the window", func(t *testing.T) {
		e, _ := sessionEnv(t)

		e.now = e.now.Add(pushring.Window)
		if err := e.h.JoinGate(ctx, cbOf(), tLegCallA); !errors.Is(err, ErrRefused) {
			t.Errorf("the join after the window must be refused: %v", err)
		}
	})

	t.Run("an app call and a SIP leg: the first lease wins", func(t *testing.T) {
		e, sid := sessionEnv(t)
		_ = e.h.Joined(ctx, cbOf(), tMasterCall) // the caller joined
		ticketCall(t, e, tDeviceA, tTicketFlowA, tTicketCallA)

		if err := e.h.JoinGate(ctx, cbOf(), tTicketCallA); err != nil {
			t.Fatalf("the app call must pass: %v", err)
		}
		if s := e.session(sid); s.LeaseOwner != tTicketCallA {
			t.Errorf("Wrong lease owner: %v", s.LeaseOwner)
		}
		if err := e.h.JoinGate(ctx, cbOf(), tLegCallA); !errors.Is(err, ErrRefused) {
			t.Errorf("the SIP leg must be refused while the app holds the lease: %v", err)
		}
	})

	t.Run("a closed session refuses its calls but not the others (add participant)", func(t *testing.T) {
		e, sid := sessionEnv(t)

		if err := e.h.Close(ctx, sid); err != nil {
			t.Fatalf("close: %v", err)
		}
		e.h.wg.Wait()

		if err := e.h.JoinGate(ctx, cbOf(), tLegCallA); !errors.Is(err, ErrRefused) {
			t.Errorf("a leg must be refused after the close: %v", err)
		}
		if err := e.h.JoinGate(ctx, cbOf(), tOtherCall); err != nil {
			t.Errorf("a call of another flow must pass: %v", err)
		}
	})

	t.Run("a session whose keys are gone does not gate", func(t *testing.T) {
		e, sid := sessionEnv(t)
		e.mr.Del("call:pushring:" + sid.String())
		e.mr.Del("call:pushring:flow:" + tFlowID.String())

		if err := e.h.JoinGate(ctx, cbOf(), tLegCallA); err != nil {
			t.Errorf("the join must pass: %v", err)
		}
	})

	t.Run("a cache failure refuses the join of a session confbridge", func(t *testing.T) {
		e, _ := sessionEnv(t)
		e.mr.Close()

		if err := e.h.JoinGate(ctx, cbOf(), tLegCallA); err == nil {
			t.Errorf("a cache failure must refuse the join")
		}
		if got := e.hangupIDs(); got[tLegCallA] != call.HangupReasonNormal {
			t.Errorf("the call must be hung up: %v", got)
		}
	})
}

func Test_Joined_callerJoined(t *testing.T) {
	ctx := context.Background()
	e, sid := sessionEnv(t)

	if s := e.session(sid); s.CallerJoined {
		t.Fatalf("the caller has not joined yet")
	}
	// the ticket is refused before the caller joined
	if _, err := e.h.TicketClaim(ctx, e.ticketOf(tDeviceA)); !errors.Is(err, pushring.ErrTicketRefused) {
		t.Errorf("the ticket must be refused before the caller joined: %v", err)
	}

	if err := e.h.Joined(ctx, cbOf(), tMasterCall); err != nil {
		t.Fatalf("joined: %v", err)
	}
	if s := e.session(sid); !s.CallerJoined || s.Winner != "" {
		t.Errorf("the caller must be marked as joined without a winner: %+v", s)
	}
	// a call that is not part of the session changes nothing
	if err := e.h.Joined(ctx, cbOf(), tOtherCall); err != nil {
		t.Errorf("another call must pass: %v", err)
	}
	if s := e.session(sid); s.Winner != "" {
		t.Errorf("another call must not win")
	}
}

func Test_Joined_sipWinner(t *testing.T) {
	ctx := context.Background()
	e, sid := sessionEnv(t)
	_ = e.h.Joined(ctx, cbOf(), tMasterCall)

	if err := e.h.JoinGate(ctx, cbOf(), tLegCallA); err != nil {
		t.Fatalf("gate: %v", err)
	}
	if err := e.h.Joined(ctx, cbOf(), tLegCallA); err != nil {
		t.Fatalf("joined: %v", err)
	}
	e.h.wg.Wait()

	s := e.session(sid)
	if s.Winner != pushring.WinnerSIP {
		t.Errorf("Wrong winner: %s", s.Winner)
	}
	for _, k := range e.mr.Keys() {
		if len(k) > len("call:pushticket:") && k[:len("call:pushticket:")] == "call:pushticket:" {
			t.Errorf("the tickets of every device must be deleted: %s", k)
		}
	}
	cancels := e.sendsOf(pmnotification.KindCallCancelled)
	if len(cancels) != 1 || cancels[0].deviceID != uuid.Nil || cancels[0].data["cancel_key"] != sid.String() {
		t.Errorf("every device must get one cancel notification: %+v", cancels)
	}

	// a second leg and an app ticket are refused after the winner
	if err := e.h.JoinGate(ctx, cbOf(), tLegCallB); !errors.Is(err, ErrRefused) {
		t.Errorf("the second leg must be refused: %v", err)
	}
	// the tickets are deleted, so the value is no ticket any more
	if claim, err := e.h.TicketClaim(ctx, e.ticketOf(tDeviceA)); claim != nil || err != nil {
		t.Errorf("the deleted ticket is no ticket. got: %v, %v", claim, err)
	}
}

func Test_Joined_appWinner(t *testing.T) {
	ctx := context.Background()
	e, sid := sessionEnv(t)
	_ = e.h.Joined(ctx, cbOf(), tMasterCall)

	ticketCall(t, e, tDeviceA, tTicketFlowA, tTicketCallA)
	// the other device already claimed its ticket too
	ticketCall(t, e, tDeviceB, tTicketFlowB, tTicketCallB)

	if err := e.h.JoinGate(ctx, cbOf(), tTicketCallA); err != nil {
		t.Fatalf("gate: %v", err)
	}
	if err := e.h.Joined(ctx, cbOf(), tTicketCallA); err != nil {
		t.Fatalf("joined: %v", err)
	}
	e.h.wg.Wait()

	s := e.session(sid)
	if s.Winner != pushring.WinnerAppPrefix+tDeviceA.String() {
		t.Errorf("Wrong winner: %s", s.Winner)
	}

	// the SIP legs are hung up through the calls of the master and through the groupcall. The winner is not.
	hang := e.hangupIDs()
	if hang[tLegCallA] != call.HangupReasonNormal || hang[tLegCallB] != call.HangupReasonNormal {
		t.Errorf("the SIP legs must be hung up: %v", hang)
	}
	if _, ok := hang[tTicketCallA]; ok {
		t.Errorf("the winner must not be hung up")
	}

	// only the other device is told to stop ringing
	cancels := e.sendsOf(pmnotification.KindCallCancelled)
	if len(cancels) != 1 || cancels[0].deviceID != tDeviceB {
		t.Errorf("only the other device gets the cancel notification: %+v", cancels)
	}

	// the second app call is refused at the gate
	if err := e.h.JoinGate(ctx, cbOf(), tTicketCallB); !errors.Is(err, ErrRefused) {
		t.Errorf("the second app must be refused: %v", err)
	}
	// and a new redemption of the first ticket is refused too (consumed)
	if _, err := e.h.TicketClaim(ctx, e.ticketOf(tDeviceA)); !errors.Is(err, pushring.ErrTicketRefused) {
		t.Errorf("the consumed ticket must be refused: %v", err)
	}
	// the ticket of the other device is gone, a redemption after the winner is refused or not found
	if claim, _ := e.h.TicketClaim(ctx, e.ticketOf(tDeviceB)); claim != nil {
		t.Errorf("a ticket may not be claimed after the winner")
	}
}

func Test_Joined_refused(t *testing.T) {
	ctx := context.Background()

	t.Run("a join after the window may not win", func(t *testing.T) {
		e, sid := sessionEnv(t)
		_ = e.h.JoinGate(ctx, cbOf(), tLegCallA)

		e.now = e.now.Add(pushring.Window)
		if err := e.h.Joined(ctx, cbOf(), tLegCallA); !errors.Is(err, ErrRefused) {
			t.Errorf("must be refused: %v", err)
		}
		if s := e.session(sid); s.Winner != "" {
			t.Errorf("no winner after the window: %s", s.Winner)
		}
	})

	t.Run("a late join whose lease was taken by another call may not win", func(t *testing.T) {
		e, _ := sessionEnv(t)
		_ = e.h.JoinGate(ctx, cbOf(), tLegCallA)
		e.now = e.now.Add(pushring.LeaseTTL + time.Second)
		if err := e.h.JoinGate(ctx, cbOf(), tLegCallB); err != nil {
			t.Fatalf("gate b: %v", err)
		}

		if err := e.h.Joined(ctx, cbOf(), tLegCallA); !errors.Is(err, ErrRefused) {
			t.Errorf("the old lease owner must be refused: %v", err)
		}
		if err := e.h.Joined(ctx, cbOf(), tLegCallB); err != nil {
			t.Errorf("the new lease owner must win: %v", err)
		}
	})

	t.Run("a late join with an expired lease that nobody took re-takes it", func(t *testing.T) {
		e, sid := sessionEnv(t)
		_ = e.h.JoinGate(ctx, cbOf(), tLegCallA)
		e.now = e.now.Add(pushring.LeaseTTL + time.Second)

		if err := e.h.Joined(ctx, cbOf(), tLegCallA); err != nil {
			t.Errorf("must win: %v", err)
		}
		if s := e.session(sid); s.Winner != pushring.WinnerSIP {
			t.Errorf("Wrong winner: %s", s.Winner)
		}
	})

	t.Run("a cache failure on the confirm refuses", func(t *testing.T) {
		e, _ := sessionEnv(t)
		_ = e.h.JoinGate(ctx, cbOf(), tLegCallA)
		e.mr.Close()

		if err := e.h.Joined(ctx, cbOf(), tLegCallA); err == nil {
			t.Errorf("a cache failure must refuse the join")
		}
	})
}

// A SIP leg and the app answer together: only one of them wins and the participants never exceed two.
func Test_Joined_race(t *testing.T) {
	ctx := context.Background()
	e, sid := sessionEnv(t)
	_ = e.h.Joined(ctx, cbOf(), tMasterCall)
	ticketCall(t, e, tDeviceA, tTicketFlowA, tTicketCallA)

	candidates := []uuid.UUID{tLegCallA, tLegCallB, tTicketCallA}
	results := make(chan error, len(candidates))
	for _, id := range candidates {
		go func(id uuid.UUID) {
			if err := e.h.JoinGate(ctx, cbOf(), id); err != nil {
				results <- err
				return
			}
			results <- e.h.Joined(ctx, cbOf(), id)
		}(id)
	}

	won := 0
	for range candidates {
		if err := <-results; err == nil {
			won++
		}
	}
	e.h.wg.Wait()

	if won != 1 {
		t.Errorf("exactly one path must win. got: %d", won)
	}
	if s := e.session(sid); s.Winner == "" {
		t.Errorf("a winner must exist")
	}
}

func Test_Release(t *testing.T) {
	ctx := context.Background()

	t.Run("another call does not free the lease", func(t *testing.T) {
		e, sid := sessionEnv(t)
		_ = e.h.JoinGate(ctx, cbOf(), tLegCallA)

		e.h.Release(ctx, tLegCallB)
		if s := e.session(sid); s.LeaseOwner != tLegCallA {
			t.Errorf("the lease must stay: %v", s.LeaseOwner)
		}
	})

	t.Run("a call that is not part of a session changes nothing", func(t *testing.T) {
		e, _ := sessionEnv(t)
		e.h.Release(ctx, tOtherCall)
		e.h.Release(ctx, uuid.Must(uuid.NewV4())) // unknown call
	})
}
