package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestConferenceCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	type test struct {
		name             string
		user             *user.User
		confType         cfconference.Type
		confName         string
		confDetail       string
		webhookURI       string
		preActions       []fmaction.Action
		postActions      []fmaction.Action
		cfConference     *cfconference.Conference
		expectConference *cfconference.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			cfconference.TypeConference,
			"test name",
			"test detail",
			"",
			[]fmaction.Action{},
			[]fmaction.Action{},
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("cea799a4-efce-11ea-9115-03d321ec6ff8"),
				Type:   cfconference.TypeConference,
				FlowID: uuid.FromStringOrNil("77b380ae-3feb-11ec-95c1-133259cb9432"),

				Status: cfconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",
			},
			&cfconference.WebhookMessage{
				ID:   uuid.FromStringOrNil("cea799a4-efce-11ea-9115-03d321ec6ff8"),
				Type: cfconference.TypeConference,

				Status: cfconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",
			},
		},
		{
			"have webhook",
			&user.User{
				ID: 1,
			},
			cfconference.TypeConference,
			"test name",
			"test detail",
			"test.com/webhook",
			[]fmaction.Action{},
			[]fmaction.Action{},
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("57916d8a-2089-11ec-98bb-9fcde2f6e0ff"),
				Type:   cfconference.TypeConference,
				FlowID: uuid.FromStringOrNil("257ba252-3fec-11ec-8a6a-f758019d5b2e"),

				Status: cfconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",

				WebhookURI: "test.com/webhook",
			},
			&cfconference.WebhookMessage{
				ID:   uuid.FromStringOrNil("57916d8a-2089-11ec-98bb-9fcde2f6e0ff"),
				Type: cfconference.TypeConference,

				Status: cfconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",

				WebhookURI: "test.com/webhook",
			},
		},
		{
			"have pre/post actions",
			&user.User{
				ID: 1,
			},
			cfconference.TypeConference,
			"test name",
			"test detail",
			"test.com/webhook",
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			[]fmaction.Action{
				{
					Type: fmaction.TypeHangup,
				},
			},
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("f63e863e-3fe7-11ec-9713-33d614df6067"),
				Type:   cfconference.TypeConference,
				FlowID: uuid.FromStringOrNil("3cd76cd8-3fec-11ec-a100-6f2e9597c4f2"),

				Status: cfconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",

				PreActions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				PostActions: []fmaction.Action{
					{
						Type: fmaction.TypeHangup,
					},
				},

				WebhookURI: "test.com/webhook",
			},
			&cfconference.WebhookMessage{
				ID:   uuid.FromStringOrNil("f63e863e-3fe7-11ec-9713-33d614df6067"),
				Type: cfconference.TypeConference,

				Status: cfconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",

				PreActions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				PostActions: []fmaction.Action{
					{
						Type: fmaction.TypeHangup,
					},
				},

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

			mockReq.EXPECT().CFV1ConferenceCreate(gomock.Any(), tt.user.ID, tt.confType, tt.confName, tt.confDetail, 0, tt.webhookURI, map[string]interface{}{}, tt.preActions, tt.postActions).Return(tt.cfConference, nil)
			res, err := h.ConferenceCreate(tt.user, tt.confType, tt.confName, tt.confDetail, tt.webhookURI, tt.preActions, tt.postActions)
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

			mockReq.EXPECT().CFV1ConferenceGet(gomock.Any(), tt.confID).Return(tt.cfConference, nil)
			mockReq.EXPECT().CFV1ConferenceDelete(gomock.Any(), tt.confID).Return(nil)

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
		expectRes []*cfconference.WebhookMessage
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

					PreActions:  []fmaction.Action{},
					PostActions: []fmaction.Action{},
				},
			},
			[]*cfconference.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("c5e87cbc-93b5-11eb-acc0-63225d983d12"),

					PreActions:  []fmaction.Action{},
					PostActions: []fmaction.Action{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().CFV1ConferenceGets(gomock.Any(), tt.user.ID, tt.token, tt.limit, "conference").Return(tt.response, nil)
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
		expectRes *cfconference.WebhookMessage
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

				PreActions:  []fmaction.Action{},
				PostActions: []fmaction.Action{},
			},
			&cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),

				PreActions:  []fmaction.Action{},
				PostActions: []fmaction.Action{},
			},
		},
		{
			"with webhook",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("b8c4d2ce-202d-11ec-97aa-43b74ed2d540"),
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("b8c4d2ce-202d-11ec-97aa-43b74ed2d540"),
				UserID: 1,

				PreActions:  []fmaction.Action{},
				PostActions: []fmaction.Action{},

				WebhookURI: "test.com/webhook",
			},
			&cfconference.WebhookMessage{
				ID: uuid.FromStringOrNil("b8c4d2ce-202d-11ec-97aa-43b74ed2d540"),

				PreActions:  []fmaction.Action{},
				PostActions: []fmaction.Action{},

				WebhookURI: "test.com/webhook",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReq.EXPECT().CFV1ConferenceGet(gomock.Any(), tt.id).Return(tt.response, nil)
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
