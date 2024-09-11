package listenhandler

import (
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/storagehandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_v1FilesPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID    uuid.UUID
		ownerID       uuid.UUID
		referenceType file.ReferenceType
		referenceID   uuid.UUID
		fileName      string
		detail        string
		filename      string
		bucketName    string
		filepath      string

		responseFile *file.File
		expectRes    *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/files",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"4d85dc7a-153e-11ef-9221-13c46bd56c4c", "owner_id":"4dc51b42-153e-11ef-94b6-63fbe2cffaae", "reference_type":"recording", "reference_id":"4df207d8-153e-11ef-8e6d-9fc4e34455ba","name":"test","detail":"test detail","filename":"test_filename.txt","bucket_name":"test_bucket","filepath":"test/file/path"}`),
			},

			customerID:    uuid.FromStringOrNil("4d85dc7a-153e-11ef-9221-13c46bd56c4c"),
			ownerID:       uuid.FromStringOrNil("4dc51b42-153e-11ef-94b6-63fbe2cffaae"),
			referenceType: file.ReferenceTypeRecording,
			referenceID:   uuid.FromStringOrNil("4df207d8-153e-11ef-8e6d-9fc4e34455ba"),
			fileName:      "test",
			detail:        "test detail",
			filename:      "test_filename.txt",
			bucketName:    "test_bucket",
			filepath:      "test/file/path",

			responseFile: &file.File{
				ID: uuid.FromStringOrNil("9de3d544-1739-11ef-acf1-e7fe99b5d7d0"),
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9de3d544-1739-11ef-acf1-e7fe99b5d7d0","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","bucket_name":"","filename":"","filepath":"","filesize":0,"uri_bucket":"","uri_download":"","tm_download_expire":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockStorage := storagehandler.NewMockStorageHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				storageHandler: mockStorage,
			}

			mockStorage.EXPECT().FileCreate(gomock.Any(), tt.customerID, tt.ownerID, tt.referenceType, tt.referenceID, tt.fileName, tt.detail, tt.filename, tt.bucketName, tt.filepath).Return(tt.responseFile, nil)

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

func Test_v1FilesGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		pageToken string
		pageSize  uint64

		responseFilters map[string]string
		responseFiles   []*file.File

		expectRes *sock.Response
	}{
		{
			"1 item",
			&sock.Request{
				URI:      "/v1/files?page_token=2020-10-10T03:30:17.000000&page_size=10&filter_customer_id=bd47c576-15ea-11ef-93f4-7b6a665b785d&filter_deleted=false",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			"2020-10-10T03:30:17.000000",
			10,
			map[string]string{
				"customer_id": "bd47c576-15ea-11ef-93f4-7b6a665b785d",
				"deleted":     "false",
			},

			[]*file.File{
				{
					ID:         uuid.FromStringOrNil("bec1be20-15ea-11ef-ab62-ab3b98e4ee3c"),
					CustomerID: uuid.FromStringOrNil("bd47c576-15ea-11ef-93f4-7b6a665b785d")},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bec1be20-15ea-11ef-ab62-ab3b98e4ee3c","customer_id":"bd47c576-15ea-11ef-93f4-7b6a665b785d","account_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","bucket_name":"","filename":"","filepath":"","filesize":0,"uri_bucket":"","uri_download":"","tm_download_expire":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockStorage := storagehandler.NewMockStorageHandler(mc)
			h := &listenHandler{
				utilHandler:    mockUtil,
				sockHandler:    mockSock,
				storageHandler: mockStorage,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockStorage.EXPECT().FileGets(gomock.Any(), tt.pageToken, tt.pageSize, tt.responseFilters).Return(tt.responseFiles, nil)

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

func Test_v1FilesIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseFile *file.File
		expectRes    *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/files/2a5db58a-15eb-11ef-b669-bba0fb7a717d",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     nil,
			},

			&file.File{
				ID: uuid.FromStringOrNil("2a5db58a-15eb-11ef-b669-bba0fb7a717d"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2a5db58a-15eb-11ef-b669-bba0fb7a717d","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","bucket_name":"","filename":"","filepath":"","filesize":0,"uri_bucket":"","uri_download":"","tm_download_expire":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockStorage := storagehandler.NewMockStorageHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				storageHandler: mockStorage,
			}

			mockStorage.EXPECT().FileGet(gomock.Any(), tt.responseFile.ID).Return(tt.responseFile, nil)

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

func Test_v1FilesIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request
		fileID  uuid.UUID

		responseFile *file.File
		expectRes    *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/files/97a4e91a-15eb-11ef-bf44-eb05a9976a61",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     nil,
			},
			uuid.FromStringOrNil("97a4e91a-15eb-11ef-bf44-eb05a9976a61"),

			&file.File{
				ID: uuid.FromStringOrNil("97a4e91a-15eb-11ef-bf44-eb05a9976a61"),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"97a4e91a-15eb-11ef-bf44-eb05a9976a61","customer_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","bucket_name":"","filename":"","filepath":"","filesize":0,"uri_bucket":"","uri_download":"","tm_download_expire":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockStorage := storagehandler.NewMockStorageHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				storageHandler: mockStorage,
			}

			mockStorage.EXPECT().FileDelete(gomock.Any(), tt.fileID).Return(tt.responseFile, nil)

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
