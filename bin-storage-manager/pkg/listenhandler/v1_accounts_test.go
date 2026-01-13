package listenhandler

import (
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/storagehandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_v1AccountsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID uuid.UUID

		responseAccount *account.Account
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/accounts",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"020a406a-1b38-11ef-87b2-037eaf8e3a2f"}`),
			},

			customerID: uuid.FromStringOrNil("020a406a-1b38-11ef-87b2-037eaf8e3a2f"),

			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("023f0570-1b38-11ef-86b9-57db70737f8f"),
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"023f0570-1b38-11ef-86b9-57db70737f8f","customer_id":"00000000-0000-0000-0000-000000000000","total_file_count":0,"total_file_size":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				accountHandler: mockAccount,
			}

			mockAccount.EXPECT().Create(gomock.Any(), tt.customerID.Return(tt.responseAccount, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1AccountsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		pageToken string
		pageSize  uint64

		expectFilters    map[account.Field]any
		responseAccounts []*account.Account

		expectRes *sock.Response
	}{
		{
			"1 item",
			&sock.Request{
				URI:      "/v1/accounts?page_token=2020-10-10T03:30:17.000000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"deleted":false}`),
			},

			"2020-10-10T03:30:17.000000",
			10,
			map[account.Field]any{
				account.FieldDeleted: false,
			},

			[]*account.Account{
				{
					ID: uuid.FromStringOrNil("d1eec710-1b38-11ef-a425-47a057cb03ee"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d1eec710-1b38-11ef-a425-47a057cb03ee","customer_id":"00000000-0000-0000-0000-000000000000","total_file_count":0,"total_file_size":0,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockStorage := storagehandler.NewMockStorageHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			h := &listenHandler{
				sockHandler:    mockSock,
				storageHandler: mockStorage,
				accountHandler: mockAccount,
			}

			mockAccount.EXPECT().Gets(gomock.Any(), tt.pageToken, tt.pageSize, gomock.Any(.Return(tt.responseAccounts, nil)

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

func Test_v1AccountsIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseFile *account.Account
		expectRes    *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/accounts/0c3d3d16-1b39-11ef-92ec-df76bb16f7c7",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},

			&account.Account{
				ID: uuid.FromStringOrNil("0c3d3d16-1b39-11ef-92ec-df76bb16f7c7"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0c3d3d16-1b39-11ef-92ec-df76bb16f7c7","customer_id":"00000000-0000-0000-0000-000000000000","total_file_count":0,"total_file_size":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockStorage := storagehandler.NewMockStorageHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			h := &listenHandler{
				sockHandler:    mockSock,
				storageHandler: mockStorage,
				accountHandler: mockAccount,
			}

			mockAccount.EXPECT().Get(gomock.Any(), tt.responseFile.ID.Return(tt.responseFile, nil)

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

func Test_v1AccountsIDDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseFile *account.Account
		expectRes    *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/accounts/308ed81e-1b39-11ef-addf-af55fc0db7b7",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},

			&account.Account{
				ID: uuid.FromStringOrNil("308ed81e-1b39-11ef-addf-af55fc0db7b7"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"308ed81e-1b39-11ef-addf-af55fc0db7b7","customer_id":"00000000-0000-0000-0000-000000000000","total_file_count":0,"total_file_size":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockStorage := storagehandler.NewMockStorageHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			h := &listenHandler{
				sockHandler:    mockSock,
				storageHandler: mockStorage,
				accountHandler: mockAccount,
			}

			mockAccount.EXPECT().Delete(gomock.Any(), tt.responseFile.ID.Return(tt.responseFile, nil)

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
