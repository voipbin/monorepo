package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	smbucketfile "gitlab.com/voipbin/bin-manager/storage-manager.git/models/bucketfile"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func TestRecordingfileGet(t *testing.T) {

	type test struct {
		name     string
		customer *cscustomer.Customer

		id uuid.UUID

		responseRecording  *cmrecording.Recording
		responseBucketFile *smbucketfile.BucketFile

		expectRes string
	}

	tests := []test{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("59a394e4-610e-11eb-b8c6-aff7333845f1"),

			&cmrecording.Recording{
				ID:         uuid.FromStringOrNil("59a394e4-610e-11eb-b8c6-aff7333845f1"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				Filenames: []string{
					"call_25b4a290-0f25-4b50-87bd-7174638ac906_2021-01-26T02:17:05Z",
				},
			},
			&smbucketfile.BucketFile{
				DownloadURI: "test.com/downloadlink.wav",
			},

			"test.com/downloadlink.wav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1RecordingGet(ctx, tt.id).Return(tt.responseRecording, nil)
			mockReq.EXPECT().StorageV1RecordingGet(ctx, tt.responseRecording.ID, 300000).Return(tt.responseBucketFile, nil)

			res, err := h.RecordingfileGet(ctx, tt.customer, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
