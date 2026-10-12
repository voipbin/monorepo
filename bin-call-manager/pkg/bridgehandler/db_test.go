package bridgehandler

import (
	"context"
	reflect "reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	type test struct {
		name string

		asteriskID    string
		id            string
		bridgeName    string
		bridgeType    bridge.Type
		tech          bridge.Tech
		class         string
		creator       string
		videoMode     string
		videoSourceID string
		referenceType bridge.ReferenceType
		referenceID   uuid.UUID

		expectBridge *bridge.Bridge

		responseBridge *bridge.Bridge
		expectRes      *bridge.Bridge
	}

	tests := []test{
		{
			"normal",

			"3e:50:6b:43:bb:30",
			"5169be58-6e0d-11ed-9bfa-1bb6e1bc670f",
			"test bridge",
			bridge.TypeMixing,
			bridge.TechSoftmix,
			"stasis",
			"Stasis",
			"none",
			"",
			bridge.ReferenceTypeUnknown,
			uuid.Nil,

			&bridge.Bridge{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "5169be58-6e0d-11ed-9bfa-1bb6e1bc670f",
				Name:       "test bridge",
				Type:       bridge.TypeMixing,
				Tech:       bridge.TechSoftmix,
				Class:      "stasis",
				Creator:    "Stasis",

				VideoMode:     "none",
				VideoSourceID: "",

				ChannelIDs: []string{},

				ReferenceType: bridge.ReferenceTypeUnknown,
				ReferenceID:   uuid.Nil,
			},

			&bridge.Bridge{
				ID: "2f9f0728-7192-11ed-9202-2fcb8e8e8f30",
			},
			&bridge.Bridge{
				ID: "2f9f0728-7192-11ed-9202-2fcb8e8e8f30",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := bridgeHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().BridgeCreate(ctx, tt.expectBridge).Return(nil)
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.id).Return(tt.responseBridge, nil)

			res, err := h.Create(
				ctx,

				tt.asteriskID,
				tt.id,
				tt.bridgeName,

				tt.bridgeType,
				tt.tech,
				tt.class,
				tt.creator,

				tt.videoMode,
				tt.videoSourceID,

				tt.referenceType,
				tt.referenceID,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	type test struct {
		name string

		id string

		responseBridge *bridge.Bridge
		expectRes      *bridge.Bridge
	}

	tests := []test{
		{
			"normal",

			"ccaceb42-7193-11ed-92f4-83ac546b4d21",

			&bridge.Bridge{
				ID: "ccaceb42-7193-11ed-92f4-83ac546b4d21",
			},
			&bridge.Bridge{
				ID: "ccaceb42-7193-11ed-92f4-83ac546b4d21",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := bridgeHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.id).Return(tt.responseBridge, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	type test struct {
		name string

		id string

		responseBridge *bridge.Bridge
		expectRes      *bridge.Bridge
	}

	tests := []test{
		{
			"normal",

			"efed02b8-7193-11ed-913f-339a78a1c39f",

			&bridge.Bridge{
				ID: "efed02b8-7193-11ed-913f-339a78a1c39f",
			},
			&bridge.Bridge{
				ID: "efed02b8-7193-11ed-913f-339a78a1c39f",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := bridgeHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().BridgeEnd(ctx, tt.id).Return(nil)
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.id).Return(tt.responseBridge, nil)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AddChannelID(t *testing.T) {

	type test struct {
		name string

		id        string
		channelID string

		responseBridge *bridge.Bridge
		expectRes      *bridge.Bridge
	}

	tests := []test{
		{
			"normal",

			"10eabc3a-7194-11ed-a900-6357cd06f37b",
			"111ea40a-7194-11ed-b6c5-bbb12546d9ab",

			&bridge.Bridge{
				ID: "10eabc3a-7194-11ed-a900-6357cd06f37b",
			},
			&bridge.Bridge{
				ID: "10eabc3a-7194-11ed-a900-6357cd06f37b",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := bridgeHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().BridgeAddChannelID(ctx, tt.id, tt.channelID).Return(nil)
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.id).Return(tt.responseBridge, nil)

			res, err := h.AddChannelID(ctx, tt.id, tt.channelID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_RemoveChannelID(t *testing.T) {

	type test struct {
		name string

		id        string
		channelID string

		responseBridge *bridge.Bridge
		expectRes      *bridge.Bridge
	}

	tests := []test{
		{
			"normal",

			"48030844-7194-11ed-a878-0fdf6c437831",
			"4826a0c4-7194-11ed-a238-ef709bc7808e",

			&bridge.Bridge{
				ID: "48030844-7194-11ed-a878-0fdf6c437831",
			},
			&bridge.Bridge{
				ID: "48030844-7194-11ed-a878-0fdf6c437831",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := bridgeHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().BridgeRemoveChannelID(ctx, tt.id, tt.channelID).Return(nil)
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.id).Return(tt.responseBridge, nil)

			res, err := h.RemoveChannelID(ctx, tt.id, tt.channelID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getWithTimeout_cancelledContext(t *testing.T) {

	type test struct {
		name string

		id         string
		iterations int
	}

	tests := []test{
		{
			name: "context is already cancelled",

			id:         "c3d1a8e2-a7e4-11f0-8b1d-4f6a2c9e7d05",
			iterations: 50000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := bridgeHandler{
				db: mockDB,
			}

			// The retry goroutine sees the cancelled context before its
			// first lookup, so BridgeGet is never called. The (nil, nil)
			// race this guards against is probabilistic, hence the loop.
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			for i := 0; i < tt.iterations; i++ {
				res, err := h.getWithTimeout(ctx, tt.id, defaultExistTimeout)
				if res != nil || err == nil {
					t.Errorf("Wrong match. iteration: %d, expect: (nil, error), got: (%v, %v)", i, res, err)
					break
				}
			}
		})
	}
}

func Test_getWithTimeout_lateSuccess(t *testing.T) {

	type test struct {
		name string

		id             string
		responseBridge *bridge.Bridge
	}

	tests := []test{
		{
			name: "lookup succeeds after the context is cancelled",

			id: "d4e2b9f3-a7e4-11f0-9c2e-5a7b3d0f8e06",
			responseBridge: &bridge.Bridge{
				ID: "d4e2b9f3-a7e4-11f0-9c2e-5a7b3d0f8e06",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := bridgeHandler{
				db: mockDB,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Give getWithTimeout time to start waiting, cancel the lookup,
			// then succeed anyway: the result arrives after the caller has
			// given up, which used to leave the goroutine blocked on send.
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.id).DoAndReturn(func(c context.Context, _ string) (*bridge.Bridge, error) {
				time.Sleep(50 * time.Millisecond)
				cancel()
				<-c.Done()
				return tt.responseBridge, nil
			}).Times(1)

			res, err := h.getWithTimeout(ctx, tt.id, 10*time.Second)
			if err == nil {
				if reflect.DeepEqual(tt.responseBridge, res) == false {
					t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseBridge, res)
				}
			} else if res != nil {
				t.Errorf("Wrong match. expect: nil result with error, got: %v", res)
			}

			deadline := time.Now().Add(2 * time.Second)
			for {
				count, first := countGetWithTimeoutGoroutines()
				if count == 0 {
					break
				}
				if time.Now().After(deadline) {
					t.Errorf("Wrong match. expect: no getWithTimeout goroutine, got: %d left\n%s", count, first)
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
		})
	}
}

// countGetWithTimeoutGoroutines returns how many goroutines are running the
// getWithTimeout retry closure, and the stack of the first one found.
func countGetWithTimeoutGoroutines() (int, string) {
	buf := make([]byte, 1<<16)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		buf = make([]byte, 2*len(buf))
	}

	count := 0
	first := ""
	for _, block := range strings.Split(string(buf), "\n\n") {
		if !strings.Contains(block, "bridgehandler.(*bridgeHandler).getWithTimeout.func") {
			continue
		}
		if count == 0 {
			first = block
		}
		count++
	}

	return count, first
}
