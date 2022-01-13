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

func TestCreate(t *testing.T) {
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

	tests := []struct {
		name string

		conferenceType conference.Type
		userID         uint64
		conferenceName string
		detail         string
		timeout        int
		webhookURI     string
		preActions     []fmaction.Action
		postActions    []fmaction.Action

		responseConfbridge *cmconfbridge.Confbridge
		responseFlow       *fmflow.Flow

		expectRes *conference.Conference
	}{
		{
			"normal",

			conference.TypeConnect,
			1,
			"test name",
			"test detail",
			86400,
			"test.com",
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

			ctx := context.Background()

			mockReq.EXPECT().CMV1ConfbridgeCreate(gomock.Any()).Return(tt.responseConfbridge, nil)
			mockReq.EXPECT().FMV1FlowCreate(gomock.Any(), gomock.Any(), fmflow.TypeConference, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseFlow, nil)
			mockDB.EXPECT().ConferenceCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().ConferenceConfbridgeSet(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().ConferenceGet(gomock.Any(), gomock.Any()).Return(tt.expectRes, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), conference.EventTypeConferenceCreated, gomock.Any(), gomock.Any())
			if tt.timeout > 0 {
				mockReq.EXPECT().CFV1ConferenceDeleteDelay(gomock.Any(), gomock.Any(), tt.timeout*1000).Return(nil)
			}

			res, err := h.Create(ctx, tt.conferenceType, tt.userID, tt.conferenceName, tt.detail, tt.timeout, tt.webhookURI, tt.preActions, tt.postActions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestCreateConferenceFlow(t *testing.T) {
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

	tests := []struct {
		name string

		userID       uint64
		conferenceID uuid.UUID
		confbridgeID uuid.UUID
		preActions   []fmaction.Action
		postActions  []fmaction.Action
		flowName     string

		responseFlow *fmflow.Flow
	}{
		{
			"normal",

			1,
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
			ctx := context.Background()

			mockReq.EXPECT().FMV1FlowCreate(gomock.Any(), tt.userID, fmflow.TypeConference, tt.flowName, "generated for conference by conference-manager.", "", gomock.Any(), true).Return(tt.responseFlow, nil)

			_, err := h.createConferenceFlow(ctx, tt.userID, tt.conferenceID, tt.confbridgeID, tt.preActions, tt.postActions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestCreateConferenceFlowActions(t *testing.T) {
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
