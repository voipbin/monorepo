package groupcallhandler

import (
	"context"
	"sync"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	rmastcontact "monorepo/bin-registrar-manager/models/astcontact"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/groupcall"
)

var covAgentID = uuid.FromStringOrNil("c0a00000-0000-4000-8000-0000000000aa")

func covGroupcall() *groupcall.Groupcall {
	return &groupcall.Groupcall{
		Identity:     commonidentity.Identity{ID: covID, CustomerID: covCustomer},
		Source:       &covTel,
		Destinations: []commonaddress.Address{covTel, covTel},
		Anonymous:    "anonymous",
	}
}

func Test_dialNextDestination_errors_cov(t *testing.T) {
	ctx := context.Background()

	t.Run("call destination error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.util.EXPECT().UUIDCreate().Return(covNewID)
		m.db.EXPECT().GroupcallSetCallIDsAndCallCountAndDialIndex(ctx, covID, gomock.Any(), gomock.Any(), 1).Return(errCov)
		if _, err := h.dialNextDestination(ctx, covGroupcall()); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("groupcall destination error", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := covGroupcall()
		gc.Destinations = []commonaddress.Address{covTel, covExt}
		m.req.EXPECT().RegistrarV1ContactList(ctx, gomock.Any()).Return(nil, errCov)
		if _, err := h.dialNextDestination(ctx, gc); err == nil {
			t.Errorf("expect error")
		}
	})
}

func Test_dialNextDestinationGroupcall_cov(t *testing.T) {
	ctx := context.Background()
	contacts := []rmastcontact.AstContact{{URI: "sip:ext1@203.0.113.1"}}

	t.Run("agent destination error", func(t *testing.T) {
		h, _ := newCovHandler(t)
		dest := commonaddress.Address{Type: commonaddress.TypeAgent, Target: "invalid"}
		if _, err := h.dialNextDestinationGroupcall(ctx, covGroupcall(), &dest); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("extension destination error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().RegistrarV1ContactList(ctx, gomock.Any()).Return(nil, errCov)
		if _, err := h.dialNextDestinationGroupcall(ctx, covGroupcall(), &covExt); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		h, _ := newCovHandler(t)
		if _, err := h.dialNextDestinationGroupcall(ctx, covGroupcall(), &covTel); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("update error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().RegistrarV1ContactList(ctx, gomock.Any()).Return(contacts, nil)
		m.util.EXPECT().UUIDCreate().Return(covNewID)
		m.db.EXPECT().GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex(ctx, covID, gomock.Any(), gomock.Any(), 1).Return(errCov)
		if _, err := h.dialNextDestinationGroupcall(ctx, covGroupcall(), &covExt); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("chained groupcall failure hangs up the groupcall", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := covGroupcall()
		m.req.EXPECT().RegistrarV1ContactList(ctx, gomock.Any()).Return(contacts, nil)
		m.util.EXPECT().UUIDCreate().Return(covNewID)
		m.db.EXPECT().GroupcallSetGroupcallIDsAndGroupcallCountAndDialIndex(ctx, covID, []uuid.UUID{covNewID}, 1, 1).Return(nil)
		m.db.EXPECT().GroupcallGet(ctx, covID).Return(gc, nil)

		var wg sync.WaitGroup
		wg.Add(2)
		m.req.EXPECT().CallV1GroupcallCreate(ctx, covNewID, covCustomer, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), covID, groupcall.RingMethodRingAll, gomock.Any(), "anonymous", gomock.Any()).DoAndReturn(
			func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, commonaddress.Address, []commonaddress.Address, uuid.UUID, uuid.UUID, groupcall.RingMethod, groupcall.AnswerMethod, string, map[string]string) (*groupcall.Groupcall, error) {
				wg.Done()
				return nil, errCov
			})
		m.db.EXPECT().GroupcallDecreaseGroupcallCount(ctx, covID).DoAndReturn(func(context.Context, uuid.UUID) error {
			wg.Done()
			return errCov
		})

		if _, err := h.dialNextDestinationGroupcall(ctx, gc, &covExt); err != nil {
			t.Fatalf("expect ok, got %v", err)
		}
		waitCov(t, &wg)
	})
}

func Test_dialNextDestinationCall_cov(t *testing.T) {
	ctx := context.Background()

	t.Run("chained call failure hangs up the call", func(t *testing.T) {
		h, m := newCovHandler(t)
		gc := covGroupcall()
		m.util.EXPECT().UUIDCreate().Return(covNewID)
		m.db.EXPECT().GroupcallSetCallIDsAndCallCountAndDialIndex(ctx, covID, []uuid.UUID{covNewID}, 1, 1).Return(nil)
		m.db.EXPECT().GroupcallGet(ctx, covID).Return(gc, nil)

		var wg sync.WaitGroup
		wg.Add(2)
		m.req.EXPECT().CallV1CallCreateWithID(ctx, covNewID, covCustomer, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), covID, false, false, "anonymous", gomock.Any(), gomock.Any()).DoAndReturn(
			func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, *commonaddress.Address, *commonaddress.Address, uuid.UUID, bool, bool, string, map[string]any, map[string]string) (*call.Call, error) {
				wg.Done()
				return nil, errCov
			})
		m.db.EXPECT().GroupcallDecreaseCallCount(ctx, covID).DoAndReturn(func(context.Context, uuid.UUID) error {
			wg.Done()
			return errCov
		})

		if _, err := h.dialNextDestinationCall(ctx, gc, &covTel); err != nil {
			t.Fatalf("expect ok, got %v", err)
		}
		waitCov(t, &wg)
	})
}

