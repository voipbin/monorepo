package transcribehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

var (
	testHostID uuid.UUID = uuid.FromStringOrNil("f65bffa8-7f69-11ed-9240-8fe157bf9db2")
)

func Test_startRecording(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		activeflowID uuid.UUID
		onEndFlowID  uuid.UUID
		recordingID  uuid.UUID
		language     string

		responseUUID       uuid.UUID
		responseTranscribe *transcribe.Transcribe

		expectTranscribe *transcribe.Transcribe
		expectRes        *transcribe.Transcribe
	}{
		{
			name: "normal",

			customerID:   uuid.FromStringOrNil("60a5cdd6-7f69-11ed-bf1e-6f78fcd020b9"),
			activeflowID: uuid.FromStringOrNil("6e17688e-0924-11f0-831d-87f333f9b9ac"),
			onEndFlowID:  uuid.FromStringOrNil("6e3fc806-0924-11f0-b967-4bcc66199182"),
			recordingID:  uuid.FromStringOrNil("60f9f97e-7f69-11ed-9ae8-b727fc26712d"),
			language:     "en-US",

			responseUUID: uuid.FromStringOrNil("c5784176-7f69-11ed-8c45-d72bf90127e9"),
			responseTranscribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6130f79e-7f69-11ed-8138-8795762544e8"),
				},
			},

			expectTranscribe: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c5784176-7f69-11ed-8c45-d72bf90127e9"),
					CustomerID: uuid.FromStringOrNil("60a5cdd6-7f69-11ed-bf1e-6f78fcd020b9"),
				},
				ActiveflowID: uuid.FromStringOrNil("6e17688e-0924-11f0-831d-87f333f9b9ac"),
				OnEndFlowID:  uuid.FromStringOrNil("6e3fc806-0924-11f0-b967-4bcc66199182"),

				ReferenceType: transcribe.ReferenceTypeRecording,
				ReferenceID:   uuid.FromStringOrNil("60f9f97e-7f69-11ed-9ae8-b727fc26712d"),

				Status:    transcribe.StatusProgressing,
				HostID:    testHostID,
				Language:  "en-US",
				Direction: transcribe.DirectionBoth,

				StreamingIDs: []uuid.UUID{},
			},
			expectRes: &transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6130f79e-7f69-11ed-8138-8795762544e8"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

			h := &transcribeHandler{
				utilHandler:       mockUtil,
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockTranscript,

				hostID: testHostID,
			}

			ctx := context.Background()

			mockDB.EXPECT().TranscribeGetByReferenceIDAndLanguage(ctx, tt.recordingID, tt.language).Return(nil, fmt.Errorf(""))

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().TranscribeCreate(ctx, tt.expectTranscribe).Return(nil)
			mockDB.EXPECT().TranscribeGet(ctx, tt.responseUUID).Return(tt.responseTranscribe, nil)
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, gomock.Any()).Return(nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTranscribe.CustomerID, transcribe.EventTypeTranscribeCreated, tt.responseTranscribe)

			mockTranscript.EXPECT().Recording(ctx, tt.customerID, tt.responseTranscribe.ID, tt.recordingID, tt.language).Return([]*transcript.Transcript{}, nil)

			// update status
			mockDB.EXPECT().TranscribeUpdate(ctx, tt.responseTranscribe.ID, map[transcribe.Field]any{
				transcribe.FieldStatus: transcribe.StatusDone,
			}).Return(nil)
			mockDB.EXPECT().TranscribeGet(ctx, tt.responseTranscribe.ID).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseTranscribe.CustomerID, transcribe.EventTypeTranscribeDone, tt.responseTranscribe)

			res, err := h.startRecording(ctx, tt.customerID, tt.activeflowID, tt.onEndFlowID, tt.recordingID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
