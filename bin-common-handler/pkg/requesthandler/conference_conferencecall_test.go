package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_ConferenceV1ConferencecallGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []cfconferencecall.Conferencecall
	}{
		{
			"normal",

			"2021-03-02 03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"/v1/conferencecalls?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10",
			"bin-manager.conference-manager.request",
			&sock.Request{
				URI:    "/v1/conferencecalls?page_token=2021-03-02+03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"99d4af42-50c8-11ee-8240-d360bb85c265"},{"id":"9a0b7f2c-50c8-11ee-a3e8-b7c427a82ef8"}]`),
			},

			[]cfconferencecall.Conferencecall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("99d4af42-50c8-11ee-8240-d360bb85c265"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("9a0b7f2c-50c8-11ee-a3e8-b7c427a82ef8"),
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			reqHandler := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.ConferenceV1ConferencecallGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferenceV1ConferencecallGet(t *testing.T) {

	type test struct {
		name             string
		conferencecallID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *cfconferencecall.Conferencecall
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("7baaa99e-14e8-11ed-8f79-f79014b94b6f"),

			"bin-manager.conference-manager.request",
			&sock.Request{
				URI:      "/v1/conferencecalls/7baaa99e-14e8-11ed-8f79-f79014b94b6f",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"7baaa99e-14e8-11ed-8f79-f79014b94b6f"}`),
			},
			&cfconferencecall.Conferencecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7baaa99e-14e8-11ed-8f79-f79014b94b6f"),
				},
			},
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	reqHandler := requestHandler{
		sock: mockSock,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

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

func Test_ConferenceV1ConferencecallKick(t *testing.T) {

	tests := []struct {
		name string

		conferencecallID uuid.UUID
		response         *sock.Response

		expectTarget  string
		expectRequest *sock.Request

		expectRes *cfconferencecall.Conferencecall
	}{
		{
			"normal",
			uuid.FromStringOrNil("dd4ff2e2-14e5-11ed-8eec-97413dd96f29"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"dd4ff2e2-14e5-11ed-8eec-97413dd96f29"}`),
			},

			"bin-manager.conference-manager.request",
			&sock.Request{
				URI:      "/v1/conferencecalls/dd4ff2e2-14e5-11ed-8eec-97413dd96f29",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			&cfconferencecall.Conferencecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dd4ff2e2-14e5-11ed-8eec-97413dd96f29"),
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
		delay            int

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request

		expectRes *cfconferencecall.Conferencecall
	}{
		{
			"normal",
			uuid.FromStringOrNil("23d64db6-94a6-11ed-9b9f-2bfedef352c1"),
			2,
			5000,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"23d64db6-94a6-11ed-9b9f-2bfedef352c1"}`),
			},

			"bin-manager.conference-manager.request",
			&sock.Request{
				URI:      "/v1/conferencecalls/23d64db6-94a6-11ed-9b9f-2bfedef352c1/health-check",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"retry_count":2}`),
			},

			&cfconferencecall.Conferencecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("23d64db6-94a6-11ed-9b9f-2bfedef352c1"),
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
			mockSock.EXPECT().RequestPublishWithDelay(
				tt.expectTarget,
				tt.expectRequest,
				tt.delay,
			).Return(nil)

			if err := reqHandler.ConferenceV1ConferencecallHealthCheck(ctx, tt.conferencecallID, tt.retryCount, tt.delay); err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}
		})
	}
}
