package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"

	_ "github.com/mattn/go-sqlite3"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/cachehandler"
)

func Test_ChannelCreate(t *testing.T) {

	type test struct {
		name string

		channel *channel.Channel

		responseCurTime string

		expectChannel *channel.Channel
	}

	tests := []test{
		{
			"test normal",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "98ff3f2a-8226-11ea-9ec5-079bcb66275c",
			},

			"2020-04-18T03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "98ff3f2a-8226-11ea-9ec5-079bcb66275c",
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},
				TMCreate:   "2020-04-18T03:22:17.995000",
				TMUpdate:   DefaultTimeStamp,
				TMDelete:   DefaultTimeStamp,
				TMAnswer:   DefaultTimeStamp,
				TMRinging:  DefaultTimeStamp,
				TMEnd:      DefaultTimeStamp,
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

			"2020-04-18T03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "fd4ed562-823f-11ea-a6b2-bbfcd3647952",
				State:      "Up",
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},
				TMCreate:   "2020-04-18T03:22:17.995000",
				TMUpdate:   DefaultTimeStamp,
				TMDelete:   DefaultTimeStamp,
				TMAnswer:   DefaultTimeStamp,
				TMRinging:  DefaultTimeStamp,
				TMEnd:      DefaultTimeStamp,
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

			"2020-04-18T03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "9b89041c-867f-11ea-813b-9f97df78ae0a",
				State:      "Up",
				Data: map[string]interface{}{
					"key1": "val1",
				},
				StasisData: map[channel.StasisDataType]string{},
				TMCreate:   "2020-04-18T03:22:17.995000",
				TMUpdate:   DefaultTimeStamp,
				TMDelete:   DefaultTimeStamp,
				TMAnswer:   DefaultTimeStamp,
				TMRinging:  DefaultTimeStamp,
				TMEnd:      DefaultTimeStamp,
			},
		},
		{
			"test normal has stasis data",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "19b5d1e2-3793-11ec-906e-e37773ea39d0",
				State:      "Up",
				StasisData: map[channel.StasisDataType]string{
					"key1": "val1",
				},
				TMCreate: "2020-04-18T03:22:17.995000",
			},

			"2020-04-18T03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "19b5d1e2-3793-11ec-906e-e37773ea39d0",
				State:      "Up",
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{
					"key1": "val1",
				},
				TMCreate:  "2020-04-18T03:22:17.995000",
				TMUpdate:  DefaultTimeStamp,
				TMDelete:  DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMRinging: DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), tt.expectChannel)
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID.Return(nil, fmt.Errorf(""))
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

func Test_ChannelGet(t *testing.T) {

	type test struct {
		name string

		channel *channel.Channel

		responseCurTime string
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"test normal",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "edcf72a4-8230-11ea-9f7f-ff89da373481",
			},

			"2020-04-18T03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "edcf72a4-8230-11ea-9f7f-ff89da373481",
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},
				TMRinging:  DefaultTimeStamp,
				TMAnswer:   DefaultTimeStamp,
				TMEnd:      DefaultTimeStamp,

				TMCreate: "2020-04-18T03:22:17.995000",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.expectRes.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), tt.expectRes)
			resChannel, err := h.ChannelGet(context.Background(), tt.expectRes.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok , got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, resChannel)
			}
		})
	}
}

func Test_ChannelEndAndDelete(t *testing.T) {
	type test struct {
		name string

		channel *channel.Channel
		hangup  ari.ChannelCause

		responseCurTime string
		expectChannel   *channel.Channel
	}

	tests := []test{
		{
			"test normal",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "810a31da-8245-11ea-881e-df4110bf6754",
			},
			ari.ChannelCauseNormalClearing,

			"2020-04-18 03:22:17.995000",
			&channel.Channel{
				AsteriskID:  "3e:50:6b:43:bb:30",
				ID:          "810a31da-8245-11ea-881e-df4110bf6754",
				Data:        map[string]interface{}{},
				StasisData:  map[channel.StasisDataType]string{},
				HangupCause: ari.ChannelCauseNormalClearing,

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     "2020-04-18 03:22:17.995000",

				TMCreate: "2020-04-18 03:22:17.995000",
				TMUpdate: "2020-04-18 03:22:17.995000",
				TMDelete: "2020-04-18 03:22:17.995000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelEndAndDelete(context.Background(), tt.channel.ID, tt.hangup); err != nil {
				t.Errorf("Wrong match. expect: ok , got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectChannel, resChannel) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectChannel, resChannel)
			}
		})
	}
}

