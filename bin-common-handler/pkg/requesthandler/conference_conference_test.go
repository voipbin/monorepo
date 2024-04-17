package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cfconference "monorepo/bin-conference-manager/models/conference"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_ConferenceV1ConferenceGet(t *testing.T) {

	type test struct {
		name         string
		conferenceID uuid.UUID

		expectQueue   string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *cfconference.Conference
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("c337c4de-4132-11ec-b076-ab42296b65d5"),

			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences/c337c4de-4132-11ec-b076-ab42296b65d5",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"c337c4de-4132-11ec-b076-ab42296b65d5","flow_id":"e0e5c2ba-4132-11ec-a38b-c7c6ccec4af6"}`),
			},
			&cfconference.Conference{
				ID:     uuid.FromStringOrNil("c337c4de-4132-11ec-b076-ab42296b65d5"),
				FlowID: uuid.FromStringOrNil("e0e5c2ba-4132-11ec-a38b-c7c6ccec4af6"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConferenceV1ConferenceGet(context.Background(), tt.conferenceID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceV1ConferenceGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes []cfconference.Conference
	}{
		{
			"normal conference",

			"2021-03-02 03:23:20.995000",
			10,
			map[string]string{
				"type": string(cfconference.TypeConference),
			},

			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10&filter_type=conference",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"281c89f0-ec61-11ec-a18d-a7389bd741ca"},{"id":"2886cafe-ec61-11ec-b982-5b047f4851d6"}]`),
			},

			[]cfconference.Conference{
				{
					ID: uuid.FromStringOrNil("281c89f0-ec61-11ec-a18d-a7389bd741ca"),
				},
				{
					ID: uuid.FromStringOrNil("2886cafe-ec61-11ec-b982-5b047f4851d6"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConferenceV1ConferenceGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
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

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cfconference.Conference
	}{
		{
			"normal",
			uuid.FromStringOrNil("2d9227a4-3d17-11ec-ab43-cfdad30eccdf"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2d9227a4-3d17-11ec-ab43-cfdad30eccdf"}`),
			},

			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences/2d9227a4-3d17-11ec-ab43-cfdad30eccdf",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&cfconference.Conference{
				ID: uuid.FromStringOrNil("2d9227a4-3d17-11ec-ab43-cfdad30eccdf"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cfconference.Conference
	}{
		{
			"normal",
			uuid.FromStringOrNil("9df75377-cffe-448a-825e-7afc7f86f9e6"),
			0,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9df75377-cffe-448a-825e-7afc7f86f9e6"}`),
			},

			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences/9df75377-cffe-448a-825e-7afc7f86f9e6/stop",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			&cfconference.Conference{
				ID: uuid.FromStringOrNil("9df75377-cffe-448a-825e-7afc7f86f9e6"),
			},
		},
		{
			"delay stop",
			uuid.FromStringOrNil("7b85487d-d251-44e6-b7c6-8cee606c9d00"),
			100000,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7b85487d-d251-44e6-b7c6-8cee606c9d00"}`),
			},

			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences/7b85487d-d251-44e6-b7c6-8cee606c9d00/stop",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			if tt.delay == 0 {
				mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
			} else {
				mockSock.EXPECT().PublishExchangeDelayedRequest(
					gomock.Any(),
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

		response         *rabbitmqhandler.Response
		expectTarget     string
		expectRequest    *rabbitmqhandler.Request
		expectConference *cfconference.Conference
	}{
		{
			"normal",
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"04432fd6-3d19-11ec-8ad9-43e6162f0953","name":"test","detail":"test detail","customer_id":"9d27750e-7f4f-11ec-b98f-839769cdfb25","timeout":86400000,"type":"connect"}`),
			},
			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"connect","customer_id":"9d27750e-7f4f-11ec-b98f-839769cdfb25","name":"test","detail":"test detail","timeout":86400000,"data":null,"pre_actions":null,"post_actions":null}`),
			},
			&cfconference.Conference{
				ID:         uuid.FromStringOrNil("04432fd6-3d19-11ec-8ad9-43e6162f0953"),
				CustomerID: uuid.FromStringOrNil("9d27750e-7f4f-11ec-b98f-839769cdfb25"),
				Type:       cfconference.TypeConnect,
				Name:       "test",
				Detail:     "test detail",
				Timeout:    86400000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			cf, err := reqHandler.ConferenceV1ConferenceCreate(ctx, tt.expectConference.CustomerID, tt.expectConference.Type, tt.expectConference.Name, tt.expectConference.Detail, tt.expectConference.Timeout, nil, nil, nil)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectConference) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectConference, cf)
			}
		})
	}
}

func Test_ConferenceV1ConferenceUpdateRecordingID(t *testing.T) {

	tests := []struct {
		name string

		id          uuid.UUID
		recordingID uuid.UUID

		response      *rabbitmqhandler.Response
		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cfconference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("6a8bb630-909e-11ed-8e51-4ba49096d3f7"),
			uuid.FromStringOrNil("6ad3b3cc-909e-11ed-b6de-bb34ce55e617"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6a8bb630-909e-11ed-8e51-4ba49096d3f7"}`),
			},
			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/6a8bb630-909e-11ed-8e51-4ba49096d3f7/recording_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"recording_id":"6ad3b3cc-909e-11ed-b6de-bb34ce55e617"}`),
			},
			&cfconference.Conference{
				ID: uuid.FromStringOrNil("6a8bb630-909e-11ed-8e51-4ba49096d3f7"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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

		id uuid.UUID

		response      *rabbitmqhandler.Response
		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cfconference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("062311b6-9107-11ed-bd31-fb8ce20a3bd7"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"062311b6-9107-11ed-bd31-fb8ce20a3bd7"}`),
			},
			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences/062311b6-9107-11ed-bd31-fb8ce20a3bd7/recording_start",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			&cfconference.Conference{
				ID: uuid.FromStringOrNil("062311b6-9107-11ed-bd31-fb8ce20a3bd7"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConferenceV1ConferenceRecordingStart(ctx, tt.id)
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

		response      *rabbitmqhandler.Response
		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cfconference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("0660ce2a-9107-11ed-8c04-93e3837ffdcd"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0660ce2a-9107-11ed-8c04-93e3837ffdcd"}`),
			},
			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences/0660ce2a-9107-11ed-8c04-93e3837ffdcd/recording_stop",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			&cfconference.Conference{
				ID: uuid.FromStringOrNil("0660ce2a-9107-11ed-8c04-93e3837ffdcd"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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

		response      *rabbitmqhandler.Response
		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cfconference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("dfa5e700-98e7-11ed-a643-4bd2f59007ae"),
			"en-US",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"dfa5e700-98e7-11ed-a643-4bd2f59007ae"}`),
			},
			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferences/dfa5e700-98e7-11ed-a643-4bd2f59007ae/transcribe_start",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"language":"en-US"}`),
			},
			&cfconference.Conference{
				ID: uuid.FromStringOrNil("dfa5e700-98e7-11ed-a643-4bd2f59007ae"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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

		response      *rabbitmqhandler.Response
		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cfconference.Conference
	}{
		{
			"normal",

			uuid.FromStringOrNil("dfda30dc-98e7-11ed-a69c-e781929a3118"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"dfda30dc-98e7-11ed-a69c-e781929a3118"}`),
			},
			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/conferences/dfda30dc-98e7-11ed-a69c-e781929a3118/transcribe_stop",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			&cfconference.Conference{
				ID: uuid.FromStringOrNil("dfda30dc-98e7-11ed-a69c-e781929a3118"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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
