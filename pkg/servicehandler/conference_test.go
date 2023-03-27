package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestConferenceCreate(t *testing.T) {

	tests := []struct {
		name             string
		customer         *cscustomer.Customer
		confType         cfconference.Type
		confName         string
		confDetail       string
		webhookURI       string
		preActions       []fmaction.Action
		postActions      []fmaction.Action
		cfConference     *cfconference.Conference
		expectConference *cfconference.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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
			},
			&cfconference.WebhookMessage{
				ID:   uuid.FromStringOrNil("57916d8a-2089-11ec-98bb-9fcde2f6e0ff"),
				Type: cfconference.TypeConference,

				Status: cfconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",
			},
		},
		{
			"have pre/post actions",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConferenceV1ConferenceCreate(ctx, tt.customer.ID, tt.confType, tt.confName, tt.confDetail, 0, map[string]interface{}{}, tt.preActions, tt.postActions).Return(tt.cfConference, nil)
			res, err := h.ConferenceCreate(ctx, tt.customer, tt.confType, tt.confName, tt.confDetail, tt.preActions, tt.postActions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectConference) != true {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectConference, res)
			}
		})
	}
}

func Test_ConferenceDelete(t *testing.T) {

	tests := []struct {
		name               string
		customer           *cscustomer.Customer
		conferenceID       uuid.UUID
		responseConference *cfconference.Conference

		expectRes *cfconference.WebhookMessage
	}{
		{
			name: "normal",
			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			conferenceID: uuid.FromStringOrNil("7bf5c33a-f086-11ea-9f7c-5f596f1dbfd0"),

			responseConference: &cfconference.Conference{
				ID:         uuid.FromStringOrNil("7bf5c33a-f086-11ea-9f7c-5f596f1dbfd0"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			expectRes: &cfconference.WebhookMessage{
				ID:         uuid.FromStringOrNil("7bf5c33a-f086-11ea-9f7c-5f596f1dbfd0"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.conferenceID).Return(tt.responseConference, nil)
			mockReq.EXPECT().ConferenceV1ConferenceDelete(ctx, tt.conferenceID).Return(tt.responseConference, nil)

			res, err := h.ConferenceDelete(ctx, tt.customer, tt.conferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestConferenceGets(t *testing.T) {

	tests := []struct {
		name      string
		customer  *cscustomer.Customer
		token     string
		limit     uint64
		response  []cfconference.Conference
		expectRes []*cfconference.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			"2020-09-20T03:23:20.995000",
			10,
			[]cfconference.Conference{
				{
					ID:         uuid.FromStringOrNil("c5e87cbc-93b5-11eb-acc0-63225d983d12"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				},
			},
			[]*cfconference.WebhookMessage{
				{
					ID:         uuid.FromStringOrNil("c5e87cbc-93b5-11eb-acc0-63225d983d12"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConferenceV1ConferenceGets(ctx, tt.customer.ID, tt.token, tt.limit, "conference").Return(tt.response, nil)
			res, err := h.ConferenceGets(ctx, tt.customer, tt.limit, tt.token)
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

	tests := []struct {
		name      string
		customer  *cscustomer.Customer
		id        uuid.UUID
		response  *cfconference.Conference
		expectRes *cfconference.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
			&cfconference.Conference{
				ID:         uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			&cfconference.WebhookMessage{
				ID:         uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
		},
		{
			"with webhook",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("b8c4d2ce-202d-11ec-97aa-43b74ed2d540"),
			&cfconference.Conference{
				ID:         uuid.FromStringOrNil("b8c4d2ce-202d-11ec-97aa-43b74ed2d540"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			&cfconference.WebhookMessage{
				ID:         uuid.FromStringOrNil("b8c4d2ce-202d-11ec-97aa-43b74ed2d540"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.id).Return(tt.response, nil)
			res, err := h.ConferenceGet(ctx, tt.customer, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceUpdate(t *testing.T) {

	tests := []struct {
		name        string
		customer    *cscustomer.Customer
		id          uuid.UUID
		updateName  string
		detail      string
		timeout     int
		preActions  []fmaction.Action
		postActions []fmaction.Action

		response  *cfconference.Conference
		expectRes *cfconference.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
			"update name",
			"update detail",
			86400,
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
				ID:         uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			&cfconference.WebhookMessage{
				ID:         uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.id).Return(tt.response, nil)
			mockReq.EXPECT().ConferenceV1ConferenceUpdate(ctx, tt.id, tt.updateName, tt.detail, tt.timeout, tt.preActions, tt.postActions).Return(tt.response, nil)
			res, err := h.ConferenceUpdate(ctx, tt.customer, tt.id, tt.updateName, tt.detail, tt.timeout, tt.preActions, tt.postActions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceRecordingStart(t *testing.T) {

	tests := []struct {
		name string

		customer     *cscustomer.Customer
		conferenceID uuid.UUID

		responseconference *cfconference.Conference
		expectRes          *cfconference.WebhookMessage
	}{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("6d48be14-910b-11ed-b644-eb3bf9ff8517"),

			&cfconference.Conference{
				ID:         uuid.FromStringOrNil("6d48be14-910b-11ed-b644-eb3bf9ff8517"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			&cfconference.WebhookMessage{
				ID:         uuid.FromStringOrNil("6d48be14-910b-11ed-b644-eb3bf9ff8517"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.conferenceID).Return(tt.responseconference, nil)
			mockReq.EXPECT().ConferenceV1ConferenceRecordingStart(ctx, tt.conferenceID).Return(tt.responseconference, nil)
			res, err := h.ConferenceRecordingStart(ctx, tt.customer, tt.conferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceRecordingStop(t *testing.T) {

	tests := []struct {
		name string

		customer     *cscustomer.Customer
		conferenceID uuid.UUID

		responseconference *cfconference.Conference
		expectRes          *cfconference.WebhookMessage
	}{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("f6e67710-910b-11ed-b11d-abaf81af53bf"),

			&cfconference.Conference{
				ID:         uuid.FromStringOrNil("f6e67710-910b-11ed-b11d-abaf81af53bf"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			&cfconference.WebhookMessage{
				ID:         uuid.FromStringOrNil("f6e67710-910b-11ed-b11d-abaf81af53bf"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.conferenceID).Return(tt.responseconference, nil)
			mockReq.EXPECT().ConferenceV1ConferenceRecordingStop(ctx, tt.conferenceID).Return(tt.responseconference, nil)
			res, err := h.ConferenceRecordingStop(ctx, tt.customer, tt.conferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceTranscribeStart(t *testing.T) {

	tests := []struct {
		name string

		customer     *cscustomer.Customer
		conferenceID uuid.UUID
		language     string

		responseconference *cfconference.Conference
		expectRes          *cfconference.WebhookMessage
	}{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("729da1a8-98eb-11ed-8fa8-1b689360397c"),
			"en-US",

			&cfconference.Conference{
				ID:         uuid.FromStringOrNil("729da1a8-98eb-11ed-8fa8-1b689360397c"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			&cfconference.WebhookMessage{
				ID:         uuid.FromStringOrNil("729da1a8-98eb-11ed-8fa8-1b689360397c"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.conferenceID).Return(tt.responseconference, nil)
			mockReq.EXPECT().ConferenceV1ConferenceTranscribeStart(ctx, tt.conferenceID, tt.language).Return(tt.responseconference, nil)
			res, err := h.ConferenceTranscribeStart(ctx, tt.customer, tt.conferenceID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceTranscribeStop(t *testing.T) {

	tests := []struct {
		name string

		customer     *cscustomer.Customer
		conferenceID uuid.UUID

		responseconference *cfconference.Conference
		expectRes          *cfconference.WebhookMessage
	}{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			uuid.FromStringOrNil("72d42a48-98eb-11ed-bd79-f3b90badd9ad"),

			&cfconference.Conference{
				ID:         uuid.FromStringOrNil("72d42a48-98eb-11ed-bd79-f3b90badd9ad"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			&cfconference.WebhookMessage{
				ID:         uuid.FromStringOrNil("72d42a48-98eb-11ed-bd79-f3b90badd9ad"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.conferenceID).Return(tt.responseconference, nil)
			mockReq.EXPECT().ConferenceV1ConferenceTranscribeStop(ctx, tt.conferenceID).Return(tt.responseconference, nil)
			res, err := h.ConferenceTranscribeStop(ctx, tt.customer, tt.conferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
