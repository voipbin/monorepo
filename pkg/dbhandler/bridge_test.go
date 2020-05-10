package dbhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

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
				ChannelIDs: []string{},
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

			res, err := h.BridgeGet(context.Background(), tt.bridge.ID)
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
				ChannelIDs: []string{},
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

			if err := h.BridgeEnd(context.Background(), tt.bridge.ID, tt.timestamp); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.BridgeGet(context.Background(), tt.bridge.ID)
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

// func TestBridgeAddChannel(t *testing.T) {
// 	type test struct {
// 		name         string
// 		bridge       *bridge.Bridge
// 		channelID    string
// 		expectBridge *bridge.Bridge
// 	}

// 	tests := []test{
// 		{
// 			"test normal",
// 			&bridge.Bridge{
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ID:         "2b45f6ee-917c-11ea-9caa-17f6fdd51eda",
// 				TMCreate:   "2020-04-18T03:22:17.995000",
// 			},
// 			"3fd26c3c-917c-11ea-a8a0-93d5d27da96a",
// 			&bridge.Bridge{
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ID:         "2b45f6ee-917c-11ea-9caa-17f6fdd51eda",
// 				Channels:   []string{"3fd26c3c-917c-11ea-a8a0-93d5d27da96a"},
// 				TMCreate:   "2020-04-18T03:22:17.995000",
// 				TMDelete:   "2020-04-18T05:22:17.995000",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			h := NewHandler(dbTest)

// 			if err := h.BridgeCreate(context.Background(), tt.bridge); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if err := h.BridgeAddChannel(context.Background(), tt.bridge.ID, tt.channelID); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			res, err := h.BridgeGet(context.Background(), tt.bridge.ID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			res.TMUpdate = ""
// 			if reflect.DeepEqual(tt.expectBridge, res) == false {
// 				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectBridge, res)
// 			}
// 		})
// 	}
// }

func TestBridgeGetUntilTimeout(t *testing.T) {
	type test struct {
		name string

		timeout time.Duration
		bridge  *bridge.Bridge
	}

	tests := []test{
		{
			"timeout",
			time.Millisecond * 100,
			&bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "75a53bae-92f9-11ea-90c9-57a00330ee42",
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

			start := time.Now()

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			_, err := h.BridgeGetUntilTimeout(ctx, tt.bridge.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			elapsed := time.Since(start)
			if tt.timeout < elapsed {
				t.Errorf("Wrong match. expect: true, got: false")
			}
		})
	}
}

func TestBridgeGetUntilTimeoutError(t *testing.T) {
	type test struct {
		name string

		timeout time.Duration
		bridge  *bridge.Bridge
	}

	tests := []test{
		{
			"timeout",
			time.Millisecond * 100,
			&bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "cd892d58-92f9-11ea-a524-8f03337a67b5",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest)

			start := time.Now()

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			_, err := h.BridgeGetUntilTimeout(ctx, tt.bridge.ID)
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
			}

			elapsed := time.Since(start)
			if elapsed < tt.timeout {
				t.Errorf("Wrong match. expect: true, got: false")
			}
		})
	}
}

func TestBridgeIsExist(t *testing.T) {
	type test struct {
		name   string
		bridge *bridge.Bridge
	}

	tests := []test{
		{
			"normal",
			&bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "cd892d58-92f9-11ea-a524-8f03337a67b5",
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

			res := h.BridgeIsExist(tt.bridge.ID, time.Second*1)
			if res != true {
				t.Errorf("Wrong match. expect: true, got: false")
			}
		})
	}
}

func TestBridgeIsExistError(t *testing.T) {
	type test struct {
		name    string
		id      string
		timeout time.Duration
	}

	tests := []test{
		{
			"normal",
			"e1b9db5e-92fb-11ea-a300-6f0c56d7b2cc",
			time.Second * 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest)

			start := time.Now()

			res := h.BridgeIsExist(tt.id, tt.timeout)
			if res != false {
				t.Errorf("Wrong match. expect: false, got: true")
			}

			elapsed := time.Since(start)
			if elapsed < tt.timeout {
				t.Errorf("Wrong match. expect: true, got: false")
			}
		})
	}
}
