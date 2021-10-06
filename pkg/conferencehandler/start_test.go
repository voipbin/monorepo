package conferencehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestStartTypeConference(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := conferenceHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
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
				ID:   uuid.FromStringOrNil("325b9d9a-a413-11ea-a6d7-ef53544faeb3"),
				Type: conference.TypeConference,
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
				Type:   conference.TypeConference,
				UserID: call.UserIDAdmin,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConferenceCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().ConferenceGet(gomock.Any(), gomock.Any()).Return(tt.conference, nil)

			if tt.reqConf.Timeout > 0 {
				mockReq.EXPECT().CallConferenceTerminate(gomock.Any(), tt.reqConf.Timeout*1000).Return(nil)
			}
			mockNotify.EXPECT().NotifyEvent(notifyhandler.EventTypeConferenceCreated, tt.conference.WebhookURI, gomock.Any())

			_, err := h.Start(tt.reqConf)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestStartTypeConnect(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := conferenceHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
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
				Type: conference.TypeConnect,
			},
			&conference.Conference{
				ID: uuid.FromStringOrNil("05d67da8-0a5c-11eb-94d6-b3c944a3bf8b"),
			},
		},
		{
			"added user id",
			&conference.Conference{
				Type:   conference.TypeConnect,
				UserID: call.UserIDAdmin,
			},
			&conference.Conference{
				ID:     uuid.FromStringOrNil("0a514944-0a5c-11eb-b450-03f3da9acf03"),
				UserID: call.UserIDAdmin,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB.EXPECT().ConferenceCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().ConferenceGet(gomock.Any(), gomock.Any()).Return(tt.conference, nil)

			if tt.reqConf.Timeout > 0 {
				mockReq.EXPECT().CallConferenceTerminate(gomock.Any(), tt.reqConf.Timeout*1000).Return(nil)
			}
			mockNotify.EXPECT().NotifyEvent(notifyhandler.EventTypeConferenceCreated, tt.conference.WebhookURI, gomock.Any())

			_, err := h.Start(tt.reqConf)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
				return
			}

		})
	}
}
