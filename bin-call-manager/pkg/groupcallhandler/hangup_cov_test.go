package groupcallhandler

import (
	"context"
	"sync"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/groupcall"
)

// expectHangupOK expects a successful h.Hangup of the covID groupcall (no master groupcall).
func expectHangupOK(m *covMocks, ctx context.Context, gc *groupcall.Groupcall) {
	m.db.EXPECT().GroupcallSetStatus(ctx, covID, groupcall.StatusHangup).Return(nil)
	m.db.EXPECT().GroupcallGet(ctx, covID).Return(gc, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, covCustomer, groupcall.EventTypeGroupcallHangup, gc)
}

func Test_Hangup_cov(t *testing.T) {
	ctx := context.Background()

	t.Run("update status error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.db.EXPECT().GroupcallSetStatus(ctx, covID, groupcall.StatusHangup).Return(errCov)
		if _, err := h.Hangup(ctx, covID); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("master groupcall hangup failure is only logged", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := &groupcall.Groupcall{Identity: commonidentity.Identity{ID: covID, CustomerID: covCustomer}, MasterGroupcallID: covMasterGC}
		expectHangupOK(m, ctx, gc)

		var wg sync.WaitGroup
		wg.Add(1)
		m.req.EXPECT().CallV1GroupcallHangupGroupcall(ctx, covMasterGC).DoAndReturn(func(context.Context, uuid.UUID) error {
			wg.Done()
			return errCov
		})

		res, err := h.Hangup(ctx, covID)
		if err != nil || res != gc {
			t.Errorf("unexpected: %v, %v", res, err)
		}
		waitCov(t, &wg)
	})
}

func Test_Hangingup_cov(t *testing.T) {
	ctx := context.Background()

	t.Run("update status error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.db.EXPECT().GroupcallSetStatus(ctx, covID, groupcall.StatusHangingup).Return(errCov)
		if _, err := h.Hangingup(ctx, covID); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("hangs up chained groupcalls and calls as a set", func(t *testing.T) {
		h, m := newCovHandler(t)
		gcID1 := uuid.FromStringOrNil("c0a00000-0000-4000-8000-0000000000b1")
		gcID2 := uuid.FromStringOrNil("c0a00000-0000-4000-8000-0000000000b2")
		callID1 := uuid.FromStringOrNil("c0a00000-0000-4000-8000-0000000000c1")
		callID2 := uuid.FromStringOrNil("c0a00000-0000-4000-8000-0000000000c2")
		gc := &groupcall.Groupcall{
			Identity:     commonidentity.Identity{ID: covID, CustomerID: covCustomer},
			GroupcallIDs: []uuid.UUID{gcID1, gcID2},
			CallIDs:      []uuid.UUID{callID1, callID2},
		}
		m.db.EXPECT().GroupcallSetStatus(ctx, covID, groupcall.StatusHangingup).Return(nil)
		m.db.EXPECT().GroupcallGet(ctx, covID).Return(gc, nil)

		var mu sync.Mutex
		gotGroupcalls := map[uuid.UUID]bool{}
		gotCalls := map[uuid.UUID]bool{}
		var wg sync.WaitGroup
		wg.Add(4)
		m.req.EXPECT().CallV1GroupcallHangup(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {
			mu.Lock()
			gotGroupcalls[id] = true
			mu.Unlock()
			wg.Done()
			return nil, nil
		}).Times(2)
		m.req.EXPECT().CallV1CallHangup(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) (*call.Call, error) {
			mu.Lock()
			gotCalls[id] = true
			mu.Unlock()
			wg.Done()
			return nil, nil
		}).Times(2)

		if _, err := h.Hangingup(ctx, covID); err != nil {
			t.Fatalf("expect ok, got %v", err)
		}
		waitCov(t, &wg)

		mu.Lock()
		defer mu.Unlock()
		if !gotGroupcalls[gcID1] || !gotGroupcalls[gcID2] || !gotCalls[callID1] || !gotCalls[callID2] {
			t.Errorf("not all ids were hung up. groupcalls: %v, calls: %v", gotGroupcalls, gotCalls)
		}
	})
}

func Test_HangupGroupcallAndCall_errors_cov(t *testing.T) {
	ctx := context.Background()

	t.Run("HangupGroupcall decrease error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.db.EXPECT().GroupcallDecreaseGroupcallCount(ctx, covID).Return(errCov)
		if _, err := h.HangupGroupcall(ctx, covID); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("HangupCall decrease error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.db.EXPECT().GroupcallDecreaseCallCount(ctx, covID).Return(errCov)
		if _, err := h.HangupCall(ctx, covID); err == nil {
			t.Errorf("expect error")
		}
	})
}

