package subscribehandler

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/pkg/arieventhandler"
)

func Test_processEventAsteriskProxy_cov(t *testing.T) {
	bridgeCreated := []byte(`{"type":"BridgeCreated","timestamp":"2020-05-09T12:41:43.591+0000","bridge":{"id":"4625f6e6-6330-48ea-9d93-5cca714322b3"},"asterisk_id":"ast-1","application":"voipbin"}`)

	tests := []struct {
		name string
		data []byte

		expectHandler bool
		handlerErr    error
		expectErr     bool
	}{
		{name: "invalid json", data: []byte(`{invalid`), expectErr: true},
		{name: "event without handler", data: []byte(`{"type":"ChannelHangupRequest","asterisk_id":"ast-1"}`)},
		{name: "handler error is swallowed", data: bridgeCreated, expectHandler: true, handlerErr: errors.New("handler")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockARI := arieventhandler.NewMockARIEventHandler(mc)
			h := &subscribeHandler{ariEventHandler: mockARI}

			if tt.expectHandler {
				mockARI.EXPECT().EventHandlerBridgeCreated(gomock.Any(), gomock.Any()).Return(tt.handlerErr)
			}

			err := h.processEventAsteriskProxy(context.Background(), &sock.Event{Publisher: "asterisk-proxy", Data: tt.data})
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_processEventAsteriskProxy_routing_cov(t *testing.T) {
	tests := []struct {
		name      string
		eventType ari.EventType
		expect    func(m *arieventhandler.MockARIEventHandler) *gomock.Call
	}{
		{"BridgeCreated", ari.EventTypeBridgeCreated, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerBridgeCreated(gomock.Any(), gomock.Any())
		}},
		{"BridgeDestroyed", ari.EventTypeBridgeDestroyed, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerBridgeDestroyed(gomock.Any(), gomock.Any())
		}},
		{"ChannelCreated", ari.EventTypeChannelCreated, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerChannelCreated(gomock.Any(), gomock.Any())
		}},
		{"ChannelDestroyed", ari.EventTypeChannelDestroyed, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerChannelDestroyed(gomock.Any(), gomock.Any())
		}},
		{"ChannelDtmfReceived", ari.EventTypeChannelDtmfReceived, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerChannelDtmfReceived(gomock.Any(), gomock.Any())
		}},
		{"ChannelEnteredBridge", ari.EventTypeChannelEnteredBridge, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerChannelEnteredBridge(gomock.Any(), gomock.Any())
		}},
		{"ChannelLeftBridge", ari.EventTypeChannelLeftBridge, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerChannelLeftBridge(gomock.Any(), gomock.Any())
		}},
		{"ChannelStateChange", ari.EventTypeChannelStateChange, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerChannelStateChange(gomock.Any(), gomock.Any())
		}},
		{"ChannelVarset", ari.EventTypeChannelVarset, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerChannelVarset(gomock.Any(), gomock.Any())
		}},
		{"ContactStatusChange", ari.EventTypeContactStatusChange, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerContactStatusChange(gomock.Any(), gomock.Any())
		}},
		{"PlaybackStarted", ari.EventTypePlaybackStarted, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerPlaybackStarted(gomock.Any(), gomock.Any())
		}},
		{"PlaybackFinished", ari.EventTypePlaybackFinished, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerPlaybackFinished(gomock.Any(), gomock.Any())
		}},
		{"RecordingFinished", ari.EventTypeRecordingFinished, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerRecordingFinished(gomock.Any(), gomock.Any())
		}},
		{"RecordingStarted", ari.EventTypeRecordingStarted, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerRecordingStarted(gomock.Any(), gomock.Any())
		}},
		{"StasisEnd", ari.EventTypeStasisEnd, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerStasisEnd(gomock.Any(), gomock.Any())
		}},
		{"StasisStart", ari.EventTypeStasisStart, func(m *arieventhandler.MockARIEventHandler) *gomock.Call {
			return m.EXPECT().EventHandlerStasisStart(gomock.Any(), gomock.Any())
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockARI := arieventhandler.NewMockARIEventHandler(mc)
			h := &subscribeHandler{ariEventHandler: mockARI}

			tt.expect(mockARI).Return(nil).Times(1)

			data := []byte(fmt.Sprintf(`{"type":%q,"asterisk_id":"ast-1","application":"voipbin"}`, string(tt.eventType)))
			if err := h.processEventAsteriskProxy(context.Background(), &sock.Event{Publisher: "asterisk-proxy", Data: data}); err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
		})
	}
}
