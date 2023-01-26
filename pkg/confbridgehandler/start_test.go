package confbridgehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_startContextIncomingTypeConference(t *testing.T) {

	tests := []struct {
		name string

		channel   *channel.Channel
		channelID string

		callID     uuid.UUID
		confbridge *confbridge.Confbridge
		data       map[string]string
	}{
		{
			"normal",

			&channel.Channel{
				ID:                "asterisk-call-5765d977d8-c4k5q-1629605410.6626",
				AsteriskID:        "80:fa:5b:5e:da:81",
				Name:              "PJSIP/in-voipbin-00000948",
				DestinationNumber: "d93fdf46-977c-11ec-8403-5b0f71484cde",
			},
			"asterisk-call-5765d977d8-c4k5q-1629605410.6626",
			uuid.FromStringOrNil("3c93499c-977e-11ec-bb53-03fb8063aed1"),
			&confbridge.Confbridge{
				ID:             uuid.FromStringOrNil("3c48edc0-977e-11ec-b248-8b86ab79e4ff"),
				Type:           confbridge.TypeConnect,
				BridgeID:       "d93fdf46-977c-11ec-8403-5b0f71484cde",
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
			},
			map[string]string{
				"context":       contextConfbridgeIncoming,
				"confbridge_id": "3c48edc0-977e-11ec-b248-8b86ab79e4ff",
				"call_id":       "3c93499c-977e-11ec-bb53-03fb8063aed1",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &confbridgeHandler{
				db:             mockDB,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				cache:          mockCache,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().Answer(ctx, tt.channelID).Return(nil)
			err := h.startContextIncomingTypeConference(ctx, tt.channelID, tt.data, tt.callID, tt.confbridge)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startContextIncomingTypeConnect(t *testing.T) {

	tests := []struct {
		name string

		channelID string

		callID     uuid.UUID
		confbridge *confbridge.Confbridge
		data       map[string]string
	}{
		{
			"call already exist",

			"asterisk-call-5765d977d8-c4k5q-1629605410.6626",

			uuid.FromStringOrNil("9ddb42ea-977e-11ec-8245-7f56a442dbdc"),
			&confbridge.Confbridge{
				ID:       uuid.FromStringOrNil("9dad7bb2-977e-11ec-935a-4f47ed77038e"),
				Type:     confbridge.TypeConnect,
				BridgeID: "9d7ecb3c-977e-11ec-bce7-abaa62ee4790",
				ChannelCallIDs: map[string]uuid.UUID{
					"a6b4bf78-9741-11ec-a688-73fbff5fe5e8": uuid.FromStringOrNil("a70ee066-9741-11ec-baba-ebfa678d2fce"),
				},
				RecordingIDs: []uuid.UUID{},
			},
			map[string]string{
				"context":       contextConfbridgeIncoming,
				"confbridge_id": "9dad7bb2-977e-11ec-935a-4f47ed77038e",
				"call_id":       "9ddb42ea-977e-11ec-8245-7f56a442dbdc",
			},
		},
		{
			"first call",

			"asterisk-call-5765d977d8-c4k5q-1629605410.6626",

			uuid.FromStringOrNil("0156d8b0-9794-11ec-80b3-7fe4ef675ba1"),
			&confbridge.Confbridge{
				ID:             uuid.FromStringOrNil("017f3f80-9794-11ec-8ebb-f7a30d24d78f"),
				Type:           confbridge.TypeConnect,
				BridgeID:       "01009e28-9794-11ec-bc54-034deaa2338f",
				ChannelCallIDs: map[string]uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
			},
			map[string]string{
				"context":       contextConfbridgeIncoming,
				"confbridge_id": "017f3f80-9794-11ec-8ebb-f7a30d24d78f",
				"call_id":       "0156d8b0-9794-11ec-80b3-7fe4ef675ba1",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &confbridgeHandler{
				db:             mockDB,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				cache:          mockCache,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			if len(tt.confbridge.ChannelCallIDs) == 0 {
				mockChannel.EXPECT().Ring(ctx, tt.channelID).Return(nil)
			} else {
				mockChannel.EXPECT().Answer(ctx, tt.channelID).Return(nil)
				for channelID := range tt.confbridge.ChannelCallIDs {
					mockChannel.EXPECT().Answer(ctx, channelID).Return(nil)
				}
			}

			err := h.startContextIncomingTypeConnect(ctx, tt.channelID, tt.data, tt.callID, tt.confbridge)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
