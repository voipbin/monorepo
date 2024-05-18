package listenhandler

import (
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/storagehandler"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_v1FilesPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		customerID    uuid.UUID
		ownerID       uuid.UUID
		referenceType file.ReferenceType
		referenceID   uuid.UUID
		fileName      string
		detail        string
		bucketName    string
		filepath      string
	}{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/files",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"4d85dc7a-153e-11ef-9221-13c46bd56c4c", "owner_id":"4dc51b42-153e-11ef-94b6-63fbe2cffaae", "reference_type":"recording", "reference_id":"4df207d8-153e-11ef-8e6d-9fc4e34455ba","name":"test","detail":"test detail","bucket_name":"test_bucket","filepath":"test/file/path"}`),
			},

			customerID:    uuid.FromStringOrNil("4d85dc7a-153e-11ef-9221-13c46bd56c4c"),
			ownerID:       uuid.FromStringOrNil("4dc51b42-153e-11ef-94b6-63fbe2cffaae"),
			referenceType: file.ReferenceTypeRecording,
			referenceID:   uuid.FromStringOrNil("4df207d8-153e-11ef-8e6d-9fc4e34455ba"),
			fileName:      "test",
			detail:        "test detail",
			bucketName:    "test_bucket",
			filepath:      "test/file/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockStorageHandler := storagehandler.NewMockStorageHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				storageHandler: mockStorageHandler,
			}

			mockStorageHandler.EXPECT().FileCreate(gomock.Any(), tt.customerID, tt.ownerID, tt.referenceType, tt.referenceID, tt.fileName, tt.detail, tt.bucketName, tt.filepath).Return(&file.File{}, nil)

			if _, err := h.processRequest(tt.request); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
