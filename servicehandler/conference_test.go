package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager/models/conference"
	"gitlab.com/voipbin/bin-manager/api-manager/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler/models/cmconference"
)

func TestConferenceCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	type test struct {
		name             string
		user             *user.User
		confType         conference.Type
		confName         string
		confDetail       string
		cmConference     *cmconference.Conference
		expectConference *conference.Conference
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			conference.TypeConference,
			"test name",
			"test detail",
			&cmconference.Conference{
				ID:       uuid.FromStringOrNil("cea799a4-efce-11ea-9115-03d321ec6ff8"),
				Type:     cmconference.TypeConference,
				BridgeID: "e7a43ad4-efce-11ea-956e-e7473d66f18f",

				Status: cmconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",
				Data:   map[string]interface{}{},

				CallIDs: []uuid.UUID{},
			},
			&conference.Conference{
				ID:   uuid.FromStringOrNil("cea799a4-efce-11ea-9115-03d321ec6ff8"),
				Type: conference.TypeConference,

				Status: conference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",

				CallIDs: []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			mockReq.EXPECT().CallConferenceCreate(tt.user.ID, cmconference.Type(tt.confType), tt.confName, tt.confDetail).Return(tt.cmConference, nil)

			res, err := h.ConferenceCreate(tt.user, tt.confType, tt.confName, tt.confDetail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectConference) != true {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectConference, res)
			}
		})
	}
}

func TestConferenceDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	type test struct {
		name         string
		user         *user.User
		confID       uuid.UUID
		cmConference *cmconference.Conference
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("7bf5c33a-f086-11ea-9f7c-5f596f1dbfd0"),
			&cmconference.Conference{
				ID:       uuid.FromStringOrNil("7bf5c33a-f086-11ea-9f7c-5f596f1dbfd0"),
				UserID:   1,
				Type:     cmconference.TypeConference,
				BridgeID: "e7a43ad4-efce-11ea-956e-e7473d66f18f",

				Status: cmconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",
				Data:   map[string]interface{}{},

				CallIDs: []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			mockReq.EXPECT().CallConferenceGet(tt.confID).Return(tt.cmConference, nil)
			mockReq.EXPECT().CallConferenceDelete(tt.confID).Return(nil)

			err := h.ConferenceDelete(tt.user, tt.confID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
