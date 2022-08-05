package conferencehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmconfbridge "gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/dbhandler"
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
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			confbridgeType := cmconfbridge.TypeConnect
			if tt.conferenceType == conference.TypeConference {
				confbridgeType = cmconfbridge.TypeConference
			}
			mockReq.EXPECT().CMV1ConfbridgeCreate(gomock.Any(), confbridgeType).Return(tt.responseConfbridge, nil)
			mockReq.EXPECT().FMV1FlowCreate(gomock.Any(), gomock.Any(), fmflow.TypeConference, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseFlow, nil)
			mockDB.EXPECT().ConferenceCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().ConferenceGet(gomock.Any(), gomock.Any()).Return(tt.expectRes, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.expectRes.CustomerID, conference.EventTypeConferenceCreated, gomock.Any())
			if tt.timeout > 0 {
				mockReq.EXPECT().CFV1ConferenceDeleteDelay(gomock.Any(), gomock.Any(), tt.timeout*1000).Return(nil)
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
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockReq.EXPECT().FMV1FlowCreate(gomock.Any(), tt.customerID, fmflow.TypeConference, tt.flowName, "generated for conference by conference-manager.", gomock.Any(), true).Return(tt.responseFlow, nil)

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
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
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
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := conferenceHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				cache:         mockCache,
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
