package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/externalmediahandler"
)

func Test_processV1ExternalMediasPost(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		expectReferenceType  externalmedia.ReferenceType
		expectReferenceID    uuid.UUID
		expectExternalHost   string
		expectEncapsulation  string
		expectTransport      string
		expectConnectionType string
		expectFormat         string
		expectDirection      string

		responseExternalMedia *externalmedia.ExternalMedia
		expectRes             *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal type connect",
			&rabbitmqhandler.Request{
				URI:      "/v1/external-medias",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"reference_type":"call","reference_id":"45832182-97b2-11ed-8f17-33590535a404","external_host":"127.0.0.1:8080","encapsulation":"rtp","transport":"udp","connection_type":"client","format":"ulaw","direction":"both"}`),
			},

			externalmedia.ReferenceTypeCall,
			uuid.FromStringOrNil("45832182-97b2-11ed-8f17-33590535a404"),
			"127.0.0.1:8080",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",

			&externalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("1fc622f4-97b3-11ed-b8f9-bfd2a55f5399"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1fc622f4-97b3-11ed-b8f9-bfd2a55f5399","asterisk_id":"","channel_id":"","reference_typee":"","reference_id":"00000000-0000-0000-0000-000000000000","local_ip":"","local_port":0,"external_host":"","encapsulation":"","transport":"","connection_type":"","format":"","direction":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &listenHandler{
				rabbitSock:           mockSock,
				externalMediaHandler: mockExternal,
			}

			mockExternal.EXPECT().Start(
				gomock.Any(),
				tt.expectReferenceType,
				tt.expectReferenceID,
				tt.expectExternalHost,
				tt.expectEncapsulation,
				tt.expectTransport,
				tt.expectConnectionType,
				tt.expectFormat,
				tt.expectDirection,
			).Return(tt.responseExternalMedia, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_processV1ExternalMediasIDGet(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		expectID uuid.UUID

		responseExternalMedia *externalmedia.ExternalMedia
		expectRes             *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal type connect",
			&rabbitmqhandler.Request{
				URI:    "/v1/external-medias/86d29aa4-97b3-11ed-a086-eb62e01c6736",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			uuid.FromStringOrNil("86d29aa4-97b3-11ed-a086-eb62e01c6736"),

			&externalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("86d29aa4-97b3-11ed-a086-eb62e01c6736"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"86d29aa4-97b3-11ed-a086-eb62e01c6736","asterisk_id":"","channel_id":"","reference_typee":"","reference_id":"00000000-0000-0000-0000-000000000000","local_ip":"","local_port":0,"external_host":"","encapsulation":"","transport":"","connection_type":"","format":"","direction":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &listenHandler{
				rabbitSock:           mockSock,
				externalMediaHandler: mockExternal,
			}

			mockExternal.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.responseExternalMedia, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1ExternalMediasIDDelete(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		expectID uuid.UUID

		responseExternalMedia *externalmedia.ExternalMedia
		expectRes             *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal type connect",
			&rabbitmqhandler.Request{
				URI:    "/v1/external-medias/bbfd5cdc-97b3-11ed-8caa-e705f8c7d343",
				Method: rabbitmqhandler.RequestMethodDelete,
			},

			uuid.FromStringOrNil("bbfd5cdc-97b3-11ed-8caa-e705f8c7d343"),

			&externalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("bbfd5cdc-97b3-11ed-8caa-e705f8c7d343"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bbfd5cdc-97b3-11ed-8caa-e705f8c7d343","asterisk_id":"","channel_id":"","reference_typee":"","reference_id":"00000000-0000-0000-0000-000000000000","local_ip":"","local_port":0,"external_host":"","encapsulation":"","transport":"","connection_type":"","format":"","direction":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &listenHandler{
				rabbitSock:           mockSock,
				externalMediaHandler: mockExternal,
			}

			mockExternal.EXPECT().Stop(gomock.Any(), tt.expectID).Return(tt.responseExternalMedia, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
