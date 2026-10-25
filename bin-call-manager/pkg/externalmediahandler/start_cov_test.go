package externalmediahandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

var errCov = fmt.Errorf("cov error")

type covMocks struct {
	util    *utilhandler.MockUtilHandler
	req     *requesthandler.MockRequestHandler
	db      *dbhandler.MockDBHandler
	channel *channelhandler.MockChannelHandler
	bridge  *bridgehandler.MockBridgeHandler
	cache   *cachehandler.MockCacheHandler
}

func newCovHandler(t *testing.T) (*externalMediaHandler, *covMocks) {
	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	m := &covMocks{
		util:    utilhandler.NewMockUtilHandler(mc),
		req:     requesthandler.NewMockRequestHandler(mc),
		db:      dbhandler.NewMockDBHandler(mc),
		channel: channelhandler.NewMockChannelHandler(mc),
		bridge:  bridgehandler.NewMockBridgeHandler(mc),
		cache:   cachehandler.NewMockCacheHandler(mc),
	}
	h := &externalMediaHandler{
		utilHandler:    m.util,
		reqHandler:     m.req,
		db:             m.db,
		channelHandler: m.channel,
		bridgeHandler:  m.bridge,
		cache:          m.cache,
		asteriskWSPort: 8088,
	}
	return h, m
}