func Test_getDialAddressesAndRingMethod_cov(t *testing.T) {
	ctx := context.Background()

	t.Run("agent error", func(t *testing.T) {
		h, _ := newCovHandler(t)
		dest := commonaddress.Address{Type: commonaddress.TypeAgent, Target: "invalid"}
		if _, _, err := h.getDialAddressesAndRingMethod(ctx, covCustomer, &dest); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("extension error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().RegistrarV1ContactList(ctx, gomock.Any()).Return(nil, errCov)
		if _, _, err := h.getDialAddressesAndRingMethod(ctx, covCustomer, &covExt); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		h, _ := newCovHandler(t)
		dest := commonaddress.Address{Type: commonaddress.TypeNone}
		if _, _, err := h.getDialAddressesAndRingMethod(ctx, covCustomer, &dest); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("tel", func(t *testing.T) {
		h, _ := newCovHandler(t)
		res, rm, err := h.getDialAddressesAndRingMethod(ctx, covCustomer, &covTel)
		if err != nil || len(res) != 1 || rm != groupcall.RingMethodRingAll {
			t.Errorf("unexpected: %v, %v, %v", res, rm, err)
		}
	})
}

func Test_getDialDestinationsAddressTypeExtension_cov(t *testing.T) {
	ctx := context.Background()

	t.Run("no registered contact", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().RegistrarV1ContactList(ctx, gomock.Any()).Return([]rmastcontact.AstContact{}, nil)
		if _, err := h.getDialDestinationsAddressTypeExtension(ctx, covCustomer, &covExt); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("uri is unescaped", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().RegistrarV1ContactList(ctx, gomock.Any()).Return([]rmastcontact.AstContact{{URI: "sip:ext1@203.0.113.1^3Btransport=udp"}}, nil)
		res, err := h.getDialDestinationsAddressTypeExtension(ctx, covCustomer, &covExt)
		if err != nil || len(res) != 1 || res[0].Target != "sip:ext1@203.0.113.1;transport=udp" {
			t.Errorf("unexpected: %v, %v", res, err)
		}
	})
}

func Test_getDialDestinationsAddressAndRingMethodTypeAgent_cov(t *testing.T) {
	ctx := context.Background()
	dest := commonaddress.Address{Type: commonaddress.TypeAgent, Target: covAgentID.String()}

	t.Run("agent get error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().AgentV1AgentGet(ctx, covAgentID).Return(nil, errCov)
		if _, _, err := h.getDialDestinationsAddressAndRingMethodTypeAgent(ctx, covCustomer, &dest); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("customer mismatch", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().AgentV1AgentGet(ctx, covAgentID).Return(&amagent.Agent{Identity: commonidentity.Identity{ID: covAgentID, CustomerID: covOther}}, nil)
		if _, _, err := h.getDialDestinationsAddressAndRingMethodTypeAgent(ctx, covCustomer, &dest); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("linear ring method", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().AgentV1AgentGet(ctx, covAgentID).Return(&amagent.Agent{
			Identity:   commonidentity.Identity{ID: covAgentID, CustomerID: covCustomer},
			RingMethod: amagent.RingMethodLinear,
			Addresses:  []commonaddress.Address{covTel},
		}, nil)
		res, rm, err := h.getDialDestinationsAddressAndRingMethodTypeAgent(ctx, covCustomer, &dest)
		if err != nil || len(res) != 1 || rm != groupcall.RingMethodLinear {
			t.Errorf("unexpected: %v, %v, %v", res, rm, err)
		}
	})
}

func Test_getAddressOwner_cov(t *testing.T) {
	ctx := context.Background()
	agentDest := commonaddress.Address{Type: commonaddress.TypeAgent, Target: covAgentID.String()}
	okAgent := &amagent.Agent{Identity: commonidentity.Identity{ID: covAgentID, CustomerID: covCustomer}}

	t.Run("agent type get error", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().AgentV1AgentGet(ctx, covAgentID).Return(nil, errCov)
		if _, _, err := h.getAddressOwner(ctx, covCustomer, &agentDest); err == nil {
			t.Errorf("expect error")
		}
	})

	t.Run("agent type ok", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().AgentV1AgentGet(ctx, covAgentID).Return(okAgent, nil)
		ot, id, err := h.getAddressOwner(ctx, covCustomer, &agentDest)
		if err != nil || ot != commonidentity.OwnerTypeAgent || id != covAgentID {
			t.Errorf("unexpected: %v, %v, %v", ot, id, err)
		}
	})

	t.Run("address lookup error is ignored", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, gomock.Any(), covCustomer, gomock.Any()).Return(nil, errCov)
		ot, id, err := h.getAddressOwner(ctx, covCustomer, &covTel)
		if err != nil || ot != commonidentity.OwnerTypeNone || id != uuid.Nil {
			t.Errorf("unexpected: %v, %v, %v", ot, id, err)
		}
	})

	t.Run("nil agent", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, gomock.Any(), covCustomer, gomock.Any()).Return(nil, nil)
		ot, _, err := h.getAddressOwner(ctx, covCustomer, &covTel)
		if err != nil || ot != commonidentity.OwnerTypeNone {
			t.Errorf("unexpected: %v, %v", ot, err)
		}
	})

	t.Run("customer mismatch", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, gomock.Any(), covCustomer, gomock.Any()).Return(&amagent.Agent{Identity: commonidentity.Identity{ID: covAgentID, CustomerID: covOther}}, nil)
		ot, _, _ := h.getAddressOwner(ctx, covCustomer, &covTel)
		if ot != commonidentity.OwnerTypeNone {
			t.Errorf("unexpected owner type: %v", ot)
		}
	})

	t.Run("address lookup ok", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.req.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(ctx, gomock.Any(), covCustomer, gomock.Any()).Return(okAgent, nil)
		ot, id, err := h.getAddressOwner(ctx, covCustomer, &covTel)
		if err != nil || ot != commonidentity.OwnerTypeAgent || id != covAgentID {
			t.Errorf("unexpected: %v, %v, %v", ot, id, err)
		}
	})
}
