package arieventhandler

import (
	"context"
	"errors"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
)

func Test_EventHandlerStasisStart_cov(t *testing.T) {
	callCh := &channel.Channel{ID: "ch-1", StasisData: map[channel.StasisDataType]string{channel.StasisDataTypeContextType: string(channel.ContextTypeCall)}}
	confCh := &channel.Channel{ID: "ch-1", StasisData: map[channel.StasisDataType]string{channel.StasisDataTypeContextType: string(channel.ContextTypeConference)}}
	unknownCh := &channel.Channel{ID: "ch-1", StasisData: map[channel.StasisDataType]string{}}

	tests := []struct {
		name string

		getErr      error
		stasisResp  *channel.Channel
		stasisErr   error
		handlerErr  error
		expectStart bool
		expectCall  bool
		expectConf  bool
		expectHang  bool // HangingUp after handler failure
		expectErr   bool
	}{
		{name: "call normal", stasisResp: callCh, expectStart: true, expectCall: true},
		{name: "confbridge normal", stasisResp: confCh, expectStart: true, expectConf: true},
		{name: "channel not found", getErr: errors.New("nf"), expectErr: true},
		{name: "channel handler failure", stasisErr: errors.New("stasis"), expectStart: true, expectErr: true},
		{name: "unknown context type", stasisResp: unknownCh, expectStart: true, expectHang: true, expectErr: true},
		{name: "call handler failure", stasisResp: callCh, expectStart: true, expectCall: true, handlerErr: errors.New("call"), expectHang: true, expectErr: true},
		{name: "confbridge handler failure", stasisResp: confCh, expectStart: true, expectConf: true, handlerErr: errors.New("conf"), expectHang: true, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConf := confbridgehandler.NewMockConfbridgeHandler(mc)
			h := eventHandler{channelHandler: mockChannel, callHandler: mockCall, confbridgeHandler: mockConf}
			ctx := context.Background()

			e := &ari.StasisStart{
				Event:   ari.Event{AsteriskID: "ast-1"},
				Channel: ari.Channel{ID: "ch-1"},
			}

			if tt.getErr != nil {
				mockChannel.EXPECT().Get(ctx, "ch-1").Return(nil, tt.getErr)
				mockChannel.EXPECT().HangingUpWithAsteriskID(ctx, "ast-1", "ch-1", ari.ChannelCauseInterworking).Return(nil)
			} else {
				mockChannel.EXPECT().Get(ctx, "ch-1").Return(&channel.Channel{ID: "ch-1"}, nil)
			}
			if tt.expectStart {
				mockChannel.EXPECT().ARIStasisStart(ctx, e).Return(tt.stasisResp, tt.stasisErr)
			}
			if tt.expectCall {
				mockCall.EXPECT().ARIStasisStart(ctx, tt.stasisResp).Return(tt.handlerErr)
			}
			if tt.expectConf {
				mockConf.EXPECT().ARIStasisStart(ctx, tt.stasisResp).Return(tt.handlerErr)
			}
			if tt.expectHang {
				mockChannel.EXPECT().HangingUp(ctx, "ch-1", ari.ChannelCauseNoRouteDestination).Return(nil, nil)
			}

			err := h.EventHandlerStasisStart(ctx, e)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}
