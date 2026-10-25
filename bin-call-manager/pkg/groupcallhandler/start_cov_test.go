package groupcallhandler

import (
	"context"
	"sync"
	"testing"

	agagent "monorepo/bin-agent-manager/models/agent"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	rmastcontact "monorepo/bin-registrar-manager/models/astcontact"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/groupcall"
)

var (
	covTel = commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000001"}
	covExt = commonaddress.Address{Type: commonaddress.TypeExtension, Target: "c0a00000-0000-4000-8000-0000000000ee", TargetName: "ext1"}
)

// expectCreateOK expects a successful h.Create.
func expectCreateOK(m *covMocks, ctx context.Context) {
	gc := &groupcall.Groupcall{Identity: commonidentity.Identity{ID: covID, CustomerID: covCustomer}}
	m.db.EXPECT().GroupcallCreate(ctx, gomock.Any()).Return(nil)
	m.db.EXPECT().GroupcallGet(ctx, gomock.Any()).Return(gc, nil)
	m.notify.EXPECT().PublishWebhookEvent(ctx, covCustomer, groupcall.EventTypeGroupcallCreated, gc)
}

func Test_Start_errors_cov(t *testing.T) {
	ctx := context.Background()

	t.Run("no destination", func(t *testing.T) {
		h, _ := newCovHandler(t)
		if _, err := h.Start(ctx, covID, covCustomer, uuid.Nil, &covTel, nil, uuid.Nil, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers, "", nil); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("nil id and unsupported ring method", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.util.EXPECT().UUIDCreate().Return(covNewID)
		if _, err := h.Start(ctx, uuid.Nil, covCustomer, uuid.Nil, &covTel, []commonaddress.Address{covTel}, uuid.Nil, uuid.Nil, groupcall.RingMethod("unknown"), groupcall.AnswerMethodHangupOthers, "", nil); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("default ring method goes to ringall and fails", func(t *testing.T) {
		h, _ := newCovHandler(t)
		// unsupported destination type => startWithDestination fails
		dest := commonaddress.Address{Type: commonaddress.TypeNone}
		if _, err := h.Start(ctx, covID, covCustomer, uuid.Nil, &covTel, []commonaddress.Address{dest}, uuid.Nil, uuid.Nil, groupcall.RingMethodNone, groupcall.AnswerMethodHangupOthers, "", nil); err == nil {
			t.Errorf("expect error")
		}
	})
}

func Test_startRingall_cov(t *testing.T) {
	ctx := context.Background()

	t.Run("create error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.util.EXPECT().UUIDCreate().Return(covNewID).Times(2)
		m.db.EXPECT().GroupcallCreate(ctx, gomock.Any()).Return(errCov)

		if _, err := h.startRingall(ctx, covID, covCustomer, &covTel, []commonaddress.Address{covTel, covTel}, uuid.Nil, uuid.Nil, uuid.Nil, groupcall.AnswerMethodHangupOthers, "", nil); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("chained call and groupcall failures hangup the groupcall", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.util.EXPECT().UUIDCreate().Return(covNewID).Times(2)
		expectCreateOK(m, ctx)

		var wg sync.WaitGroup
		wg.Add(3)
		m.req.EXPECT().RegistrarV1ContactList(ctx, gomock.Any()).DoAndReturn(func(context.Context, map[string]any) ([]rmastcontact.AstContact, error) {
			wg.Done()
			return nil, errCov
		})
		m.req.EXPECT().CallV1CallCreateWithID(ctx, gomock.Any(), covCustomer, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), covID, false, false, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, *commonaddress.Address, *commonaddress.Address, uuid.UUID, bool, bool, string, map[string]any, map[string]string) (*call.Call, error) {
				wg.Done()
				return nil, errCov
			})
		// both failing goroutines call HangupGroupcall => DecreaseGroupcallCount (set of 2 calls, order independent)
		m.db.EXPECT().GroupcallDecreaseGroupcallCount(ctx, covID).DoAndReturn(func(context.Context, uuid.UUID) error {
			wg.Done()
			return errCov
		}).Times(2)
		wg.Add(1)

		res, err := h.startRingall(ctx, covID, covCustomer, &covTel, []commonaddress.Address{covTel, covExt}, uuid.Nil, uuid.Nil, uuid.Nil, groupcall.AnswerMethodHangupOthers, "", nil)
		if err != nil || res == nil {
			t.Fatalf("expect ok, got %v", err)
		}
		waitCov(t, &wg)
	})
}

func Test_startLinear_cov(t *testing.T) {
	ctx := context.Background()

	t.Run("create error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.util.EXPECT().UUIDCreate().Return(covNewID)
		m.db.EXPECT().GroupcallCreate(ctx, gomock.Any()).Return(errCov)

		if _, err := h.startLinear(ctx, covID, covCustomer, uuid.Nil, &covTel, []commonaddress.Address{covTel}, uuid.Nil, uuid.Nil, groupcall.AnswerMethodHangupOthers, "", nil); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("call creation failure", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.util.EXPECT().UUIDCreate().Return(covNewID)
		expectCreateOK(m, ctx)

		var wg sync.WaitGroup
		wg.Add(2)
		m.req.EXPECT().CallV1CallCreateWithID(ctx, covNewID, covCustomer, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), covID, false, false, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, *commonaddress.Address, *commonaddress.Address, uuid.UUID, bool, bool, string, map[string]any, map[string]string) (*call.Call, error) {
				wg.Done()
				return nil, errCov
			})
		m.db.EXPECT().GroupcallDecreaseGroupcallCount(ctx, covID).DoAndReturn(func(context.Context, uuid.UUID) error {
			wg.Done()
			return errCov
		})

		if _, err := h.startLinear(ctx, covID, covCustomer, uuid.Nil, &covTel, []commonaddress.Address{covTel}, uuid.Nil, uuid.Nil, groupcall.AnswerMethodHangupOthers, "", nil); err != nil {
			t.Fatalf("expect ok, got %v", err)
		}
		waitCov(t, &wg)
	})

	t.Run("groupcall creation failure", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.util.EXPECT().UUIDCreate().Return(covNewID)
		expectCreateOK(m, ctx)

		var wg sync.WaitGroup
		wg.Add(2)
		m.req.EXPECT().RegistrarV1ContactList(ctx, gomock.Any()).DoAndReturn(func(context.Context, map[string]any) ([]rmastcontact.AstContact, error) {
			wg.Done()
			return nil, errCov
		})
		m.db.EXPECT().GroupcallDecreaseGroupcallCount(ctx, covID).DoAndReturn(func(context.Context, uuid.UUID) error {
			wg.Done()
			return errCov
		})

		if _, err := h.startLinear(ctx, covID, covCustomer, uuid.Nil, &covTel, []commonaddress.Address{covExt}, uuid.Nil, uuid.Nil, groupcall.AnswerMethodHangupOthers, "", nil); err != nil {
			t.Fatalf("expect ok, got %v", err)
		}
		waitCov(t, &wg)
	})
}

