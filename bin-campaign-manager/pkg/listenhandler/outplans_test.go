package listenhandler

import (
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
)

func Test_v1OutplansPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID   uuid.UUID
		outplanName  string
		detail       string
		source       *commonaddress.Address
		dialTimeout  int
		tryInterval  int
		maxTryCount0 int
		maxTryCount1 int
		maxTryCount2 int
		maxTryCount3 int
		maxTryCount4 int

		responseOutplan *outplan.Outplan

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outplans",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"99f40c64-c462-11ec-b1e4-57c64041ff4e","name":"test name","detail":"test detail","source":{"type":"tel","target":"+821100000001"},"dial_timeout":30000,"try_interval":600000,"max_try_count_0":5,"max_try_count_1":5,"max_try_count_2":5,"max_try_count_3":5,"max_try_count_4":5}`),
			},

			uuid.FromStringOrNil("99f40c64-c462-11ec-b1e4-57c64041ff4e"),
			"test name",
			"test detail",
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			30000,
			600000,
			5,
			5,
			5,
			5,
			5,

			&outplan.Outplan{
				ID: uuid.FromStringOrNil("3ca9708e-c463-11ec-95e7-53d07a0e8bb2"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3ca9708e-c463-11ec-95e7-53d07a0e8bb2","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				outplanHandler: mockOutplan,
			}

			mockOutplan.EXPECT().Create(gomock.Any(), tt.customerID, tt.outplanName, tt.detail, tt.source, tt.dialTimeout, tt.tryInterval, tt.maxTryCount0, tt.maxTryCount1, tt.maxTryCount2, tt.maxTryCount3, tt.maxTryCount4).Return(tt.responseOutplan, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutplansGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		pageToken  string
		pageSize   uint64
		customerID uuid.UUID

		responseOutplans []*outplan.Outplan

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outplans?page_token=2020-10-10%2003:30:17.000000&page_size=10&customer_id=4890c13c-c467-11ec-add3-77ebc79554b0",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			"2020-10-10 03:30:17.000000",
			10,
			uuid.FromStringOrNil("4890c13c-c467-11ec-add3-77ebc79554b0"),

			[]*outplan.Outplan{
				{
					ID: uuid.FromStringOrNil("4d100e66-c467-11ec-b6c1-eb8c7f0b8f24"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"4d100e66-c467-11ec-b6c1-eb8c7f0b8f24","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				outplanHandler: mockOutplan,
			}

			mockOutplan.EXPECT().GetsByCustomerID(gomock.Any(), tt.customerID, tt.pageToken, tt.pageSize).Return(tt.responseOutplans, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutplansIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		campaignID uuid.UUID

		responseOutplan *outplan.Outplan

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outplans/a96cd4d2-c467-11ec-b1b4-0f35d821b46c",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("a96cd4d2-c467-11ec-b1b4-0f35d821b46c"),

			&outplan.Outplan{
				ID: uuid.FromStringOrNil("a96cd4d2-c467-11ec-b1b4-0f35d821b46c"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a96cd4d2-c467-11ec-b1b4-0f35d821b46c","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				outplanHandler: mockOutplan,
			}

			mockOutplan.EXPECT().Get(gomock.Any(), tt.campaignID).Return(tt.responseOutplan, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutplansIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		campaignID uuid.UUID

		responseOutplan *outplan.Outplan

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outplans/dfce10d6-c467-11ec-b1d3-0fb0c67e2f11",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("dfce10d6-c467-11ec-b1d3-0fb0c67e2f11"),

			&outplan.Outplan{
				ID: uuid.FromStringOrNil("dfce10d6-c467-11ec-b1d3-0fb0c67e2f11"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"dfce10d6-c467-11ec-b1d3-0fb0c67e2f11","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				outplanHandler: mockOutplan,
			}

			mockOutplan.EXPECT().Delete(gomock.Any(), tt.campaignID).Return(tt.responseOutplan, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutplansIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		outplanID   uuid.UUID
		outplanName string
		detail      string

		responseOutplan *outplan.Outplan

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outplans/1819e97e-c468-11ec-bf3c-63e7e7c996db",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name","detail":"update detail"}`),
			},

			uuid.FromStringOrNil("1819e97e-c468-11ec-bf3c-63e7e7c996db"),
			"update name",
			"update detail",

			&outplan.Outplan{
				ID: uuid.FromStringOrNil("1819e97e-c468-11ec-bf3c-63e7e7c996db"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1819e97e-c468-11ec-bf3c-63e7e7c996db","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				outplanHandler: mockOutplan,
			}

			mockOutplan.EXPECT().UpdateBasicInfo(gomock.Any(), tt.outplanID, tt.outplanName, tt.detail).Return(tt.responseOutplan, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutplansIDDialsPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		outplanID    uuid.UUID
		source       *commonaddress.Address
		dialTimeout  int
		tryInterval  int
		maxTryCount0 int
		maxTryCount1 int
		maxTryCount2 int
		maxTryCount3 int
		maxTryCount4 int

		responseOutplan *outplan.Outplan

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outplans/703adb5e-c468-11ec-b8ff-f3c00713cce4/dials",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"source":{"type":"tel","target":"+821100000001"},"dial_timeout":30000,"try_interval":600000,"max_try_count_0":5,"max_try_count_1":5,"max_try_count_2":5,"max_try_count_3":5,"max_try_count_4":5}`),
			},

			uuid.FromStringOrNil("703adb5e-c468-11ec-b8ff-f3c00713cce4"),
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			30000,
			600000,
			5,
			5,
			5,
			5,
			5,

			&outplan.Outplan{
				ID: uuid.FromStringOrNil("703adb5e-c468-11ec-b8ff-f3c00713cce4"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"703adb5e-c468-11ec-b8ff-f3c00713cce4","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","source":null,"dial_timeout":0,"try_interval":0,"max_try_count_0":0,"max_try_count_1":0,"max_try_count_2":0,"max_try_count_3":0,"max_try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				outplanHandler: mockOutplan,
			}

			mockOutplan.EXPECT().UpdateDialInfo(gomock.Any(), tt.outplanID, tt.source, tt.dialTimeout, tt.tryInterval, tt.maxTryCount0, tt.maxTryCount1, tt.maxTryCount2, tt.maxTryCount3, tt.maxTryCount4).Return(tt.responseOutplan, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
