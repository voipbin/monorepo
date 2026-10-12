package channelhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	type test struct {
		name string

		id          string
		asteriskID  string
		channelName string
		channelType channel.Type
		tech        channel.Tech

		// sip information
		sipCallID    string
		sipTransport channel.SIPTransport

		// source/destination
		sourceName        string
		sourceNumber      string
		destinationName   string
		destinationNumber string

		state ari.ChannelState

		responseChannel *channel.Channel
		expectChannel   *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"5169be58-6e0d-11ed-9bfa-1bb6e1bc670f",
			"3e:50:6b:43:bb:30",
			"PJSIP/call-in-00000000",
			channel.TypeCall,
			channel.TechPJSIP,

			"7592d5bc-6e0d-11ed-936f-13316491bdb7",
			channel.SIPTransportTCP,

			"test source name",
			"+821100000001",
			"test destination name",
			"+821100000002",

			ari.ChannelStateRing,

			&channel.Channel{
				ID: "5169be58-6e0d-11ed-9bfa-1bb6e1bc670f",
			},
			&channel.Channel{
				ID:                "5169be58-6e0d-11ed-9bfa-1bb6e1bc670f",
				AsteriskID:        "3e:50:6b:43:bb:30",
				Name:              "PJSIP/call-in-00000000",
				Type:              channel.TypeCall,
				Tech:              channel.TechPJSIP,
				SIPCallID:         "",
				SIPTransport:      "",
				SourceName:        "test source name",
				SourceNumber:      "+821100000001",
				DestinationName:   "test destination name",
				DestinationNumber: "+821100000002",
				State:             ari.ChannelStateRing,
				Data:              map[string]interface{}{},
				StasisName:        "",
				StasisData:        map[channel.StasisDataType]string{},
				BridgeID:          "",
				PlaybackID:        "",
				DialResult:        "",
				HangupCause:       ari.ChannelCauseUnknown,
				Direction:         channel.DirectionNone,
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelCreate(ctx, tt.expectChannel).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(tt.responseChannel, nil)
			mockReq.EXPECT().CallV1ChannelHealth(gomock.Any(), tt.responseChannel.ID, defaultHealthDelay, 0)

			res, err := h.Create(
				ctx,

				tt.id,
				tt.asteriskID,
				tt.channelName,
				tt.channelType,
				tt.tech,

				tt.sourceName,
				tt.sourceNumber,
				tt.destinationName,
				tt.destinationNumber,

				tt.state,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.responseChannel, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChannel, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	type test struct {
		name string

		id string

		responseChannel *channel.Channel
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"c946395c-6e11-11ed-a1ab-73bb9ac04094",

			&channel.Channel{
				ID: "c946395c-6e11-11ed-a1ab-73bb9ac04094",
			},
			&channel.Channel{
				ID: "c946395c-6e11-11ed-a1ab-73bb9ac04094",
			},
		},
	}

	for _, tt := range tests {

		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(tt.responseChannel, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Microsecond * 100)

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getWithTimeout(t *testing.T) {

	type test struct {
		name string

		id string

		responseChannel *channel.Channel
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"f40fd888-9b88-11ed-b227-bfa210c766fb",

			&channel.Channel{
				ID: "f40fd888-9b88-11ed-b227-bfa210c766fb",
			},
			&channel.Channel{
				ID: "f40fd888-9b88-11ed-b227-bfa210c766fb",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(tt.responseChannel, nil)

			res, err := h.getWithTimeout(ctx, tt.id, defaultExistTimeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getWithTimeout_lateSuccess(t *testing.T) {

	type test struct {
		name string

		id              string
		timeout         time.Duration
		responseChannel *channel.Channel
	}

	tests := []test{
		{
			name: "lookup succeeds after the deadline",

			id:      "6a1f3c2e-a7f1-11f0-8d2b-6b8c4e1f9a07",
			timeout: 100 * time.Millisecond,
			responseChannel: &channel.Channel{
				ID: "6a1f3c2e-a7f1-11f0-8d2b-6b8c4e1f9a07",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := channelHandler{
				db: mockDB,
			}

			// The lookup is still in flight when the deadline passes and then
			// succeeds: the caller has already given up, which used to leave
			// the caller and the retry goroutine waiting on each other forever.
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).DoAndReturn(func(c context.Context, _ string) (*channel.Channel, error) {
				<-c.Done()
				time.Sleep(50 * time.Millisecond)
				return tt.responseChannel, nil
			}).Times(1)

			type result struct {
				res *channel.Channel
				err error
			}
			chanResult := make(chan result, 1)
			go func() {
				res, err := h.getWithTimeout(context.Background(), tt.id, tt.timeout)
				chanResult <- result{res: res, err: err}
			}()

			select {
			case r := <-chanResult:
				if r.res != nil || r.err == nil {
					t.Errorf("Wrong match. expect: (nil, error), got: (%v, %v)", r.res, r.err)
				}
			case <-time.After(2 * time.Second):
				t.Errorf("Wrong match. expect: return after the deadline, got: no return within 2s")
			}

			checkNoGetWithTimeoutGoroutine(t)
		})
	}
}

func Test_getWithTimeout_returnsAtDeadline(t *testing.T) {

	type test struct {
		name string

		id      string
		timeout time.Duration
	}

	tests := []test{
		{
			name: "lookup in flight at the deadline",

			id:      "7b2e4d3f-a7f1-11f0-9e3c-7c9d5f2a0b18",
			timeout: 50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := channelHandler{
				db: mockDB,
			}

			// The lookup outlives the deadline until the test releases it. The
			// caller must return at the deadline instead of waiting for the
			// retry goroutine to finish the lookup.
			release := make(chan struct{})
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).DoAndReturn(func(c context.Context, _ string) (*channel.Channel, error) {
				<-c.Done()
				<-release
				return nil, fmt.Errorf("channel not found")
			}).Times(1)

			type result struct {
				res *channel.Channel
				err error
			}
			chanResult := make(chan result, 1)
			go func() {
				res, err := h.getWithTimeout(context.Background(), tt.id, tt.timeout)
				chanResult <- result{res: res, err: err}
			}()

			select {
			case r := <-chanResult:
				if r.res != nil || r.err == nil {
					t.Errorf("Wrong match. expect: (nil, error), got: (%v, %v)", r.res, r.err)
				}
			case <-time.After(2 * time.Second):
				t.Errorf("Wrong match. expect: return at the deadline, got: no return within 2s while the lookup is in flight")
			}
			close(release)

			checkNoGetWithTimeoutGoroutine(t)
		})
	}
}

// checkNoGetWithTimeoutGoroutine waits up to 2 seconds for every
// getWithTimeout retry goroutine to exit, and reports the first one left.
func checkNoGetWithTimeoutGoroutine(t *testing.T) {
	deadline := time.Now().Add(2 * time.Second)
	for {
		count, first := countGetWithTimeoutGoroutines()
		if count == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("Wrong match. expect: no getWithTimeout goroutine, got: %d left\n%s", count, first)
			return
		}
		time.Sleep(10 * time.Millisecond)
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
		if !strings.Contains(block, "channelhandler.(*channelHandler).getWithTimeout.func") {
			continue
		}
		if count == 0 {
			first = block
		}
		count++
	}

	return count, first
}

func Test_Delete(t *testing.T) {

	type test struct {
		name string

		id    string
		cause ari.ChannelCause

		responseChannel *channel.Channel
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"f18b019a-6e11-11ed-beaf-9bbcc1d7b210",
			ari.ChannelCauseNormalClearing,

			&channel.Channel{
				ID: "f18b019a-6e11-11ed-beaf-9bbcc1d7b210",
			},
			&channel.Channel{
				ID: "f18b019a-6e11-11ed-beaf-9bbcc1d7b210",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelEndAndDelete(ctx, tt.id, tt.cause).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(tt.responseChannel, nil)
			res, err := h.Delete(ctx, tt.id, tt.cause)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SetDataItem(t *testing.T) {

	type test struct {
		name string

		id     string
		key    string
		valuse interface{}

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"5c43140a-6e12-11ed-9711-b3bf155a9ed8",
			"key1",
			"value1",

			&channel.Channel{
				ID: "5c43140a-6e12-11ed-9711-b3bf155a9ed8",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetDataItem(ctx, tt.id, tt.key, tt.valuse).Return(nil)
			if err := h.SetDataItem(ctx, tt.id, tt.key, tt.valuse); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_SetSIPTransport(t *testing.T) {

	type test struct {
		name string

		id        string
		transport channel.SIPTransport

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"65ed1bae-6e5d-11ed-a91c-dbd67fe25ab5",
			channel.SIPTransportTCP,

			&channel.Channel{
				ID: "65ed1bae-6e5d-11ed-a91c-dbd67fe25ab5",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetSIPTransport(ctx, tt.id, tt.transport).Return(nil)

			// goroutine
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(&channel.Channel{}, nil)

			if err := h.SetSIPTransport(ctx, tt.id, tt.transport); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Microsecond * 100)
		})
	}
}

func Test_SetDirection(t *testing.T) {

	type test struct {
		name string

		id        string
		direction channel.Direction

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"6643e632-6e5d-11ed-b753-1b5e0ad635df",
			channel.DirectionOutgoing,

			&channel.Channel{
				ID: "6643e632-6e5d-11ed-b753-1b5e0ad635df",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetDirection(ctx, tt.id, tt.direction).Return(nil)

			// goroutine
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(&channel.Channel{}, nil)

			if err := h.SetDirection(ctx, tt.id, tt.direction); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Microsecond * 100)
		})
	}
}

func Test_SetType(t *testing.T) {

	type test struct {
		name string

		id          string
		channelType channel.Type

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"48f30b8e-6e5e-11ed-8355-27bff5c24a53",
			channel.TypeCall,

			&channel.Channel{
				ID: "48f30b8e-6e5e-11ed-8355-27bff5c24a53",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetType(ctx, tt.id, tt.channelType).Return(nil)
			if err := h.SetType(ctx, tt.id, tt.channelType); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateState(t *testing.T) {

	type test struct {
		name string

		id    string
		state ari.ChannelState

		responseChannel *channel.Channel
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"state up",

			"49299d7a-6e5e-11ed-8cf1-2fc1c1f5b170",
			ari.ChannelStateUp,

			&channel.Channel{
				ID: "49299d7a-6e5e-11ed-8cf1-2fc1c1f5b170",
			},
			&channel.Channel{
				ID: "49299d7a-6e5e-11ed-8cf1-2fc1c1f5b170",
			},
		},
		{
			"state ring",

			"690e29bc-6e6d-11ed-b10b-0f521d477a9e",
			ari.ChannelStateRing,

			&channel.Channel{
				ID: "690e29bc-6e6d-11ed-b10b-0f521d477a9e",
			},
			&channel.Channel{
				ID: "690e29bc-6e6d-11ed-b10b-0f521d477a9e",
			},
		},
		{
			"state ringing",

			"690e29bc-6e6d-11ed-b10b-0f521d477a9e",
			ari.ChannelStateRinging,

			&channel.Channel{
				ID: "69525ac4-6e6d-11ed-b750-2ba5cf3aa1d6",
			},
			&channel.Channel{
				ID: "69525ac4-6e6d-11ed-b750-2ba5cf3aa1d6",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			switch tt.state {
			case ari.ChannelStateUp:
				mockDB.EXPECT().ChannelSetStateAnswer(ctx, tt.id, tt.state).Return(nil)
			case ari.ChannelStateRing, ari.ChannelStateRinging:
				mockDB.EXPECT().ChannelSetStateRinging(ctx, tt.id, tt.state).Return(nil)
			}
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(tt.responseChannel, nil)

			res, err := h.UpdateState(ctx, tt.id, tt.state)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_setSIPCallID(t *testing.T) {

	type test struct {
		name string

		id        string
		sipCallID string
	}

	tests := []test{
		{
			name: "normal",

			id:        "04245724-f1d5-11ed-a080-2f94d1616a75",
			sipCallID: "0c1e1b40-f1d5-11ed-8d96-c37af7788faa",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetSIPCallID(ctx, tt.id, tt.sipCallID).Return(nil)

			if err := h.setSIPCallID(ctx, tt.id, tt.sipCallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_setSIPPai(t *testing.T) {

	type test struct {
		name string

		id     string
		sipPai string
	}

	tests := []test{
		{
			name: "normal",

			id:     "04245724-f1d5-11ed-a080-2f94d1616a75",
			sipPai: "0c1e1b40-f1d5-11ed-8d96-c37af7788faa",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetDataItem(ctx, tt.id, "sip_pai", tt.sipPai).Return(nil)

			if err := h.setSIPPai(ctx, tt.id, tt.sipPai); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_setSIPPrivacy(t *testing.T) {

	type test struct {
		name string

		id         string
		sipPrivacy string
	}

	tests := []test{
		{
			name: "normal",

			id:         "88d8c6bc-f1d5-11ed-911b-972e99d029f5",
			sipPrivacy: "id",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetDataItem(ctx, tt.id, "sip_privacy", tt.sipPrivacy).Return(nil)

			if err := h.setSIPPrivacy(ctx, tt.id, tt.sipPrivacy); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateStasisName(t *testing.T) {

	type test struct {
		name string

		id         string
		stasisName string

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			name: "normal",

			id:         "b7e84cf2-f1d5-11ed-a3ad-4fc6f2acba8c",
			stasisName: "voipbin",

			responseChannel: &channel.Channel{
				ID: "b7e84cf2-f1d5-11ed-a3ad-4fc6f2acba8c",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetStasis(ctx, tt.id, tt.stasisName).Return(nil)
			mockDB.EXPECT().ChannelGet(ctx, tt.id).Return(tt.responseChannel, nil)

			res, err := h.UpdateStasisName(ctx, tt.id, tt.stasisName)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseChannel, res) {
				t.Errorf("Wrong match.\nexpect: %vgot: %v", tt.responseChannel, res)
			}
		})
	}
}

func Test_UpdateBridgeID(t *testing.T) {

	type test struct {
		name string

		id       string
		bridgeID string

		responseChannel *channel.Channel
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"bcb4bf86-6e5e-11ed-8165-bb45c04f334c",
			"bcdc5050-6e5e-11ed-9b50-6fc652dbf99a",

			&channel.Channel{
				ID: "bcb4bf86-6e5e-11ed-8165-bb45c04f334c",
			},
			&channel.Channel{
				ID: "bcb4bf86-6e5e-11ed-8165-bb45c04f334c",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetBridgeID(ctx, tt.id, tt.bridgeID).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(tt.responseChannel, nil)

			res, err := h.UpdateBridgeID(ctx, tt.id, tt.bridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_UpdatePlaybackID(t *testing.T) {

	type test struct {
		name string

		id         string
		playbackID string

		responseChannel *channel.Channel
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"2b7b2f90-f1d6-11ed-afd3-f38ba952ed15",
			"2ba4b180-f1d6-11ed-b331-07ed2083bb9c",

			&channel.Channel{
				ID: "2b7b2f90-f1d6-11ed-afd3-f38ba952ed15",
			},
			&channel.Channel{
				ID: "2b7b2f90-f1d6-11ed-afd3-f38ba952ed15",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetPlaybackID(ctx, tt.id, tt.playbackID).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(tt.responseChannel, nil)

			res, err := h.UpdatePlaybackID(ctx, tt.id, tt.playbackID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_UpdateMuteDirection(t *testing.T) {

	type test struct {
		name string

		id            string
		muteDirection channel.MuteDirection

		responseChannel *channel.Channel
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"e07af06a-d246-11ed-abee-7b29895cb95c",
			channel.MuteDirectionBoth,

			&channel.Channel{
				ID: "e07af06a-d246-11ed-abee-7b29895cb95c",
			},
			&channel.Channel{
				ID: "e07af06a-d246-11ed-abee-7b29895cb95c",
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

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetMuteDirection(ctx, tt.id, tt.muteDirection).Return(nil)
			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(tt.responseChannel, nil)

			res, err := h.UpdateMuteDirection(ctx, tt.id, tt.muteDirection)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
