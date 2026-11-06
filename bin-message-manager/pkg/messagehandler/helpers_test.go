package messagehandler

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/dbhandler"
)

// outs builds provider results from outcomes.
func outs(o ...SendOutcome) []SendResult {
	res := make([]SendResult, 0, len(o))
	for _, v := range o {
		res = append(res, SendResult{Outcome: v})
	}

	return res
}

// memDB is an in memory message row behind a MockDBHandler. It behaves like the real compare and set: a row has a
// version, tm_update is nil while the version is 0, and MessageUpdateTargetsIf writes only if the caller read the
// current version.
type memDB struct {
	mu sync.Mutex

	msg     *message.Message
	version int

	// hooks, called without the lock before a conditional write is evaluated; the call number starts at 1.
	beforeIf func(call int)

	ifCalls           int
	mergeCalls        int
	getCalls          int
	forceConflicts    int   // the next n conditional writes answer false without writing
	failIfErr         error // the conditional writes answer this error
	failMergeErr      error // the unconditional write answers this error
	failGetFromDB     error // the reads answer this error
	failGetAfterWrite bool  // the reads answer an error once a row was written
}

var memDBBase = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

func (d *memDB) stamp() *time.Time {
	if d.version == 0 {
		return nil
	}
	t := memDBBase.Add(time.Duration(d.version) * time.Second)
	return &t
}

func (d *memDB) snapshot() *message.Message {
	cp := *d.msg
	cp.Targets = append([]target.Target(nil), d.msg.Targets...)
	cp.TMUpdate = d.stamp()
	return &cp
}

func (d *memDB) targets() []target.Target {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]target.Target(nil), d.msg.Targets...)
}

func (d *memDB) providerName() message.ProviderName {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.msg.ProviderName
}

func (d *memDB) writes() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.version
}

// setTargets changes the stored targets as another writer would, and bumps the version.
func (d *memDB) setTargets(f func(targets []target.Target)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	f(d.msg.Targets)
	d.version++
}

func newMemDB(db *dbhandler.MockDBHandler, m *message.Message) *memDB {
	d := &memDB{msg: m}

	db.EXPECT().MessageGetFromDB(gomock.Any(), m.ID).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*message.Message, error) {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.getCalls++
		if d.failGetFromDB != nil {
			return nil, d.failGetFromDB
		}
		if d.failGetAfterWrite && d.version > 0 {
			return nil, errors.New("read failed")
		}
		return d.snapshot(), nil
	}).AnyTimes()

	db.EXPECT().MessageUpdateTargetsIf(gomock.Any(), m.ID, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, prev *time.Time, provider *message.ProviderName, targets []target.Target) (bool, error) {
			d.mu.Lock()
			d.ifCalls++
			call := d.ifCalls
			hook := d.beforeIf
			d.mu.Unlock()
			if hook != nil {
				hook(call)
			}

			d.mu.Lock()
			defer d.mu.Unlock()
			if d.failIfErr != nil {
				return false, d.failIfErr
			}
			if d.forceConflicts > 0 {
				d.forceConflicts--
				return false, nil
			}

			cur := d.stamp()
			if (prev == nil) != (cur == nil) || (prev != nil && !prev.Equal(*cur)) {
				return false, nil
			}

			d.msg.Targets = append([]target.Target(nil), targets...)
			if provider != nil {
				d.msg.ProviderName = *provider
			}
			d.version++
			return true, nil
		}).AnyTimes()

	db.EXPECT().MessageUpdateTargets(gomock.Any(), m.ID, gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, provider message.ProviderName, targets []target.Target) error {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.mergeCalls++
			if d.failMergeErr != nil {
				return d.failMergeErr
			}
			d.msg.Targets = append([]target.Target(nil), targets...)
			d.msg.ProviderName = provider
			d.version++
			return nil
		}).AnyTimes()

	return d
}

// storedMessage returns an outbound message with the given targets.
func storedMessage(targets []target.Target) *message.Message {
	return &message.Message{
		Identity:     commonidentity.Identity{ID: testMessageID, CustomerID: testCustomerID},
		Type:         message.TypeSMS,
		Targets:      targets,
		ProviderName: message.ProviderNameTelnyx,
		Direction:    message.DirectionOutbound,
	}
}

// counterValue reads a counter of the default registry by name and labels.
func promValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()

	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Could not gather the metrics: %v", err)
	}

	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
	metric:
		for _, m := range mf.GetMetric() {
			got := map[string]string{}
			for _, l := range m.GetLabel() {
				got[l.GetName()] = l.GetValue()
			}
			for k, v := range labels {
				if got[k] != v {
					continue metric
				}
			}
			return m.GetCounter().GetValue()
		}
	}

	return 0
}

// events counts the message_updated events and keeps the published bodies.
type events struct {
	mu     sync.Mutex
	bodies []*message.Message
	ctxErr []error // the error of the context of each publish call
}

func (e *events) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.bodies)
}

func expectEvents(n *notifyhandler.MockNotifyHandler) *events {
	e := &events{}
	n.EXPECT().PublishWebhookEvent(gomock.Any(), testCustomerID, message.EventTypeMessageUpdated, gomock.Any()).Do(
		func(ctx context.Context, _ uuid.UUID, _ string, body any) {
			e.mu.Lock()
			defer e.mu.Unlock()
			e.ctxErr = append(e.ctxErr, ctx.Err())
			if m, ok := body.(*message.Message); ok {
				e.bodies = append(e.bodies, m)
			}
		}).AnyTimes()
	return e
}

func newTestUtil(mc *gomock.Controller) *utilhandler.MockUtilHandler {
	u := utilhandler.NewMockUtilHandler(mc)
	now := time.Date(2026, 11, 6, 1, 2, 3, 0, time.UTC)
	u.EXPECT().TimeNow().DoAndReturn(func() *time.Time { t := now; return &t }).AnyTimes()
	return u
}

func addr(number string) *commonaddress.Address {
	return &commonaddress.Address{Type: commonaddress.TypeTel, Target: number}
}
