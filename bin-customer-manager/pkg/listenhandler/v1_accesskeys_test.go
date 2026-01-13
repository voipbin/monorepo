package listenhandler

import (
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/pkg/accesskeyhandler"
	reflect "reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_processV1AccesskeysGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request
		size    uint64
		token   string

		responseFilters    map[string]string
		expectFilters      map[accesskey.Field]any
		responseAccesskeys []*accesskey.Accesskey
		expectRes          *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/accesskeys?page_size=10&page_token=2021-11-23%2017:55:39.712000",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			10,
			"2021-11-23 17:55:39.712000",

			map[string]string{
				"deleted": "false",
			},
			map[accesskey.Field]any{
				accesskey.FieldDeleted: false,
			},
			[]*accesskey.Accesskey{
				{
					ID: uuid.FromStringOrNil("62c8a72c-ab32-11ef-ab05-d7fd85b6e924"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"62c8a72c-ab32-11ef-ab05-d7fd85b6e924","customer_id":"00000000-0000-0000-0000-000000000000","token":""}]`),
			},
		},
		{
			"2 accesskeys",
			&sock.Request{
				URI:      "/v1/accesskeys?page_size=10&page_token=2021-11-23%2017:55:39.712000",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			10,
			"2021-11-23 17:55:39.712000",

			map[string]string{
				"deleted": "false",
			},
			map[accesskey.Field]any{
				accesskey.FieldDeleted: false,
			},
			[]*accesskey.Accesskey{
				{
					ID: uuid.FromStringOrNil("635ae524-ab32-11ef-a2be-139c3573025a"),
				},
				{
					ID: uuid.FromStringOrNil("6386db7a-ab32-11ef-907a-5bb0d3b3ee89"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"635ae524-ab32-11ef-a2be-139c3573025a","customer_id":"00000000-0000-0000-0000-000000000000","token":""},{"id":"6386db7a-ab32-11ef-907a-5bb0d3b3ee89","customer_id":"00000000-0000-0000-0000-000000000000","token":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				reqHandler:       mockReq,
				utilHandler:      mockUtil,
				accesskeyHandler: mockAccesskey,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockAccesskey.EXPECT().Gets(gomock.Any(), tt.size, tt.token, gomock.Any()).Return(tt.responseAccesskeys, nil)

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

func Test_processV1AccesskeysPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAccesskey *accesskey.Accesskey

		expectedCustomerID uuid.UUID
		expectedName       string
		expectedDetail     string
		expectedExpire     time.Duration
		expectedRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/accesskeys",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"0324e804-ab36-11ef-8a73-971d5f0ad5ee", "name": "name test", "detail": "detail test", "expire": 86400000}`),
			},

			responseAccesskey: &accesskey.Accesskey{
				ID: uuid.FromStringOrNil("bd461f74-ab35-11ef-80dd-67cc72d376f4"),
			},

			expectedCustomerID: uuid.FromStringOrNil("0324e804-ab36-11ef-8a73-971d5f0ad5ee"),
			expectedName:       "name test",
			expectedDetail:     "detail test",
			expectedExpire:     time.Second * 86400000,
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"bd461f74-ab35-11ef-80dd-67cc72d376f4","customer_id":"00000000-0000-0000-0000-000000000000","token":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAcesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				reqHandler:       mockReq,
				accesskeyHandler: mockAcesskey,
			}

			mockAcesskey.EXPECT().Create(
				gomock.Any(),
				tt.expectedCustomerID,
				tt.expectedName,
				tt.expectedDetail,
				tt.expectedExpire,
			).Return(tt.responseAccesskey, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_processV1AccesskeysIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		id uuid.UUID

		responseAccesskey *accesskey.Accesskey
		expectedRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/accesskeys/89af5ea4-ab36-11ef-b925-0b839d161400",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			id: uuid.FromStringOrNil("89af5ea4-ab36-11ef-b925-0b839d161400"),

			responseAccesskey: &accesskey.Accesskey{
				ID: uuid.FromStringOrNil("89af5ea4-ab36-11ef-b925-0b839d161400"),
			},
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"89af5ea4-ab36-11ef-b925-0b839d161400","customer_id":"00000000-0000-0000-0000-000000000000","token":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				reqHandler:       mockReq,
				accesskeyHandler: mockAccesskey,
			}

			mockAccesskey.EXPECT().Get(gomock.Any(), tt.id).Return(tt.responseAccesskey, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_processV1AccesskeysIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		accesskeyID       uuid.UUID
		responseAccesskey *accesskey.Accesskey

		expectedRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/accesskeys/012ed66c-ab37-11ef-9ea1-b359d0d62fed",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			accesskeyID: uuid.FromStringOrNil("012ed66c-ab37-11ef-9ea1-b359d0d62fed"),
			responseAccesskey: &accesskey.Accesskey{
				ID: uuid.FromStringOrNil("012ed66c-ab37-11ef-9ea1-b359d0d62fed"),
			},

			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"012ed66c-ab37-11ef-9ea1-b359d0d62fed","customer_id":"00000000-0000-0000-0000-000000000000","token":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				reqHandler:       mockReq,
				accesskeyHandler: mockAccesskey,
			}

			mockAccesskey.EXPECT().Delete(gomock.Any(), tt.accesskeyID).Return(tt.responseAccesskey, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_processV1AccesskeysIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseAccesskey *accesskey.Accesskey

		expectedID     uuid.UUID
		expectedName   string
		expectedDetail string
		expectedRes    *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/accesskeys/6c132852-ab37-11ef-870d-578e6cdd7c37",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name", "detail": "update detail"}`),
			},

			responseAccesskey: &accesskey.Accesskey{
				ID: uuid.FromStringOrNil("6c132852-ab37-11ef-870d-578e6cdd7c37"),
			},
			expectedID:     uuid.FromStringOrNil("6c132852-ab37-11ef-870d-578e6cdd7c37"),
			expectedName:   "update name",
			expectedDetail: "update detail",
			expectedRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6c132852-ab37-11ef-870d-578e6cdd7c37","customer_id":"00000000-0000-0000-0000-000000000000","token":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockAccesskey := accesskeyhandler.NewMockAccesskeyHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				reqHandler:       mockReq,
				accesskeyHandler: mockAccesskey,
			}

			mockAccesskey.EXPECT().UpdateBasicInfo(gomock.Any(), tt.expectedID, tt.expectedName, tt.expectedDetail).Return(tt.responseAccesskey, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}
