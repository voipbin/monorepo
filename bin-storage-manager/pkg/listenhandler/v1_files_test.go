package listenhandler

import (
	"errors"
	"strings"

	logrustest "github.com/sirupsen/logrus/hooks/test"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/storagehandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_v1FilesPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		customerID    uuid.UUID
		ownerID       uuid.UUID
		referenceType file.ReferenceType
		referenceID   uuid.UUID
		fileType      file.Type
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
			fileType:      file.TypeNone,
			fileName:      "test",
			detail:        "test detail",
			filename:      "test_filename.txt",
			bucketName:    "test_bucket",
			filepath:      "test/file/path",

			responseFile: &file.File{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9de3d544-1739-11ef-acf1-e7fe99b5d7d0"),
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9de3d544-1739-11ef-acf1-e7fe99b5d7d0","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","bucket_name":"","filename":"","filepath":"","filesize":0,"uri_bucket":"","uri_download":"","tm_download_expire":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockStorage := storagehandler.NewMockStorageHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				storageHandler: mockStorage,
			}

			mockStorage.EXPECT().FileCreate(gomock.Any(), tt.customerID, tt.ownerID, tt.referenceType, tt.referenceID, tt.fileType, tt.fileName, tt.detail, tt.filename, tt.bucketName, tt.filepath).Return(tt.responseFile, nil)

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

		expectFilters map[file.Field]any
		responseFiles []*file.File

		expectRes *sock.Response
	}{
		{
			"1 item",
			&sock.Request{
				URI:      "/v1/files?page_token=2020-10-10T03:30:17.000000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"bd47c576-15ea-11ef-93f4-7b6a665b785d","deleted":false}`),
			},

			"2020-10-10T03:30:17.000000Z",
			10,
			map[file.Field]any{
				file.FieldCustomerID: uuid.FromStringOrNil("bd47c576-15ea-11ef-93f4-7b6a665b785d"),
				file.FieldDeleted:    false,
			},

			[]*file.File{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("bec1be20-15ea-11ef-ab62-ab3b98e4ee3c"),
						CustomerID: uuid.FromStringOrNil("bd47c576-15ea-11ef-93f4-7b6a665b785d")},
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"bec1be20-15ea-11ef-ab62-ab3b98e4ee3c","customer_id":"bd47c576-15ea-11ef-93f4-7b6a665b785d","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","bucket_name":"","filename":"","filepath":"","filesize":0,"uri_bucket":"","uri_download":"","tm_download_expire":null,"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockStorage := storagehandler.NewMockStorageHandler(mc)
			h := &listenHandler{
				sockHandler:    mockSock,
				storageHandler: mockStorage,
			}

			mockStorage.EXPECT().FileList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(tt.responseFiles, nil)

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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a5db58a-15eb-11ef-b669-bba0fb7a717d"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2a5db58a-15eb-11ef-b669-bba0fb7a717d","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","bucket_name":"","filename":"","filepath":"","filesize":0,"uri_bucket":"","uri_download":"","tm_download_expire":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockStorage := storagehandler.NewMockStorageHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				storageHandler: mockStorage,
			}

			mockStorage.EXPECT().FileGet(gomock.Any(), tt.responseFile.Identity.ID).Return(tt.responseFile, nil)

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
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("97a4e91a-15eb-11ef-bf44-eb05a9976a61"),
				},
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"97a4e91a-15eb-11ef-bf44-eb05a9976a61","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","account_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","bucket_name":"","filename":"","filepath":"","filesize":0,"uri_bucket":"","uri_download":"","tm_download_expire":null,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
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

func Test_v1FilesContentPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockStorage := storagehandler.NewMockStorageHandler(mc)
	h := &listenHandler{
		sockHandler:    sockhandler.NewMockSockHandler(mc),
		storageHandler: mockStorage,
	}

	customerID := uuid.FromStringOrNil("4d85dc7a-153e-11ef-9221-13c46bd56c4c")
	ownerID := uuid.FromStringOrNil("4dc51b42-153e-11ef-94b6-63fbe2cffaae")

	mockStorage.EXPECT().FileCreateContent(
		gomock.Any(), customerID, ownerID, file.TypeMedia, "test", "test detail", "media-0.jpg", "image/jpeg", []byte("abc"),
	).Return(&file.File{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("9de3d544-1739-11ef-acf1-e7fe99b5d7d0")}}, nil)

	res, err := h.processRequest(&sock.Request{
		URI:      "/v1/files/content",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"customer_id":"4d85dc7a-153e-11ef-9221-13c46bd56c4c","owner_id":"4dc51b42-153e-11ef-94b6-63fbe2cffaae","type":"media","name":"test","detail":"test detail","filename":"media-0.jpg","content_type":"image/jpeg","content":"YWJj"}`),
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res == nil || res.StatusCode != 200 || !strings.Contains(string(res.Data), "9de3d544-1739-11ef-acf1-e7fe99b5d7d0") {
		t.Errorf("Wrong match. got: %v", res)
	}
}

func Test_v1FilesContentPost_errors(t *testing.T) {
	t.Run("malformed body", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &listenHandler{storageHandler: storagehandler.NewMockStorageHandler(mc)}
		_, err := h.processRequest(&sock.Request{URI: "/v1/files/content", Method: sock.RequestMethodPost, Data: []byte(`{"content": 5`)})
		if err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("handler error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockStorage := storagehandler.NewMockStorageHandler(mc)
		h := &listenHandler{storageHandler: mockStorage}
		mockStorage.EXPECT().FileCreateContent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("simulated"))

		_, err := h.processRequest(&sock.Request{URI: "/v1/files/content", Method: sock.RequestMethodPost, Data: []byte(`{}`)})
		if err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})
}

// Test_processRequest_contentNotLogged checks that neither the content handler nor the unknown route line
// writes the request data to the log.
func Test_processRequest_contentNotLogged(t *testing.T) {
	logHook := logrustest.NewGlobal()
	defer logHook.Reset()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockStorage := storagehandler.NewMockStorageHandler(mc)
	h := &listenHandler{storageHandler: mockStorage}
	mockStorage.EXPECT().FileCreateContent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("simulated"))

	fakeContent := "ZmFrZQ=="
	_, _ = h.processRequest(&sock.Request{URI: "/v1/files/content", Method: sock.RequestMethodPost, Data: []byte(`{"content":"` + fakeContent + `"}`)})
	_, _ = h.processRequest(&sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodPost, Data: []byte(`{"content":"` + fakeContent + `"}`)})

	for _, e := range logHook.AllEntries() {
		line, _ := e.String()
		if strings.Contains(line, fakeContent) {
			t.Errorf("Wrong match. the log carries the request data: %s", line)
		}
	}
}
