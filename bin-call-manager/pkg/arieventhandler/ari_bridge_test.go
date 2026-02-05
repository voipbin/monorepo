package arieventhandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/testhelper"
)

func Test_EventHandlerBridgeCreated(t *testing.T) {

	tests := []struct {
		name  string
		event *ari.BridgeCreated

		expectAsteriskID    string
		expectID            string
		expectName          string
		expectType          bridge.Type
		expectTech          bridge.Tech
		expectClass         string
		expectCreator       string
		expectVideoMode     string
		expectVideoSourceID string
		expectReferenceType bridge.ReferenceType
		expectReferenceID   uuid.UUID

		responseBridge *bridge.Bridge
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

			"42:01:0a:a4:00:05",
			"4625f6e6-6330-48ea-9d93-5cca714322b3",
			"echo",
			bridge.TypeMixing,
			bridge.TechSimple,
			"stasis",
			"Stasis",
			"none",
			"",
			bridge.ReferenceTypeUnknown,
			uuid.Nil,

			&bridge.Bridge{
				ID: "4625f6e6-6330-48ea-9d93-5cca714322b3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockRequest := requesthandler.NewMockRequestHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := eventHandler{
				db:            mockDB,
				sockHandler:   mockSock,
				reqHandler:    mockRequest,
				callHandler:   mockCall,
				bridgeHandler: mockBridge,
			}

			ctx := context.Background()

			mockBridge.EXPECT().Create(
				ctx,
				tt.expectAsteriskID,
				tt.expectID,
				tt.expectName,
				tt.expectType,
				tt.expectTech,
				tt.expectClass,
				tt.expectCreator,
				tt.expectVideoMode,
				tt.expectVideoSourceID,
				tt.expectReferenceType,
				tt.expectReferenceID,
			).Return(tt.responseBridge, nil)

			if err := h.EventHandlerBridgeCreated(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventHandlerBridgeDestroyed(t *testing.T) {

	tests := []struct {
		name string

		event           *ari.BridgeDestroyed
		expectBridgeID  string
		expectTimestamp *time.Time

		responseBridge *bridge.Bridge
	}{
		{
			"normal",
			&ari.BridgeDestroyed{
				Event: ari.Event{
					Type:        ari.EventTypeBridgeDestroyed,
					Application: "voipbin",
					Timestamp:   "2020-05-04T00:27:59.747Z",
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
			testhelper.TimePtr("2020-05-04T00:27:59.747Z"),

			&bridge.Bridge{
				ID: "17174a5e-91f6-11ea-b637-fb223e63cedf",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockRequest := requesthandler.NewMockRequestHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := eventHandler{
				db:                mockDB,
				sockHandler:       mockSock,
				reqHandler:        mockRequest,
				callHandler:       mockCall,
				bridgeHandler:     mockBridge,
				confbridgeHandler: mockConfbridge,
			}

			ctx := context.Background()

			mockBridge.EXPECT().Get(ctx, tt.expectBridgeID).Return(tt.responseBridge, nil)
			mockBridge.EXPECT().Delete(ctx, tt.expectBridgeID).Return(tt.responseBridge, nil)
			mockConfbridge.EXPECT().ARIBridgeDestroyed(ctx, tt.responseBridge).Return(nil)

			if err := h.EventHandlerBridgeDestroyed(ctx, tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
