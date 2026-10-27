package confbridgehandler

import (
	"errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
)

var (
	errCov2     = errors.New("cov2 error")
	cov2CBID    = uuid.FromStringOrNil("c2b00000-0000-4000-8000-000000000001")
	cov2CallID  = uuid.FromStringOrNil("c2b00000-0000-4000-8000-000000000002")
	cov2CallID2 = uuid.FromStringOrNil("c2b00000-0000-4000-8000-000000000003")
	cov2ExtID   = uuid.FromStringOrNil("c2b00000-0000-4000-8000-000000000004")
	cov2RecID   = uuid.FromStringOrNil("c2b00000-0000-4000-8000-000000000005")
)

type cov2Mocks struct {
	util    *utilhandler.MockUtilHandler
	req     *requesthandler.MockRequestHandler
	notify  *notifyhandler.MockNotifyHandler
	db      *dbhandler.MockDBHandler
	cache   *cachehandler.MockCacheHandler
	channel *channelhandler.MockChannelHandler
	bridge  *bridgehandler.MockBridgeHandler
	rec     *recordinghandler.MockRecordingHandler
	ext     *externalmediahandler.MockExternalMediaHandler
}

func newCov2Handler(t *testing.T) (*confbridgeHandler, *cov2Mocks) {
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &cov2Mocks{
		util:    utilhandler.NewMockUtilHandler(mc),
		req:     requesthandler.NewMockRequestHandler(mc),
		notify:  notifyhandler.NewMockNotifyHandler(mc),
		db:      dbhandler.NewMockDBHandler(mc),
		cache:   cachehandler.NewMockCacheHandler(mc),
		channel: channelhandler.NewMockChannelHandler(mc),
		bridge:  bridgehandler.NewMockBridgeHandler(mc),
		rec:     recordinghandler.NewMockRecordingHandler(mc),
		ext:     externalmediahandler.NewMockExternalMediaHandler(mc),
	}
	h := &confbridgeHandler{
		utilHandler:          m.util,
		reqHandler:           m.req,
		notifyHandler:        m.notify,
		db:                   m.db,
		cache:                m.cache,
		channelHandler:       m.channel,
		bridgeHandler:        m.bridge,
		recordingHandler:     m.rec,
		externalMediaHandler: m.ext,
	}
	return h, m
}

func Test_NewConfbridgeHandler_cov2_fields(t *testing.T) {
	_, m := newCov2Handler(t)

	res := NewConfbridgeHandler(m.req, m.notify, m.db, m.cache, m.channel, m.bridge, m.rec, m.ext)
	h, ok := res.(*confbridgeHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.utilHandler == nil {
		t.Errorf("Expected util handler to be set")
	}
	if h.reqHandler != m.req || h.notifyHandler != m.notify || h.db != m.db || h.cache != m.cache ||
		h.channelHandler != m.channel || h.bridgeHandler != m.bridge || h.recordingHandler != m.rec ||
		h.externalMediaHandler != m.ext {
		t.Errorf("Wrong dependency wiring")
	}
}

// cov2CB returns a confbridge with the default test id, modified by the given function.
func cov2CB(modify func(c *confbridge.Confbridge)) *confbridge.Confbridge {
	res := &confbridge.Confbridge{}
	res.ID = cov2CBID
	if modify != nil {
		modify(res)
	}
	return res
}

// gomockAnyUUID matches any uuid.UUID argument.
func gomockAnyUUID() gomock.Matcher {
	return gomock.AssignableToTypeOf(uuid.UUID{})
}

func uuidNil() uuid.UUID {
	return uuid.Nil
}

func identityOf(id uuid.UUID) commonidentity.Identity {
	return commonidentity.Identity{ID: id}
}
