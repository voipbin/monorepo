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

func Test_EventHandlerChannelDestroyed_cov2(t *testing.T) {
	tests := []struct {
		name string

		deleteErr  error
		chType     channel.Type
		callErr    error
		expectCall bool
		expectErr  error
	}{
		{name: "delete error", deleteErr: errors.New("del"), expectErr: errors.New("del")},
		{name: "type call success", chType: channel.TypeCall, expectCall: true},
		{name: "type call handler error", chType: channel.TypeCall, callErr: errors.New("call"), expectCall: true, expectErr: errors.New("call")},
		{name: "type confbridge ignored", chType: channel.TypeConfbridge},
		{name: "type application ignored", chType: channel.TypeApplication},
		{name: "unsupported type ignored", chType: channel.Type("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			h := eventHandler{channelHandler: mockChannel, callHandler: mockCall}

			e := &ari.ChannelDestroyed{Channel: ari.Channel{ID: "ch-1"}, Cause: 16}
			cn := &channel.Channel{ID: "ch-1", Type: tt.chType}

			if tt.deleteErr != nil {
				mockChannel.EXPECT().Delete(gomock.Any(), "ch-1", ari.ChannelCause(16)).Return(nil, tt.deleteErr)
			} else {
				mockChannel.EXPECT().Delete(gomock.Any(), "ch-1", ari.ChannelCause(16)).Return(cn, nil)
			}
			if tt.expectCall {
				mockCall.EXPECT().ARIChannelDestroyed(gomock.Any(), cn).Return(tt.callErr)
			}

			err := h.EventHandlerChannelDestroyed(context.Background(), e)
			if (err == nil) != (tt.expectErr == nil) || (err != nil && err.Error() != tt.expectErr.Error()) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_EventHandlerChannelDtmfReceived_cov2(t *testing.T) {
	tests := []struct {
		name string

		getErr  error
		dtmfErr error
	}{
		{name: "get error", getErr: errors.New("get")},
		{name: "dtmf error", dtmfErr: errors.New("dtmf")},
		{name: "success"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			h := eventHandler{channelHandler: mockChannel, callHandler: mockCall}

			e := &ari.ChannelDtmfReceived{Channel: ari.Channel{ID: "ch-1"}, Digit: "5", Duration: 100}
			cn := &channel.Channel{ID: "ch-1"}

			if tt.getErr != nil {
				mockChannel.EXPECT().Get(gomock.Any(), "ch-1").Return(nil, tt.getErr)
			} else {
				mockChannel.EXPECT().Get(gomock.Any(), "ch-1").Return(cn, nil)
				mockCall.EXPECT().ARIChannelDtmfReceived(gomock.Any(), cn, "5", 100).Return(tt.dtmfErr)
			}

			err := h.EventHandlerChannelDtmfReceived(context.Background(), e)
			expect := tt.getErr
			if expect == nil {
				expect = tt.dtmfErr
			}
			if err != expect {
				t.Errorf("Wrong match. expect: %v, got: %v", expect, err)
			}
		})
	}
}

func Test_EventHandlerChannelEnteredBridge_cov2(t *testing.T) {
	tests := []struct {
		name string

		chType    channel.Type
		updateErr error
		addErr    error
		confErr   error

		expectHangup bool
		expectAdd    bool
		expectConf   bool
		expectErr    error
	}{
		{name: "update error", updateErr: errors.New("upd"), expectHangup: true, expectErr: errors.New("upd")},
		{name: "add channel error", addErr: errors.New("add"), expectAdd: true, expectHangup: true, expectErr: errors.New("add")},
		{name: "non confbridge success", chType: channel.TypeCall, expectAdd: true},
		{name: "confbridge success", chType: channel.TypeConfbridge, expectAdd: true, expectConf: true},
		{name: "confbridge error", chType: channel.TypeConfbridge, expectAdd: true, expectConf: true, confErr: errors.New("conf"), expectErr: errors.New("conf")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)
			mockConf := confbridgehandler.NewMockConfbridgeHandler(mc)
			h := eventHandler{reqHandler: mockReq, channelHandler: mockChannel, bridgeHandler: mockBridge, confbridgeHandler: mockConf}

			e := &ari.ChannelEnteredBridge{
				Event:   ari.Event{AsteriskID: "ast-1"},
				Channel: ari.Channel{ID: "ch-1"},
				Bridge:  ari.Bridge{ID: "br-1"},
			}
			cn := &channel.Channel{ID: "ch-1", Type: tt.chType}
			br := &bridge.Bridge{ID: "br-1"}

			if tt.updateErr != nil {
				mockChannel.EXPECT().UpdateBridgeID(gomock.Any(), "ch-1", "br-1").Return(nil, tt.updateErr)
			} else {
				mockChannel.EXPECT().UpdateBridgeID(gomock.Any(), "ch-1", "br-1").Return(cn, nil)
			}
			if tt.expectAdd {
				if tt.addErr != nil {
					mockBridge.EXPECT().AddChannelID(gomock.Any(), "br-1", "ch-1").Return(nil, tt.addErr)
				} else {
					mockBridge.EXPECT().AddChannelID(gomock.Any(), "br-1", "ch-1").Return(br, nil)
				}
			}
			if tt.expectHangup {
				mockReq.EXPECT().AstChannelHangup(gomock.Any(), "ast-1", "ch-1", ari.ChannelCauseInterworking, 0).Return(nil)
			}
			if tt.expectConf {
				mockConf.EXPECT().ARIChannelEnteredBridge(gomock.Any(), cn, br).Return(tt.confErr)
			}

			err := h.EventHandlerChannelEnteredBridge(context.Background(), e)
			if (err == nil) != (tt.expectErr == nil) || (err != nil && err.Error() != tt.expectErr.Error()) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_EventHandlerChannelStateChange_cov2(t *testing.T) {
	tests := []struct {
		name string

		chType    channel.Type
		stateErr  error
		callErr   error
		expectCal bool
		expectJoi bool
	}{
		{name: "channel state error", stateErr: errors.New("state")},
		{name: "call handler error", callErr: errors.New("call"), expectCal: true},
		{name: "join channel notifies confbridge", chType: channel.TypeJoin, expectCal: true, expectJoi: true},
		{name: "call channel success", chType: channel.TypeCall, expectCal: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConf := confbridgehandler.NewMockConfbridgeHandler(mc)
			h := eventHandler{channelHandler: mockChannel, callHandler: mockCall, confbridgeHandler: mockConf}

			e := &ari.ChannelStateChange{Channel: ari.Channel{ID: "ch-1"}}
			cn := &channel.Channel{ID: "ch-1", Type: tt.chType}

			if tt.stateErr != nil {
				mockChannel.EXPECT().ARIChannelStateChange(gomock.Any(), e).Return(nil, tt.stateErr)
			} else {
				mockChannel.EXPECT().ARIChannelStateChange(gomock.Any(), e).Return(cn, nil)
			}
			if tt.expectCal {
				mockCall.EXPECT().ARIChannelStateChange(gomock.Any(), cn).Return(tt.callErr)
			}
			if tt.expectJoi {
				mockConf.EXPECT().ARIChannelStateChange(gomock.Any(), cn)
			}

			err := h.EventHandlerChannelStateChange(context.Background(), e)
			expect := tt.stateErr
			if expect == nil {
				expect = tt.callErr
			}
			if err != expect {
				t.Errorf("Wrong match. expect: %v, got: %v", expect, err)
			}
		})
	}
}

func Test_EventHandlerChannelVarset_cov2_setError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockChannel := channelhandler.NewMockChannelHandler(mc)
	h := eventHandler{channelHandler: mockChannel}

	e := &ari.ChannelVarset{Channel: ari.Channel{ID: "ch-1"}, Variable: "key", Value: "val"}
	mockChannel.EXPECT().SetDataItem(gomock.Any(), "ch-1", "key", "val").Return(errors.New("set"))

	err := h.EventHandlerChannelVarset(context.Background(), e)
	if err == nil || err.Error() != "could not set the variable: set" {
		t.Errorf("Wrong match. got: %v", err)
	}
}
