package arieventhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func TestEventHandlerBridgeCreated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	h := eventHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		reqHandler:  mockRequest,
		callHandler: mockCall,
	}

	tests := []struct {
		name        string
		event       *ari.BridgeCreated
		expectEvent *bridge.Bridge
	}{
		{
			"normal",
			&ari.BridgeCreated{
				Event: ari.Event{
					Type:        ari.EventTypeBridgeCreated,
					Application: "voipbin",
					AsteriskID:  "42:01:0a:a4:00:05",
				},

				Bridge: ari.Bridge{
					ID:   "4625f6e6-6330-48ea-9d93-5cca714322b3",
					Name: "echo",

					BridgeType:  "mixing",
					Technology:  "simple_bridge",
					BridgeClass: "stasis",
					Creator:     "Stasis",

					VideoMode: "none",

					Channels: []string{},

					CreationTime: "2020-05-09T12:41:43.591+0000",
				},
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

				ReferenceType: bridge.ReferenceTypeUnknown,
				ReferenceID:   uuid.Nil,

				TMCreate: "2020-05-09T12:41:43.591",
				TMUpdate: defaultTimeStamp,
				TMDelete: defaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockDB.EXPECT().BridgeCreate(gomock.Any(), tt.expectEvent)

			if err := h.EventHandlerBridgeCreated(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestEventHandlerBridgeDestroyed(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockRequest := requesthandler.NewMockRequestHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	h := eventHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		reqHandler:  mockRequest,
		callHandler: mockCall,
	}

	tests := []struct {
		name            string
		event           *ari.BridgeDestroyed
		expectBridgeID  string
		expectTimestamp string
	}{
		{
			"normal",
			&ari.BridgeDestroyed{
				Event: ari.Event{
					Type:        ari.EventTypeBridgeDestroyed,
					Application: "voipbin",
					Timestamp:   "2020-05-04T00:27:59.747",
					AsteriskID:  "42:01:0a:a4:00:03",
				},

				Bridge: ari.Bridge{
					ID:   "17174a5e-91f6-11ea-b637-fb223e63cedf",
					Name: "test",

					BridgeType:  "mixing",
					Technology:  "simple_bridge",
					BridgeClass: "stasis",
					Creator:     "Stasis",

					VideoMode: "talker",

					Channels: []string{},

					CreationTime: "2020-05-03T23:37:49.233+0000",
				},
			},
			"17174a5e-91f6-11ea-b637-fb223e63cedf",
			"2020-05-04T00:27:59.747",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().BridgeIsExist(tt.expectBridgeID, defaultExistTimeout).Return(true)
			mockDB.EXPECT().BridgeEnd(gomock.Any(), tt.expectBridgeID, tt.expectTimestamp)

			if err := h.EventHandlerBridgeDestroyed(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