func Test_hangupCommon_cov(t *testing.T) {
	ctx := context.Background()
	base := func() *groupcall.Groupcall {
		return &groupcall.Groupcall{Identity: commonidentity.Identity{ID: covID, CustomerID: covCustomer}}
	}

	t.Run("still ongoing calls", func(t *testing.T) {
		h, _ := newCovHandler(t)
		gc := base()
		gc.CallCount = 1
		res, err := h.hangupCommon(ctx, gc)
		if err != nil || res != gc {
			t.Errorf("unexpected: %v, %v", res, err)
		}
	})

	t.Run("still ongoing groupcalls", func(t *testing.T) {
		h, _ := newCovHandler(t)
		gc := base()
		gc.GroupcallCount = 1
		res, err := h.hangupCommon(ctx, gc)
		if err != nil || res != gc {
			t.Errorf("unexpected: %v, %v", res, err)
		}
	})

	t.Run("answered call hangs up", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := base()
		gc.AnswerCallID = covOther
		expectHangupOK(m, ctx, gc)
		if res, err := h.hangupCommon(ctx, gc); err != nil || res != gc {
			t.Errorf("unexpected: %v, %v", res, err)
		}
	})

	t.Run("answered call hangup error", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := base()
		gc.AnswerCallID = covOther
		m.db.EXPECT().GroupcallSetStatus(ctx, covID, groupcall.StatusHangup).Return(errCov)
		if _, err := h.hangupCommon(ctx, gc); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("hangingup status hangs up", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := base()
		gc.Status = groupcall.StatusHangingup
		expectHangupOK(m, ctx, gc)
		if res, err := h.hangupCommon(ctx, gc); err != nil || res != gc {
			t.Errorf("unexpected: %v, %v", res, err)
		}
	})

	t.Run("ringall hangup error", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := base()
		gc.RingMethod = groupcall.RingMethodRingAll
		m.db.EXPECT().GroupcallSetStatus(ctx, covID, groupcall.StatusHangup).Return(errCov)
		if _, err := h.hangupCommon(ctx, gc); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("ringall hangup ok", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := base()
		gc.RingMethod = groupcall.RingMethodRingAll
		expectHangupOK(m, ctx, gc)
		if res, err := h.hangupCommon(ctx, gc); err != nil || res != gc {
			t.Errorf("unexpected: %v, %v", res, err)
		}
	})

	t.Run("unsupported ring method", func(t *testing.T) {
		h, _ := newCovHandler(t)
		gc := base()
		gc.RingMethod = groupcall.RingMethod("unknown")
		if _, err := h.hangupCommon(ctx, gc); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("linear error is wrapped", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := base()
		gc.RingMethod = groupcall.RingMethodLinear
		gc.MasterCallID = covMasterCL
		m.req.EXPECT().CallV1CallGet(ctx, covMasterCL).Return(nil, errCov)
		if _, err := h.hangupCommon(ctx, gc); err == nil {
			t.Errorf("expect error")
		}
	})
}

