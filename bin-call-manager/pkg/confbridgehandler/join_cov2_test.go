package confbridgehandler

import (
	"context"
	stderrors "errors"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/common"
	"monorepo/bin-call-manager/models/confbridge"
)

func Test_Join_cov2_errors(t *testing.T) {
	ctx := context.Background()

	cbWithBridge := &confbridge.Confbridge{BridgeID: "c2-bridge-1", Type: confbridge.TypeConference}
	cbWithBridge.ID = cov2CBID
	c := &call.Call{ChannelID: "c2-call-channel", BridgeID: "c2-call-bridge"}
	c.ID = cov2CallID

	tests := []struct {
		name  string
		setup func(m *cov2Mocks)
	}{
		{
			name: "confbridge get failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
		},
		{
			name: "call get failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cbWithBridge, nil)
				m.req.EXPECT().CallV1CallGet(ctx, cov2CallID).Return(nil, errCov2)
			},
		},
		{
			name: "confbridge bridge missing and recreating failed",
			setup: func(m *cov2Mocks) {
				cb := &confbridge.Confbridge{BridgeID: ""}
				cb.ID = cov2CBID
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cb, nil)
				m.req.EXPECT().CallV1CallGet(ctx, cov2CallID).Return(c, nil)
				// createConfbridgeBridge reads the confbridge again, which fails.
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
		},
		{
			name: "bridge get for endpoint failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cbWithBridge, nil)
				m.req.EXPECT().CallV1CallGet(ctx, cov2CallID).Return(c, nil)
				m.bridge.EXPECT().IsExist(ctx, "c2-bridge-1").Return(true)
				m.bridge.EXPECT().Get(ctx, "c2-bridge-1").Return(nil, errCov2)
			},
		},
		{
			name: "asterisk address lookup failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cbWithBridge, nil)
				m.req.EXPECT().CallV1CallGet(ctx, cov2CallID).Return(c, nil)
				m.bridge.EXPECT().IsExist(ctx, "c2-bridge-1").Return(true)
				m.bridge.EXPECT().Get(ctx, "c2-bridge-1").Return(&bridge.Bridge{ID: "c2-bridge-1", AsteriskID: "c2-ast"}, nil)
				m.cache.EXPECT().AsteriskAddressInternalGet(ctx, "c2-ast").Return("", errCov2)
			},
		},
		{
			name: "start join channel failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cbWithBridge, nil)
				m.req.EXPECT().CallV1CallGet(ctx, cov2CallID).Return(c, nil)
				m.bridge.EXPECT().IsExist(ctx, "c2-bridge-1").Return(true)
				m.bridge.EXPECT().Get(ctx, "c2-bridge-1").Return(&bridge.Bridge{ID: "c2-bridge-1", AsteriskID: "c2-ast"}, nil)
				m.cache.EXPECT().AsteriskAddressInternalGet(ctx, "c2-ast").Return("192.0.2.1", nil)
				m.util.EXPECT().UUIDCreate().Return(cov2ExtID)
				m.channel.EXPECT().StartChannelWithBaseChannel(
					ctx, "c2-call-channel", cov2ExtID.String(),
					"context_type=call,context=call-join,confbridge_id="+cov2CBID.String()+",bridge_id=c2-call-bridge,call_id="+cov2CallID.String(),
					"PJSIP/conf-join/sip:c2-bridge-1@192.0.2.1:5060", "", "vp8", "", map[string]string{
						"PJSIP_HEADER(add," + common.SIPHeaderCallID + ")":       cov2CallID.String(),
						"PJSIP_HEADER(add," + common.SIPHeaderConfbridgeID + ")": cov2CBID.String(),
					},
				).Return(nil, errCov2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			err := h.Join(ctx, cov2CBID, cov2CallID)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !stderrors.Is(err, errCov2) {
				t.Errorf("Wrong match. expect: wrapped errCov2, got: %v", err)
			}
		})
	}
}

func Test_createConfbridgeBridge_cov2_errors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		cbType confbridge.Type
		setup  func(m *cov2Mocks)

		expectIs  error
		expectMsg string
	}{
		{
			name:   "get confbridge failed",
			cbType: confbridge.TypeConference,
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
			expectIs: errCov2,
		},
		{
			name:   "unsupported confbridge type",
			cbType: confbridge.Type("unknown"),
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cov2CB(func(c *confbridge.Confbridge) { c.Type = "unknown" }), nil)
			},
			expectMsg: "unsupported confbridge type. confbridge_type: unknown",
		},
		{
			name:   "bridge start failed",
			cbType: confbridge.TypeConference,
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cov2CB(func(c *confbridge.Confbridge) { c.Type = confbridge.TypeConference }), nil)
				m.util.EXPECT().UUIDCreate().Return(cov2ExtID)
				m.bridge.EXPECT().Start(ctx, requesthandler.AsteriskIDConference, cov2ExtID.String(),
					"reference_type=confbridge,reference_id="+cov2CBID.String(), []bridge.Type{bridge.TypeMixing}).Return(nil, errCov2)
			},
			expectIs: errCov2,
		},
		{
			name:   "update bridge id failed",
			cbType: confbridge.TypeConnect,
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cov2CB(func(c *confbridge.Confbridge) { c.Type = confbridge.TypeConnect }), nil)
				m.util.EXPECT().UUIDCreate().Return(cov2ExtID)
				m.bridge.EXPECT().Start(ctx, requesthandler.AsteriskIDConference, cov2ExtID.String(),
					"reference_type=confbridge,reference_id="+cov2CBID.String(), []bridge.Type{bridge.TypeMixing, bridge.TypeProxyMedia}).Return(&bridge.Bridge{ID: "c2-new-bridge"}, nil)
				m.db.EXPECT().ConfbridgeSetBridgeID(ctx, cov2CBID, "c2-new-bridge").Return(errCov2)
			},
			expectIs: errCov2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			res, err := h.createConfbridgeBridge(ctx, cov2CBID)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
			if tt.expectIs != nil && !stderrors.Is(err, tt.expectIs) {
				t.Errorf("Wrong match. expect: wrapped %v, got: %v", tt.expectIs, err)
			}
			if tt.expectMsg != "" && !reflect.DeepEqual(err.Error(), tt.expectMsg) {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectMsg, err.Error())
			}
		})
	}
}
