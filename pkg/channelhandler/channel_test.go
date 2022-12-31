package channelhandler

import (
	"context"
	reflect "reflect"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
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

		state      ari.ChannelState
		data       map[string]interface{}
		stasisName string
		stasisData map[string]string
		bridgeID   string
		playbackID string

		dialResult  string
		hangupCause ari.ChannelCause

		direction channel.Direction

		responseChannel *channel.Channel
		expectRes       *channel.Channel
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
			map[string]interface{}{
				"key1": "value1",
			},
			"voipbin",
			map[string]string{
				"call_id": "75c37398-6e0d-11ed-94a6-8f255cf3934e",
			},
			"75f01be6-6e0d-11ed-a7c9-7335c3fa383a",
			"761b21f6-6e0d-11ed-b651-c37f76e7d8aa",

			"answer",
			ari.ChannelCauseNormalClearing,

			channel.DirectionOutgoing,

			&channel.Channel{
				ID: "764818be-6e0d-11ed-a193-33bbe6697279",
			},
			&channel.Channel{
				ID: "764818be-6e0d-11ed-a193-33bbe6697279",
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

			c := &channel.Channel{
				ID:         tt.id,
				AsteriskID: tt.asteriskID,
				Name:       tt.channelName,
				Type:       tt.channelType,
				Tech:       tt.tech,

				SIPCallID:    tt.sipCallID,
				SIPTransport: tt.sipTransport,

				SourceName:        tt.sourceName,
				SourceNumber:      tt.sourceNumber,
				DestinationName:   tt.destinationName,
				DestinationNumber: tt.destinationNumber,

				State:      tt.state,
				Data:       tt.data,
				StasisName: tt.stasisName,
				StasisData: tt.stasisData,
				BridgeID:   tt.bridgeID,
				PlaybackID: tt.playbackID,

				DialResult:  tt.dialResult,
				HangupCause: tt.hangupCause,
				Direction:   tt.direction,
			}
			mockDB.EXPECT().ChannelCreate(ctx, c).Return(nil)
			mockDB.EXPECT().ChannelGet(ctx, c.ID).Return(tt.responseChannel, nil)

			res, err := h.Create(
				ctx,

				tt.id,
				tt.asteriskID,
				tt.channelName,
				tt.channelType,
				tt.tech,

				tt.sipCallID,
				tt.sipTransport,

				tt.sourceName,
				tt.sourceNumber,
				tt.destinationName,
				tt.destinationNumber,

				tt.state,
				tt.data,
				tt.stasisName,
				tt.stasisData,
				tt.bridgeID,
				tt.playbackID,

				tt.dialResult,
				tt.hangupCause,

				tt.direction,
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

			mockDB.EXPECT().ChannelGet(ctx, tt.id).Return(tt.responseChannel, nil)

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
			mockDB.EXPECT().ChannelGet(ctx, tt.id).Return(tt.responseChannel, nil)
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
			mockDB.EXPECT().ChannelGet(ctx, tt.id).Return(&channel.Channel{}, nil)

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
			mockDB.EXPECT().ChannelGet(ctx, tt.id).Return(&channel.Channel{}, nil)

			if err := h.SetDirection(ctx, tt.id, tt.direction); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Microsecond * 100)
		})
	}
}

func Test_SetSIPCallID(t *testing.T) {

	type test struct {
		name string

		id        string
		sipCallID string

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			"normal",

			"1e5c8b0c-6e5e-11ed-8582-a3186cbc997e",
			"2009157e-6e5e-11ed-8a8e-bbb3856712ec",

			&channel.Channel{
				ID: "1e5c8b0c-6e5e-11ed-8582-a3186cbc997e",
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

			mockDB.EXPECT().ChannelSetSIPCallID(ctx, tt.id, tt.sipCallID).Return(nil)
			if err := h.SetSIPCallID(ctx, tt.id, tt.sipCallID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
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
			mockDB.EXPECT().ChannelGet(ctx, tt.id).Return(tt.responseChannel, nil)

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
			mockDB.EXPECT().ChannelGet(ctx, tt.id).Return(tt.responseChannel, nil)

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
