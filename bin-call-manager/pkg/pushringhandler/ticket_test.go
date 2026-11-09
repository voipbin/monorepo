package pushringhandler

import (
	"github.com/gofrs/uuid"
	pmnotification "monorepo/bin-push-manager/models/notification"

	"context"
	"errors"
	"testing"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/pushring"
)

func Test_TicketClaim(t *testing.T) {
	ctx := context.Background()

	t.Run("a value that is no ticket continues with the direct lookup", func(t *testing.T) {
		e, _ := sessionEnv(t)

		for _, v := range []string{"", "direct.abc123", "direct.0123456789ab", directPrefix + "0123456789abcdef0123456789abcdef"} {
			claim, err := e.h.TicketClaim(ctx, v)
			if claim != nil || err != nil {
				t.Errorf("value %q must fall through. got: %v, %v", v, claim, err)
			}
		}
	})

	t.Run("ok after the caller joined, and the bound ticket identifies the call", func(t *testing.T) {
		e, sid := sessionEnv(t)
		_ = e.h.Joined(ctx, cbOf(), tMasterCall)

		claim, err := e.h.TicketClaim(ctx, e.ticketOf(tDeviceA))
		if err != nil || claim == nil || claim.SessionID != sid || claim.DeviceID != tDeviceA {
			t.Fatalf("wrong claim: %+v, %v", claim, err)
		}

		s, err := e.h.TicketSessionGet(ctx, claim)
		if err != nil || s.ID != sid {
			t.Errorf("wrong session: %v, %v", s, err)
		}

		// a concurrent INVITE with the same ticket is refused
		if _, errSecond := e.h.TicketClaim(ctx, e.ticketOf(tDeviceA)); !errors.Is(errSecond, pushring.ErrTicketRefused) {
			t.Errorf("the second claim must be refused: %v", errSecond)
		}
	})

	t.Run("refused after the window, with its own ticket of the other device unaffected", func(t *testing.T) {
		e, _ := sessionEnv(t)
		_ = e.h.Joined(ctx, cbOf(), tMasterCall)

		e.now = e.now.Add(pushring.Window)
		if _, err := e.h.TicketClaim(ctx, e.ticketOf(tDeviceA)); !errors.Is(err, pushring.ErrTicketRefused) {
			t.Errorf("must be refused after the window: %v", err)
		}
	})

	t.Run("a ticket returns to issued when its call hung up before it joined, and the retry is allowed", func(t *testing.T) {
		e, sid := sessionEnv(t)
		_ = e.h.Joined(ctx, cbOf(), tMasterCall)

		ticketCall(t, e, tDeviceA, tTicketFlowA, tTicketCallA)

		// the first attempt fails before the join: the call hangs up
		e.h.CallHangup(ctx, e.calls[tTicketCallA])

		claim, err := e.h.TicketClaim(ctx, e.ticketOf(tDeviceA))
		if err != nil || claim == nil || claim.SessionID != sid {
			t.Errorf("the retry must be allowed: %v, %v", claim, err)
		}
	})

	t.Run("a ticket in use with no call ever created returns by time", func(t *testing.T) {
		e, _ := sessionEnv(t)
		_ = e.h.Joined(ctx, cbOf(), tMasterCall)

		if _, err := e.h.TicketClaim(ctx, e.ticketOf(tDeviceA)); err != nil {
			t.Fatalf("claim: %v", err)
		}
		e.now = e.now.Add(pushring.TicketInUseTTL + 1)
		if claim, err := e.h.TicketClaim(ctx, e.ticketOf(tDeviceA)); err != nil || claim == nil {
			t.Errorf("the retry must be allowed after the in use time: %v, %v", claim, err)
		}
	})

	t.Run("a cache failure falls through to the direct lookup", func(t *testing.T) {
		e, _ := sessionEnv(t)
		ticket := e.ticketOf(tDeviceA)
		e.mr.Close()

		if claim, err := e.h.TicketClaim(ctx, ticket); claim != nil || err != nil {
			t.Errorf("a cache failure must fall through. got: %v, %v", claim, err)
		}
	})
}

func Test_CallHangup(t *testing.T) {
	ctx := context.Background()

	t.Run("a lease owner leg that hung up releases the lease", func(t *testing.T) {
		e, sid := sessionEnv(t)
		_ = e.h.JoinGate(ctx, cbOf(), tLegCallA)

		e.h.CallHangup(ctx, e.calls[tLegCallA])
		if s := e.session(sid); s.LeaseOwner != uuid.Nil {
			t.Errorf("the lease must be released: %v", s.LeaseOwner)
		}
		if err := e.h.JoinGate(ctx, cbOf(), tLegCallB); err != nil {
			t.Errorf("the other leg must pass: %v", err)
		}
	})

	t.Run("a hangup after the winner changes nothing", func(t *testing.T) {
		e, sid := sessionEnv(t)
		_ = e.h.JoinGate(ctx, cbOf(), tLegCallA)
		_ = e.h.Joined(ctx, cbOf(), tLegCallA)
		e.h.wg.Wait()

		e.h.CallHangup(ctx, e.calls[tLegCallA])
		if s := e.session(sid); s.Winner != pushring.WinnerSIP || s.LeaseOwner != tLegCallA {
			t.Errorf("the winner must stay: %+v", s)
		}
	})

	t.Run("a call that is not part of a session is ignored", func(t *testing.T) {
		e, sid := sessionEnv(t)

		e.h.CallHangup(ctx, e.calls[tOtherCall])
		if s := e.session(sid); s.State != pushring.StateRinging {
			t.Errorf("the session must stay open: %s", s.State)
		}
	})

	t.Run("a cache failure is not fatal", func(t *testing.T) {
		e, _ := sessionEnv(t)
		e.mr.Close()

		e.h.CallHangup(ctx, e.calls[tMasterCall])
	})

	t.Run("the hangup of the master call closes the session without hanging up the master", func(t *testing.T) {
		e, sid := sessionEnv(t)
		_ = e.h.Joined(ctx, cbOf(), tMasterCall)

		e.h.CallHangup(ctx, e.calls[tMasterCall])
		e.h.wg.Wait()

		if s := e.session(sid); s.State != pushring.StateClosed {
			t.Errorf("the session must be closed: %s", s.State)
		}
		hang := e.hangupIDs()
		if _, ok := hang[tMasterCall]; ok {
			t.Errorf("the master call is gone and must not be hung up again: %v", hang)
		}
		if hang[tLegCallA] != call.HangupReasonNormal || hang[tLegCallB] != call.HangupReasonNormal {
			t.Errorf("the SIP legs must be hung up: %v", hang)
		}
		if len(e.terminated) != 1 || e.terminated[0] != tConfbridge {
			t.Errorf("the confbridge must be terminated: %v", e.terminated)
		}
		if n := len(e.sendsOf(pmnotification.KindCallCancelled)); n != 1 {
			t.Errorf("the devices must get a cancel notification. count: %d", n)
		}
	})
}
