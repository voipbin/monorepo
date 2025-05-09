package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_CallV1ExternalMediaGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     []cmexternalmedia.ExternalMedia
	}{
		{
			"normal",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"reference_id": "6ddd7aa8-e82c-11ee-9ae3-23cca4c32454",
			},

			"/v1/external-medias?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.call-manager.request",
			&sock.Request{
				URI:    "/v1/external-medias?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_reference_id=6ddd7aa8-e82c-11ee-9ae3-23cca4c32454",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"7e4a0f64-e82c-11ee-8e4f-cf15aa8ffd9e"}]`),
			},
			[]cmexternalmedia.ExternalMedia{
				{
					ID: uuid.FromStringOrNil("7e4a0f64-e82c-11ee-8e4f-cf15aa8ffd9e"),
				},
			},
		},
		{
			"2 results",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"reference_id": "a188209c-e82c-11ee-9a12-2f13b7edeb5f",
			},

			"/v1/external-medias?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.call-manager.request",
			&sock.Request{
				URI:    "/v1/external-medias?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_reference_id=a188209c-e82c-11ee-9a12-2f13b7edeb5f",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"a1ef1b08-e82c-11ee-bf49-e7fd70d542c4"},{"id":"a21fb5a6-e82c-11ee-8d35-d7e4c2dfa582"}]`),
			},
			[]cmexternalmedia.ExternalMedia{
				{
					ID: uuid.FromStringOrNil("a1ef1b08-e82c-11ee-bf49-e7fd70d542c4"),
				},
				{
					ID: uuid.FromStringOrNil("a21fb5a6-e82c-11ee-8d35-d7e4c2dfa582"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			reqHandler := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1ExternalMediaGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1ExternalMediaStart(t *testing.T) {

	tests := []struct {
		name string

		externalMediaID uuid.UUID
		referenceType   cmexternalmedia.ReferenceType
		referenceID     uuid.UUID
		noInsertMedia   bool
		externalHost    string
		encapsulation   string
		transport       string
		connectionType  string
		format          string
		direction       string

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *cmexternalmedia.ExternalMedia
	}{
		{
			"normal",

			uuid.FromStringOrNil("7f655194-b336-11ef-ad61-e340f855ae0d"),
			cmexternalmedia.ReferenceTypeCall,
			uuid.FromStringOrNil("94a6ec48-97c2-11ed-bd66-afb196d5c598"),
			true,
			"localhost:5060",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e8337d9a-97c2-11ed-93ad-5bcba5332622"}`),
			},

			&sock.Request{
				URI:      "/v1/external-medias",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"7f655194-b336-11ef-ad61-e340f855ae0d","reference_type":"call","reference_id":"94a6ec48-97c2-11ed-bd66-afb196d5c598","no_insert_media":true,"external_host":"localhost:5060","encapsulation":"rtp","transport":"udp","connection_type":"client","format":"ulaw","direction":"both"}`),
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1ExternalMediaStart(ctx, tt.externalMediaID, tt.referenceType, tt.referenceID, tt.noInsertMedia, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, tt.direction)
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
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *cmexternalmedia.ExternalMedia
	}{
		{
			"normal",

			uuid.FromStringOrNil("0a90d6b2-97c3-11ed-a114-3b7fc4677d36"),

			"bin-manager.call-manager.request",
			&sock.Request{
				URI:    "/v1/external-medias/0a90d6b2-97c3-11ed-a114-3b7fc4677d36",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *cmexternalmedia.ExternalMedia
	}{
		{
			"normal",

			uuid.FromStringOrNil("2f4d5390-97c3-11ed-9dc6-47ce92d2b198"),

			"bin-manager.call-manager.request",
			&sock.Request{
				URI:    "/v1/external-medias/2f4d5390-97c3-11ed-9dc6-47ce92d2b198",
				Method: sock.RequestMethodDelete,
			},
			&sock.Response{
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

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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
