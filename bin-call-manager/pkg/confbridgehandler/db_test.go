package confbridgehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customer       uuid.UUID
		activeflowID   uuid.UUID
		referenceType  confbridge.ReferenceType
		referenceID    uuid.UUID
		confbridgeType confbridge.Type

		responseUUID       uuid.UUID
		responseConfbridge *confbridge.Confbridge

		expectConfbridge *confbridge.Confbridge
	}{
		{
			name: "all",

			customer:       uuid.FromStringOrNil("c050fd9c-9c73-11ed-85ab-e77970de1f56"),
			activeflowID:   uuid.FromStringOrNil("a3ee4d58-06ac-11f0-a3b5-83deb166428d"),
			referenceType:  confbridge.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("a44a072e-06ac-11f0-9408-87d5d88111cf"),
			confbridgeType: confbridge.TypeConference,

			responseUUID: uuid.FromStringOrNil("c08d26e6-9c73-11ed-ba48-ab8447f05e1d"),
			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c08d26e6-9c73-11ed-ba48-ab8447f05e1d"),
				},
			},

			expectConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c08d26e6-9c73-11ed-ba48-ab8447f05e1d"),
					CustomerID: uuid.FromStringOrNil("c050fd9c-9c73-11ed-85ab-e77970de1f56"),
				},
				ActiveflowID:  uuid.FromStringOrNil("a3ee4d58-06ac-11f0-a3b5-83deb166428d"),
				ReferenceType: confbridge.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a44a072e-06ac-11f0-9408-87d5d88111cf"),

				Type:     confbridge.TypeConference,
				Status:   confbridge.StatusProgressing,
				BridgeID: "",
				Flags:    []confbridge.Flag{},

				ChannelCallIDs: map[string]uuid.UUID{},

				RecordingID:  uuid.Nil,
				RecordingIDs: []uuid.UUID{},

				ExternalMediaID: uuid.Nil,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &confbridgeHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				cache:          mockCache,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUID)
			mockDB.EXPECT().ConfbridgeCreate(ctx, tt.expectConfbridge.Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.responseUUID.Return(tt.responseConfbridge, nil)
			mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeCreated, tt.responseConfbridge)

			res, err := h.Create(ctx, tt.customer, tt.activeflowID, tt.referenceType, tt.referenceID, tt.confbridgeType)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConfbridge) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConfbridge, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[string]string

		responseGets []*confbridge.Confbridge
		expectRes    []*confbridge.Confbridge
	}{
		{
			name: "normal",

			size:  10,
			token: "2020-05-03%2021:35:02.809",
			filters: map[string]string{
				"customer_id": "78a0debc-f0ce-11ee-8de6-9b2ff94e8b94",
				"deleted":     "false",
			},

			responseGets: []*confbridge.Confbridge{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("7904c314-f0ce-11ee-bc13-1789810328f5"),
					},
				},
			},
			expectRes: []*confbridge.Confbridge{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("7904c314-f0ce-11ee-bc13-1789810328f5"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &confbridgeHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGets(ctx, tt.size, tt.token, gomock.Any().Return(tt.responseGets, nil)

			res, err := h.Gets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_UpdateRecordingID(t *testing.T) {

	tests := []struct {
		name string

		id          uuid.UUID
		recordingID uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("857422e8-9c74-11ed-a9a0-af3ca863e4b3"),
			uuid.FromStringOrNil("859c0d4e-9c74-11ed-845d-171122093d82"),
		},
		{
			"update to nil",

			uuid.FromStringOrNil("e7397ed8-9c74-11ed-8047-f342528da527"),
			uuid.Nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &confbridgeHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				cache:          mockCache,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeSetRecordingID(ctx, tt.id, tt.recordingID.Return(nil)
			if tt.recordingID != uuid.Nil {
				mockDB.EXPECT().ConfbridgeAddRecordingIDs(ctx, tt.id, tt.recordingID.Return(nil)
			}
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id.Return(&confbridge.Confbridge{}, nil)

			_, err := h.UpdateRecordingID(ctx, tt.id, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateExternalMediaID(t *testing.T) {

	tests := []struct {
		name string

		id              uuid.UUID
		externalMediaID uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("e6bc4418-9c74-11ed-bd1f-43a5f1baaebe"),
			uuid.FromStringOrNil("e6e3a12a-9c74-11ed-a0fc-ff1ad2ae6a02"),
		},
		{
			"update to nil",

			uuid.FromStringOrNil("e70f1f3a-9c74-11ed-a5bc-076ae0e2cdbe"),
			uuid.Nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &confbridgeHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				cache:          mockCache,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeSetExternalMediaID(ctx, tt.id, tt.externalMediaID.Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id.Return(&confbridge.Confbridge{}, nil)

			_, err := h.UpdateExternalMediaID(ctx, tt.id, tt.externalMediaID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_AddChannelCallID(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		channelID string
		callID    uuid.UUID

		responseConfbridge *confbridge.Confbridge
		expectEvent        *confbridge.EventConfbridgeJoined
	}{
		{
			name: "normal",

			id:        uuid.FromStringOrNil("be4c1f74-a3bf-11ed-a9e6-4b424bee3fa9"),
			channelID: "becfe048-a3bf-11ed-9a79-139a910fe7a0",
			callID:    uuid.FromStringOrNil("bea13a86-a3bf-11ed-b7b9-efc804ecc73e"),

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("be4c1f74-a3bf-11ed-a9e6-4b424bee3fa9"),
				},
			},
			expectEvent: &confbridge.EventConfbridgeJoined{
				Confbridge: confbridge.Confbridge{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("be4c1f74-a3bf-11ed-a9e6-4b424bee3fa9"),
					},
				},
				JoinedCallID: uuid.FromStringOrNil("bea13a86-a3bf-11ed-b7b9-efc804ecc73e"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &confbridgeHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				cache:          mockCache,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeAddChannelCallID(ctx, tt.id, tt.channelID, tt.callID.Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id.Return(tt.responseConfbridge, nil)
			mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeJoined, tt.expectEvent)

			res, err := h.AddChannelCallID(ctx, tt.id, tt.channelID, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConfbridge) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConfbridge, res)
			}
		})
	}
}

func Test_dbDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConfbridge *confbridge.Confbridge
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("1d170c9a-bce2-11ed-9315-370c4af8e8c4"),

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1d170c9a-bce2-11ed-9315-370c4af8e8c4"),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &confbridgeHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				cache:          mockCache,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeDelete(ctx, tt.id.Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id.Return(tt.responseConfbridge, nil)
			mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeDeleted, tt.responseConfbridge)

			res, err := h.dbDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConfbridge) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConfbridge, res)
			}
		})
	}
}

func Test_UpdateStatus(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		status confbridge.Status

		responseConfbridge *confbridge.Confbridge
		expectEventType    string
	}{
		{
			name: "normal",

			id:     uuid.FromStringOrNil("49331904-83e7-4cd9-a9e2-75c7406554cf"),
			status: confbridge.StatusTerminating,

			responseConfbridge: &confbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("49331904-83e7-4cd9-a9e2-75c7406554cf"),
				},
			},
			expectEventType: confbridge.EventTypeConfbridgeTerminating,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &confbridgeHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				cache:          mockCache,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeSetStatus(ctx, tt.id, tt.status.Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id.Return(tt.responseConfbridge, nil)
			mockNotify.EXPECT().PublishEvent(ctx, tt.expectEventType, tt.responseConfbridge)

			res, err := h.UpdateStatus(ctx, tt.id, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConfbridge) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConfbridge, res)
			}
		})
	}
}
