package arieventhandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
)

func Test_EventHandlerBridgeCreated_cov2(t *testing.T) {
	refID := uuid.FromStringOrNil("7444bf66-13d9-11ec-9a1a-ab73ca01d2c6")

	tests := []struct {
		name string

		bridgeName string
		createErr  error

		expectRefType bridge.ReferenceType
		expectRefID   uuid.UUID
	}{
		{name: "name with reference", bridgeName: "reference_type=call,reference_id=" + refID.String(), expectRefType: bridge.ReferenceTypeCall, expectRefID: refID},
		{name: "name without reference", bridgeName: "", expectRefType: bridge.ReferenceTypeUnknown, expectRefID: uuid.Nil},
		{name: "create error", bridgeName: "", createErr: errors.New("create"), expectRefType: bridge.ReferenceTypeUnknown, expectRefID: uuid.Nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockBridge := bridgehandler.NewMockBridgeHandler(mc)
			h := eventHandler{bridgeHandler: mockBridge}

			e := &ari.BridgeCreated{
				Event:  ari.Event{AsteriskID: "ast-1"},
				Bridge: ari.Bridge{ID: "br-1", Name: tt.bridgeName, BridgeType: "mixing", Technology: "simple_bridge", BridgeClass: "stasis", Creator: "Stasis"},
			}
			var br *bridge.Bridge
			if tt.createErr == nil {
				br = &bridge.Bridge{ID: "br-1"}
			}

			mockBridge.EXPECT().Create(gomock.Any(), "ast-1", "br-1", tt.bridgeName, bridge.Type("mixing"), bridge.Tech("simple_bridge"), "stasis", "Stasis", "", "", tt.expectRefType, tt.expectRefID).Return(br, tt.createErr)

			err := h.EventHandlerBridgeCreated(context.Background(), e)
			if err != tt.createErr {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.createErr, err)
			}
		})
	}
}

func Test_EventHandlerBridgeDestroyed_cov2(t *testing.T) {
	tests := []struct {
		name string

		getErr    error
		deleteErr error
		confErr   error

		expectErr string
	}{
		{name: "get error", getErr: errors.New("get"), expectErr: "no bridge found"},
		{name: "delete error", deleteErr: errors.New("del"), expectErr: "del"},
		{name: "confbridge error", confErr: errors.New("conf"), expectErr: "could not handle the event by the confbridgehandler: conf"},
		{name: "success"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockBridge := bridgehandler.NewMockBridgeHandler(mc)
			mockConf := confbridgehandler.NewMockConfbridgeHandler(mc)
			h := eventHandler{bridgeHandler: mockBridge, confbridgeHandler: mockConf}

			e := &ari.BridgeDestroyed{Bridge: ari.Bridge{ID: "br-1"}}
			br := &bridge.Bridge{ID: "br-1"}

			if tt.getErr != nil {
				mockBridge.EXPECT().Get(gomock.Any(), "br-1").Return(nil, tt.getErr)
			} else {
				mockBridge.EXPECT().Get(gomock.Any(), "br-1").Return(br, nil)
				if tt.deleteErr != nil {
					mockBridge.EXPECT().Delete(gomock.Any(), "br-1").Return(nil, tt.deleteErr)
				} else {
					mockBridge.EXPECT().Delete(gomock.Any(), "br-1").Return(br, nil)
					mockConf.EXPECT().ARIBridgeDestroyed(gomock.Any(), br).Return(tt.confErr)
				}
			}

			err := h.EventHandlerBridgeDestroyed(context.Background(), e)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tt.expectErr {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectErr, got)
			}
		})
	}
}
