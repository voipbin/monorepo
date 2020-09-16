package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler/models/conference"
	dbhandler "gitlab.com/voipbin/bin-manager/common-handler.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
)

func TestStartTypeConference(t *testing.T) {
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
		reqConf    *conference.Conference
		conference *conference.Conference
	}

	tests := []test{
		{
			"normal",
			&conference.Conference{
				Type: conference.TypeConference,
			},
			&conference.Conference{
				ID: uuid.FromStringOrNil("325b9d9a-a413-11ea-a6d7-ef53544faeb3"),
			},
		},
		{
			"added user id",
			&conference.Conference{
				Type:   conference.TypeConference,
				UserID: call.UserIDAdmin,
			},
			&conference.Conference{
				ID:     uuid.FromStringOrNil("12cb0fd8-f148-11ea-b1c3-878d11a76a13"),
				UserID: call.UserIDAdmin,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConferenceCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().ConferenceGet(gomock.Any(), gomock.Any()).Return(tt.conference, nil)

			if tt.reqConf.Timeout > 0 {
				mockReq.EXPECT().CallConferenceTerminate(gomock.Any(), "timeout", tt.reqConf.Timeout*1000).Return(nil)
			}

			h.startTypeConference(tt.reqConf)
		})
	}
}
