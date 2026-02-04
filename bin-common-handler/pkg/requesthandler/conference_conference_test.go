package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cmrecording "monorepo/bin-call-manager/models/recording"
	cfconference "monorepo/bin-conference-manager/models/conference"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_ConferenceV1ConferenceGet(t *testing.T) {

	type test struct {
		name         string
		conferenceID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *cfconference.Conference
	}

	tests := []test{
		{
			name:         "normal",
			conferenceID: uuid.FromStringOrNil("c337c4de-4132-11ec-b076-ab42296b65d5"),

			expectQueue: "bin-manager.conference-manager.request",
			expectRequest: &sock.Request{
				URI:    "/v1/conferences/c337c4de-4132-11ec-b076-ab42296b65d5",
				Method: sock.RequestMethodGet,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"c337c4de-4132-11ec-b076-ab42296b65d5"}`),
			},
			expectRes: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c337c4de-4132-11ec-b076-ab42296b65d5"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConferenceV1ConferenceGet(context.Background(), tt.conferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceV1ConferenceList(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[cfconference.Field]any

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []cfconference.Conference
	}{
		{
			"normal conference",

			"2021-03-02T03:23:20.995000Z",
			10,
			map[cfconference.Field]any{
				cfconference.FieldType: string(cfconference.TypeConference),
			},

			"bin-manager.conference-manager.request",
			&sock.Request{
				URI:      "/v1/conferences?page_token=2021-03-02T03%3A23%3A20.995000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"type":"conference"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"281c89f0-ec61-11ec-a18d-a7389bd741ca"},{"id":"2886cafe-ec61-11ec-b982-5b047f4851d6"}]`),
			},

			[]cfconference.Conference{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("281c89f0-ec61-11ec-a18d-a7389bd741ca"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2886cafe-ec61-11ec-b982-5b047f4851d6"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.ConferenceV1ConferenceList(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceV1ConferenceDelete(t *testing.T) {

	tests := []struct {
		name string

		conferenceID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cfconference.Conference
	}{
		{
			"normal",
			uuid.FromStringOrNil("2d9227a4-3d17-11ec-ab43-cfdad30eccdf"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2d9227a4-3d17-11ec-ab43-cfdad30eccdf"}`),
			},

			"bin-manager.conference-manager.request",
			&sock.Request{
				URI:    "/v1/conferences/2d9227a4-3d17-11ec-ab43-cfdad30eccdf",
				Method: sock.RequestMethodDelete,
			},
			&cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2d9227a4-3d17-11ec-ab43-cfdad30eccdf"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConferenceV1ConferenceDelete(ctx, tt.conferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ConferenceV1ConferenceStop(t *testing.T) {

	tests := []struct {
		name string

		conferenceID uuid.UUID
		delay        int

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cfconference.Conference
	}{
		{
			name:         "normal",
			conferenceID: uuid.FromStringOrNil("9df75377-cffe-448a-825e-7afc7f86f9e6"),
			delay:        0,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9df75377-cffe-448a-825e-7afc7f86f9e6"}`),
			},

			expectTarget: "bin-manager.conference-manager.request",
			expectRequest: &sock.Request{
				URI:    "/v1/conferences/9df75377-cffe-448a-825e-7afc7f86f9e6/stop",
				Method: sock.RequestMethodPost,
			},
			expectRes: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9df75377-cffe-448a-825e-7afc7f86f9e6"),
				},
			},
		},
		{
			name:         "delay stop",
			conferenceID: uuid.FromStringOrNil("7b85487d-d251-44e6-b7c6-8cee606c9d00"),
			delay:        100000,

			response: nil,

			expectTarget: "bin-manager.conference-manager.request",
			expectRequest: &sock.Request{
				URI:    "/v1/conferences/7b85487d-d251-44e6-b7c6-8cee606c9d00/stop",
				Method: sock.RequestMethodPost,
			},
			expectRes: &cfconference.Conference{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			if tt.delay == 0 {
				mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
			} else {
				mockSock.EXPECT().RequestPublishWithDelay(
					tt.expectTarget,
					tt.expectRequest,
					tt.delay,
				).Return(nil)
			}

			res, err := reqHandler.ConferenceV1ConferenceStop(ctx, tt.conferenceID, tt.delay)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ConferenceV1ConferenceCreate(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		customerID     uuid.UUID
		conferenceType cfconference.Type
		conferenceName string
		detail         string
		data           map[string]any
		timeout        int
		preFlowID      uuid.UUID
		postFlowID     uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cfconference.Conference
	}{
		{
			name: "normal",

			id:             uuid.FromStringOrNil("356d3738-1e15-11f0-b30f-7316bf9e4453"),
			customerID:     uuid.FromStringOrNil("9d27750e-7f4f-11ec-b98f-839769cdfb25"),
			conferenceType: cfconference.TypeConnect,
			conferenceName: "test",
			detail:         "test detail",
			data:           map[string]any{"key1": "val1"},
			timeout:        86400000,
			preFlowID:      uuid.FromStringOrNil("35dfdb9e-1e15-11f0-9c43-a7aa985a8d9e"),
			postFlowID:     uuid.FromStringOrNil("3611cafa-1e15-11f0-9d4e-9fb24b4c8272"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"04432fd6-3d19-11ec-8ad9-43e6162f0953"}`),
			},

			expectTarget: "bin-manager.conference-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/conferences",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"356d3738-1e15-11f0-b30f-7316bf9e4453","customer_id":"9d27750e-7f4f-11ec-b98f-839769cdfb25","type":"connect","name":"test","detail":"test detail","data":{"key1":"val1"},"timeout":86400000,"pre_flow_id":"35dfdb9e-1e15-11f0-9c43-a7aa985a8d9e","post_flow_id":"3611cafa-1e15-11f0-9d4e-9fb24b4c8272"}`),
			},
			expectRes: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("04432fd6-3d19-11ec-8ad9-43e6162f0953"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			cf, err := reqHandler.ConferenceV1ConferenceCreate(
				ctx,
				tt.id,
				tt.customerID,
				tt.conferenceType,
				tt.conferenceName,
				tt.detail,
				tt.data,
				tt.timeout,
				tt.preFlowID,
				tt.postFlowID,
			)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}

func Test_ConferenceV1ConferenceUpdate(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		conferenceName string
		detail         string
		data           map[string]any
		timeout        int
		preFlowID      uuid.UUID
		postFlowID     uuid.UUID

		response      *sock.Response
		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cfconference.Conference
	}{
		{
			name: "normal",

			id:             uuid.FromStringOrNil("77ebcd6c-1e16-11f0-9bb7-c3dbf388b8ac"),
			conferenceName: "test",
			detail:         "test detail",
			data:           map[string]any{"key1": "val1"},
			timeout:        86400000,
			preFlowID:      uuid.FromStringOrNil("781eb272-1e16-11f0-9cb8-9b794114fb25"),
			postFlowID:     uuid.FromStringOrNil("7847a3da-1e16-11f0-8549-9f04e3676942"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"77ebcd6c-1e16-11f0-9bb7-c3dbf388b8ac"}`),
			},
			expectTarget: "bin-manager.conference-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/conferences/77ebcd6c-1e16-11f0-9bb7-c3dbf388b8ac",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"test","detail":"test detail","data":{"key1":"val1"},"timeout":86400000,"pre_flow_id":"781eb272-1e16-11f0-9cb8-9b794114fb25","post_flow_id":"7847a3da-1e16-11f0-8549-9f04e3676942"}`),
			},
			expectRes: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("77ebcd6c-1e16-11f0-9bb7-c3dbf388b8ac"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConferenceV1ConferenceUpdate(ctx, tt.id, tt.conferenceName, tt.detail, tt.data, tt.timeout, tt.preFlowID, tt.postFlowID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceV1ConferenceUpdateRecordingID(t *testing.T) {

	tests := []struct {
		name string

		id          uuid.UUID
		recordingID uuid.UUID

		response      *sock.Response
		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cfconference.Conference
	}{
		{
			name: "normal",

			id:          uuid.FromStringOrNil("6a8bb630-909e-11ed-8e51-4ba49096d3f7"),
			recordingID: uuid.FromStringOrNil("6ad3b3cc-909e-11ed-b6de-bb34ce55e617"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6a8bb630-909e-11ed-8e51-4ba49096d3f7"}`),
			},
			expectTarget: "bin-manager.conference-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/conferences/6a8bb630-909e-11ed-8e51-4ba49096d3f7/recording_id",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"recording_id":"6ad3b3cc-909e-11ed-b6de-bb34ce55e617"}`),
			},
			expectRes: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6a8bb630-909e-11ed-8e51-4ba49096d3f7"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConferenceV1ConferenceUpdateRecordingID(ctx, tt.id, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceV1ConferenceRecordingStart(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		activeflowID uuid.UUID
		format       cmrecording.Format
		duration     int
		onEndFlowID  uuid.UUID

		response      *sock.Response
		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cfconference.Conference
	}{
		{
			name: "normal",

			id:           uuid.FromStringOrNil("062311b6-9107-11ed-bd31-fb8ce20a3bd7"),
			activeflowID: uuid.FromStringOrNil("a129ba1c-075b-11f0-9356-b3b5e89e14f0"),
			format:       cmrecording.FormatWAV,
			duration:     600,
			onEndFlowID:  uuid.FromStringOrNil("01eac468-055e-11f0-b60a-2753cc705cdb"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"062311b6-9107-11ed-bd31-fb8ce20a3bd7"}`),
			},
			expectTarget: "bin-manager.conference-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/conferences/062311b6-9107-11ed-bd31-fb8ce20a3bd7/recording_start",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"activeflow_id":"a129ba1c-075b-11f0-9356-b3b5e89e14f0","format":"wav","duration":600,"on_end_flow_id":"01eac468-055e-11f0-b60a-2753cc705cdb"}`),
			},
			expectRes: &cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("062311b6-9107-11ed-bd31-fb8ce20a3bd7"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConferenceV1ConferenceRecordingStart(ctx, tt.id, tt.activeflowID, tt.format, tt.duration, tt.onEndFlowID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceV1ConferenceRecordingStop(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response      *sock.Response
		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cfconference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("0660ce2a-9107-11ed-8c04-93e3837ffdcd"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0660ce2a-9107-11ed-8c04-93e3837ffdcd"}`),
			},
			"bin-manager.conference-manager.request",
			&sock.Request{
				URI:    "/v1/conferences/0660ce2a-9107-11ed-8c04-93e3837ffdcd/recording_stop",
				Method: sock.RequestMethodPost,
			},
			&cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0660ce2a-9107-11ed-8c04-93e3837ffdcd"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConferenceV1ConferenceRecordingStop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceV1ConferenceTranscribeStart(t *testing.T) {

	tests := []struct {
		name string

		id       uuid.UUID
		language string

		response      *sock.Response
		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cfconference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("dfa5e700-98e7-11ed-a643-4bd2f59007ae"),
			"en-US",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"dfa5e700-98e7-11ed-a643-4bd2f59007ae"}`),
			},
			"bin-manager.conference-manager.request",
			&sock.Request{
				URI:      "/v1/conferences/dfa5e700-98e7-11ed-a643-4bd2f59007ae/transcribe_start",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"language":"en-US"}`),
			},
			&cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dfa5e700-98e7-11ed-a643-4bd2f59007ae"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConferenceV1ConferenceTranscribeStart(ctx, tt.id, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceV1ConferenceTranscribeStop(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response      *sock.Response
		expectTarget  string
		expectRequest *sock.Request
		expectRes     *cfconference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("dfda30dc-98e7-11ed-a69c-e781929a3118"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"dfda30dc-98e7-11ed-a69c-e781929a3118"}`),
			},
			"bin-manager.conference-manager.request",
			&sock.Request{
				URI:    "/v1/conferences/dfda30dc-98e7-11ed-a69c-e781929a3118/transcribe_stop",
				Method: sock.RequestMethodPost,
			},
			&cfconference.Conference{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dfda30dc-98e7-11ed-a69c-e781929a3118"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConferenceV1ConferenceTranscribeStop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
