package pushringhandler

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	pmdevice "monorepo/bin-push-manager/models/device"
	pmnotification "monorepo/bin-push-manager/models/notification"
	rmastcontact "monorepo/bin-registrar-manager/models/astcontact"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/models/pushring"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

// fixed ids of the tests.
var (
	tCustomerID  = uuid.FromStringOrNil("e1000000-0000-4000-8000-000000000001")
	tAgentID     = uuid.FromStringOrNil("e1000000-0000-4000-8000-000000000002")
	tExtensionID = uuid.FromStringOrNil("e1000000-0000-4000-8000-000000000003")
	tConfbridge  = uuid.FromStringOrNil("e1000000-0000-4000-8000-000000000004")
	tMasterCall  = uuid.FromStringOrNil("e1000000-0000-4000-8000-000000000005")
	tFlowID      = uuid.FromStringOrNil("e1000000-0000-4000-8000-000000000006")
	tLegCallA    = uuid.FromStringOrNil("e1000000-0000-4000-8000-0000000000a1")
	tLegCallB    = uuid.FromStringOrNil("e1000000-0000-4000-8000-0000000000a2")
	tDeviceA     = uuid.FromStringOrNil("e1000000-0000-4000-8000-0000000000d1")
	tDeviceB     = uuid.FromStringOrNil("e1000000-0000-4000-8000-0000000000d2")
)

type hangupRecord struct {
	id     uuid.UUID
	reason call.HangupReason
}

type sendRecord struct {
	deviceID uuid.UUID
	kind     pmnotification.Kind
	data     map[string]string
}

// env is the set of the fakes of one test. The cache is a real cache handler on a fake Redis, so the state machine of the
// session is the real one.
type env struct {
	t     *testing.T
	req   *requesthandler.MockRequestHandler
	db    *dbhandler.MockDBHandler
	cache cachehandler.CacheHandler
	mr    *miniredis.Miniredis
	h     *pushRingHandler

	now time.Time

	mu      sync.Mutex
	hangups []hangupRecord
	sends   []sendRecord
	expires map[int]int // delay -> count
	// terminated holds the ids of the terminated confbridges
	terminated []uuid.UUID

	// answers of the lookups of the start
	agent      *amagent.Agent
	agentErr   error
	agentBlock bool // the lookup does not answer until its context ends
	devices    []pmdevice.Device
	deviceErr  error
	contacts   []rmastcontact.AstContact
	contactErr error
	flagErr    error
	cb         *confbridge.Confbridge
	sendResult func(kind pmnotification.Kind, deviceID uuid.UUID) (*pmnotification.SendResult, error)

	calls      map[uuid.UUID]*call.Call
	groupcalls []*groupcall.Groupcall
}

