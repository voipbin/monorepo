package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestLeavedConferenceTypeConferenceEmptyChannels(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := conferenceHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name       string
		channel    *channel.Channel
		bridge     *bridge.Bridge
		conference *conference.Conference
	}

	tests := []test{
		{
			"terminating conference empty channels",
			&channel.Channel{
				ID:         "9cad54d0-a2c5-11ea-8936-47d2d40af59c",
				AsteriskID: "00:11:22:33:44:55",
				Data:       map[string]interface{}{},
				Type:       channel.TypeConf,
			},
			&bridge.Bridge{
				ID:            "9cfb2fac-a2c5-11ea-9f48-f742f07a1551",
				ReferenceType: bridge.ReferenceTypeConference,
				ReferenceID:   uuid.FromStringOrNil("9d1df140-a2c5-11ea-a4e2-87034be20188"),
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("9d1df140-a2c5-11ea-a4e2-87034be20188"),
				Type:     conference.TypeConference,
				Status:   conference.StatusTerminating,
				BridgeID: "9cfb2fac-a2c5-11ea-9f48-f742f07a1551",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.bridge.ReferenceID).Return(tt.conference, nil)

			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
			mockDB.EXPECT().BridgeGet(gomock.Any(), tt.bridge.ID).Return(tt.bridge, nil)
			mockReq.EXPECT().AstBridgeDelete(tt.bridge.AsteriskID, tt.bridge.ID).Return(nil)
			mockDB.EXPECT().ConferenceEnd(gomock.Any(), tt.conference.ID).Return(nil)
			h.leaved(tt.channel, tt.bridge)
		})
	}
}

func TestLeavedConferenceTypeConferenceWithChannels(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := conferenceHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name       string
		channel    *channel.Channel
		bridge     *bridge.Bridge
		conference *conference.Conference
	}

	tests := []test{
		{
			"terminating conference many channels",
			&channel.Channel{
				ID:         "21a00a76-1542-11ec-ba3c-5726fdf2d299",
				AsteriskID: "00:11:22:33:44:55",
				Data:       map[string]interface{}{},
				Type:       channel.TypeConf,
			},
			&bridge.Bridge{
				ID:            "21df6194-1542-11ec-a6a6-9bd398ba1f89",
				ReferenceType: bridge.ReferenceTypeConference,
				ReferenceID:   uuid.FromStringOrNil("220c0f0a-1542-11ec-af93-9fe3b014b556"),
				ChannelIDs: []string{
					"22354d3e-1542-11ec-8b80-379f316d459b",
					"22661d42-1542-11ec-bc73-2b6284eceef3",
				},
			},
			&conference.Conference{
				ID:       uuid.FromStringOrNil("220c0f0a-1542-11ec-af93-9fe3b014b556"),
				Type:     conference.TypeConference,
				Status:   conference.StatusTerminating,
				BridgeID: "21df6194-1542-11ec-a6a6-9bd398ba1f89",
			},
		}}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.bridge.ReferenceID).Return(tt.conference, nil)
			if err := h.leaved(tt.channel, tt.bridge); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