func Test_ChannelSetStateAnswer(t *testing.T) {

	type test struct {
		name string

		channel *channel.Channel
		state   ari.ChannelState

		responseCurTime string
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "bbed0da6-6e6d-11ed-9544-937fb1cf3a60",
				State:      ari.ChannelStateDown,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			ari.ChannelStateUp,

			"2020-04-20 03:23:20.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "bbed0da6-6e6d-11ed-9544-937fb1cf3a60",
				State:      ari.ChannelStateUp,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},

				TMAnswer:  "2020-04-20 03:23:20.995000",
				TMRinging: DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:23:20.995000",
				TMUpdate: "2020-04-20 03:23:20.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			// prepare
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(ctx, gomock.Any())
			if err := h.ChannelCreate(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(ctx, gomock.Any())
			if err := h.ChannelSetStateAnswer(ctx, tt.channel.ID, tt.state); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(ctx, tt.channel.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(ctx, gomock.Any())
			resChannel, err := h.ChannelGet(ctx, tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, resChannel)
			}
		})
	}
}

func Test_ChannelSetStateRinging(t *testing.T) {

	type test struct {
		name string

		channel *channel.Channel
		state   ari.ChannelState

		responseCurTime string
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"state ring",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "dbb6d036-6e6d-11ed-8256-7b4b5eef5694",
				State:      ari.ChannelStateDown,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			ari.ChannelStateRing,

			"2020-04-20 03:23:20.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "dbb6d036-6e6d-11ed-8256-7b4b5eef5694",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},

				TMAnswer:  DefaultTimeStamp,
				TMRinging: "2020-04-20 03:23:20.995000",
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:23:20.995000",
				TMUpdate: "2020-04-20 03:23:20.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"state ringing",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "f03c8a28-6e6d-11ed-a20b-bfaa56fb5a4c",
				State:      ari.ChannelStateDown,
				TMCreate:   "2020-04-20T03:22:17.995000",
			},
			ari.ChannelStateRing,

			"2020-04-20 03:23:20.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "f03c8a28-6e6d-11ed-a20b-bfaa56fb5a4c",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},

				TMAnswer:  DefaultTimeStamp,
				TMRinging: "2020-04-20 03:23:20.995000",
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:23:20.995000",
				TMUpdate: "2020-04-20 03:23:20.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			// prepare
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(ctx, gomock.Any())
			if err := h.ChannelCreate(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(ctx, gomock.Any())
			if err := h.ChannelSetStateRinging(ctx, tt.channel.ID, tt.state); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(ctx, tt.channel.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(ctx, gomock.Any())
			resChannel, err := h.ChannelGet(ctx, tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, resChannel)
			}
		})
	}
}

func Test_ChannelSetStasis(t *testing.T) {

	type test struct {
		name string

		channel *channel.Channel
		stasis  string

		responseCurTime string
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"test normal ringing",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "6b2d1f2e-8fd5-11ea-9c77-fbd302019a8f",
				State:      ari.ChannelStateRing,
			},
			"voipbin",

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "6b2d1f2e-8fd5-11ea-9c77-fbd302019a8f",
				State:      ari.ChannelStateRing,
				StasisName: "voipbin",
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			// prepare
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetStasis(context.Background(), tt.channel.ID, tt.stasis); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, resChannel) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, resChannel)
			}
		})
	}
}

func Test_ChannelSetType(t *testing.T) {

	type test struct {
		name string

		channel *channel.Channel
		cType   channel.Type

		responseCurTime string
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"type none",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "6dac9dec-e548-11ea-945f-7b58ec7f18f5",
				State:      ari.ChannelStateRing,
			},
			channel.TypeNone,

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "6dac9dec-e548-11ea-945f-7b58ec7f18f5",
				State:      ari.ChannelStateRing,
				Type:       channel.TypeNone,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"type call",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "a9891886-e548-11ea-bd56-2f7c4e2675d0",
				State:      ari.ChannelStateRing,
			},
			channel.TypeCall,

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "a9891886-e548-11ea-bd56-2f7c4e2675d0",
				State:      ari.ChannelStateRing,
				Type:       channel.TypeCall,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"type conf",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "b88dea32-e548-11ea-8fd0-9f74b211e14a",
				State:      ari.ChannelStateRing,
			},
			channel.TypeConfbridge,

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "b88dea32-e548-11ea-8fd0-9f74b211e14a",
				State:      ari.ChannelStateRing,
				Type:       channel.TypeConfbridge,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"type join",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "c6e3c3b8-e548-11ea-b3d1-131c49931114",
				State:      ari.ChannelStateRing,
			},
			channel.TypeJoin,

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "c6e3c3b8-e548-11ea-b3d1-131c49931114",
				State:      ari.ChannelStateRing,
				Type:       channel.TypeJoin,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			// prepare
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetType(context.Background(), tt.channel.ID, tt.cType); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, resChannel)
			}
		})
	}
}