func Test_hangupRingMethodLinear_cov(t *testing.T) {
	ctx := context.Background()
	base := func() *groupcall.Groupcall {
		return &groupcall.Groupcall{
			Identity:     commonidentity.Identity{ID: covID, CustomerID: covCustomer},
			RingMethod:   groupcall.RingMethodLinear,
			Destinations: []commonaddress.Address{covTel, covTel},
		}
	}

	t.Run("master call get error", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := base()
		gc.MasterCallID = covMasterCL
		m.req.EXPECT().CallV1CallGet(ctx, covMasterCL).Return(nil, errCov)
		if _, err := h.hangupRingMethodLinear(ctx, gc); err == nil {
			t.Errorf("expect error")
		}
	})

	for _, status := range []call.Status{call.StatusHangup, call.StatusCanceling, call.StatusTerminating} {
		t.Run("master call "+string(status), func(t *testing.T) {
			h, m := newCovHandler(t)
			gc := base()
			gc.MasterCallID = covMasterCL
			m.req.EXPECT().CallV1CallGet(ctx, covMasterCL).Return(&call.Call{Identity: commonidentity.Identity{ID: covMasterCL}, Status: status}, nil)
			expectHangupOK(m, ctx, gc)
			if res, err := h.hangupRingMethodLinear(ctx, gc); err != nil || res != gc {
				t.Errorf("unexpected: %v, %v", res, err)
			}
		})
	}

	t.Run("master groupcall get error", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := base()
		gc.MasterGroupcallID = covMasterGC
		m.db.EXPECT().GroupcallGet(ctx, covMasterGC).Return(nil, errCov)
		if _, err := h.hangupRingMethodLinear(ctx, gc); err == nil {
			t.Errorf("expect error")
		}
	})

	for _, status := range []groupcall.Status{groupcall.StatusHangingup, groupcall.StatusHangup} {
		t.Run("master groupcall "+string(status), func(t *testing.T) {
			h, m := newCovHandler(t)
			gc := base()
			gc.MasterGroupcallID = covMasterGC
			m.db.EXPECT().GroupcallGet(ctx, covMasterGC).Return(&groupcall.Groupcall{Identity: commonidentity.Identity{ID: covMasterGC}, Status: status}, nil)
			// the hung up result must not carry the master groupcall id (no extra goroutine)
			resp := base()
			expectHangupOK(m, ctx, resp)
			if res, err := h.hangupRingMethodLinear(ctx, gc); err != nil || res != resp {
				t.Errorf("unexpected: %v, %v", res, err)
			}
		})
	}

	t.Run("no more destination hangs up", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := base()
		gc.DialIndex = 1
		expectHangupOK(m, ctx, gc)
		if res, err := h.hangupRingMethodLinear(ctx, gc); err != nil || res != gc {
			t.Errorf("unexpected: %v, %v", res, err)
		}
	})

	t.Run("dial next destination error", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := base()
		m.util.EXPECT().UUIDCreate().Return(covNewID)
		m.db.EXPECT().GroupcallSetCallIDsAndCallCountAndDialIndex(ctx, covID, gomock.Any(), gomock.Any(), 1).Return(errCov)
		if _, err := h.hangupRingMethodLinear(ctx, gc); err == nil {
			t.Errorf("expect error")
		}
	})
}

func Test_HangingupOthers_cov(t *testing.T) {
	ctx := context.Background()
	h, m := newCovHandler(t)

	gcID1 := uuid.FromStringOrNil("c0a00000-0000-4000-8000-0000000000d1")
	answerID := uuid.FromStringOrNil("c0a00000-0000-4000-8000-0000000000e1")
	callID1 := uuid.FromStringOrNil("c0a00000-0000-4000-8000-0000000000e2")
	callID2 := uuid.FromStringOrNil("c0a00000-0000-4000-8000-0000000000e3")
	gc := &groupcall.Groupcall{
		Identity:     commonidentity.Identity{ID: covID, CustomerID: covCustomer},
		AnswerCallID: answerID,
		GroupcallIDs: []uuid.UUID{gcID1},
		CallIDs:      []uuid.UUID{callID1, answerID, callID2},
	}

	var mu sync.Mutex
	gotGroupcalls := map[uuid.UUID]bool{}
	gotCalls := map[uuid.UUID]bool{}
	var wg sync.WaitGroup
	wg.Add(3)
	m.req.EXPECT().CallV1GroupcallHangupOthers(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) error {
		mu.Lock()
		gotGroupcalls[id] = true
		mu.Unlock()
		wg.Done()
		return nil
	}).Times(1)
	m.req.EXPECT().CallV1CallHangup(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) (*call.Call, error) {
		mu.Lock()
		gotCalls[id] = true
		mu.Unlock()
		wg.Done()
		return nil, nil
	}).Times(2)

	if err := h.HangingupOthers(ctx, gc); err != nil {
		t.Fatalf("expect ok, got %v", err)
	}
	waitCov(t, &wg)

	mu.Lock()
	defer mu.Unlock()
	if !gotGroupcalls[gcID1] || len(gotGroupcalls) != 1 {
		t.Errorf("Wrong match. expect groupcalls: {%s}, got: %v", gcID1, gotGroupcalls)
	}
	if !gotCalls[callID1] || !gotCalls[callID2] || gotCalls[answerID] || len(gotCalls) != 2 {
		t.Errorf("Wrong match. expect calls: {%s, %s} without %s, got: %v", callID1, callID2, answerID, gotCalls)
	}
}
