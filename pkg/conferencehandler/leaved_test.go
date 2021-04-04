package conferencehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestGetTerminateType(t *testing.T) {
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
		expactRes  termType
	}

	tests := []test{
		{
			"conference has call",
			&conference.Conference{
				ID:   uuid.FromStringOrNil("19b80886-a2c5-11ea-a4c9-7bdb045195b8"),
				Type: conference.TypeConference,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("21510778-a2c5-11ea-8109-8313c4dd63a6"),
				},
			},
			termTypeNone,
		},
		{
			"conference has no call",
			&conference.Conference{
				ID:      uuid.FromStringOrNil("2174c0dc-a2c5-11ea-a265-93c7fc3f9dd1"),
				Type:    conference.TypeConference,
				CallIDs: []uuid.UUID{},
			},
			termTypeNone,
		},
		{
			"conference is terminating has a call",
			&conference.Conference{
				ID:     uuid.FromStringOrNil("2174c0dc-a2c5-11ea-a265-93c7fc3f9dd1"),
				Type:   conference.TypeConference,
				Status: conference.StatusTerminating,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("219dfc18-a2c5-11ea-8482-af13d9f6b2dd"),
				},
			},
			termTypeTerminatable,
		},
		{
			"conference is terminating has no call",
			&conference.Conference{
				ID:      uuid.FromStringOrNil("2174c0dc-a2c5-11ea-a265-93c7fc3f9dd1"),
				Type:    conference.TypeConference,
				Status:  conference.StatusTerminating,
				CallIDs: []uuid.UUID{},
			},
			termTypeDestroyable,
		},
		{
			"connect has a call",
			&conference.Conference{
				ID:   uuid.FromStringOrNil("162c9192-94e5-11eb-b0e8-0be2148bceaf"),
				Type: conference.TypeConnect,
				CallIDs: []uuid.UUID{
					uuid.FromStringOrNil("19737adc-94e5-11eb-a434-075840d0564d"),
				},
			},
			termTypeTerminatable,
		},
		{
			"connect has no call",
			&conference.Conference{
				ID:      uuid.FromStringOrNil("2a19047e-94e5-11eb-bd12-df8f679653a1"),
				Type:    conference.TypeConnect,
				CallIDs: []uuid.UUID{},
			},
			termTypeDestroyable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
			res := h.getTerminateType(context.Background(), tt.conference.ID)
			if tt.expactRes != res {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expactRes, res)
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
				Data:       map[string]interface{}{},
				Type:       channel.TypeCall,
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
