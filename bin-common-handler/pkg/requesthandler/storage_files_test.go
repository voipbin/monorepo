package requesthandler

import (
	"context"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	smfile "monorepo/bin-storage-manager/models/file"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_StorageV1FileCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		ownerID        uuid.UUID
		referenceType  smfile.ReferenceType
		referenceID    uuid.UUID
		fileName       string
		detail         string
		filename       string
		bucketName     string
		filepath       string
		requestTimeout int

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult *smfile.File
	}{
		{
			name: "normal",

			customerID:     uuid.FromStringOrNil("4edf2f7e-160e-11ef-9cee-7f2de117897d"),
			ownerID:        uuid.FromStringOrNil("4f3b8ecc-160e-11ef-8ec2-0bcbadd66f6f"),
			referenceType:  smfile.ReferenceTypeRecording,
			referenceID:    uuid.FromStringOrNil("4f6d6000-160e-11ef-a051-a7a6e34953db"),
			fileName:       "test name",
			detail:         "test detail",
			filename:       "test_filename.txt",
			bucketName:     "test_bucket",
			filepath:       "tmp/file/path",
			requestTimeout: 5000,

			expectTarget: "bin-manager.storage-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/files",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"4edf2f7e-160e-11ef-9cee-7f2de117897d","owner_id":"4f3b8ecc-160e-11ef-8ec2-0bcbadd66f6f","reference_type":"recording","reference_id":"4f6d6000-160e-11ef-a051-a7a6e34953db","name":"test name","detail":"test detail","filename":"test_filename.txt","bucket_name":"test_bucket","filepath":"tmp/file/path"}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e4784552-160e-11ef-b04c-73e4a6f7f798"}`),
			},
			expectResult: &smfile.File{
				ID: uuid.FromStringOrNil("e4784552-160e-11ef-b04c-73e4a6f7f798"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.StorageV1FileCreate(ctx, tt.customerID, tt.ownerID, tt.referenceType, tt.referenceID, tt.fileName, tt.detail, tt.filename, tt.bucketName, tt.filepath, tt.requestTimeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_StorageV1FileCreateWithDelay(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		ownerID       uuid.UUID
		referenceType smfile.ReferenceType
		referenceID   uuid.UUID
		fileName      string
		detail        string
		filename      string
		bucketName    string
		filepath      string
		delay         int

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult *smfile.File
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("785e19ca-1d91-11ef-8d6a-3bfa80d939d5"),
			ownerID:       uuid.FromStringOrNil("78a2b97c-1d91-11ef-8897-bb58d4f1853d"),
			referenceType: smfile.ReferenceTypeRecording,
			referenceID:   uuid.FromStringOrNil("78d6f98a-1d91-11ef-80d5-937f9fac88bd"),
			fileName:      "test name",
			detail:        "test detail",
			filename:      "test_filename.txt",
			bucketName:    "test_bucket",
			filepath:      "tmp/file/path",
			delay:         5000,

			expectTarget: "bin-manager.storage-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/files",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"785e19ca-1d91-11ef-8d6a-3bfa80d939d5","owner_id":"78a2b97c-1d91-11ef-8897-bb58d4f1853d","reference_type":"recording","reference_id":"78d6f98a-1d91-11ef-80d5-937f9fac88bd","name":"test name","detail":"test detail","filename":"test_filename.txt","bucket_name":"test_bucket","filepath":"tmp/file/path"}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"790613d2-1d91-11ef-b443-1f88ddd0d95b"}`),
			},
			expectResult: &smfile.File{
				ID: uuid.FromStringOrNil("790613d2-1d91-11ef-b443-1f88ddd0d95b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().PublishExchangeDelayedRequest(gomock.Any(), tt.expectTarget, tt.expectRequest, tt.delay).Return(nil)

			err := reqHandler.StorageV1FileCreateWithDelay(ctx, tt.customerID, tt.ownerID, tt.referenceType, tt.referenceID, tt.fileName, tt.detail, tt.filename, tt.bucketName, tt.filepath, tt.delay)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_StorageV1FileGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		response *sock.Response

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		expectResult  []smfile.File
	}{
		{
			"normal",

			"2020-09-20 03:23:20.995000",
			10,
			map[string]string{
				"customer_id": "31237c7c-1610-11ef-84b3-f728e90c5c3e",
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"308d1de0-1610-11ef-af26-cf10522110e0"},{"id":"30f09a46-1610-11ef-99a4-4b9fef0d8729"}]`),
			},

			"/v1/files?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10",
			"bin-manager.storage-manager.request",
			&sock.Request{
				URI:      "/v1/files?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10&filter_customer_id=31237c7c-1610-11ef-84b3-f728e90c5c3e",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			[]smfile.File{
				{
					ID: uuid.FromStringOrNil("308d1de0-1610-11ef-af26-cf10522110e0"),
				},
				{
					ID: uuid.FromStringOrNil("30f09a46-1610-11ef-99a4-4b9fef0d8729"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			reqHandler := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.StorageV1FileGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_StorageV1FileGet(t *testing.T) {

	tests := []struct {
		name string

		fileID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *smfile.File
	}{
		{
			"normal",

			uuid.FromStringOrNil("846be5e0-1610-11ef-9d6d-cfa226c15144"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"846be5e0-1610-11ef-9d6d-cfa226c15144"}`),
			},

			"bin-manager.storage-manager.request",
			&sock.Request{
				URI:      "/v1/files/846be5e0-1610-11ef-9d6d-cfa226c15144",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},
			&smfile.File{
				ID: uuid.FromStringOrNil("846be5e0-1610-11ef-9d6d-cfa226c15144"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.StorageV1FileGet(ctx, tt.fileID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_StorageV1FileDelete(t *testing.T) {

	tests := []struct {
		name string

		fileID         uuid.UUID
		requestTimeout int

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *smfile.File
	}{
		{
			"normal",

			uuid.FromStringOrNil("b0cf0e3c-1610-11ef-8e33-0b8cfeddd4f8"),
			5000,
			&sock.Response{
				StatusCode: 200,
				Data:       []byte(`{"id":"b0cf0e3c-1610-11ef-8e33-0b8cfeddd4f8"}`),
			},

			"bin-manager.storage-manager.request",
			&sock.Request{
				URI:      "/v1/files/b0cf0e3c-1610-11ef-8e33-0b8cfeddd4f8",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeNone,
			},
			&smfile.File{
				ID: uuid.FromStringOrNil("b0cf0e3c-1610-11ef-8e33-0b8cfeddd4f8"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.StorageV1FileDelete(ctx, tt.fileID, tt.requestTimeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
