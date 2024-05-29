package listenhandler

import (
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	compressfile "monorepo/bin-storage-manager/models/compressfile"
	"monorepo/bin-storage-manager/pkg/storagehandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_v1CompressfilesPost(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		referenceIDs []uuid.UUID
		fileIDs      []uuid.UUID

		responseCompress *compressfile.CompressFile
		expectRes        *rabbitmqhandler.Response
	}{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/compressfiles",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"reference_ids":["f2525e0e-1d6d-11ef-8d33-f3f47b464f43","25c5729e-1d6e-11ef-940b-0fa28944ca27"], "file_ids":["f27dacc6-1d6d-11ef-954e-73482b2c50cb","25e8ab92-1d6e-11ef-a6e2-8f552c006c72"]}`),
			},

			referenceIDs: []uuid.UUID{
				uuid.FromStringOrNil("f2525e0e-1d6d-11ef-8d33-f3f47b464f43"),
				uuid.FromStringOrNil("25c5729e-1d6e-11ef-940b-0fa28944ca27"),
			},
			fileIDs: []uuid.UUID{
				uuid.FromStringOrNil("f27dacc6-1d6d-11ef-954e-73482b2c50cb"),
				uuid.FromStringOrNil("25e8ab92-1d6e-11ef-a6e2-8f552c006c72"),
			},

			responseCompress: &compressfile.CompressFile{
				FileIDs: []uuid.UUID{
					uuid.FromStringOrNil("f2525e0e-1d6d-11ef-8d33-f3f47b464f43"),
					uuid.FromStringOrNil("25c5729e-1d6e-11ef-940b-0fa28944ca27"),
					uuid.FromStringOrNil("f27dacc6-1d6d-11ef-954e-73482b2c50cb"),
					uuid.FromStringOrNil("25e8ab92-1d6e-11ef-a6e2-8f552c006c72"),
				},
			},
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"file_ids":["f2525e0e-1d6d-11ef-8d33-f3f47b464f43","25c5729e-1d6e-11ef-940b-0fa28944ca27","f27dacc6-1d6d-11ef-954e-73482b2c50cb","25e8ab92-1d6e-11ef-a6e2-8f552c006c72"],"download_uri":"","tm_download_expire":""}`),
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
				rabbitSock:     mockSock,
				storageHandler: mockStorage,
			}

			mockStorage.EXPECT().CompressfileCreate(gomock.Any(), tt.referenceIDs, tt.fileIDs).Return(tt.responseCompress, nil)

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
