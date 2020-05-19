package conferencehandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/call"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/conference"
	dbhandler "gitlab.com/voipbin/bin-manager/call-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/requesthandler"
)

func TestLeavedNoTerminate(t *testing.T) {
	type test struct {
		name       string
		conference *conference.Conference
		call       *call.Call
	}

	tests := []test{
		{
			"echo leaved but",
			&conference.Conference{
				ID:      uuid.FromStringOrNil("1b0f9e5e-9246-11ea-a764-53e61c9fef34"),
				Type:    conference.TypeEcho,
				CallIDs: []uuid.UUID{uuid.Nil},
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("2dde2e70-9245-11ea-a1e5-1b4f44d33983"),
				AsteriskID: "80:fa:5b:5e:da:81",
				ChannelID:  "14ddefea-9246-11ea-bcc6-4bbba9c0b195",
			},
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := conferenceHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConferenceRemoveCallID(gomock.Any(), tt.conference.ID, tt.call.ID).Return(nil)
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)
			mockDB.EXPECT().ConferenceGet(gomock.Any(), tt.conference.ID).Return(tt.conference, nil)

			h.Leaved(tt.conference.ID, tt.call.ID)
		})
	}
}

func TestIsTerminatable(t *testing.T) {
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
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := conferenceHandler{
		reqHandler: mockReq,
		db:         mockDB,
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