func Test_ChannelSetData(t *testing.T) {

	type test struct {
		name string

		channel *channel.Channel
		data    map[string]interface{}

		responseCurTime string
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"empty data",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "f7ca1534-8fd7-11ea-8626-438559ccdb88",
				State:      ari.ChannelStateRing,
			},
			map[string]interface{}{},

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "f7ca1534-8fd7-11ea-8626-438559ccdb88",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"have some data",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "77f761e4-8fd8-11ea-ab40-37a48b9e8971",
				State:      ari.ChannelStateRing,
			},
			map[string]interface{}{"DOMAIN": "sip-service.voipbin.net", "SOURCE": "213.127.79.161", "CONTEXT": "in-voipbin", "SIP_PAI": "", "SIP_CALLID": "AWV705JjED", "SIP_PRIVACY": ""},

			"2020-04-20 03:22:17.995000",
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
				StasisData: map[channel.StasisDataType]string{},

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			// prepare
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetData(context.Background(), tt.channel.ID, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, resChannel)
			}
		})
	}
}

func Test_ChannelSetStasisInfo(t *testing.T) {

	type test struct {
		name    string
		channel *channel.Channel

		id string

		channelType channel.Type
		stasisName  string
		stasisData  map[channel.StasisDataType]string

		direction channel.Direction

		responseCurTime string
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			name: "normal",

			channel: &channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "670c5460-1b3f-4548-80c9-d3c96cef6a58",
				State:      ari.ChannelStateRing,
			},

			id: "670c5460-1b3f-4548-80c9-d3c96cef6a58",

			channelType: channel.TypeCall,
			stasisName:  "voipbin",
			stasisData: map[channel.StasisDataType]string{
				channel.StasisDataTypeContextType: "call",
				channel.StasisDataTypeContext:     "call-in",
				channel.StasisDataTypeDomain:      "sip-service.voipbin.net",
				channel.StasisDataTypeSource:      "213.127.79.161",
				channel.StasisDataTypeSIPPAI:      "",
				channel.StasisDataTypeSIPCallID:   "AWV705JjED",
				channel.StasisDataTypeSIPPrivacy:  "",
			},
			direction: channel.DirectionIncoming,

			responseCurTime: "2020-04-20 03:22:17.995000",
			expectRes: &channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "670c5460-1b3f-4548-80c9-d3c96cef6a58",
				Type:       channel.TypeCall,

				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				StasisName: "voipbin",
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeContextType: "call",
					channel.StasisDataTypeContext:     "call-in",
					channel.StasisDataTypeDomain:      "sip-service.voipbin.net",
					channel.StasisDataTypeSource:      "213.127.79.161",
					channel.StasisDataTypeSIPPAI:      "",
					channel.StasisDataTypeSIPCallID:   "AWV705JjED",
					channel.StasisDataTypeSIPPrivacy:  "",
				},
				Direction: channel.DirectionIncoming,

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			// prepare
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetStasisInfo(
				ctx,
				tt.channel.ID,
				tt.channelType,
				tt.stasisName,
				tt.stasisData,
				tt.direction,
			); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			res, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ChannelSetBridgeID(t *testing.T) {

	type test struct {
		name string

		channel  *channel.Channel
		bridgeID string

		responseCurTime string
		expectRes       *channel.Channel
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

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "4c10052c-9177-11ea-bee2-8f5a79d2f22b",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},
				BridgeID:   "",

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"have bridge id",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "463ea0ea-9177-11ea-8893-a396f178d2b6",
				State:      ari.ChannelStateRing,
			},
			"506009d8-9177-11ea-8793-e70255f860f8",

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "463ea0ea-9177-11ea-8893-a396f178d2b6",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},
				BridgeID:   "506009d8-9177-11ea-8793-e70255f860f8",

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			// prepare
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(context.Background(), tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetBridgeID(context.Background(), tt.channel.ID, tt.bridgeID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, resChannel)
			}
		})
	}
}

