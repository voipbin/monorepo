package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

// func TestARIChannelLeftBridge(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockReq := requesthandler.NewMockRequestHandler(mc)
// 	mockDB := dbhandler.NewMockDBHandler(mc)
// 	mockCache := cachehandler.NewMockCacheHandler(mc)

// 	h := &conferenceHandler{
// 		db:         mockDB,
// 		reqHandler: mockReq,
// 		cache:      mockCache,
// 	}

// 	type test struct {
// 		name       string
// 		channel    *channel.Channel
// 		bridge     *bridge.Bridge
// 		conference *conference.Conference
// 		call       *call.Call
// 	}

// 	tests := []test{
// 		{
// 			"echo left",
// 			&channel.Channel{
// 				ID:         "e03dc034-9566-11ea-ad83-1f7a1993587b",
// 				AsteriskID: "80:fa:5b:5e:da:81",
// 				Data: map[string]interface{}{
// 					"CONTEXT": "call-in",
// 				},
// 			},
// 			&bridge.Bridge{
// 				ID:             "e41948fe-9566-11ea-a4fe-db788b6b6d7b",
// 				ConferenceID:   uuid.FromStringOrNil("e9051ac8-9566-11ea-bde6-331b8236a4c2"),
// 				ConferenceType: conference.TypeEcho,
// 			},
// 			&conference.Conference{
// 				ID:   uuid.FromStringOrNil("e9051ac8-9566-11ea-bde6-331b8236a4c2"),
// 				Type: conference.TypeEcho,
// 			},
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("ec4371b2-9566-11ea-bfd3-13a7a033d235"),
// 				ConfID:     uuid.FromStringOrNil("454cb52a-9567-11ea-91be-3b3c3d7249b6"),
// 				ChannelID:  "e03dc034-9566-11ea-ad83-1f7a1993587b",
// 				AsteriskID: "80:fa:5b:5e:da:81",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID, tt.channel.AsteriskID).Return(tt.call, nil)
// 			mockDB.EXPECT().CallSetConferenceID(gomock.Any(), tt.call.ID, uuid.Nil)
// 			mockDB.EXPECT().ConferenceRemoveCallID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
// 			mockReq.EXPECT().CallCallActionNext(gomock.Any()).Return(nil)

// 			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.bridge.ConferenceID).Return(tt.conference, nil)
// 			mockDB.EXPECT().ConferenceEnd(gomock.Any(), tt.conference.ID).Return(nil)
// 			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.bridge.ConferenceID).Return(tt.conference, nil)
// 			mockReq.EXPECT().AstBridgeDelete()

// 			mockReq.EXPECT().AstChannelHangup(tt.call.AsteriskID, tt.call.ChannelID, ari.ChannelCauseNormalClearing)

// 			err := h.ARIChannelLeftBridge(tt.channel, tt.bridge)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }

func TestARIChannelEnteredBridge(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &conferenceHandler{
		db:         mockDB,
		reqHandler: mockReq,
		cache:      mockCache,
	}

	type test struct {
		name    string
		channel *channel.Channel
		bridge  *bridge.Bridge
		call    *call.Call
	}

	tests := []test{
		{
			"normal",
			&channel.Channel{
				ID:         "f7fb3c7a-9565-11ea-976f-c7f5e818313e",
				AsteriskID: "80:fa:5b:5e:da:81",
				Data: map[string]interface{}{
					"CONTEXT": "call-in",
				},
			},
			&bridge.Bridge{
				ID:           "feae07aa-9565-11ea-8905-4b0058aac916",
				ConferenceID: uuid.FromStringOrNil("99292922-9566-11ea-972a-1f51774dac7e"),
			},
			&call.Call{
				ID: uuid.FromStringOrNil("63e5903e-9566-11ea-80d3-3739f385fd3f"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().CallGetByChannelID(gomock.Any(), tt.channel.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallSetConferenceID(gomock.Any(), tt.call.ID, tt.bridge.ConferenceID)
			mockDB.EXPECT().ConferenceAddCallID(gomock.Any(), tt.bridge.ConferenceID, tt.call.ID).Return(nil)

			err := h.ARIChannelEnteredBridge(tt.channel, tt.bridge)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
