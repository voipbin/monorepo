package eventhandler

import (
	"testing"

	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/eventhandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/rabbitmq"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestEventHandlerBridgeCreated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)

	type test struct {
		name         string
		event        *rabbitmq.Event
		expectBridge *bridge.Bridge
	}

	tests := []test{
		{
			"normal",
			&rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     []byte(`{"type":"BridgeCreated","timestamp":"2020-05-09T12:41:43.591+0000","bridge":{"id":"4625f6e6-6330-48ea-9d93-5cca714322b3","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"echo","channels":[],"creationtime":"2020-05-09T12:41:43.591+0000","video_mode":"none"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`),
			},
			&bridge.Bridge{
				ID:         "4625f6e6-6330-48ea-9d93-5cca714322b3",
				AsteriskID: "42:01:0a:a4:00:05",
				Name:       "echo",
				Type:       bridge.TypeMixing,
				Tech:       bridge.TechSimple,
				Class:      "stasis",
				Creator:    "Stasis",
				VideoMode:  "none",

				ChannelIDs: []string{},

				TMCreate: "2020-05-09T12:41:43.591",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := eventHandler{
				db:          mockDB,
				rabbitSock:  mockSock,
				reqHandler:  mockRequest,
				callHandler: mockCall,
			}

			mockDB.EXPECT().BridgeCreate(gomock.Any(), tt.expectBridge)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestEventHandlerBridgeDestroyed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmq.NewMockRabbit(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)

	type test struct {
		name            string
		event           *rabbitmq.Event
		expectBridgeID  string
		expectTimestamp string
	}

	tests := []test{
		{
			"normal",
			&rabbitmq.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     []byte(`{"type":"BridgeDestroyed","timestamp":"2020-05-04T00:27:59.747+0000","bridge":{"id":"17174a5e-91f6-11ea-b637-fb223e63cedf","technology":"simple_bridge","bridge_type":"mixing","bridge_class":"stasis","creator":"Stasis","name":"test","channels":[],"creationtime":"2020-05-03T23:37:49.233+0000","video_mode":"talker"},"asterisk_id":"42:01:0a:a4:00:03","application":"voipbin"}`),
			},
			"17174a5e-91f6-11ea-b637-fb223e63cedf",
			"2020-05-04T00:27:59.747",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := eventHandler{
				db:          mockDB,
				rabbitSock:  mockSock,
				reqHandler:  mockRequest,
				callHandler: mockCall,
			}

			mockDB.EXPECT().BridgeIsExist(tt.expectBridgeID, defaultExistTimeout).Return(true)
			mockDB.EXPECT().BridgeEnd(gomock.Any(), tt.expectBridgeID, tt.expectTimestamp)

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
