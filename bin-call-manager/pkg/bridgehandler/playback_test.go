package bridgehandler

import (
	"context"
	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/bridge"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"testing"

	gomock "go.uber.org/mock/gomock"
)

func Test_Play(t *testing.T) {

	tests := []struct {
		name string

		id         string
		playbackID string
		medias     []string
		language   string
		offsetms   int
		skipms     int

		responseBridge *bridge.Bridge
	}{
		{
			name: "normal",

			id:         "b57850aa-7f0b-11f0-be7c-4b710b288213",
			playbackID: "b3ecb76c-a911-11ed-ba75-1fd6a1f4a8dc",
			medias: []string{
				"sound:https://test.com/b5b3d29c-7f0b-11f0-9617-67da2ef9cf13.wav",
				"sound:https://test.com/b5d3f69e-7f0b-11f0-bf07-13c4351845ca.wav",
			},
			language: "",
			offsetms: 0,
			skipms:   0,

			responseBridge: &bridge.Bridge{
				ID:       "b57850aa-7f0b-11f0-be7c-4b710b288213",
				TMDelete: dbhandler.DefaultTimeStamp,
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
			mockReq.EXPECT().AstBridgePlay(ctx, tt.responseBridge.AsteriskID, tt.responseBridge.ID, tt.medias, tt.language, tt.offsetms, tt.skipms, tt.playbackID).Return(&ari.Playback{}, nil)

			if err := h.Play(ctx, tt.id, tt.playbackID, tt.medias, tt.language, tt.offsetms, tt.skipms); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
