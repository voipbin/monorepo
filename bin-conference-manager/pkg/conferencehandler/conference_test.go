package conferencehandler

import (
	"context"
	"reflect"
	"testing"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conference"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		customerID     uuid.UUID
		conferenceType conference.Type
		conferenceName string
		detail         string
		data           map[string]any
		timeout        int
		preFlowID      uuid.UUID
		postFlowID     uuid.UUID

		responseUUID       uuid.UUID
		responseConfbridge *cmconfbridge.Confbridge
		responseFlow       *fmflow.Flow

		expectConference *conference.Conference
		expectRes        *conference.Conference
	}{
		{
			name: "normal",

			id:             uuid.Nil,
			customerID:     uuid.FromStringOrNil("4fa8d53a-8057-11ec-9e7c-2310213dc857"),
			conferenceType: conference.TypeConnect,
			conferenceName: "test name",
			detail:         "test detail",
			data: map[string]any{
				"key1": "value1",
			},
			timeout:    86400,
			preFlowID:  uuid.FromStringOrNil("4b33ead2-1e0e-11f0-925e-2b4560faf69c"),
			postFlowID: uuid.FromStringOrNil("4b73ade8-1e0e-11f0-99e3-2388d8687539"),

			responseUUID: uuid.FromStringOrNil("944e0272-06b5-11f0-8fb6-43b650d2e25d"),
			responseConfbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a5aab3aa-5b8a-11ec-bf89-432b7557fb8b"),
				},
			},
			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a5d4bad8-5b8a-11ec-a510-57d74c7f1270"),
				},
			},

			expectConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("944e0272-06b5-11f0-8fb6-43b650d2e25d"),
					CustomerID: uuid.FromStringOrNil("4fa8d53a-8057-11ec-9e7c-2310213dc857"),
				},
				ConfbridgeID: uuid.FromStringOrNil("a5aab3aa-5b8a-11ec-bf89-432b7557fb8b"),
				Type:         conference.TypeConnect,
				Status:       conference.StatusProgressing,
				Name:         "test name",
				Detail:       "test detail",
				Data: map[string]any{
					"key1": "value1",
				},
				Timeout:           86400,
				PreFlowID:         uuid.FromStringOrNil("4b33ead2-1e0e-11f0-925e-2b4560faf69c"),
				PostFlowID:        uuid.FromStringOrNil("4b73ade8-1e0e-11f0-99e3-2388d8687539"),
				ConferencecallIDs: []uuid.UUID{},
				RecordingIDs:      []uuid.UUID{},
				TranscribeIDs:     []uuid.UUID{},
			},
			expectRes: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("944e0272-06b5-11f0-8fb6-43b650d2e25d"),
				},
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			confbridgeType := cmconfbridge.TypeConnect
			if tt.conferenceType == conference.TypeConference {
				confbridgeType = cmconfbridge.TypeConference
			}
			mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, tt.customerID, uuid.Nil, cmconfbridge.ReferenceTypeConference, tt.responseUUID, confbridgeType).Return(tt.responseConfbridge, nil)
			mockDB.EXPECT().ConferenceCreate(ctx, tt.expectConference).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, gomock.Any()).Return(tt.expectRes, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectRes.CustomerID, conference.EventTypeConferenceCreated, gomock.Any())
			if tt.timeout > 0 {
				mockReq.EXPECT().ConferenceV1ConferenceDeleteDelay(ctx, gomock.Any(), tt.timeout*1000).Return(nil)
			}

			res, err := h.Create(ctx, tt.id, tt.customerID, tt.conferenceType, tt.conferenceName, tt.detail, tt.data, tt.timeout, tt.preFlowID, tt.postFlowID)
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3d72398e-13bd-11ed-9abc-67fe710b5f19"),
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

