package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler"
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
		webhookURI       string
		cfConference     *cfconference.Conference
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
			"",
			&cfconference.Conference{
				ID:   uuid.FromStringOrNil("cea799a4-efce-11ea-9115-03d321ec6ff8"),
				Type: cfconference.TypeConference,

				Status: cfconference.StatusProgressing,
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
		{
			"have webhook",
			&user.User{
				ID: 1,
			},
			conference.TypeConference,
			"test name",
			"test detail",
			"test.com/webhook",
			&cfconference.Conference{
				ID:   uuid.FromStringOrNil("57916d8a-2089-11ec-98bb-9fcde2f6e0ff"),
				Type: cfconference.TypeConference,

				Status: cfconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",
				Data:   map[string]interface{}{},

				WebhookURI: "test.com/webhook",

				CallIDs: []uuid.UUID{},
			},
			&conference.Conference{
				ID:   uuid.FromStringOrNil("57916d8a-2089-11ec-98bb-9fcde2f6e0ff"),
				Type: conference.TypeConference,

				Status: conference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",

				CallIDs:    []uuid.UUID{},
				WebhookURI: "test.com/webhook",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			mockReq.EXPECT().CFConferenceCreate(tt.user.ID, cfconference.Type(tt.confType), tt.confName, tt.confDetail, tt.webhookURI).Return(tt.cfConference, nil)

			res, err := h.ConferenceCreate(tt.user, tt.confType, tt.confName, tt.confDetail, tt.webhookURI)
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
		cfConference *cfconference.Conference
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("7bf5c33a-f086-11ea-9f7c-5f596f1dbfd0"),
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("7bf5c33a-f086-11ea-9f7c-5f596f1dbfd0"),
				UserID: 1,
				Type:   cfconference.TypeConference,

				Status: cfconference.StatusProgressing,
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

			mockReq.EXPECT().CFConferenceGet(tt.confID).Return(tt.cfConference, nil)
			mockReq.EXPECT().CFConferenceDelete(tt.confID).Return(nil)

			err := h.ConferenceDelete(tt.user, tt.confID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestConferenceGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name      string
		user      *user.User
		token     string
		limit     uint64
		response  []cfconference.Conference
		expectRes []*conference.Conference
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			"2020-09-20T03:23:20.995000",
			10,
			[]cfconference.Conference{
				{
					ID:     uuid.FromStringOrNil("c5e87cbc-93b5-11eb-acc0-63225d983d12"),
					UserID: 1,
				},
			},
			[]*conference.Conference{
				{
					ID:     uuid.FromStringOrNil("c5e87cbc-93b5-11eb-acc0-63225d983d12"),
					UserID: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().CFConferenceGets(tt.user.ID, tt.token, tt.limit, "conference").Return(tt.response, nil)
			res, err := h.ConferenceGets(tt.user, tt.limit, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func TestConferenceGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	type test struct {
		name      string
		user      *user.User
		id        uuid.UUID
		response  *cfconference.Conference
		expectRes *conference.Conference
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
				UserID: 1,
			},
			&conference.Conference{
				ID:     uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
				UserID: 1,
			},
		},
		{
			"with webhook",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("b8c4d2ce-202d-11ec-97aa-43b74ed2d540"),
			&cfconference.Conference{
				ID:         uuid.FromStringOrNil("b8c4d2ce-202d-11ec-97aa-43b74ed2d540"),
				UserID:     1,
				WebhookURI: "test.com/webhook",
			},
			&conference.Conference{
				ID:         uuid.FromStringOrNil("b8c4d2ce-202d-11ec-97aa-43b74ed2d540"),
				UserID:     1,
				WebhookURI: "test.com/webhook",
			},
		},
		{
			"with bridge id",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("b8e3118a-202d-11ec-b9e8-03c7f800eaf8"),
			&cfconference.Conference{
				ID:         uuid.FromStringOrNil("b8e3118a-202d-11ec-b9e8-03c7f800eaf8"),
				UserID:     1,
				WebhookURI: "test.com/webhook",
			},
			&conference.Conference{
				ID:         uuid.FromStringOrNil("b8e3118a-202d-11ec-b9e8-03c7f800eaf8"),
				UserID:     1,
				WebhookURI: "test.com/webhook",
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().CFConferenceGet(tt.id).Return(tt.response, nil)
			res, err := h.ConferenceGet(tt.user, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
