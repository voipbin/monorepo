package servicehandler

import (
	"context"
	"reflect"
	"testing"

	cmrecording "monorepo/bin-call-manager/models/recording"

	"monorepo/bin-common-handler/pkg/requesthandler"

	amagent "monorepo/bin-agent-manager/models/agent"
	smcompressfile "monorepo/bin-storage-manager/models/compressfile"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_RecordingfileGet(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent

		id uuid.UUID

		responseRecording   *cmrecording.Recording
		responseComressfile *smcompressfile.CompressFile

		expectReferenceIDs []uuid.UUID
		expectRes          string
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("59a394e4-610e-11eb-b8c6-aff7333845f1"),

			&cmrecording.Recording{
				ID:         uuid.FromStringOrNil("59a394e4-610e-11eb-b8c6-aff7333845f1"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Filenames: []string{
					"call_25b4a290-0f25-4b50-87bd-7174638ac906_2021-01-26T02:17:05Z",
				},
				TMDelete: defaultTimestamp,
			},
			&smcompressfile.CompressFile{
				DownloadURI: "test.com/downloadlink.wav",
			},

			[]uuid.UUID{
				uuid.FromStringOrNil("59a394e4-610e-11eb-b8c6-aff7333845f1"),
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
			mockReq.EXPECT().StorageV1CompressfileCreate(ctx, tt.expectReferenceIDs, []uuid.UUID{}, 300000).Return(tt.responseComressfile, nil)

			res, err := h.RecordingfileGet(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
