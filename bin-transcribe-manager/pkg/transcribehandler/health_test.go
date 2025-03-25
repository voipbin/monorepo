package transcribehandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-call-manager/models/call"
	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_HealthCheck(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		retryCount int

		responseTranscribe *transcribe.Transcribe
		responseCall       *cmcall.Call
		responseConfbridge *cmconfbridge.Confbridge

		expectRetryCount int
	}{
		{
			name: "reference type call",

			id:         uuid.FromStringOrNil("d9560fc8-fcfd-4e86-a336-aa9e2110bf51"),
			retryCount: 2,

			responseTranscribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d9560fc8-fcfd-4e86-a336-aa9e2110bf51"),
				},
				ReferenceType: transcribe.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("3f7775c7-af75-4fa7-85f2-3e6e9d27663f"),
				Status:        transcribe.StatusProgressing,
				TMDelete:      dbhandler.DefaultTimeStamp,
			},
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3f7775c7-af75-4fa7-85f2-3e6e9d27663f"),
				},
				Status:   cmcall.StatusProgressing,
				TMHangup: dbhandler.DefaultTimeStamp,
				TMDelete: dbhandler.DefaultTimeStamp,
			},

			expectRetryCount: 0,
		},
		{
			name: "reference type confbridge",

			id:         uuid.FromStringOrNil("1e04c9d8-2cc6-4b17-a0a4-0dbd0355ff2e"),
			retryCount: 2,

			responseTranscribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1e04c9d8-2cc6-4b17-a0a4-0dbd0355ff2e"),
				},
				ReferenceType: transcribe.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("fe812e35-b30e-4b38-9705-4cc22cbe3678"),
				Status:        transcribe.StatusProgressing,
				TMDelete:      dbhandler.DefaultTimeStamp,
			},
			responseConfbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fe812e35-b30e-4b38-9705-4cc22cbe3678"),
				},
				TMDelete: dbhandler.DefaultTimeStamp,
			},

			expectRetryCount: 0,
		},
		{
			name: "reference call ended",

			id:         uuid.FromStringOrNil("99b7a33f-a411-4d86-a613-f317036ef5aa"),
			retryCount: 0,

			responseTranscribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("99b7a33f-a411-4d86-a613-f317036ef5aa"),
				},
				ReferenceType: transcribe.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("6f459316-678f-4c22-aa16-5f91cd8c4a2d"),
				TMDelete:      dbhandler.DefaultTimeStamp,
			},
			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6f459316-678f-4c22-aa16-5f91cd8c4a2d"),
				},
				Status:   call.StatusHangup,
				TMDelete: dbhandler.DefaultTimeStamp,
			},

			expectRetryCount: 1,
		},
		{
			name: "reference confbridge ended",

			id:         uuid.FromStringOrNil("113e33b2-4ad5-4b35-aefd-e9674c9109bc"),
			retryCount: 0,

			responseTranscribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("113e33b2-4ad5-4b35-aefd-e9674c9109bc"),
				},
				ReferenceType: transcribe.ReferenceTypeConfbridge,
				ReferenceID:   uuid.FromStringOrNil("cdc3edd9-ee22-43ec-a598-4f27c896a4ca"),
				TMDelete:      dbhandler.DefaultTimeStamp,
			},
			responseConfbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdc3edd9-ee22-43ec-a598-4f27c896a4ca"),
				},
			},

			expectRetryCount: 1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)

			h := &transcribeHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotfiy,
			}
			ctx := context.Background()

			mockDB.EXPECT().TranscribeGet(ctx, tt.id).Return(tt.responseTranscribe, nil)
			switch tt.responseTranscribe.ReferenceType {
			case transcribe.ReferenceTypeCall:
				mockReq.EXPECT().CallV1CallGet(ctx, tt.responseTranscribe.ReferenceID).Return(tt.responseCall, nil)

			case transcribe.ReferenceTypeConfbridge:
				mockReq.EXPECT().CallV1ConfbridgeGet(ctx, tt.responseTranscribe.ReferenceID).Return(tt.responseConfbridge, nil)
			}

			mockReq.EXPECT().TranscribeV1TranscribeHealthCheck(ctx, tt.id, defaultHealthDelay, tt.expectRetryCount).Return(nil)

			h.HealthCheck(ctx, tt.id, tt.retryCount)

			time.Sleep(time.Millisecond * 100)
		})
	}
}
