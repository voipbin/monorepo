package externalmediahandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Stop(t *testing.T) {

	tests := []struct {
		name                  string
		externalMediaID       uuid.UUID
		responseExternalMedia *externalmedia.ExternalMedia
	}{
		{
			name:            "normal",
			externalMediaID: uuid.FromStringOrNil("f325901e-96ee-11ed-bf37-abb6beffa198"),

			responseExternalMedia: &externalmedia.ExternalMedia{
				ID:              uuid.FromStringOrNil("f325901e-96ee-11ed-bf37-abb6beffa198"),
				AsteriskID:      "42:01:0a:a4:00:05",
				ChannelID:       "f3b36ede-96ee-11ed-8213-53b58f84fb99",
				LocalIP:         "",
				LocalPort:       0,
				ExternalHost:    "",
				Encapsulation:   "",
				Transport:       "",
				ConnectionType:  "",
				Format:          "",
				DirectionListen: "",
			},
		},
		{
			name:            "reference type call removes from parent",
			externalMediaID: uuid.FromStringOrNil("a1b2c3d4-1111-2222-3333-444455556666"),

			responseExternalMedia: &externalmedia.ExternalMedia{
				ID:            uuid.FromStringOrNil("a1b2c3d4-1111-2222-3333-444455556666"),
				AsteriskID:    "42:01:0a:a4:00:05",
				ChannelID:     "b2c3d4e5-1111-2222-3333-444455556666",
				ReferenceType: externalmedia.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("c3d4e5f6-1111-2222-3333-444455556666"),
			},
		},
		{
			name:            "reference type confbridge removes from parent",
			externalMediaID: uuid.FromStringOrNil("d4e5f6a7-1111-2222-3333-444455556666"),

			responseExternalMedia: &externalmedia.ExternalMedia{
				ID:            uuid.FromStringOrNil("d4e5f6a7-1111-2222-3333-444455556666"),
				AsteriskID:    "42:01:0a:a4:00:05",
				ChannelID:     "e5f6a7b8-1111-2222-3333-444455556666",
				ReferenceType: externalmedia.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("f6a7b8c9-1111-2222-3333-444455556666"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &externalMediaHandler{
				utilHandler: mockUtil,
				reqHandler:  mockReq,
				db:          mockDB,

				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockDB.EXPECT().ExternalMediaGet(ctx, tt.externalMediaID).Return(tt.responseExternalMedia, nil)
			mockDB.EXPECT().ExternalMediaSet(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ExternalMediaGet(ctx, tt.externalMediaID).Return(tt.responseExternalMedia, nil)
			mockChannel.EXPECT().HangingUpWithAsteriskID(ctx, tt.responseExternalMedia.AsteriskID, tt.responseExternalMedia.ChannelID, ari.ChannelCauseNormalClearing).Return(nil)
			mockDB.EXPECT().ExternalMediaDelete(ctx, tt.externalMediaID).Return(nil)

			switch tt.responseExternalMedia.ReferenceType {
			case externalmedia.ReferenceTypeCall:
				mockDB.EXPECT().CallRemoveExternalMediaID(ctx, tt.responseExternalMedia.ReferenceID, tt.externalMediaID).Return(nil)
			case externalmedia.ReferenceTypeConfbridge:
				mockDB.EXPECT().ConfbridgeRemoveExternalMediaID(ctx, tt.responseExternalMedia.ReferenceID, tt.externalMediaID).Return(nil)
			}

			res, err := h.Stop(ctx, tt.externalMediaID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseExternalMedia, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseExternalMedia, res)
			}
		})
	}
}