func Test_ChannelSetSIPTransport(t *testing.T) {

	type test struct {
		name string

		channel      *channel.Channel
		sipTransport channel.SIPTransport

		responseCurTime string

		expectRes *channel.Channel
	}

	tests := []test{
		{
			"empty transport",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "fbded60a-e46e-11ea-902e-df33108e8067",
				State:      ari.ChannelStateRing,
			},
			channel.SIPTransportNone,

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID:   "3e:50:6b:43:bb:30",
				ID:           "fbded60a-e46e-11ea-902e-df33108e8067",
				State:        ari.ChannelStateRing,
				Data:         map[string]interface{}{},
				StasisData:   map[channel.StasisDataType]string{},
				BridgeID:     "",
				SIPTransport: channel.SIPTransportNone,

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"transport udp",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "02aafd92-e46f-11ea-b2fa-47bf7497a896",
				State:      ari.ChannelStateRing,
			},
			channel.SIPTransportUDP,

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID:   "3e:50:6b:43:bb:30",
				ID:           "02aafd92-e46f-11ea-b2fa-47bf7497a896",
				State:        ari.ChannelStateRing,
				Data:         map[string]interface{}{},
				StasisData:   map[channel.StasisDataType]string{},
				BridgeID:     "",
				SIPTransport: channel.SIPTransportUDP,

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"transport tcp",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "08c3dc4e-e46f-11ea-9485-9b1b4d3b6eff",
				State:      ari.ChannelStateRing,
			},
			channel.SIPTransportTCP,

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID:   "3e:50:6b:43:bb:30",
				ID:           "08c3dc4e-e46f-11ea-9485-9b1b4d3b6eff",
				State:        ari.ChannelStateRing,
				Data:         map[string]interface{}{},
				StasisData:   map[channel.StasisDataType]string{},
				BridgeID:     "",
				SIPTransport: channel.SIPTransportTCP,

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"transport tls",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "0de1d6cc-e46f-11ea-b74a-8367c248db58",
				State:      ari.ChannelStateRing,
			},
			channel.SIPTransportTLS,

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID:   "3e:50:6b:43:bb:30",
				ID:           "0de1d6cc-e46f-11ea-b74a-8367c248db58",
				State:        ari.ChannelStateRing,
				Data:         map[string]interface{}{},
				StasisData:   map[channel.StasisDataType]string{},
				BridgeID:     "",
				SIPTransport: channel.SIPTransportTLS,

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"transport wss",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "14465b0a-e46f-11ea-bde1-7bd4574e50ee",
				State:      ari.ChannelStateRing,
			},
			channel.SIPTransportWSS,

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID:   "3e:50:6b:43:bb:30",
				ID:           "14465b0a-e46f-11ea-bde1-7bd4574e50ee",
				State:        ari.ChannelStateRing,
				Data:         map[string]interface{}{},
				StasisData:   map[channel.StasisDataType]string{},
				BridgeID:     "",
				SIPTransport: channel.SIPTransportWSS,

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			// prepare
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetSIPTransport(ctx, tt.channel.ID, tt.sipTransport); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, resChannel)
			}
		})
	}
}

func Test_ChannelSetSIPCallID(t *testing.T) {

	type test struct {
		name string

		channel   *channel.Channel
		sipCallID string

		responseCurTime string
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "865526ea-e46f-11ea-8149-5b36febf5766",
				State:      ari.ChannelStateRing,
			},
			"8b647c44-e46f-11ea-8015-97545f4bc809",

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "865526ea-e46f-11ea-8149-5b36febf5766",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},
				BridgeID:   "",
				SIPCallID:  "8b647c44-e46f-11ea-8015-97545f4bc809",

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			// prepare
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetSIPCallID(ctx, tt.channel.ID, tt.sipCallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, resChannel)
			}
		})
	}
}

