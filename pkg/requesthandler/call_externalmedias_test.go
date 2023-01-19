package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmexternalmedia "gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_CallV1ExternalMediaStart(t *testing.T) {

	tests := []struct {
		name string

		referenceType  cmexternalmedia.ReferenceType
		referenceID    uuid.UUID
		externalHost   string
		encapsulation  string
		transport      string
		connectionType string
		format         string
		direction      string

		response *rabbitmqhandler.Response

		expectRequest *rabbitmqhandler.Request
		expectRes     *cmexternalmedia.ExternalMedia
	}{
		{
			"normal",

			cmexternalmedia.ReferenceTypeCall,
			uuid.FromStringOrNil("94a6ec48-97c2-11ed-bd66-afb196d5c598"),
			"localhost:5060",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e8337d9a-97c2-11ed-93ad-5bcba5332622"}`),
			},

			&rabbitmqhandler.Request{
				URI:      "/v1/external-medias",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"reference_type":"call","reference_id":"94a6ec48-97c2-11ed-bd66-afb196d5c598","external_host":"localhost:5060","encapsulation":"rtp","transport":"udp","connection_type":"client","format":"ulaw","direction":"both"}`),
			},
			&cmexternalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("e8337d9a-97c2-11ed-93ad-5bcba5332622"),
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

			mockSock.EXPECT().PublishRPC(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1ExternalMediaStart(ctx, tt.referenceType, tt.referenceID, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1ExternalMediaGet(t *testing.T) {

	tests := []struct {
		name string

		externalMediaID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *cmexternalmedia.ExternalMedia
	}{
		{
			"normal",

			uuid.FromStringOrNil("0a90d6b2-97c3-11ed-a114-3b7fc4677d36"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/external-medias/0a90d6b2-97c3-11ed-a114-3b7fc4677d36",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0a90d6b2-97c3-11ed-a114-3b7fc4677d36"}`),
			},
			&cmexternalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("0a90d6b2-97c3-11ed-a114-3b7fc4677d36"),
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

			res, err := reqHandler.CallV1ExternalMediaGet(ctx, tt.externalMediaID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1ExternalMediaStop(t *testing.T) {

	tests := []struct {
		name string

		externalMediaID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *cmexternalmedia.ExternalMedia
	}{
		{
			"normal",

			uuid.FromStringOrNil("2f4d5390-97c3-11ed-9dc6-47ce92d2b198"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/external-medias/2f4d5390-97c3-11ed-9dc6-47ce92d2b198",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2f4d5390-97c3-11ed-9dc6-47ce92d2b198"}`),
			},
			&cmexternalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("2f4d5390-97c3-11ed-9dc6-47ce92d2b198"),
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

			res, err := reqHandler.CallV1ExternalMediaStop(ctx, tt.externalMediaID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
