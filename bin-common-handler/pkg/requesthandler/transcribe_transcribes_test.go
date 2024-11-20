package requesthandler

import (
	"context"
	"reflect"
	"testing"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_TranscribeV1TranscribeGet(t *testing.T) {

	type test struct {
		name string

		transcribeID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult *tmtranscribe.Transcribe
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("32b71878-8093-11ed-8578-775276ea57cf"),

			"bin-manager.transcribe-manager.request",
			&sock.Request{
				URI:      "/v1/transcribes/32b71878-8093-11ed-8578-775276ea57cf",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"32b71878-8093-11ed-8578-775276ea57cf"}`),
			},
			&tmtranscribe.Transcribe{
				ID: uuid.FromStringOrNil("32b71878-8093-11ed-8578-775276ea57cf"),
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

			res, err := reqHandler.TranscribeV1TranscribeGet(ctx, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectResult, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_TranscribeV1TranscribeGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     []tmtranscribe.Transcribe
	}{
		{
			"1 item",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"customer_id": "adddce70-8093-11ed-9a79-530f80f428d8",
			},

			"/v1/transcribes?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.transcribe-manager.request",
			&sock.Request{
				URI:      "/v1/transcribes?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_customer_id=adddce70-8093-11ed-9a79-530f80f428d8",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ae0a7cfe-8093-11ed-963d-abb334c8e6d8"}]`),
			},
			[]tmtranscribe.Transcribe{
				{
					ID: uuid.FromStringOrNil("ae0a7cfe-8093-11ed-963d-abb334c8e6d8"),
				},
			},
		},
		{
			"2 items",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"customer_id": "bb3c9146-8093-11ed-a0df-6fbf1a76cbd3",
			},

			"/v1/transcribes?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.transcribe-manager.request",
			&sock.Request{
				URI:      "/v1/transcribes?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_customer_id=bb3c9146-8093-11ed-a0df-6fbf1a76cbd3",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bb6c13bc-8093-11ed-b647-5f3b613e1180"},{"id":"bb8fc46a-8093-11ed-9ea7-9304ab751b40"}]`),
			},
			[]tmtranscribe.Transcribe{
				{
					ID: uuid.FromStringOrNil("bb6c13bc-8093-11ed-b647-5f3b613e1180"),
				},
				{
					ID: uuid.FromStringOrNil("bb8fc46a-8093-11ed-9ea7-9304ab751b40"),
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
			h := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.TranscribeV1TranscribeGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TranscribeV1TranscribeStart(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		referenceType tmtranscribe.ReferenceType
		referenceID   uuid.UUID
		language      string
		direction     tmtranscribe.Direction

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *tmtranscribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("2ab9c63a-8227-11ed-928b-1b90501adbe2"),
			tmtranscribe.ReferenceTypeCall,
			uuid.FromStringOrNil("2ae8944c-8227-11ed-acb4-c3e23ea3a2a4"),
			"en-US",
			tmtranscribe.DirectionBoth,

			"bin-manager.transcribe-manager.request",
			&sock.Request{
				URI:      "/v1/transcribes",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"2ab9c63a-8227-11ed-928b-1b90501adbe2","reference_type":"call","reference_id":"2ae8944c-8227-11ed-acb4-c3e23ea3a2a4","language":"en-US","direction":"both"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2b13a4ca-8227-11ed-8bad-b7bb9aa7f185"}`),
			},
			&tmtranscribe.Transcribe{
				ID: uuid.FromStringOrNil("2b13a4ca-8227-11ed-8bad-b7bb9aa7f185"),
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

			res, err := reqHandler.TranscribeV1TranscribeStart(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.language, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TranscribeV1TranscribeStop(t *testing.T) {

	tests := []struct {
		name string

		transcribeID uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes *tmtranscribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("2622b04a-8228-11ed-98f0-6bfc284cdb95"),

			"bin-manager.transcribe-manager.request",
			&sock.Request{
				URI:      "/v1/transcribes/2622b04a-8228-11ed-98f0-6bfc284cdb95/stop",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2622b04a-8228-11ed-98f0-6bfc284cdb95"}`),
			},
			&tmtranscribe.Transcribe{
				ID: uuid.FromStringOrNil("2622b04a-8228-11ed-98f0-6bfc284cdb95"),
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

			res, err := reqHandler.TranscribeV1TranscribeStop(ctx, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TranscribeV1TranscribeHealthCheck(t *testing.T) {

	tests := []struct {
		name string

		transcribeID uuid.UUID
		delay        int
		retryCount   int

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("273d1fa4-e9ac-46cc-920e-34e163eb0e73"),
			0,
			3,

			"bin-manager.transcribe-manager.request",
			&sock.Request{
				URI:      "/v1/transcribes/273d1fa4-e9ac-46cc-920e-34e163eb0e73/health-check",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"retry_count":3}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
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

			err := reqHandler.TranscribeV1TranscribeHealthCheck(ctx, tt.transcribeID, tt.delay, tt.retryCount)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
