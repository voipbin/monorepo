package listenhandler

import (
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
)

func Test_processV1ExternalMediasPost(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		expectID             uuid.UUID
		expectReferenceType  externalmedia.ReferenceType
		expectReferenceID    uuid.UUID
		expectNoInsertMedia  bool
		expectExternalHost   string
		expectEncapsulation  externalmedia.Encapsulation
		expectTransport      externalmedia.Transport
		expectConnectionType string
		expectFormat         string
		expectDirection      string

		responseExternalMedia *externalmedia.ExternalMedia
		expectRes             *sock.Response
	}

	tests := []test{
		{
			"normal type connect",
			&sock.Request{
				URI:      "/v1/external-medias",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"077a8dce-b332-11ef-a775-d39c4839f5a6","reference_type":"call","reference_id":"45832182-97b2-11ed-8f17-33590535a404","no_insert_media":false,"external_host":"127.0.0.1:8080","encapsulation":"rtp","transport":"udp","connection_type":"client","format":"ulaw","direction":"both"}`),
			},

			uuid.FromStringOrNil("077a8dce-b332-11ef-a775-d39c4839f5a6"),
			externalmedia.ReferenceTypeCall,
			uuid.FromStringOrNil("45832182-97b2-11ed-8f17-33590535a404"),
			false,
			"127.0.0.1:8080",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",

			&externalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("077a8dce-b332-11ef-a775-d39c4839f5a6"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"077a8dce-b332-11ef-a775-d39c4839f5a6","asterisk_id":"","channel_id":"","reference_typee":"","reference_id":"00000000-0000-0000-0000-000000000000","local_ip":"","local_port":0,"external_host":"","encapsulation":"","transport":"","connection_type":"","format":"","direction":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				externalMediaHandler: mockExternal,
			}

			mockExternal.EXPECT().Start(
				gomock.Any(),
				tt.expectID,
				tt.expectReferenceType,
				tt.expectReferenceID,
				tt.expectNoInsertMedia,
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

func Test_processV1ExternalMediasGet(t *testing.T) {
	tests := []struct {
		name string

		request   *sock.Request
		pageSize  uint64
		pageToken string

		responseExternalMedias []*externalmedia.ExternalMedia
		responseFilters        map[string]string
		expectRes              *sock.Response
	}{
		{
			"normal",

			&sock.Request{
				URI:    "/v1/external-medias?page_size=10&page_token=2020-05-03%2021:35:02.809&reference_id=0971d7c4-e829-11ee-a17d-b320c527e478",
				Method: sock.RequestMethodGet,
			},
			10,
			"2020-05-03 21:35:02.809",

			[]*externalmedia.ExternalMedia{
				{
					ID: uuid.FromStringOrNil("28db3628-e829-11ee-a39e-83e2f12ec29f"),
				},
			},
			map[string]string{
				"reference_id": "0971d7c4-e829-11ee-a17d-b320c527e478",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"28db3628-e829-11ee-a39e-83e2f12ec29f","asterisk_id":"","channel_id":"","reference_typee":"","reference_id":"00000000-0000-0000-0000-000000000000","local_ip":"","local_port":0,"external_host":"","encapsulation":"","transport":"","connection_type":"","format":"","direction":""}]`),
			},
		},
		{
			"2 items",

			&sock.Request{
				URI:    "/v1/external-medias?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_reference_id=98d20344-e829-11ee-992d-fbe3942f7a49",
				Method: sock.RequestMethodGet,
			},
			10,
			"2020-05-03 21:35:02.809",

			[]*externalmedia.ExternalMedia{
				{
					ID: uuid.FromStringOrNil("98fda9f4-e829-11ee-83bd-233ee47d5cb3"),
				},
				{
					ID: uuid.FromStringOrNil("992a4cca-e829-11ee-83e5-4bc5ace56a63"),
				},
			},
			map[string]string{
				"reference_id": "98d20344-e829-11ee-992d-fbe3942f7a49",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"98fda9f4-e829-11ee-83bd-233ee47d5cb3","asterisk_id":"","channel_id":"","reference_typee":"","reference_id":"00000000-0000-0000-0000-000000000000","local_ip":"","local_port":0,"external_host":"","encapsulation":"","transport":"","connection_type":"","format":"","direction":""},{"id":"992a4cca-e829-11ee-83e5-4bc5ace56a63","asterisk_id":"","channel_id":"","reference_typee":"","reference_id":"00000000-0000-0000-0000-000000000000","local_ip":"","local_port":0,"external_host":"","encapsulation":"","transport":"","connection_type":"","format":"","direction":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockExternalMedia := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &listenHandler{
				utilHandler:          mockUtil,
				sockHandler:          mockSock,
				callHandler:          mockCall,
				externalMediaHandler: mockExternalMedia,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockExternalMedia.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.responseFilters).Return(tt.responseExternalMedias, nil)
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
		request *sock.Request

		expectID uuid.UUID

		responseExternalMedia *externalmedia.ExternalMedia
		expectRes             *sock.Response
	}

	tests := []test{
		{
			"normal type connect",
			&sock.Request{
				URI:    "/v1/external-medias/86d29aa4-97b3-11ed-a086-eb62e01c6736",
				Method: sock.RequestMethodGet,
			},

			uuid.FromStringOrNil("86d29aa4-97b3-11ed-a086-eb62e01c6736"),

			&externalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("86d29aa4-97b3-11ed-a086-eb62e01c6736"),
			},
			&sock.Response{
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
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
		request *sock.Request

		expectID uuid.UUID

		responseExternalMedia *externalmedia.ExternalMedia
		expectRes             *sock.Response
	}

	tests := []test{
		{
			"normal type connect",
			&sock.Request{
				URI:    "/v1/external-medias/bbfd5cdc-97b3-11ed-8caa-e705f8c7d343",
				Method: sock.RequestMethodDelete,
			},

			uuid.FromStringOrNil("bbfd5cdc-97b3-11ed-8caa-e705f8c7d343"),

			&externalmedia.ExternalMedia{
				ID: uuid.FromStringOrNil("bbfd5cdc-97b3-11ed-8caa-e705f8c7d343"),
			},
			&sock.Response{
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
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
