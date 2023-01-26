package confbridgehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/bridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customer       uuid.UUID
		confbridgeType confbridge.Type

		responseUUID uuid.UUID

		expectConfbridge *confbridge.Confbridge
	}{
		{
			"normal",

			uuid.FromStringOrNil("c050fd9c-9c73-11ed-85ab-e77970de1f56"),
			confbridge.TypeConference,

			uuid.FromStringOrNil("c08d26e6-9c73-11ed-ba48-ab8447f05e1d"),

			&confbridge.Confbridge{
				ID:             uuid.FromStringOrNil("c08d26e6-9c73-11ed-ba48-ab8447f05e1d"),
				CustomerID:     uuid.FromStringOrNil("c050fd9c-9c73-11ed-85ab-e77970de1f56"),
				Type:           confbridge.TypeConference,
				RecordingIDs:   []uuid.UUID{},
				ChannelCallIDs: map[string]uuid.UUID{},
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

			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUID)
			mockDB.EXPECT().ConfbridgeCreate(ctx, tt.expectConfbridge).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.responseUUID).Return(&confbridge.Confbridge{}, nil)
			mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeCreated, &confbridge.Confbridge{})

			_, err := h.Create(ctx, tt.customer, tt.confbridgeType)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
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

			mockDB.EXPECT().ConfbridgeSetRecordingID(ctx, tt.id, tt.recordingID).Return(nil)
			if tt.recordingID != uuid.Nil {
				mockDB.EXPECT().ConfbridgeAddRecordingIDs(ctx, tt.id, tt.recordingID).Return(nil)
			}
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(&confbridge.Confbridge{}, nil)

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

			mockDB.EXPECT().ConfbridgeSetExternalMediaID(ctx, tt.id, tt.externalMediaID).Return(nil)
			mockDB.EXPECT().ConfbridgeGet(ctx, tt.id).Return(&confbridge.Confbridge{}, nil)

			_, err := h.UpdateExternalMediaID(ctx, tt.id, tt.externalMediaID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
