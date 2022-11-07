package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
)

func TestBridgeCreate(t *testing.T) {

	type test struct {
		name      string
		bridge    *bridge.Bridge
		expectRes *bridge.Bridge
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
		{
			"reference type call",
			&bridge.Bridge{
				AsteriskID:    "3e:50:6b:43:bb:30",
				ID:            "36d8b0be-9316-11ea-b829-6be92ca1faee",
				ReferenceType: bridge.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("23c83b3e-9316-11ea-91c3-ef8d90e0ec42"),
				TMCreate:      "2020-04-18T03:22:17.995000",
			},
			&bridge.Bridge{
				AsteriskID:    "3e:50:6b:43:bb:30",
				ID:            "36d8b0be-9316-11ea-b829-6be92ca1faee",
				ChannelIDs:    []string{},
				ReferenceType: bridge.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("23c83b3e-9316-11ea-91c3-ef8d90e0ec42"),
				TMCreate:      "2020-04-18T03:22:17.995000",
			},
		},
		{
			"reference type conference",
			&bridge.Bridge{
				AsteriskID:    "3e:50:6b:43:bb:30",
				ID:            "5149007a-9316-11ea-9de0-5f9cb2e8c235",
				ReferenceType: bridge.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("560448b8-9316-11ea-a651-b78c9ee8e874"),
				TMCreate:      "2020-04-18T03:22:17.995000",
			},
			&bridge.Bridge{
				AsteriskID:    "3e:50:6b:43:bb:30",
				ID:            "5149007a-9316-11ea-9de0-5f9cb2e8c235",
				ChannelIDs:    []string{},
				ReferenceType: bridge.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("560448b8-9316-11ea-a651-b78c9ee8e874"),
				TMCreate:      "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().BridgeSet(gomock.Any(), gomock.Any())
			if err := h.BridgeCreate(context.Background(), tt.bridge); err != nil {
				t.Errorf("Wrong match. BridgeCreate expect: ok, got: %v", err)
			}

			mockCache.EXPECT().BridgeGet(gomock.Any(), tt.bridge.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().BridgeSet(gomock.Any(), gomock.Any())
			res, err := h.BridgeGet(context.Background(), tt.bridge.ID)
			if err != nil {
				t.Errorf("Wrong match. BridgeGet expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestBridgeEnd(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

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
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().BridgeSet(gomock.Any(), gomock.Any())
			if err := h.BridgeCreate(context.Background(), tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().BridgeSet(gomock.Any(), gomock.Any())
			if err := h.BridgeEnd(context.Background(), tt.bridge.ID, tt.timestamp); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().BridgeGet(gomock.Any(), tt.bridge.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().BridgeSet(gomock.Any(), gomock.Any())
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

func TestBridgeGetUntilTimeout(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

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
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().BridgeSet(gomock.Any(), gomock.Any())
			if err := h.BridgeCreate(context.Background(), tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			start := time.Now()

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			mockCache.EXPECT().BridgeGet(gomock.Any(), tt.bridge.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().BridgeSet(gomock.Any(), gomock.Any())
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
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

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
			h := NewHandler(dbTest, mockCache)

			start := time.Now()

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			mockCache.EXPECT().BridgeGet(gomock.Any(), tt.bridge.ID).Return(nil, fmt.Errorf("")).AnyTimes()
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
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

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
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().BridgeSet(gomock.Any(), gomock.Any())
			if err := h.BridgeCreate(context.Background(), tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().BridgeGet(gomock.Any(), tt.bridge.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().BridgeSet(gomock.Any(), gomock.Any())
			res := h.BridgeIsExist(tt.bridge.ID, time.Second*1)
			if res != true {
				t.Errorf("Wrong match. expect: true, got: false")
			}
		})
	}
}

func TestBridgeIsExistError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

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
			h := NewHandler(dbTest, mockCache)

			start := time.Now()

			mockCache.EXPECT().BridgeGet(gomock.Any(), tt.id).Return(nil, fmt.Errorf("")).AnyTimes()
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
