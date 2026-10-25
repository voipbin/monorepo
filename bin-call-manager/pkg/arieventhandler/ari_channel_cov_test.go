package arieventhandler

import (
	"context"
	"errors"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
)

func Test_EventHandlerChannelLeftBridge_cov(t *testing.T) {
	cn := &channel.Channel{ID: "ch-1"}

	tests := []struct {
		name string

		updateErr error
		removeErr error
		bridgeRef bridge.ReferenceType

		expectRemove bool
		expectHangup bool
		expectCall   bool
		expectConf   bool
		handlerErr   error
		expectErr    bool
	}{
		{name: "update bridge id failure", updateErr: errors.New("upd"), expectHangup: true, expectErr: true},
		{name: "remove channel failure", removeErr: errors.New("rm"), expectRemove: true, expectHangup: true, expectErr: true},
		{name: "unknown reference type", bridgeRef: bridge.ReferenceTypeUnknown, expectRemove: true},
		{name: "call handler error propagates", bridgeRef: bridge.ReferenceTypeCall, expectRemove: true, expectCall: true, handlerErr: errors.New("c"), expectErr: true},
		{name: "confbridge handler error propagates", bridgeRef: bridge.ReferenceTypeConfbridge, expectRemove: true, expectConf: true, handlerErr: errors.New("c"), expectErr: true},
		{name: "call snoop reference routes to call handler", bridgeRef: bridge.ReferenceTypeCallSnoop, expectRemove: true, expectCall: true},
		{name: "confbridge snoop reference routes to confbridge handler", bridgeRef: bridge.ReferenceTypeConfbridgeSnoop, expectRemove: true, expectConf: true},
		{name: "call snoop handler error propagates", bridgeRef: bridge.ReferenceTypeCallSnoop, expectRemove: true, expectCall: true, handlerErr: errors.New("c"), expectErr: true},
		{name: "confbridge snoop handler error propagates", bridgeRef: bridge.ReferenceTypeConfbridgeSnoop, expectRemove: true, expectConf: true, handlerErr: errors.New("c"), expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConf := confbridgehandler.NewMockConfbridgeHandler(mc)
			h := eventHandler{
				reqHandler: mockReq, channelHandler: mockChannel, bridgeHandler: mockBridge,
				callHandler: mockCall, confbridgeHandler: mockConf,
			}
			ctx := context.Background()

			e := &ari.ChannelLeftBridge{
				Event:   ari.Event{AsteriskID: "ast-1"},
				Channel: ari.Channel{ID: "ch-1"},
				Bridge:  ari.Bridge{ID: "br-1"},
			}
			br := &bridge.Bridge{ID: "br-1", ReferenceType: tt.bridgeRef}

			if tt.updateErr != nil {
				mockChannel.EXPECT().UpdateBridgeID(ctx, "ch-1", "").Return(nil, tt.updateErr)
			} else {
				mockChannel.EXPECT().UpdateBridgeID(ctx, "ch-1", "").Return(cn, nil)
			}
			if tt.expectRemove {
				if tt.removeErr != nil {
					mockBridge.EXPECT().RemoveChannelID(ctx, "br-1", "ch-1").Return(nil, tt.removeErr)
				} else {
					mockBridge.EXPECT().RemoveChannelID(ctx, "br-1", "ch-1").Return(br, nil)
				}
			}
			if tt.expectHangup {
				mockReq.EXPECT().AstChannelHangup(ctx, "ast-1", "ch-1", ari.ChannelCauseInterworking, 0).Return(nil)
			}
			if tt.expectCall {
				mockCall.EXPECT().ARIChannelLeftBridge(ctx, cn, br).Return(tt.handlerErr)
			}
			if tt.expectConf {
				mockConf.EXPECT().ARIChannelLeftBridge(ctx, cn, br).Return(tt.handlerErr)
			}

			err := h.EventHandlerChannelLeftBridge(ctx, e)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}
