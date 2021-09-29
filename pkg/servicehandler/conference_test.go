package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler"
	cmconference "gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
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
			"",
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
		{
			"have webhook",
			&user.User{
				ID: 1,
			},
			conference.TypeConference,
			"test name",
			"test detail",
			"test.com/webhook",
			&cmconference.Conference{
				ID:       uuid.FromStringOrNil("57916d8a-2089-11ec-98bb-9fcde2f6e0ff"),
				Type:     cmconference.TypeConference,
				BridgeID: "57ea39d8-2089-11ec-8c10-1ffbbf9317a4",

				Status: cmconference.StatusProgressing,
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

			mockReq.EXPECT().CMConferenceCreate(tt.user.ID, cmconference.Type(tt.confType), tt.confName, tt.confDetail, tt.webhookURI).Return(tt.cmConference, nil)

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

			mockReq.EXPECT().CMConferenceGet(tt.confID).Return(tt.cmConference, nil)
			mockReq.EXPECT().CMConferenceDelete(tt.confID).Return(nil)

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
		response  []cmconference.Conference
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
			[]cmconference.Conference{
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
			mockReq.EXPECT().CMConferenceGets(tt.user.ID, tt.token, tt.limit).Return(tt.response, nil)
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
		response  *cmconference.Conference
		expectRes *conference.Conference
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
			&cmconference.Conference{
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
			&cmconference.Conference{
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
			&cmconference.Conference{
				ID:         uuid.FromStringOrNil("b8e3118a-202d-11ec-b9e8-03c7f800eaf8"),
				UserID:     1,
				BridgeID:   "c565dde8-202d-11ec-870d-9b697a133315",
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
			mockReq.EXPECT().CMConferenceGet(tt.id).Return(tt.response, nil)
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
