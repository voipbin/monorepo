package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	_ "github.com/mattn/go-sqlite3"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/eventhandler/models/channel"
)

func TestChannelCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		channel       *channel.Channel
		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"test normal",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "98ff3f2a-8226-11ea-9ec5-079bcb66275c",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "98ff3f2a-8226-11ea-9ec5-079bcb66275c",
				Data:       map[string]interface{}{},
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
		},
		{
			"test normal has state",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "fd4ed562-823f-11ea-a6b2-bbfcd3647952",
				State:      "Up",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "fd4ed562-823f-11ea-a6b2-bbfcd3647952",
				State:      "Up",
				Data:       map[string]interface{}{},
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
		},
		{
			"test normal has data",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "9b89041c-867f-11ea-813b-9f97df78ae0a",
				State:      "Up",
				Data: map[string]interface{}{
					"key1": "val1",
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "9b89041c-867f-11ea-813b-9f97df78ae0a",
				State:      "Up",
				Data: map[string]interface{}{
					"key1": "val1",
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().ChannelSet(gomock.Any(), tt.expectChannel)
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), tt.expectChannel)
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectChannel, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		queryChannel  *channel.Channel
		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"test normal",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "edcf72a4-8230-11ea-9f7f-ff89da373481",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "edcf72a4-8230-11ea-9f7f-ff89da373481",
				Data:       map[string]interface{}{},
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.queryChannel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.expectChannel.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), tt.expectChannel)
			resChannel, err := h.ChannelGet(context.Background(), tt.expectChannel.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok , got: %v", err)
			}

			if reflect.DeepEqual(tt.expectChannel, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelStasisGetUntilTimeout(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		timeout       time.Duration
		channel       *channel.Channel
		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"test normal",
			time.Second * 1,
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "86858c0a-90ae-11ea-950d-2bf631eba312",
				Stasis:     "voipbin",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "86858c0a-90ae-11ea-950d-2bf631eba312",
				Data:       map[string]interface{}{},
				Stasis:     "voipbin",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGetUntilTimeoutWithStasis(ctx, tt.channel.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok , got: %v", err)
			}

			if reflect.DeepEqual(tt.expectChannel, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelStasisGetUntilTimeoutError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		timeout time.Duration
		channel *channel.Channel
	}

	tests := []test{
		{
			"timeout",
			time.Millisecond * 100,
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "c703a640-90ae-11ea-9e20-235745594c22",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any()).AnyTimes()
			_, err := h.ChannelGetUntilTimeoutWithStasis(ctx, tt.channel.ID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func TestChannelEnd(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		channel   *channel.Channel
		hangup    ari.ChannelCause
		timestamp string

		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"test normal",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "810a31da-8245-11ea-881e-df4110bf6754",
				TMCreate:   "2020-04-18T03:22:17.995000",
			},
			ari.ChannelCauseNormalClearing,
			"2020-04-18T03:23:20.995000",
			&channel.Channel{
				AsteriskID:  "3e:50:6b:43:bb:30",
				ID:          "810a31da-8245-11ea-881e-df4110bf6754",
				Data:        map[string]interface{}{},
				HangupCause: ari.ChannelCauseNormalClearing,
				TMCreate:    "2020-04-18T03:22:17.995000",
				TMUpdate:    "2020-04-18T03:23:20.995000",
				TMEnd:       "2020-04-18T03:23:20.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			// prepare
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelEnd(context.Background(), tt.channel.ID, tt.timestamp, tt.hangup); err != nil {
				t.Errorf("Wrong match. expect: ok , got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectChannel, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelSetState(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		channel   *channel.Channel
		state     ari.ChannelState
		timestamp string

		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"test normal ringing",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "bb0010a8-8727-11ea-ae7b-83dba3060609",
				State:      ari.ChannelStateDown,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			ari.ChannelStateRinging,
			"2020-04-20T03:23:20.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "bb0010a8-8727-11ea-ae7b-83dba3060609",
				State:      ari.ChannelStateRinging,
				Data:       map[string]interface{}{},
				TMCreate:   "2020-04-20T03:22:17.995000",
				TMUpdate:   "2020-04-20T03:23:20.995000",
				TMRinging:  "2020-04-20T03:23:20.995000",
			},
		},
		{
			"test normal up",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "d485219e-8727-11ea-b467-83397e16f8da",
				State:      ari.ChannelStateDown,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			ari.ChannelStateUp,
			"2020-04-20T03:23:20.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "d485219e-8727-11ea-b467-83397e16f8da",
				State:      ari.ChannelStateUp,
				Data:       map[string]interface{}{},
				TMCreate:   "2020-04-20T03:22:17.995000",
				TMUpdate:   "2020-04-20T03:23:20.995000",
				TMAnswer:   "2020-04-20T03:23:20.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			// prepare
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetState(context.Background(), tt.channel.ID, tt.timestamp, tt.state); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectChannel, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelSetStasis(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		channel *channel.Channel
		stasis  string

		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"test normal ringing",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "6b2d1f2e-8fd5-11ea-9c77-fbd302019a8f",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			"voipbin",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "6b2d1f2e-8fd5-11ea-9c77-fbd302019a8f",
				State:      ari.ChannelStateRing,
				Stasis:     "voipbin",
				Data:       map[string]interface{}{},
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			// prepare
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetStasis(context.Background(), tt.channel.ID, tt.stasis); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			resChannel.TMUpdate = ""
			if reflect.DeepEqual(tt.expectChannel, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelSetData(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		channel *channel.Channel
		data    map[string]interface{}

		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"empty data",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "f7ca1534-8fd7-11ea-8626-438559ccdb88",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			map[string]interface{}{},
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "f7ca1534-8fd7-11ea-8626-438559ccdb88",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
		},
		{
			"have some data",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "77f761e4-8fd8-11ea-ab40-37a48b9e8971",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			map[string]interface{}{"DOMAIN": "sip-service.voipbin.net", "SOURCE": "213.127.79.161", "CONTEXT": "in-voipbin", "SIP_PAI": "", "SIP_CALLID": "AWV705JjED", "SIP_PRIVACY": ""},
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "77f761e4-8fd8-11ea-ab40-37a48b9e8971",
				State:      ari.ChannelStateRing,
				Data: map[string]interface{}{
					"DOMAIN":      "sip-service.voipbin.net",
					"SOURCE":      "213.127.79.161",
					"CONTEXT":     "in-voipbin",
					"SIP_PAI":     "",
					"SIP_CALLID":  "AWV705JjED",
					"SIP_PRIVACY": "",
				},
				TMCreate: "2020-04-20T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			// prepare
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetData(context.Background(), tt.channel.ID, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			resChannel.TMUpdate = ""
			if reflect.DeepEqual(tt.expectChannel, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelSetDataAndStasis(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		channel *channel.Channel
		data    map[string]interface{}
		stasis  string

		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"empty data",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "e27e0d7e-8fd8-11ea-9b19-5b7e412d9d1c",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			map[string]interface{}{},
			"voipbin",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "e27e0d7e-8fd8-11ea-9b19-5b7e412d9d1c",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				Stasis:     "voipbin",
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
		},
		{
			"have some data",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "de94c572-8fd8-11ea-8a51-cfff145aaab5",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			map[string]interface{}{"DOMAIN": "sip-service.voipbin.net", "SOURCE": "213.127.79.161", "CONTEXT": "in-voipbin", "SIP_PAI": "", "SIP_CALLID": "AWV705JjED", "SIP_PRIVACY": ""},
			"voipbin",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "de94c572-8fd8-11ea-8a51-cfff145aaab5",
				State:      ari.ChannelStateRing,
				Data: map[string]interface{}{
					"DOMAIN":      "sip-service.voipbin.net",
					"SOURCE":      "213.127.79.161",
					"CONTEXT":     "in-voipbin",
					"SIP_PAI":     "",
					"SIP_CALLID":  "AWV705JjED",
					"SIP_PRIVACY": "",
				},
				Stasis:   "voipbin",
				TMCreate: "2020-04-20T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			// prepare
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetDataAndStasis(context.Background(), tt.channel.ID, tt.data, tt.stasis); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			resChannel.TMUpdate = ""
			if reflect.DeepEqual(tt.expectChannel, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelSetBridgeID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		channel  *channel.Channel
		bridgeID string

		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"empty bridgeID",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "4c10052c-9177-11ea-bee2-8f5a79d2f22b",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			"",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "4c10052c-9177-11ea-bee2-8f5a79d2f22b",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				BridgeID:   "",
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
		},
		{
			"have bridge id",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "463ea0ea-9177-11ea-8893-a396f178d2b6",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			"506009d8-9177-11ea-8793-e70255f860f8",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "463ea0ea-9177-11ea-8893-a396f178d2b6",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				BridgeID:   "506009d8-9177-11ea-8793-e70255f860f8",
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			// prepare
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetBridgeID(context.Background(), tt.channel.ID, tt.bridgeID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			resChannel.TMUpdate = ""
			if reflect.DeepEqual(tt.expectChannel, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelSetSIPTransport(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		channel      *channel.Channel
		sipTransport channel.SIPTransport

		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"empty transport",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "fbded60a-e46e-11ea-902e-df33108e8067",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			channel.SIPTransportNone,
			&channel.Channel{
				AsteriskID:   "3e:50:6b:43:bb:30",
				ID:           "fbded60a-e46e-11ea-902e-df33108e8067",
				State:        ari.ChannelStateRing,
				Data:         map[string]interface{}{},
				BridgeID:     "",
				SIPTransport: channel.SIPTransportNone,
				TMCreate:     "2020-04-20T03:22:17.995000",
			},
		},
		{
			"transport udp",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "02aafd92-e46f-11ea-b2fa-47bf7497a896",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			channel.SIPTransportUDP,
			&channel.Channel{
				AsteriskID:   "3e:50:6b:43:bb:30",
				ID:           "02aafd92-e46f-11ea-b2fa-47bf7497a896",
				State:        ari.ChannelStateRing,
				Data:         map[string]interface{}{},
				BridgeID:     "",
				SIPTransport: channel.SIPTransportUDP,
				TMCreate:     "2020-04-20T03:22:17.995000",
			},
		},
		{
			"transport tcp",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "08c3dc4e-e46f-11ea-9485-9b1b4d3b6eff",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			channel.SIPTransportTCP,
			&channel.Channel{
				AsteriskID:   "3e:50:6b:43:bb:30",
				ID:           "08c3dc4e-e46f-11ea-9485-9b1b4d3b6eff",
				State:        ari.ChannelStateRing,
				Data:         map[string]interface{}{},
				BridgeID:     "",
				SIPTransport: channel.SIPTransportTCP,
				TMCreate:     "2020-04-20T03:22:17.995000",
			},
		},
		{
			"transport tls",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "0de1d6cc-e46f-11ea-b74a-8367c248db58",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			channel.SIPTransportTLS,
			&channel.Channel{
				AsteriskID:   "3e:50:6b:43:bb:30",
				ID:           "0de1d6cc-e46f-11ea-b74a-8367c248db58",
				State:        ari.ChannelStateRing,
				Data:         map[string]interface{}{},
				BridgeID:     "",
				SIPTransport: channel.SIPTransportTLS,
				TMCreate:     "2020-04-20T03:22:17.995000",
			},
		},
		{
			"transport wss",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "14465b0a-e46f-11ea-bde1-7bd4574e50ee",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			channel.SIPTransportWSS,
			&channel.Channel{
				AsteriskID:   "3e:50:6b:43:bb:30",
				ID:           "14465b0a-e46f-11ea-bde1-7bd4574e50ee",
				State:        ari.ChannelStateRing,
				Data:         map[string]interface{}{},
				BridgeID:     "",
				SIPTransport: channel.SIPTransportWSS,
				TMCreate:     "2020-04-20T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)
			ctx := context.Background()

			// prepare
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetSIPTransport(ctx, tt.channel.ID, tt.sipTransport); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			resChannel.TMUpdate = ""
			if reflect.DeepEqual(tt.expectChannel, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelSetSIPCallID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		channel   *channel.Channel
		sipCallID string

		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "865526ea-e46f-11ea-8149-5b36febf5766",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			"8b647c44-e46f-11ea-8015-97545f4bc809",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "865526ea-e46f-11ea-8149-5b36febf5766",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				BridgeID:   "",
				SIPCallID:  "8b647c44-e46f-11ea-8015-97545f4bc809",
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)
			ctx := context.Background()

			// prepare
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetSIPCallID(ctx, tt.channel.ID, tt.sipCallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			resChannel.TMUpdate = ""
			if reflect.DeepEqual(tt.expectChannel, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelSetDirection(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		channel   *channel.Channel
		direction channel.Direction

		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"empty direction",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "ca2738ea-dfd3-11ea-8083-971809e1ac12",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			channel.DirectionNone,
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "ca2738ea-dfd3-11ea-8083-971809e1ac12",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				BridgeID:   "",
				Direction:  channel.DirectionNone,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
		},
		{
			"incoming",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "1db9f1d2-dfd4-11ea-b001-7bdfb0d41751",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			channel.DirectionIncoming,
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "1db9f1d2-dfd4-11ea-b001-7bdfb0d41751",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				BridgeID:   "",
				Direction:  channel.DirectionIncoming,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
		},
		{
			"outgoing",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "5dd41c2e-dfd5-11ea-abd7-ef8fe2a633c4",
				State:      ari.ChannelStateRing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			channel.DirectionOutgoing,
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "5dd41c2e-dfd5-11ea-abd7-ef8fe2a633c4",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				BridgeID:   "",
				Direction:  channel.DirectionOutgoing,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)
			ctx := context.Background()

			// prepare
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetDirection(ctx, tt.channel.ID, tt.direction); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			resChannel.TMUpdate = ""
			if reflect.DeepEqual(tt.expectChannel, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func TestChannelGetUntilTimeout(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		timeout time.Duration
		channel *channel.Channel
	}

	tests := []test{
		{
			"timeout",
			time.Millisecond * 100,
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "75a53bae-92f9-11ea-90c9-57a00330ee42",
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

			go func() {
				time.Sleep(time.Millisecond * 10)
				mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
				if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}()

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			_, err := h.ChannelGetUntilTimeout(ctx, tt.channel.ID)
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

func TestChannelGetUntilTimeoutError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		timeout time.Duration
		channel *channel.Channel
	}

	tests := []test{
		{
			"timeout",
			time.Millisecond * 100,
			&channel.Channel{
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

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID).Return(nil, fmt.Errorf("")).AnyTimes()
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any()).AnyTimes()
			_, err := h.ChannelGetUntilTimeout(ctx, tt.channel.ID)
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
