package conferencehandler

import (
	"context"
	"reflect"
	"testing"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		conferenceType conference.Type
		customerID     uuid.UUID
		conferenceName string
		detail         string
		timeout        int
		preActions     []fmaction.Action
		postActions    []fmaction.Action

		responseConfbridge *cmconfbridge.Confbridge
		responseFlow       *fmflow.Flow

		expectRes *conference.Conference
	}{
		{
			"normal",

			conference.TypeConnect,
			uuid.FromStringOrNil("4fa8d53a-8057-11ec-9e7c-2310213dc857"),
			"test name",
			"test detail",
			86400,
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			[]fmaction.Action{
				{
					Type: fmaction.TypeTalk,
				},
			},

			&cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("a5aab3aa-5b8a-11ec-bf89-432b7557fb8b"),
			},
			&fmflow.Flow{
				ID: uuid.FromStringOrNil("a5d4bad8-5b8a-11ec-a510-57d74c7f1270"),
			},

			&conference.Conference{
				ID:           uuid.FromStringOrNil("a5f5c12e-5b8a-11ec-9358-f381c6b41b9f"),
				Type:         conference.TypeConference,
				ConfbridgeID: uuid.FromStringOrNil("a5aab3aa-5b8a-11ec-bf89-432b7557fb8b"),
				Timeout:      86400,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			ctx := context.Background()

			confbridgeType := cmconfbridge.TypeConnect
			if tt.conferenceType == conference.TypeConference {
				confbridgeType = cmconfbridge.TypeConference
			}
			mockReq.EXPECT().CallV1ConfbridgeCreate(gomock.Any(), tt.customerID, confbridgeType).Return(tt.responseConfbridge, nil)
			mockReq.EXPECT().FlowV1FlowCreate(gomock.Any(), gomock.Any(), fmflow.TypeConference, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseFlow, nil)
			mockDB.EXPECT().ConferenceCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().ConferenceGet(gomock.Any(), gomock.Any()).Return(tt.expectRes, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.expectRes.CustomerID, conference.EventTypeConferenceCreated, gomock.Any())
			if tt.timeout > 0 {
				mockReq.EXPECT().ConferenceV1ConferenceDeleteDelay(gomock.Any(), gomock.Any(), tt.timeout*1000).Return(nil)
			}

			res, err := h.Create(ctx, tt.conferenceType, tt.customerID, tt.conferenceName, tt.detail, tt.timeout, tt.preActions, tt.postActions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_createConferenceFlow(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		conferenceID uuid.UUID
		confbridgeID uuid.UUID
		preActions   []fmaction.Action
		postActions  []fmaction.Action
		flowName     string

		responseFlow *fmflow.Flow
	}{
		{
			"normal",

			uuid.FromStringOrNil("4fa8d53a-8057-11ec-9e7c-2310213dc857"),
			uuid.FromStringOrNil("28d88218-5b89-11ec-afc1-f38e08436159"),
			uuid.FromStringOrNil("2902463e-5b89-11ec-abcf-33e30ef231fb"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			[]fmaction.Action{
				{
					Type: fmaction.TypeTalk,
				},
			},
			"conference-28d88218-5b89-11ec-afc1-f38e08436159",

			&fmflow.Flow{
				ID: uuid.FromStringOrNil("65038316-5b87-11ec-b9ce-875d97997cbb"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowCreate(gomock.Any(), tt.customerID, fmflow.TypeConference, tt.flowName, "generated for conference by conference-manager.", gomock.Any(), true).Return(tt.responseFlow, nil)

			_, err := h.createConferenceFlow(ctx, tt.customerID, tt.conferenceID, tt.confbridgeID, tt.preActions, tt.postActions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_createConferenceFlowActions(t *testing.T) {

	tests := []struct {
		name string

		confbridgeID uuid.UUID
		preActions   []fmaction.Action
		postActions  []fmaction.Action

		expectRes []fmaction.Action
	}{
		{
			"normal",

			uuid.FromStringOrNil("3cc2ca6e-5b88-11ec-8e91-9f93e28a48df"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			[]fmaction.Action{
				{
					Type: fmaction.TypeTalk,
				},
			},

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type:   fmaction.TypeConfbridgeJoin,
					Option: []byte(`{"confbridge_id":"3cc2ca6e-5b88-11ec-8e91-9f93e28a48df"}`),
				},
				{
					Type: fmaction.TypeTalk,
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			res, err := h.createConferenceFlowActions(tt.confbridgeID, tt.preActions, tt.postActions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_GetByConfbridgeID(t *testing.T) {

	tests := []struct {
		name string

		confbridgeID uuid.UUID

		responseConference *conference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("3d72398e-13bd-11ed-9abc-67fe710b5f19"),

			&conference.Conference{
				ID: uuid.FromStringOrNil("3d72398e-13bd-11ed-9abc-67fe710b5f19"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			ctx := context.Background()

			mockDB.EXPECT().ConferenceGetByConfbridgeID(ctx, tt.confbridgeID).Return(tt.responseConference, nil)

			res, err := h.GetByConfbridgeID(ctx, tt.confbridgeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConference) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConference, res)
			}
		})
	}
}

func Test_UpdateRecordingID(t *testing.T) {

	tests := []struct {
		name string

		id          uuid.UUID
		recordingID uuid.UUID

		responseConference *conference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("cbbf9c7c-9090-11ed-b005-b76aad1ce504"),
			uuid.FromStringOrNil("cbf14380-9090-11ed-8ae5-0bdda69156ed"),

			&conference.Conference{
				ID: uuid.FromStringOrNil("cbbf9c7c-9090-11ed-b005-b76aad1ce504"),
			},
		},
		{
			"update to nil",

			uuid.FromStringOrNil("f036af26-98c1-11ed-a796-0753c08f103e"),
			uuid.Nil,

			&conference.Conference{
				ID: uuid.FromStringOrNil("f036af26-98c1-11ed-a796-0753c08f103e"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			ctx := context.Background()

			mockDB.EXPECT().ConferenceSetRecordingID(ctx, tt.id, tt.recordingID).Return(nil)
			if tt.recordingID != uuid.Nil {
				mockDB.EXPECT().ConferenceAddRecordingIDs(ctx, tt.id, tt.recordingID).Return(nil)
			}
			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)

			res, err := h.UpdateRecordingID(ctx, tt.id, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConference) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConference, res)
			}
		})
	}
}

func Test_UpdateTranscribeID(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		transcribeID uuid.UUID

		responseConference *conference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("d5e141cc-98c1-11ed-8cce-6b7c71104332"),
			uuid.FromStringOrNil("d629b3ee-98c1-11ed-8d37-d7fa7c56235d"),

			&conference.Conference{
				ID: uuid.FromStringOrNil("d5e141cc-98c1-11ed-8cce-6b7c71104332"),
			},
		},
		{
			"update to nil",

			uuid.FromStringOrNil("f0138906-98c1-11ed-8b15-77a88eafac75"),
			uuid.Nil,

			&conference.Conference{
				ID: uuid.FromStringOrNil("f0138906-98c1-11ed-8b15-77a88eafac75"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			ctx := context.Background()

			mockDB.EXPECT().ConferenceSetTranscribeID(ctx, tt.id, tt.transcribeID).Return(nil)
			if tt.transcribeID != uuid.Nil {
				mockDB.EXPECT().ConferenceAddTranscribeIDs(ctx, tt.id, tt.transcribeID).Return(nil)
			}
			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)

			res, err := h.UpdateTranscribeID(ctx, tt.id, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConference) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConference, res)
			}
		})
	}
}

func Test_AddConferencecallID(t *testing.T) {

	tests := []struct {
		name string

		id               uuid.UUID
		conferencecallID uuid.UUID

		responseConference *conference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("3a9eb896-94a9-11ed-b58c-af21495e92d6"),
			uuid.FromStringOrNil("3b1b4a50-94a9-11ed-a34c-13244d5ec3ce"),

			&conference.Conference{
				ID: uuid.FromStringOrNil("3a9eb896-94a9-11ed-b58c-af21495e92d6"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			ctx := context.Background()

			mockDB.EXPECT().ConferenceAddConferencecallID(ctx, tt.id, tt.conferencecallID).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseConference.CustomerID, conference.EventTypeConferenceUpdated, tt.responseConference)

			res, err := h.AddConferencecallID(ctx, tt.id, tt.conferencecallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConference) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConference, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConference *conference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("9e43cc72-94e2-11ed-b2f6-b3ea29f01f60"),

			&conference.Conference{
				ID:     uuid.FromStringOrNil("9e43cc72-94e2-11ed-b2f6-b3ea29f01f60"),
				Status: conference.StatusTerminating,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			ctx := context.Background()

			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)

			mockDB.EXPECT().ConferenceDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseConference.CustomerID, conference.EventTypeConferenceDeleted, tt.responseConference)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConference) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseConference, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string
		filters    map[string]string

		responseConference []*conference.Conference

		expectRes []*conference.Conference
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("c7dc2ef0-afd3-11ee-a624-3fa2cdf1cb55"),
			size:       10,
			token:      "2023-01-03 21:35:02.809",
			filters: map[string]string{
				"type": string(conference.TypeConnect),
			},

			responseConference: []*conference.Conference{
				{
					ID: uuid.FromStringOrNil("c831941c-afd3-11ee-b91b-8fdf2766ea5e"),
				},
			},

			expectRes: []*conference.Conference{
				{
					ID: uuid.FromStringOrNil("c831941c-afd3-11ee-b91b-8fdf2766ea5e"),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConferenceGets(ctx, tt.size, tt.token, tt.filters).Return(tt.responseConference, nil)

			res, err := h.Gets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
