package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cfconferencecall "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_ConferenceV1ConferencecallGet(t *testing.T) {

	type test struct {
		name             string
		conferencecallID uuid.UUID

		expectQueue   string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *cfconferencecall.Conferencecall
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("7baaa99e-14e8-11ed-8f79-f79014b94b6f"),

			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferencecalls/7baaa99e-14e8-11ed-8f79-f79014b94b6f",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"7baaa99e-14e8-11ed-8f79-f79014b94b6f"}`),
			},
			&cfconferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("7baaa99e-14e8-11ed-8f79-f79014b94b6f"),
			},
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConferenceV1ConferencecallGet(context.Background(), tt.conferencecallID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_ConferenceV1ConferencecallCreate(t *testing.T) {

	tests := []struct {
		name string

		conferenceID  uuid.UUID
		referenceType cfconferencecall.ReferenceType
		referenceID   uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes *cfconferencecall.Conferencecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("bfd19d92-1553-11ed-b35a-2f1b276b3437"),
			cfconferencecall.ReferenceTypeCall,
			uuid.FromStringOrNil("c079c990-1553-11ed-b46c-0bfa5493534b"),

			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferencecalls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"conference_id":"bfd19d92-1553-11ed-b35a-2f1b276b3437","reference_type":"call","reference_id":"c079c990-1553-11ed-b46c-0bfa5493534b"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c0f79c94-1553-11ed-98d9-978c6abc4479"}`),
			},

			&cfconferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("c0f79c94-1553-11ed-98d9-978c6abc4479"),
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

			cf, err := reqHandler.ConferenceV1ConferencecallCreate(ctx, tt.conferenceID, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(cf, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, cf)
			}
		})
	}
}

func Test_ConferenceV1ConferencecallKick(t *testing.T) {

	tests := []struct {
		name string

		conferencecallID uuid.UUID
		response         *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		expectRes *cfconferencecall.Conferencecall
	}{
		{
			"normal",
			uuid.FromStringOrNil("dd4ff2e2-14e5-11ed-8eec-97413dd96f29"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"dd4ff2e2-14e5-11ed-8eec-97413dd96f29"}`),
			},

			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferencecalls/dd4ff2e2-14e5-11ed-8eec-97413dd96f29",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			&cfconferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("dd4ff2e2-14e5-11ed-8eec-97413dd96f29"),
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

			res, err := reqHandler.ConferenceV1ConferencecallKick(ctx, tt.conferencecallID)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}

func Test_ConferenceV1ConferencecallHealthCheck(t *testing.T) {

	tests := []struct {
		name string

		conferencecallID uuid.UUID
		retryCount       int

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		expectRes *cfconferencecall.Conferencecall
	}{
		{
			"normal",
			uuid.FromStringOrNil("23d64db6-94a6-11ed-9b9f-2bfedef352c1"),
			2,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"23d64db6-94a6-11ed-9b9f-2bfedef352c1"}`),
			},

			"bin-manager.conference-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/conferencecalls/23d64db6-94a6-11ed-9b9f-2bfedef352c1/health-check",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"retry_count":2}`),
			},

			&cfconferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("23d64db6-94a6-11ed-9b9f-2bfedef352c1"),
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

			res, err := reqHandler.ConferenceV1ConferencecallHealthCheck(ctx, tt.conferencecallID, tt.retryCount)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}
