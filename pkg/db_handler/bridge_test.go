package dbhandler

import (
	"context"
	"reflect"
	"testing"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/bridge"
)

func TestBridgeCreate(t *testing.T) {
	type test struct {
		name         string
		bridge       *bridge.Bridge
		expectBridge *bridge.Bridge
	}

	tests := []test{
		{
			"test normal",
			&bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "98ff3f2a-8226-11ea-9ec5-079bcb66275c",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			&bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "98ff3f2a-8226-11ea-9ec5-079bcb66275c",
				Channels:   []string{},
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest)

			if err := h.BridgeCreate(context.Background(), tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.BridgeGet(context.Background(), tt.bridge.AsteriskID, tt.bridge.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectBridge, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectBridge, res)
			}
		})
	}
}

func TestBridgeEnd(t *testing.T) {
	type test struct {
		name         string
		bridge       *bridge.Bridge
		timestamp    string
		expectBridge *bridge.Bridge
	}

	tests := []test{
		{
			"test normal",
			&bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "208a5bbe-8ee3-11ea-b267-174c3bd0a842",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			"2020-04-18T05:22:17.995000",
			&bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "208a5bbe-8ee3-11ea-b267-174c3bd0a842",
				Channels:   []string{},
				TMCreate:   "2020-04-18T03:22:17.995000",
				TMDelete:   "2020-04-18T05:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest)

			if err := h.BridgeCreate(context.Background(), tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if err := h.BridgeEnd(context.Background(), tt.bridge.AsteriskID, tt.bridge.ID, tt.timestamp); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.BridgeGet(context.Background(), tt.bridge.AsteriskID, tt.bridge.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMUpdate = ""
			if reflect.DeepEqual(tt.expectBridge, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectBridge, res)
			}
		})
	}
}
