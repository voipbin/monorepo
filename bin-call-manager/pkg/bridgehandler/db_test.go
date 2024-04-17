package bridgehandler

import (
	"context"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

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
