package conferencehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/arihandler/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conferencehandler/models/conference"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestIsTerminatable(t *testing.T) {
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
		conference *conference.Conference
		expactRes  bool
	}

	tests := []test{
		{
			"finished echo",
			&conference.Conference{
				ID:      uuid.FromStringOrNil("1b0f9e5e-9246-11ea-a764-53e61c9fef34"),
				Type:    conference.TypeEcho,
				CallIDs: []uuid.UUID{},
			},
			true,
		},
		{
			"echo not finish",
			&conference.Conference{
				ID:   uuid.FromStringOrNil("5e105594-924c-11ea-8218-5348c72b7ef9"),
				Type: conference.TypeEcho,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("f9c82130-924a-11ea-903b-ffe67f3c4d82"),
				},
			},
			false,
		},
		{
			"conference has call",
			&conference.Conference{
				ID:   uuid.FromStringOrNil("19b80886-a2c5-11ea-a4c9-7bdb045195b8"),
				Type: conference.TypeConference,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("21510778-a2c5-11ea-8109-8313c4dd63a6"),
				},
			},
			false,
		},
		{
			"conference has no call",
			&conference.Conference{
				ID:   uuid.FromStringOrNil("2174c0dc-a2c5-11ea-a265-93c7fc3f9dd1"),
				Type: conference.TypeConference,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("219dfc18-a2c5-11ea-8482-af13d9f6b2dd"),
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
			res := h.isTerminatable(context.Background(), tt.conference.ID)
			if tt.expactRes != res {
				t.Errorf("Wrong match. expect: %t, got: %t", tt.expactRes, res)
			}
		})
	}
}

func TestLeavedConferenceTypeConference(t *testing.T) {
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
			"conference normal",
			&channel.Channel{
				ID:         "9cad54d0-a2c5-11ea-8936-47d2d40af59c",
				AsteriskID: "00:11:22:33:44:55",
				Data: map[string]interface{}{
					"CONTEXT": "call-in",
				},
			},
			&bridge.Bridge{
				ID:             "9cfb2fac-a2c5-11ea-9f48-f742f07a1551",
				ConferenceID:   uuid.FromStringOrNil("9d1df140-a2c5-11ea-a4e2-87034be20188"),
				ConferenceType: conference.TypeConference,
			},
			&conference.Conference{
				ID:   uuid.FromStringOrNil("9d1df140-a2c5-11ea-a4e2-87034be20188"),
				Type: conference.TypeConference,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.bridge.ConferenceID).Return(tt.conference, nil)

			h.leavedConference(tt.channel, tt.bridge)
		})
	}
}

func TestLeavedConferenceTypeEcho(t *testing.T) {
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
			"conference normal",
			&channel.Channel{
				ID:         "9cad54d0-a2c5-11ea-8936-47d2d40af59c",
				AsteriskID: "00:11:22:33:44:55",
				Data: map[string]interface{}{
					"CONTEXT": "call-in",
				},
			},
			&bridge.Bridge{
				ID:             "9cfb2fac-a2c5-11ea-9f48-f742f07a1551",
				ConferenceID:   uuid.FromStringOrNil("9d1df140-a2c5-11ea-a4e2-87034be20188"),
				ConferenceType: conference.TypeEcho,
			},
			&conference.Conference{
				ID:   uuid.FromStringOrNil("9d1df140-a2c5-11ea-a4e2-87034be20188"),
				Type: conference.TypeEcho,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.bridge.ConferenceID).Return(tt.conference, nil)
			mockReq.EXPECT().CallConferenceTerminate(tt.bridge.ConferenceID, "normal terminating", requesthandler.DelayNow).Return(nil)

			h.leavedConference(tt.channel, tt.bridge)
		})
	}

}