func Test_Update(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		conferenceName string
		detail         string
		data           map[string]any
		timeout        int
		preFlowID      uuid.UUID
		postFlowID     uuid.UUID

		expectFields       map[conference.Field]any
		responseConference *conference.Conference
	}{
		{
			name: "normal",

			id:             uuid.FromStringOrNil("c40d48ac-1e10-11f0-a6b3-276e2a2df365"),
			conferenceName: "update name",
			detail:         "update detail",
			data: map[string]any{
				"key1": "value1",
			},
			timeout:    86400,
			preFlowID:  uuid.FromStringOrNil("c4642da2-1e10-11f0-9c2d-9fadec265c1d"),
			postFlowID: uuid.FromStringOrNil("c48cc2e4-1e10-11f0-9e00-bbd2a97e0a8e"),

			expectFields: map[conference.Field]any{
				conference.FieldName:       "update name",
				conference.FieldDetail:     "update detail",
				conference.FieldData:       map[string]any{"key1": "value1"},
				conference.FieldTimeout:    86400,
				conference.FieldPreFlowID:  uuid.FromStringOrNil("c4642da2-1e10-11f0-9c2d-9fadec265c1d"),
				conference.FieldPostFlowID: uuid.FromStringOrNil("c48cc2e4-1e10-11f0-9e00-bbd2a97e0a8e"),
			},
			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c40d48ac-1e10-11f0-a6b3-276e2a2df365"),
				},
			},
		},
		{
			name: "update to nil",

			id: uuid.FromStringOrNil("c4bfe75a-1e10-11f0-802b-0769590697e5"),

			expectFields: map[conference.Field]any{
				conference.FieldName:       "",
				conference.FieldDetail:     "",
				conference.FieldData:       map[string]any(nil),
				conference.FieldTimeout:    0,
				conference.FieldPreFlowID:  uuid.Nil,
				conference.FieldPostFlowID: uuid.Nil,
			},
			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c4bfe75a-1e10-11f0-802b-0769590697e5"),
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

			mockDB.EXPECT().ConferenceUpdate(ctx, tt.id, tt.expectFields).Return(nil)
			mockDB.EXPECT().ConferenceGet(ctx, tt.id).Return(tt.responseConference, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseConference.CustomerID, conference.EventTypeConferenceUpdated, tt.responseConference)

			res, err := h.Update(ctx, tt.id, tt.conferenceName, tt.detail, tt.data, tt.timeout, tt.preFlowID, tt.postFlowID)
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

		expectFields       map[conference.Field]any
		responseConference *conference.Conference
	}{
		{
			name: "normal",

			id:          uuid.FromStringOrNil("cbbf9c7c-9090-11ed-b005-b76aad1ce504"),
			recordingID: uuid.FromStringOrNil("cbf14380-9090-11ed-8ae5-0bdda69156ed"),

			expectFields: map[conference.Field]any{
				conference.FieldRecordingID: uuid.FromStringOrNil("cbf14380-9090-11ed-8ae5-0bdda69156ed"),
			},
			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cbbf9c7c-9090-11ed-b005-b76aad1ce504"),
				},
			},
		},
		{
			name: "update to nil",

			id:          uuid.FromStringOrNil("f036af26-98c1-11ed-a796-0753c08f103e"),
			recordingID: uuid.Nil,

			expectFields: map[conference.Field]any{
				conference.FieldRecordingID: uuid.Nil,
			},
			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f036af26-98c1-11ed-a796-0753c08f103e"),
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

			mockDB.EXPECT().ConferenceUpdate(ctx, tt.id, tt.expectFields).Return(nil)
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

		expectFields       map[conference.Field]any
		responseConference *conference.Conference
	}{
		{
			name: "normal",

			id:           uuid.FromStringOrNil("d5e141cc-98c1-11ed-8cce-6b7c71104332"),
			transcribeID: uuid.FromStringOrNil("d629b3ee-98c1-11ed-8d37-d7fa7c56235d"),

			expectFields: map[conference.Field]any{
				conference.FieldTranscribeID: uuid.FromStringOrNil("d629b3ee-98c1-11ed-8d37-d7fa7c56235d"),
			},
			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d5e141cc-98c1-11ed-8cce-6b7c71104332"),
				},
			},
		},
		{
			name: "update to nil",

			id:           uuid.FromStringOrNil("f0138906-98c1-11ed-8b15-77a88eafac75"),
			transcribeID: uuid.Nil,

			expectFields: map[conference.Field]any{
				conference.FieldTranscribeID: uuid.Nil,
			},
			responseConference: &conference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f0138906-98c1-11ed-8b15-77a88eafac75"),
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

			mockDB.EXPECT().ConferenceUpdate(ctx, tt.id, tt.expectFields).Return(nil)
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3a9eb896-94a9-11ed-b58c-af21495e92d6"),
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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9e43cc72-94e2-11ed-b2f6-b3ea29f01f60"),
				},
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
		filters    map[conference.Field]any

		responseConference []*conference.Conference

		expectRes []*conference.Conference
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("c7dc2ef0-afd3-11ee-a624-3fa2cdf1cb55"),
			size:       10,
			token:      "2023-01-03 21:35:02.809",
			filters: map[conference.Field]any{
				conference.FieldType: conference.TypeConnect,
			},

			responseConference: []*conference.Conference{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c831941c-afd3-11ee-b91b-8fdf2766ea5e"),
					},
				},
			},

			expectRes: []*conference.Conference{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c831941c-afd3-11ee-b91b-8fdf2766ea5e"),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConferenceList(ctx, tt.size, tt.token, tt.filters).Return(tt.responseConference, nil)

			res, err := h.List(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