func Test_ChannelSetDirection(t *testing.T) {

	type test struct {
		name string

		channel   *channel.Channel
		direction channel.Direction

		responseCurTime string
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"empty direction",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "ca2738ea-dfd3-11ea-8083-971809e1ac12",
				State:      ari.ChannelStateRing,
			},
			channel.DirectionNone,

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "ca2738ea-dfd3-11ea-8083-971809e1ac12",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},
				BridgeID:   "",
				Direction:  channel.DirectionNone,

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"incoming",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "1db9f1d2-dfd4-11ea-b001-7bdfb0d41751",
				State:      ari.ChannelStateRing,
			},
			channel.DirectionIncoming,

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "1db9f1d2-dfd4-11ea-b001-7bdfb0d41751",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},
				BridgeID:   "",
				Direction:  channel.DirectionIncoming,

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
		{
			"outgoing",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "5dd41c2e-dfd5-11ea-abd7-ef8fe2a633c4",
				State:      ari.ChannelStateRing,
			},
			channel.DirectionOutgoing,

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "5dd41c2e-dfd5-11ea-abd7-ef8fe2a633c4",
				State:      ari.ChannelStateRing,
				Data:       map[string]interface{}{},
				StasisData: map[channel.StasisDataType]string{},
				BridgeID:   "",
				Direction:  channel.DirectionOutgoing,

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			// prepare
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetDirection(ctx, tt.channel.ID, tt.direction); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, resChannel)
			}
		})
	}
}

func Test_ChannelSetMuteDirection(t *testing.T) {

	type test struct {
		name string

		channel       *channel.Channel
		muteDirection channel.MuteDirection

		responseCurTime string
		expectRes       *channel.Channel
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				AsteriskID: "3e:50:6b:43:bb:30",
				ID:         "7ac68c3a-d245-11ed-b6dd-53479be2c198",
			},
			channel.MuteDirectionBoth,

			"2020-04-20 03:22:17.995000",
			&channel.Channel{
				AsteriskID:    "3e:50:6b:43:bb:30",
				ID:            "7ac68c3a-d245-11ed-b6dd-53479be2c198",
				Data:          map[string]interface{}{},
				StasisData:    map[channel.StasisDataType]string{},
				MuteDirection: channel.MuteDirectionBoth,

				TMRinging: DefaultTimeStamp,
				TMAnswer:  DefaultTimeStamp,
				TMEnd:     DefaultTimeStamp,

				TMCreate: "2020-04-20 03:22:17.995000",
				TMUpdate: "2020-04-20 03:22:17.995000",
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			// prepare
			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelCreate(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTime)
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			if err := h.ChannelSetMuteDirection(ctx, tt.channel.ID, tt.muteDirection); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ChannelGet(gomock.Any(), tt.channel.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ChannelSet(gomock.Any(), gomock.Any())
			resChannel, err := h.ChannelGet(context.Background(), tt.channel.ID)
			if err != nil {
				t.Errorf("Could not get channel. err: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, resChannel) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, resChannel)
			}
		})
	}
}

