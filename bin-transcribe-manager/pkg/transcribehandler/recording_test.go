package transcribehandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"reflect"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
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
		provider     transcribe.Provider

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
			provider:     transcribe.ProviderEmpty,

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
				Provider:  transcribe.ProviderEmpty,

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

			res, err := h.startRecording(ctx, tt.responseUUID, false, tt.customerID, tt.activeflowID, tt.onEndFlowID, tt.recordingID, tt.language, tt.provider)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

// Test_startRecording_idOmitted_existingRow covers: id omitted + an
// existing transcribe for the same recording/language -- unchanged
// idempotent-return behavior: the existing row is returned as-is, no
// TRANSCRIBE_ALREADY_EXISTS_DIFFERENT_ID error.
func Test_startRecording_idOmitted_existingRow(t *testing.T) {
	customerID := uuid.FromStringOrNil("d1a1a1a1-0000-11ed-0000-000000000001")
	activeflowID := uuid.FromStringOrNil("d1a1a1a1-0000-11ed-0000-000000000002")
	onEndFlowID := uuid.FromStringOrNil("d1a1a1a1-0000-11ed-0000-000000000003")
	recordingID := uuid.FromStringOrNil("d1a1a1a1-0000-11ed-0000-000000000004")
	existingID := uuid.FromStringOrNil("d1a1a1a1-0000-11ed-0000-000000000005")

	existingTranscribe := &transcribe.Transcribe{
		Identity: commonidentity.Identity{ID: existingID},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &transcribeHandler{
		db: mockDB,
	}

	ctx := context.Background()

	mockDB.EXPECT().TranscribeGetByReferenceIDAndLanguage(ctx, recordingID, "en-US").Return(existingTranscribe, nil)

	// callerSpecifiedID == false: the resolved id (uuid.Nil is never passed
	// here since Start always resolves id before dispatch) is irrelevant --
	// only callerSpecifiedID drives the branch.
	res, err := h.startRecording(ctx, uuid.FromStringOrNil("d1a1a1a1-0000-11ed-0000-0000000000ff"), false, customerID, activeflowID, onEndFlowID, recordingID, "en-US", transcribe.ProviderEmpty)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, existingTranscribe) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", existingTranscribe, res)
	}
}

// Test_startRecording_callerSpecifiedID_alwaysConflicts covers: id provided
// + an existing transcribe for the same recording/language -- this is
// always a conflict (TRANSCRIBE_ALREADY_EXISTS_DIFFERENT_ID), never the
// idempotent return, per the B6/B10 corrections. The "matching id" case is
// unreachable (Start's pre-check guarantees no row has this id yet) and is
// intentionally not a test case here.
func Test_startRecording_callerSpecifiedID_alwaysConflicts(t *testing.T) {
	customerID := uuid.FromStringOrNil("d2a1a1a1-0000-11ed-0000-000000000001")
	activeflowID := uuid.FromStringOrNil("d2a1a1a1-0000-11ed-0000-000000000002")
	onEndFlowID := uuid.FromStringOrNil("d2a1a1a1-0000-11ed-0000-000000000003")
	recordingID := uuid.FromStringOrNil("d2a1a1a1-0000-11ed-0000-000000000004")
	callerID := uuid.FromStringOrNil("d2a1a1a1-0000-11ed-0000-000000000005")
	existingID := uuid.FromStringOrNil("d2a1a1a1-0000-11ed-0000-000000000006")

	existingTranscribe := &transcribe.Transcribe{
		Identity: commonidentity.Identity{ID: existingID},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &transcribeHandler{
		db: mockDB,
	}

	ctx := context.Background()

	mockDB.EXPECT().TranscribeGetByReferenceIDAndLanguage(ctx, recordingID, "en-US").Return(existingTranscribe, nil)

	res, err := h.startRecording(ctx, callerID, true, customerID, activeflowID, onEndFlowID, recordingID, "en-US", transcribe.ProviderEmpty)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok. res: %v", res)
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Errorf("Wrong error type. expect: VoipbinError, got: %T", err)
	} else {
		if ve.Status != cerrors.StatusFailedPrecondition {
			t.Errorf("Wrong status. expect: %s, got: %s", cerrors.StatusFailedPrecondition, ve.Status)
		}
		if ve.Reason != "TRANSCRIBE_ALREADY_EXISTS_DIFFERENT_ID" {
			t.Errorf("Wrong reason. expect: TRANSCRIBE_ALREADY_EXISTS_DIFFERENT_ID, got: %s", ve.Reason)
		}
	}
}
