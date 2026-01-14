package servicehandler

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	cmrecording "monorepo/bin-call-manager/models/recording"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	cfconference "monorepo/bin-conference-manager/models/conference"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/websockhandler"
)

func Test_ConferenceCreate(t *testing.T) {

	tests := []struct {
		name             string
		agent            *amagent.Agent
		id               uuid.UUID
		confType         cfconference.Type
		confName         string
		confDetail       string
		timeout          int
		data             map[string]any
		preFlowID        uuid.UUID
		postFlowID       uuid.UUID
		cfConference     *cfconference.Conference
		expectConference *cfconference.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			id:         uuid.FromStringOrNil("b6573ab8-1e1c-11f0-9350-cb7934970eb4"),
			confType:   cfconference.TypeConference,
			confName:   "test name",
			confDetail: "test detail",
			timeout:    100,
			data: map[string]any{
				"key1": "hello",
				"kwy2": float64(300),
			},
			preFlowID:  uuid.FromStringOrNil("b5f74aea-1e1c-11f0-9686-3b0ba2b0b81f"),
			postFlowID: uuid.FromStringOrNil("b62ea512-1e1c-11f0-ab82-ff2f37918a3b"),
			cfConference: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cea799a4-efce-11ea-9115-03d321ec6ff8"),
				},
				Type: cfconference.TypeConference,

				Status: cfconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",
			},
			expectConference: &cfconference.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cea799a4-efce-11ea-9115-03d321ec6ff8"),
				},
				Type: cfconference.TypeConference,

				Status: cfconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",
			},
		},
		{
			name: "empty",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			id:         uuid.Nil,
			confType:   cfconference.TypeConference,
			confName:   "",
			confDetail: "",
			timeout:    0,
			data:       map[string]any{},
			preFlowID:  uuid.Nil,
			postFlowID: uuid.Nil,
			cfConference: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f63e863e-3fe7-11ec-9713-33d614df6067"),
				},
				Type: cfconference.TypeConference,

				Status: cfconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",
			},
			expectConference: &cfconference.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f63e863e-3fe7-11ec-9713-33d614df6067"),
				},
				Type: cfconference.TypeConference,

				Status: cfconference.StatusProgressing,
				Name:   "test name",
				Detail: "test detail",
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

			mockReq.EXPECT().ConferenceV1ConferenceCreate(
				ctx,
				tt.id,
				tt.agent.CustomerID,
				tt.confType,
				tt.confName,
				tt.confDetail,
				tt.data,
				tt.timeout,
				tt.preFlowID,
				tt.postFlowID,
			).Return(tt.cfConference, nil)
			res, err := h.ConferenceCreate(
				ctx,
				tt.agent,
				tt.id,
				tt.confType,
				tt.confName,
				tt.confDetail,
				tt.data,
				tt.timeout,
				tt.preFlowID,
				tt.postFlowID,
			)
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
		agent              *amagent.Agent
		conferenceID       uuid.UUID
		responseConference *cfconference.Conference

		expectRes *cfconference.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			conferenceID: uuid.FromStringOrNil("7bf5c33a-f086-11ea-9f7c-5f596f1dbfd0"),

			responseConference: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7bf5c33a-f086-11ea-9f7c-5f596f1dbfd0"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &cfconference.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7bf5c33a-f086-11ea-9f7c-5f596f1dbfd0"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.conferenceID).Return(tt.responseConference, nil)
			mockReq.EXPECT().ConferenceV1ConferenceDelete(ctx, tt.conferenceID).Return(tt.responseConference, nil)

			res, err := h.ConferenceDelete(ctx, tt.agent, tt.conferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceGets(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		token string
		limit uint64

		response      []cfconference.Conference
		expectFilters map[cfconference.Field]any
		expectRes     []*cfconference.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			"2020-09-20T03:23:20.995000",
			10,

			[]cfconference.Conference{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c5e87cbc-93b5-11eb-acc0-63225d983d12"),
						CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
					},
				},
			},
			map[cfconference.Field]any{
				cfconference.FieldCustomerID: "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				cfconference.FieldDeleted:    false,
				cfconference.FieldType:       string(cfconference.TypeConference),
			},
			[]*cfconference.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c5e87cbc-93b5-11eb-acc0-63225d983d12"),
						CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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

			mockReq.EXPECT().ConferenceV1ConferenceGets(ctx, tt.token, tt.limit, tt.expectFilters).Return(tt.response, nil)
			res, err := h.ConferenceGets(ctx, tt.agent, tt.limit, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceGet(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		id        uuid.UUID
		response  *cfconference.Conference
		expectRes *cfconference.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
			&cfconference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cfconference.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.id).Return(tt.response, nil)
			res, err := h.ConferenceGet(ctx, tt.agent, tt.id)
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
		name       string
		agent      *amagent.Agent
		id         uuid.UUID
		updateName string
		detail     string
		data       map[string]any
		timeout    int
		preFlowID  uuid.UUID
		postFlowID uuid.UUID

		response  *cfconference.Conference
		expectRes *cfconference.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			id:         uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
			updateName: "update name",
			detail:     "update detail",
			timeout:    86400,
			preFlowID:  uuid.FromStringOrNil("3c9c8bc8-1e1d-11f0-a370-03530069d812"),
			postFlowID: uuid.FromStringOrNil("3cc09626-1e1d-11f0-ab8a-6be43ca1e9e9"),

			response: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &cfconference.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("78396a1c-202d-11ec-a85f-67fefb00b6a7"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.id).Return(tt.response, nil)
			mockReq.EXPECT().ConferenceV1ConferenceUpdate(ctx, tt.id, tt.updateName, tt.detail, tt.data, tt.timeout, tt.preFlowID, tt.postFlowID).Return(tt.response, nil)
			res, err := h.ConferenceUpdate(ctx, tt.agent, tt.id, tt.updateName, tt.detail, tt.data, tt.timeout, tt.preFlowID, tt.postFlowID)
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

		agent        *amagent.Agent
		conferenceID uuid.UUID
		format       cmrecording.Format
		duration     int
		onEndFlowID  uuid.UUID

		responseconference *cfconference.Conference
		expectRes          *cfconference.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			conferenceID: uuid.FromStringOrNil("6d48be14-910b-11ed-b644-eb3bf9ff8517"),
			format:       cmrecording.FormatWAV,
			duration:     3600,
			onEndFlowID:  uuid.FromStringOrNil("9c050072-0563-11f0-ac0d-3b5ae35ace28"),

			responseconference: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6d48be14-910b-11ed-b644-eb3bf9ff8517"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &cfconference.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6d48be14-910b-11ed-b644-eb3bf9ff8517"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.conferenceID).Return(tt.responseconference, nil)
			mockReq.EXPECT().ConferenceV1ConferenceRecordingStart(ctx, tt.conferenceID, uuid.Nil, tt.format, tt.duration, tt.onEndFlowID).Return(tt.responseconference, nil)
			res, err := h.ConferenceRecordingStart(ctx, tt.agent, tt.conferenceID, tt.format, tt.duration, tt.onEndFlowID)
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

		agent        *amagent.Agent
		conferenceID uuid.UUID

		responseconference *cfconference.Conference
		expectRes          *cfconference.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("f6e67710-910b-11ed-b11d-abaf81af53bf"),

			&cfconference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f6e67710-910b-11ed-b11d-abaf81af53bf"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cfconference.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f6e67710-910b-11ed-b11d-abaf81af53bf"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.conferenceID).Return(tt.responseconference, nil)
			mockReq.EXPECT().ConferenceV1ConferenceRecordingStop(ctx, tt.conferenceID).Return(tt.responseconference, nil)
			res, err := h.ConferenceRecordingStop(ctx, tt.agent, tt.conferenceID)
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

		agent        *amagent.Agent
		conferenceID uuid.UUID
		language     string

		responseconference *cfconference.Conference
		expectRes          *cfconference.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("729da1a8-98eb-11ed-8fa8-1b689360397c"),
			"en-US",

			&cfconference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("729da1a8-98eb-11ed-8fa8-1b689360397c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cfconference.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("729da1a8-98eb-11ed-8fa8-1b689360397c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.conferenceID).Return(tt.responseconference, nil)
			mockReq.EXPECT().ConferenceV1ConferenceTranscribeStart(ctx, tt.conferenceID, tt.language).Return(tt.responseconference, nil)
			res, err := h.ConferenceTranscribeStart(ctx, tt.agent, tt.conferenceID, tt.language)
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

		agent        *amagent.Agent
		conferenceID uuid.UUID

		responseconference *cfconference.Conference
		expectRes          *cfconference.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("72d42a48-98eb-11ed-bd79-f3b90badd9ad"),

			&cfconference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72d42a48-98eb-11ed-bd79-f3b90badd9ad"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cfconference.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72d42a48-98eb-11ed-bd79-f3b90badd9ad"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.conferenceID).Return(tt.responseconference, nil)
			mockReq.EXPECT().ConferenceV1ConferenceTranscribeStop(ctx, tt.conferenceID).Return(tt.responseconference, nil)
			res, err := h.ConferenceTranscribeStop(ctx, tt.agent, tt.conferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceMediaStreamStart(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		conferenceID  uuid.UUID
		encapsulation string
		writer        http.ResponseWriter
		request       *http.Request

		responseConference *cfconference.Conference

		expectRes []*cfconference.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("9543aae2-eb4a-11ee-987e-e725fbe471f2"),
			"rtp",
			&mockResponseWriter{},
			&http.Request{},

			&cfconference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9543aae2-eb4a-11ee-987e-e725fbe471f2"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
			[]*cfconference.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("9543aae2-eb4a-11ee-987e-e725fbe471f2"),
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
			mockWebsock := websockhandler.NewMockWebsockHandler(mc)

			h := serviceHandler{
				reqHandler:     mockReq,
				dbHandler:      mockDB,
				websockHandler: mockWebsock,
			}
			ctx := context.Background()

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.conferenceID).Return(tt.responseConference, nil)
			mockWebsock.EXPECT().RunMediaStream(ctx, tt.writer, tt.request, cmexternalmedia.ReferenceTypeConfbridge, tt.responseConference.ConfbridgeID, tt.encapsulation).Return(nil)

			if err := h.ConferenceMediaStreamStart(ctx, tt.agent, tt.conferenceID, tt.encapsulation, tt.writer, tt.request); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