func Test_startExternalMedia_cov(t *testing.T) {
	id := uuid.FromStringOrNil("11111111-0001-11f0-8000-000000000001")
	refID := uuid.FromStringOrNil("11111111-0002-11f0-8000-000000000002")
	extChID := uuid.FromStringOrNil("11111111-0003-11f0-8000-000000000003")
	asteriskID := "42:01:0a:a4:00:05"

	tests := []struct {
		name string

		id            uuid.UUID
		encapsulation externalmedia.Encapsulation
		transport     externalmedia.Transport
		connType      string
		format        string

		// mock behaviors
		setErr          error // first ExternalMediaSet (Create)
		startErr        error
		startCh         *channel.Channel
		getErr          error // ExternalMediaGet inside UpdateLocalAddress
		updateSetErr    error
		cacheErr        error
		finalSetErr     error
		expectCleanup   bool
		expectCache     bool
		expectFinalSet  bool
		expectUUIDForID bool

		expectErr    bool
		expectMedia  string
		expectIP     string
		expectPort   int
		expectChData string
	}{
		{
			name:         "defaults are applied and local address is parsed",
			id:           id,
			startCh:      &channel.Channel{Data: map[string]any{ChannelValiableExternalMediaLocalAddress: "10.0.0.1", ChannelValiableExternalMediaLocalPort: "5000"}},
			expectIP:     "10.0.0.1",
			expectPort:   5000,
			expectChData: "", // not checked for rtp
		},
		{
			name:            "nil id is generated",
			id:              uuid.Nil,
			startCh:         &channel.Channel{Data: map[string]any{}},
			expectUUIDForID: true,
		},
		{
			name:          "audiosocket uses the id as channel data",
			id:            id,
			encapsulation: externalmedia.EncapsulationAudioSocket,
			startCh:       &channel.Channel{Data: map[string]any{}},
			expectChData:  id.String(),
		},
		{
			name:      "create error",
			id:        id,
			setErr:    errCov,
			expectErr: true,
		},
		{
			name:      "start external media channel error removes the record",
			id:        id,
			startErr:  errCov,
			expectErr: true,
		},
		{
			name:          "update local address error cleans up",
			id:            id,
			startCh:       &channel.Channel{Data: map[string]any{}},
			getErr:        errCov,
			expectCleanup: true,
			expectErr:     true,
		},
		{
			name:      "websocket normal",
			id:        id,
			transport: externalmedia.TransportWebsocket,
			startCh: &channel.Channel{Data: map[string]any{
				ChannelVariableWebSocketConnectionID: "conn-1",
			}},
			expectCache:    true,
			expectFinalSet: true,
			expectMedia:    "ws://192.0.2.10:8088/media/conn-1",
		},
		{
			name:          "websocket without connection id cleans up",
			id:            id,
			transport:     externalmedia.TransportWebsocket,
			startCh:       &channel.Channel{Data: map[string]any{}},
			expectCleanup: true,
			expectErr:     true,
		},
		{
			name:      "websocket cache error cleans up",
			id:        id,
			transport: externalmedia.TransportWebsocket,
			startCh: &channel.Channel{Data: map[string]any{
				ChannelVariableWebSocketConnectionID: "conn-1",
			}},
			cacheErr:      errCov,
			expectCache:   true,
			expectCleanup: true,
			expectErr:     true,
		},
		{
			name:      "websocket media uri persist error cleans up",
			id:        id,
			transport: externalmedia.TransportWebsocket,
			startCh: &channel.Channel{Data: map[string]any{
				ChannelVariableWebSocketConnectionID: "conn-1",
			}},
			finalSetErr:    errCov,
			expectCache:    true,
			expectFinalSet: true,
			expectCleanup:  true,
			expectErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCovHandler(t)
			ctx := context.Background()

			useID := tt.id
			if tt.expectUUIDForID {
				useID = id
				m.util.EXPECT().UUIDCreate().Return(id)
			}
			m.util.EXPECT().UUIDCreate().Return(extChID)

			stored := &externalmedia.ExternalMedia{ID: useID}

			// Create
			m.db.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(tt.setErr)
			if tt.setErr != nil {
				goto run
			}

			m.channel.EXPECT().StartExternalMedia(ctx, asteriskID, extChID.String(), "ext:1234", gomock.Any(), gomock.Any(), "", gomock.Any(), gomock.Any(), defaultDirection, gomock.Any(), nil).
				DoAndReturn(func(_ context.Context, _, _, _, encapsulation, transport, _, connType, format, _, chData string, _ map[string]string) (*channel.Channel, error) {
					if tt.startErr != nil {
						return nil, tt.startErr
					}
					wantEncap := defaultEncapsulation
					if tt.encapsulation != "" {
						wantEncap = tt.encapsulation
					}
					if encapsulation != string(wantEncap) {
						t.Errorf("Wrong encapsulation. expect: %s, got: %s", wantEncap, encapsulation)
					}
					if connType != defaultConnectionType || format != defaultFormat {
						t.Errorf("Wrong defaults. connType: %s, format: %s", connType, format)
					}
					if tt.expectChData != "" && chData != tt.expectChData {
						t.Errorf("Wrong channel data. expect: %s, got: %s", tt.expectChData, chData)
					}
					return tt.startCh, nil
				})
			if tt.startErr != nil {
				m.db.EXPECT().ExternalMediaDelete(ctx, useID).Return(nil)
				goto run
			}

			// UpdateLocalAddress
			if tt.getErr != nil {
				m.db.EXPECT().ExternalMediaGet(ctx, useID).Return(nil, tt.getErr)
			} else {
				m.db.EXPECT().ExternalMediaGet(ctx, useID).Return(stored, nil).Times(2)
				m.db.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(tt.updateSetErr)
			}

			if tt.expectCache {
				m.cache.EXPECT().AsteriskAddressInternalGet(ctx, asteriskID).Return("192.0.2.10", tt.cacheErr)
			}
			if tt.expectFinalSet {
				m.db.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(tt.finalSetErr)
			}
			if tt.expectCleanup {
				m.channel.EXPECT().HangingUpWithAsteriskID(ctx, asteriskID, extChID.String(), ari.ChannelCauseNormalClearing).Return(nil)
				m.db.EXPECT().ExternalMediaDelete(ctx, useID).Return(nil)
			}

		run:
			res, err := h.startExternalMedia(ctx, tt.id, asteriskID, "bridge-1", "", externalmedia.ReferenceTypeCall, refID, "ext:1234", tt.encapsulation, tt.transport, "", "", "", externalmedia.DirectionBoth, externalmedia.DirectionBoth)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr {
				return
			}
			if res.LocalIP != tt.expectIP || res.LocalPort != tt.expectPort {
				t.Errorf("Wrong local address. expect: %s:%d, got: %s:%d", tt.expectIP, tt.expectPort, res.LocalIP, res.LocalPort)
			}
			if res.MediaURI != tt.expectMedia {
				t.Errorf("Wrong media uri. expect: %s, got: %s", tt.expectMedia, res.MediaURI)
			}
		})
	}
}