func Test_ChannelGets(t *testing.T) {

	type test struct {
		name     string
		channels []*channel.Channel

		filters map[string]string

		responseCurTimes []string

		expectRes []*channel.Channel
	}

	tests := []test{
		{
			name: "normal",
			channels: []*channel.Channel{
				{
					ID:         "3b29f23c-42ec-11f0-ad0b-9360df8ed5c7",
					AsteriskID: "3e:50:6b:43:bb:31",
				},
				{
					ID:         "3b733fc8-42ec-11f0-9fa5-4f77e55ccdbd",
					AsteriskID: "3e:50:6b:43:bb:31",
				},
			},

			filters: map[string]string{
				"deleted":     "false",
				"asterisk_id": "3e:50:6b:43:bb:31",
			},

			responseCurTimes: []string{
				"2020-04-18 03:22:17.995000",
				"2020-04-18 03:22:18.995000",
			},

			expectRes: []*channel.Channel{
				{
					ID:         "3b733fc8-42ec-11f0-9fa5-4f77e55ccdbd",
					AsteriskID: "3e:50:6b:43:bb:31",

					Data:       map[string]interface{}{},
					StasisData: map[channel.StasisDataType]string{},

					TMAnswer:  DefaultTimeStamp,
					TMRinging: DefaultTimeStamp,
					TMEnd:     DefaultTimeStamp,
					TMCreate:  "2020-04-18 03:22:18.995000",
					TMUpdate:  DefaultTimeStamp,
					TMDelete:  DefaultTimeStamp,
				},
				{
					ID:         "3b29f23c-42ec-11f0-ad0b-9360df8ed5c7",
					AsteriskID: "3e:50:6b:43:bb:31",

					Data:       map[string]interface{}{},
					StasisData: map[channel.StasisDataType]string{},

					TMAnswer:  DefaultTimeStamp,
					TMRinging: DefaultTimeStamp,
					TMEnd:     DefaultTimeStamp,
					TMCreate:  "2020-04-18 03:22:17.995000",
					TMUpdate:  DefaultTimeStamp,
					TMDelete:  DefaultTimeStamp,
				},
			},
		},
		{
			name:     "empty",
			channels: []*channel.Channel{},

			filters: map[string]string{
				"deleted":     "true",
				"asterisk_id": "3e:50:6b:43:bb:32",
			},

			responseCurTimes: []string{},
			expectRes:        []*channel.Channel{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// creates calls for test
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for i, c := range tt.channels {
				mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTimes[i])
				mockCache.EXPECT().ChannelSet(ctx, gomock.Any())
				_ = h.ChannelCreate(ctx, c)
			}

			res, err := h.ChannelGets(ctx, 10, utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_ChannelGetsForRecovery(t *testing.T) {

	type test struct {
		name     string
		channels []*channel.Channel

		asteriskID  string
		channelType channel.Type
		startTime   string
		endTime     string
		size        uint64

		responseCurTimes []string

		expectRes []*channel.Channel
	}

	tests := []test{
		{
			name: "normal",
			channels: []*channel.Channel{
				{
					ID:         "94e47248-48dc-11f0-95bc-e7af7d0649ff",
					AsteriskID: "3e:50:6b:43:bb:32",
					Type:       channel.TypeCall,
				},
				{
					ID:         "951aff48-48dc-11f0-b5a4-ef7f3e74ce09",
					AsteriskID: "3e:50:6b:43:bb:32",
					Type:       channel.TypeCall,
				},
				{
					ID:         "9543bb4a-48dc-11f0-be5c-47c96fc104ff",
					AsteriskID: "3e:50:6b:43:bb:32",
					Type:       channel.TypeCall,
				},
				{
					ID:         "9568a1bc-48dc-11f0-80d6-3facb800a905",
					AsteriskID: "3e:50:6b:43:bb:32",
					Type:       channel.TypeCall,
				},
			},

			asteriskID:  "3e:50:6b:43:bb:32",
			channelType: channel.TypeCall,
			startTime:   "2025-06-14 01:30:00.000000",
			endTime:     "2025-06-14 03:30:00.000000",
			size:        10,

			responseCurTimes: []string{
				"2025-06-14 01:00:00.000000",
				"2025-06-14 02:00:00.000000",
				"2025-06-14 03:00:00.000000",
				"2025-06-14 04:00:00.000000",
			},

			expectRes: []*channel.Channel{
				{
					ID:         "9543bb4a-48dc-11f0-be5c-47c96fc104ff",
					AsteriskID: "3e:50:6b:43:bb:32",
					Type:       channel.TypeCall,

					Data:       map[string]any{},
					StasisData: map[channel.StasisDataType]string{},

					TMAnswer:  DefaultTimeStamp,
					TMRinging: DefaultTimeStamp,
					TMEnd:     DefaultTimeStamp,
					TMCreate:  "2025-06-14 03:00:00.000000",
					TMUpdate:  DefaultTimeStamp,
					TMDelete:  DefaultTimeStamp,
				},
				{
					ID:         "951aff48-48dc-11f0-b5a4-ef7f3e74ce09",
					AsteriskID: "3e:50:6b:43:bb:32",
					Type:       channel.TypeCall,

					Data:       map[string]any{},
					StasisData: map[channel.StasisDataType]string{},

					TMAnswer:  DefaultTimeStamp,
					TMRinging: DefaultTimeStamp,
					TMEnd:     DefaultTimeStamp,
					TMCreate:  "2025-06-14 02:00:00.000000",
					TMUpdate:  DefaultTimeStamp,
					TMDelete:  DefaultTimeStamp,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// creates calls for test
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}
			ctx := context.Background()

			for i, c := range tt.channels {
				mockUtil.EXPECT().TimeGetCurTime(.Return(tt.responseCurTimes[i])
				mockCache.EXPECT().ChannelSet(ctx, gomock.Any())
				_ = h.ChannelCreate(ctx, c)
			}

			res, err := h.ChannelGetsForRecovery(ctx, tt.asteriskID, tt.channelType, tt.startTime, tt.endTime, tt.size)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}
