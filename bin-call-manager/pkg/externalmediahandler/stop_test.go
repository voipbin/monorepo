package externalmediahandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_Stop(t *testing.T) {

	tests := []struct {
		name                  string
		externalMediaID       uuid.UUID
		responseExternalMedia *externalmedia.ExternalMedia
	}{
		{
			"normal",
			uuid.FromStringOrNil("f325901e-96ee-11ed-bf37-abb6beffa198"),

			&externalmedia.ExternalMedia{
				ID:             uuid.FromStringOrNil("f325901e-96ee-11ed-bf37-abb6beffa198"),
				AsteriskID:     "42:01:0a:a4:00:05",
				ChannelID:      "f3b36ede-96ee-11ed-8213-53b58f84fb99",
				LocalIP:        "",
				LocalPort:      0,
				ExternalHost:   "",
				Encapsulation:  "",
				Transport:      "",
				ConnectionType: "",
				Format:         "",
				Direction:      "",
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

			h := &externalMediaHandler{
				utilHandler: mockUtil,
				reqHandler:  mockReq,
				db:          mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().ExternalMediaGet(ctx, tt.externalMediaID).Return(tt.responseExternalMedia, nil)
			mockReq.EXPECT().AstChannelHangup(ctx, tt.responseExternalMedia.AsteriskID, tt.responseExternalMedia.ChannelID, ari.ChannelCauseNormalClearing, 0).Return(nil)
			mockDB.EXPECT().ExternalMediaDelete(ctx, tt.externalMediaID).Return(nil)

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
