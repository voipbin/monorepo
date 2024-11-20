package listenhandler

import (
	"reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/models/conferencecall"
	"monorepo/bin-conference-manager/pkg/conferencecallhandler"
	"monorepo/bin-conference-manager/pkg/conferencehandler"
)

func Test_processV1ConferencecallsGet(t *testing.T) {

	tests := []struct {
		name string

		request   *sock.Request
		pageSize  uint64
		pageToken string

		responseFilters     map[string]string
		responseConferences []*conferencecall.Conferencecall
		expectRes           *sock.Response
	}{
		{
			"normal",

			&sock.Request{
				URI:    "/v1/conferencecalls?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_customer_id=54197ee2-50c3-11ee-ba48-af437ce87cbf&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},
			10,
			"2020-05-03 21:35:02.809",

			map[string]string{
				"customer_id": "54197ee2-50c3-11ee-ba48-af437ce87cbf",
				"deleted":     "false",
			},

			[]*conferencecall.Conferencecall{
				{
					ID: uuid.FromStringOrNil("544b3fea-50c3-11ee-86bb-6fe1c82ac8b3"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"544b3fea-50c3-11ee-86bb-6fe1c82ac8b3","customer_id":"00000000-0000-0000-0000-000000000000","conference_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := &listenHandler{
				utilHandler:           mockUtil,
				sockHandler:           mockSock,
				conferencecallHandler: mockConf,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockConf.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.responseFilters).Return(tt.responseConferences, nil)
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

func Test_processV1ConferencecallsIDGet(t *testing.T) {

	tests := []struct {
		name             string
		request          *sock.Request
		conferencecallID uuid.UUID

		responseConferencecall *conferencecall.Conferencecall

		expectRes *sock.Response
	}{
		{
			"normal",

			&sock.Request{
				URI:    "/v1/conferencecalls/1015da76-14cc-11ed-b156-5b7904da0071",
				Method: sock.RequestMethodGet,
			},
			uuid.FromStringOrNil("1015da76-14cc-11ed-b156-5b7904da0071"),

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("1015da76-14cc-11ed-b156-5b7904da0071"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1015da76-14cc-11ed-b156-5b7904da0071","customer_id":"00000000-0000-0000-0000-000000000000","conference_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)
			mockConferencecall := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := &listenHandler{
				utilHandler:           mockUtil,
				sockHandler:           mockSock,
				conferenceHandler:     mockConf,
				conferencecallHandler: mockConferencecall,
			}

			mockConferencecall.EXPECT().Get(gomock.Any(), tt.conferencecallID).Return(tt.responseConferencecall, nil)
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

func Test_processV1ConferencecallsIDDelete(t *testing.T) {

	tests := []struct {
		name             string
		request          *sock.Request
		conferencecallID uuid.UUID

		responseConferencecall *conferencecall.Conferencecall

		expectRes *sock.Response
	}{
		{
			"normal",

			&sock.Request{
				URI:    "/v1/conferencecalls/8a1fd900-3bf3-11ec-bd15-eb0c54c84612",
				Method: sock.RequestMethodDelete,
			},
			uuid.FromStringOrNil("8a1fd900-3bf3-11ec-bd15-eb0c54c84612"),

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("8a1fd900-3bf3-11ec-bd15-eb0c54c84612"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8a1fd900-3bf3-11ec-bd15-eb0c54c84612","customer_id":"00000000-0000-0000-0000-000000000000","conference_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConfcall := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := &listenHandler{
				sockHandler:           mockSock,
				conferencecallHandler: mockConfcall,
			}

			mockConfcall.EXPECT().Terminate(gomock.Any(), tt.conferencecallID).Return(tt.responseConferencecall, nil)
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

func Test_processV1ConferencecallsIDHealthCheckPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		expectConferencecallID uuid.UUID
		expectRetyCount        int
		expectDelay            int
		expectRes              *sock.Response
	}{
		{
			"normal",

			&sock.Request{
				URI:      "/v1/conferencecalls/14fb5cf8-94a3-11ed-8a92-2b5c1e7d925b/health-check",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"retry_count": 2}`),
			},

			uuid.FromStringOrNil("14fb5cf8-94a3-11ed-8a92-2b5c1e7d925b"),
			2,
			5000,
			&sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockConfcall := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := &listenHandler{
				sockHandler:           mockSock,
				conferencecallHandler: mockConfcall,
			}

			mockConfcall.EXPECT().HealthCheck(gomock.Any(), tt.expectConferencecallID, tt.expectRetyCount)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