func newEnv(t *testing.T) *env {
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mr := miniredis.RunT(t)
	cache := cachehandler.NewHandler(mr.Addr(), "", 0)

	e := &env{
		t:       t,
		req:     requesthandler.NewMockRequestHandler(mc),
		db:      dbhandler.NewMockDBHandler(mc),
		cache:   cache,
		mr:      mr,
		now:     time.Now().Truncate(time.Millisecond),
		expires: map[int]int{},
		agent: &amagent.Agent{
			Identity: commonidentity.Identity{ID: tAgentID, CustomerID: tCustomerID},
		},
		devices: []pmdevice.Device{
			{Identity: commonidentity.Identity{ID: tDeviceA, CustomerID: tCustomerID}, AgentID: tAgentID},
			{Identity: commonidentity.Identity{ID: tDeviceB, CustomerID: tCustomerID}, AgentID: tAgentID},
		},
		contacts: []rmastcontact.AstContact{{ID: "contact-1"}},
		cb: &confbridge.Confbridge{
			Identity: commonidentity.Identity{ID: tConfbridge, CustomerID: tCustomerID},
			Type:     confbridge.TypeConnect,
		},
		sendResult: func(pmnotification.Kind, uuid.UUID) (*pmnotification.SendResult, error) {
			return &pmnotification.SendResult{DeviceCount: 1, SentCount: 1}, nil
		},
		calls: map[uuid.UUID]*call.Call{},
	}

	h := NewPushRingHandler(e.req, e.db, e.cache).(*pushRingHandler)
	h.now = func() time.Time { return e.now }
	h.SetHangup(func(ctx context.Context, id uuid.UUID, reason call.HangupReason) error {
		e.mu.Lock()
		defer e.mu.Unlock()
		e.hangups = append(e.hangups, hangupRecord{id: id, reason: reason})
		return nil
	})
	e.h = h

	e.req.EXPECT().RegistrarV1ContactList(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, filters map[string]any) ([]rmastcontact.AstContact, error) {
			return e.contacts, e.contactErr
		}).AnyTimes()
	e.req.EXPECT().AgentV1AgentGetByCustomerIDAndAddress(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, timeout int, customerID uuid.UUID, addr commonaddress.Address) (*amagent.Agent, error) {
			if e.agentBlock {
				<-ctx.Done()
				return nil, ctx.Err()
			}
			return e.agent, e.agentErr
		}).AnyTimes()
	e.req.EXPECT().PushV1DeviceList(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, customerID, agentID uuid.UUID) ([]pmdevice.Device, error) {
			return e.devices, e.deviceErr
		}).AnyTimes()
	e.req.EXPECT().PushV1NotificationSend(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, customerID, agentID, deviceID uuid.UUID, kind pmnotification.Kind, data map[string]string) (*pmnotification.SendResult, error) {
			e.mu.Lock()
			e.sends = append(e.sends, sendRecord{deviceID: deviceID, kind: kind, data: data})
			e.mu.Unlock()
			return e.sendResult(kind, deviceID)
		}).AnyTimes()
	e.req.EXPECT().CallV1PushRingExpire(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, id uuid.UUID, delay int) error {
			e.mu.Lock()
			e.expires[delay]++
			e.mu.Unlock()
			return nil
		}).AnyTimes()
	e.req.EXPECT().CallV1ConfbridgeTerminate(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
			e.mu.Lock()
			e.terminated = append(e.terminated, id)
			e.mu.Unlock()
			return &confbridge.Confbridge{}, nil
		}).AnyTimes()

	e.db.EXPECT().ConfbridgeGet(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error) {
			return e.cb, nil
		}).AnyTimes()
	e.db.EXPECT().ConfbridgeSetFlags(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, id uuid.UUID, flags []confbridge.Flag) error {
			if e.flagErr == nil {
				e.cb.Flags = flags
			}
			return e.flagErr
		}).AnyTimes()
	e.db.EXPECT().CallGet(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, id uuid.UUID) (*call.Call, error) {
			if c, ok := e.calls[id]; ok {
				return c, nil
			}
			return nil, dbhandler.ErrNotFound
		}).AnyTimes()
	e.db.EXPECT().CallList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, size uint64, token string, filters map[call.Field]any) ([]*call.Call, error) {
			res := []*call.Call{}
			for _, c := range e.calls {
				if v, ok := filters[call.FieldMasterCallID]; ok && c.MasterCallID != v.(uuid.UUID) {
					continue
				}
				if v, ok := filters[call.FieldFlowID]; ok && c.FlowID != v.(uuid.UUID) {
					continue
				}
				res = append(res, c)
			}
			return res, nil
		}).AnyTimes()
	e.db.EXPECT().GroupcallList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, size uint64, token string, filters map[groupcall.Field]any) ([]*groupcall.Groupcall, error) {
			res := []*groupcall.Groupcall{}
			for _, g := range e.groupcalls {
				if g.MasterCallID != filters[groupcall.FieldMasterCallID].(uuid.UUID) || g.FlowID != filters[groupcall.FieldFlowID].(uuid.UUID) {
					continue
				}
				res = append(res, g)
			}
			return res, nil
		}).AnyTimes()

	return e
}

// start starts a session for the standard call and waits for the notifications.
func (e *env) start() *pushring.StartResult {
	e.t.Helper()

	res, err := e.h.Start(context.Background(), tCustomerID,
		commonaddress.Address{Type: commonaddress.TypeExtension, Target: tExtensionID.String(), TargetName: "2000"},
		tConfbridge, tMasterCall, tFlowID,
		commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550100"},
	)
	if err != nil {
		e.t.Fatalf("start: %v", err)
	}
	e.h.wg.Wait()

	return res
}

// addCall registers a call that the db fake returns.
func (e *env) addCall(id, flowID uuid.UUID) *call.Call {
	c := &call.Call{
		Identity:     commonidentity.Identity{ID: id, CustomerID: tCustomerID},
		FlowID:       flowID,
		MasterCallID: tMasterCall,
	}
	e.calls[id] = c
	return c
}

// ticketOf returns the ticket that was pushed to the device.
func (e *env) ticketOf(deviceID uuid.UUID) string {
	e.t.Helper()

	e.mu.Lock()
	defer e.mu.Unlock()
	for _, s := range e.sends {
		if s.kind == pmnotification.KindCallIncoming && s.deviceID == deviceID {
			addr := s.data["join_address"]
			return strings.TrimSuffix(strings.TrimPrefix(addr, "sip:"), "@sip.voipbin.net")
		}
	}
	e.t.Fatalf("no incoming notification for the device %s", deviceID)
	return ""
}

func (e *env) sendsOf(kind pmnotification.Kind) []sendRecord {
	e.mu.Lock()
	defer e.mu.Unlock()

	res := []sendRecord{}
	for _, s := range e.sends {
		if s.kind == kind {
			res = append(res, s)
		}
	}
	return res
}

func (e *env) hangupIDs() map[uuid.UUID]call.HangupReason {
	e.mu.Lock()
	defer e.mu.Unlock()

	res := map[uuid.UUID]call.HangupReason{}
	for _, hu := range e.hangups {
		res[hu.id] = hu.reason
	}
	return res
}

func (e *env) session(id uuid.UUID) *pushring.Session {
	e.t.Helper()

	s, err := e.cache.PushRingSessionGet(context.Background(), id)
	if err != nil {
		e.t.Fatalf("session get: %v", err)
	}
	return s
}