func Test_startWithDestination_cov(t *testing.T) {
	ctx := context.Background()

	t.Run("unsupported destination type", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.util.EXPECT().UUIDCreate().Return(covNewID) // nil id => new id
		dest := commonaddress.Address{Type: commonaddress.TypeNone}
		if _, err := h.startWithDestination(ctx, uuid.Nil, covCustomer, uuid.Nil, &covTel, &dest, uuid.Nil, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers, "", nil); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("create error with owner lookup failure", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.util.EXPECT().UUIDCreate().Return(covNewID)
		m.req.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, gomock.Any(), covCustomer, gomock.Any()).Return(nil, errCov)
		m.db.EXPECT().GroupcallCreate(ctx, gomock.Any()).Return(errCov)

		if _, err := h.startWithDestination(ctx, covID, covCustomer, uuid.Nil, &covTel, &covTel, uuid.Nil, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers, "", nil); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("chained call failure hangs up the call", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.util.EXPECT().UUIDCreate().Return(covNewID)
		m.req.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, gomock.Any(), covCustomer, gomock.Any()).Return(nil, nil)
		expectCreateOK(m, ctx)

		var wg sync.WaitGroup
		wg.Add(2)
		m.req.EXPECT().CallV1CallCreateWithID(ctx, covNewID, covCustomer, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), covID, false, false, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, *commonaddress.Address, *commonaddress.Address, uuid.UUID, bool, bool, string, map[string]any, map[string]string) (*call.Call, error) {
				wg.Done()
				return nil, errCov
			})
		m.db.EXPECT().GroupcallDecreaseCallCount(ctx, covID).DoAndReturn(func(context.Context, uuid.UUID) error {
			wg.Done()
			return errCov
		})

		if _, err := h.startWithDestination(ctx, covID, covCustomer, uuid.Nil, &covTel, &covTel, uuid.Nil, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers, "", nil); err != nil {
			t.Fatalf("expect ok, got %v", err)
		}
		waitCov(t, &wg)
	})

	t.Run("chained groupcall failure from agent addresses", func(t *testing.T) {
		h, m := newCovHandler(t)
		agentID := uuid.FromStringOrNil("c0a00000-0000-4000-8000-0000000000aa")
		dest := commonaddress.Address{Type: commonaddress.TypeAgent, Target: agentID.String()}
		ag := &agagent.Agent{
			Identity:  commonidentity.Identity{ID: agentID, CustomerID: covCustomer},
			Addresses: []commonaddress.Address{covExt},
		}

		m.util.EXPECT().UUIDCreate().Return(covNewID)
		// dial addresses lookup and owner lookup
		m.req.EXPECT().AgentV1AgentGet(ctx, agentID).Return(ag, nil).Times(2)
		expectCreateOK(m, ctx)

		var wg sync.WaitGroup
		wg.Add(1)
		m.req.EXPECT().CallV1GroupcallCreate(ctx, covNewID, covCustomer, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), covID, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers, gomock.Any(), gomock.Any()).DoAndReturn(
			func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, commonaddress.Address, []commonaddress.Address, uuid.UUID, uuid.UUID, groupcall.RingMethod, groupcall.AnswerMethod, string, map[string]string) (*groupcall.Groupcall, error) {
				wg.Done()
				return nil, errCov
			})

		if _, err := h.startWithDestination(ctx, covID, covCustomer, uuid.Nil, &covTel, &dest, uuid.Nil, uuid.Nil, groupcall.RingMethodRingAll, groupcall.AnswerMethodHangupOthers, "", nil); err != nil {
			t.Fatalf("expect ok, got %v", err)
		}
		waitCov(t, &wg)
	})
}