func Test_Start_errors_cov(t *testing.T) {
	id := uuid.FromStringOrNil("22222222-0001-11f0-8000-000000000001")
	refID := uuid.FromStringOrNil("22222222-0002-11f0-8000-000000000002")
	extChID := uuid.FromStringOrNil("22222222-0003-11f0-8000-000000000003")
	asteriskID := "42:01:0a:a4:00:05"
	snoopCall := &call.Call{Identity: commonidentity.Identity{ID: refID}, ChannelID: "ch-1", Status: call.StatusProgressing, BridgeID: "br-call"}
	ch := &channel.Channel{ID: "ch-1", AsteriskID: asteriskID}

	t.Run("nil id is generated and unsupported reference type fails", func(t *testing.T) {
		h, m := newCovHandler(t)
		m.util.EXPECT().UUIDCreate().Return(id)

		if _, err := h.Start(context.Background(), uuid.Nil, externalmedia.ReferenceType("unknown"), refID, "ext:1", "", "", "", "", "", externalmedia.DirectionBoth, externalmedia.DirectionBoth); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("call get error", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()
		m.req.EXPECT().CallV1CallGet(ctx, refID).Return(nil, errCov)

		if _, err := h.Start(ctx, id, externalmedia.ReferenceTypeCall, refID, "ext:1", "", "", "", "", "", externalmedia.DirectionBoth, externalmedia.DirectionBoth); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("channel get error", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()
		m.req.EXPECT().CallV1CallGet(ctx, refID).Return(snoopCall, nil)
		m.channel.EXPECT().Get(ctx, "ch-1").Return(nil, errCov)

		if _, err := h.Start(ctx, id, externalmedia.ReferenceTypeCall, refID, "ext:1", "", "", "", "", "", externalmedia.DirectionBoth, externalmedia.DirectionBoth); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("snoop bridge start error", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()
		m.req.EXPECT().CallV1CallGet(ctx, refID).Return(snoopCall, nil)
		m.channel.EXPECT().Get(ctx, "ch-1").Return(ch, nil)
		m.util.EXPECT().UUIDCreate().Return(extChID)
		m.bridge.EXPECT().Start(ctx, asteriskID, extChID.String(), gomock.Any(), gomock.Any()).Return(nil, errCov)

		if _, err := h.Start(ctx, id, externalmedia.ReferenceTypeCall, refID, "ext:1", "", "", "", "", "", externalmedia.DirectionBoth, externalmedia.DirectionBoth); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("snoop start error destroys the bridge", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()
		m.req.EXPECT().CallV1CallGet(ctx, refID).Return(snoopCall, nil)
		m.channel.EXPECT().Get(ctx, "ch-1").Return(ch, nil)
		m.util.EXPECT().UUIDCreate().Return(extChID).Times(2)
		m.bridge.EXPECT().Start(ctx, asteriskID, extChID.String(), gomock.Any(), gomock.Any()).Return(&bridge.Bridge{ID: "br-snoop"}, nil)
		m.channel.EXPECT().StartSnoop(ctx, "ch-1", extChID.String(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errCov)
		m.bridge.EXPECT().Destroy(ctx, "br-snoop").Return(nil)

		if _, err := h.Start(ctx, id, externalmedia.ReferenceTypeCall, refID, "ext:1", "", "", "", "", "", externalmedia.DirectionBoth, externalmedia.DirectionBoth); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("start external media error hangs up the snoop and destroys the bridge", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()
		m.req.EXPECT().CallV1CallGet(ctx, refID).Return(snoopCall, nil)
		m.channel.EXPECT().Get(ctx, "ch-1").Return(ch, nil)
		m.util.EXPECT().UUIDCreate().Return(extChID).Times(3)
		m.bridge.EXPECT().Start(ctx, asteriskID, extChID.String(), gomock.Any(), gomock.Any()).Return(&bridge.Bridge{ID: "br-snoop"}, nil)
		m.channel.EXPECT().StartSnoop(ctx, "ch-1", extChID.String(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&channel.Channel{ID: "snoop-1"}, nil)
		m.db.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(errCov)
		m.channel.EXPECT().HangingUp(ctx, "snoop-1", ari.ChannelCauseNormalClearing).Return(nil, nil)
		m.bridge.EXPECT().Destroy(ctx, "br-snoop").Return(nil)

		if _, err := h.Start(ctx, id, externalmedia.ReferenceTypeCall, refID, "ext:1", "", "", "", "", "", externalmedia.DirectionBoth, externalmedia.DirectionBoth); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})
}

func Test_startReferenceTypeConfbridge_errors_cov(t *testing.T) {
	id := uuid.FromStringOrNil("33333333-0001-11f0-8000-000000000001")
	cbID := uuid.FromStringOrNil("33333333-0002-11f0-8000-000000000002")
	extChID := uuid.FromStringOrNil("33333333-0003-11f0-8000-000000000003")
	cb := &confbridge.Confbridge{ID: cbID, BridgeID: "br-cb"}

	t.Run("confbridge get error", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()
		m.req.EXPECT().CallV1ConfbridgeGet(ctx, cbID).Return(nil, errCov)

		if _, err := h.startReferenceTypeConfbridge(ctx, id, cbID, "ext:1", "", "", "", "", ""); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("bridge get error", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()
		m.req.EXPECT().CallV1ConfbridgeGet(ctx, cbID).Return(cb, nil)
		m.bridge.EXPECT().Get(ctx, "br-cb").Return(nil, errCov)

		if _, err := h.startReferenceTypeConfbridge(ctx, id, cbID, "ext:1", "", "", "", "", ""); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("start external media error", func(t *testing.T) {
		h, m := newCovHandler(t)
		ctx := context.Background()
		m.req.EXPECT().CallV1ConfbridgeGet(ctx, cbID).Return(cb, nil)
		m.bridge.EXPECT().Get(ctx, "br-cb").Return(&bridge.Bridge{ID: "br-cb", AsteriskID: "ast-1"}, nil)
		m.util.EXPECT().UUIDCreate().Return(extChID)
		m.db.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(errCov)

		if _, err := h.startReferenceTypeConfbridge(ctx, id, cbID, "ext:1", "", "", "", "", ""); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})
}

func Test_isCallBridgeEligible_cov(t *testing.T) {
	bridgeID := "51bf770e-7f1b-11f0-ac50-971ccbe0a7ba"
	otherID := uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071")

	tests := []struct {
		name string

		call         *call.Call
		externalHost string
		listen       externalmedia.Direction
		speak        externalmedia.Direction

		expectRes    bool
		expectReason string
	}{
		{"speak out listen none", &call.Call{Status: call.StatusProgressing, BridgeID: bridgeID}, externalHostIncoming, externalmedia.DirectionNone, externalmedia.DirectionOut, true, ""},
		{"speak out listen in", &call.Call{Status: call.StatusProgressing, BridgeID: bridgeID}, externalHostIncoming, externalmedia.DirectionIn, externalmedia.DirectionOut, true, ""},
		{"speak out listen out", &call.Call{Status: call.StatusProgressing, BridgeID: bridgeID}, externalHostIncoming, externalmedia.DirectionOut, externalmedia.DirectionOut, false, "listen direction is not none or in"},
		{"speak out listen both", &call.Call{Status: call.StatusProgressing, BridgeID: bridgeID}, externalHostIncoming, externalmedia.DirectionBoth, externalmedia.DirectionOut, false, "listen direction is not none or in"},
		{"speak in listen none", &call.Call{Status: call.StatusProgressing, BridgeID: bridgeID}, externalHostIncoming, externalmedia.DirectionNone, externalmedia.DirectionIn, false, "speak direction is not out"},
		{"speak none listen none", &call.Call{Status: call.StatusProgressing, BridgeID: bridgeID}, externalHostIncoming, externalmedia.DirectionNone, externalmedia.DirectionNone, false, "speak direction is not out"},
		{"speak both listen in", &call.Call{Status: call.StatusProgressing, BridgeID: bridgeID}, externalHostIncoming, externalmedia.DirectionIn, externalmedia.DirectionBoth, false, "speak direction is not out"},
		{"customer host", &call.Call{Status: call.StatusProgressing, BridgeID: bridgeID}, "example.com", externalmedia.DirectionNone, externalmedia.DirectionOut, false, "external host is not platform internal"},
		{"not progressing", &call.Call{Status: call.StatusRinging, BridgeID: bridgeID}, externalHostIncoming, externalmedia.DirectionNone, externalmedia.DirectionOut, false, "call is not progressing"},
		{"no bridge", &call.Call{Status: call.StatusProgressing}, externalHostIncoming, externalmedia.DirectionNone, externalmedia.DirectionOut, false, "call has no bridge"},
		{"in confbridge", &call.Call{Status: call.StatusProgressing, BridgeID: bridgeID, ConfbridgeID: otherID}, externalHostIncoming, externalmedia.DirectionNone, externalmedia.DirectionOut, false, "call is in a conference"},
		{"slave leg", &call.Call{Status: call.StatusProgressing, BridgeID: bridgeID, MasterCallID: otherID}, externalHostIncoming, externalmedia.DirectionNone, externalmedia.DirectionOut, false, "call is the slave leg of a connect"},
		{"master leg", &call.Call{Status: call.StatusProgressing, BridgeID: bridgeID, ChainedCallIDs: []uuid.UUID{otherID}}, externalHostIncoming, externalmedia.DirectionNone, externalmedia.DirectionOut, false, "call is the master leg of a connect"},
		{"groupcall leg", &call.Call{Status: call.StatusProgressing, BridgeID: bridgeID, GroupcallID: otherID}, externalHostIncoming, externalmedia.DirectionNone, externalmedia.DirectionOut, false, "call is a groupcall leg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, reason := isCallBridgeEligible(tt.call, tt.externalHost, tt.listen, tt.speak)
			if res != tt.expectRes || reason != tt.expectReason {
				t.Errorf("Wrong match. expect: (%v, %q), got: (%v, %q)", tt.expectRes, tt.expectReason, res, reason)
			}
		})
	}
}
