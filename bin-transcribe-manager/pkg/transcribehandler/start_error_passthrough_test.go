package transcribehandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
)

// Test_Start_errorPassthrough checks that the failures of the inner steps of Start reach the
// caller as the same error (wrapped with context where the code adds it), and that nothing is
// created or kept when a step fails.
func Test_Start_errorPassthrough(t *testing.T) {
	customerID := uuid.FromStringOrNil("d1a1a1a1-0000-11ed-0000-000000000001")
	referenceID := uuid.FromStringOrNil("d1a1a1a1-0000-11ed-0000-000000000004")
	callerID := uuid.FromStringOrNil("d1a1a1a1-0000-11ed-0000-000000000099")
	generatedID := uuid.FromStringOrNil("d1a1a1a1-0000-11ed-0000-000000000098")
	streamingID := uuid.FromStringOrNil("d1a1a1a1-0000-11ed-0000-0000000000aa")

	boom := fmt.Errorf("boom")

	tests := []struct {
		name string

		id            uuid.UUID
		referenceType transcribe.ReferenceType

		setup func(ctx context.Context, mockReq *requesthandler.MockRequestHandler, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockStreaming *streaminghandler.MockStreamingHandler)

		expectErrContains string
	}{
		{
			name:          "the existing id lookup fails with an error other than not found",
			id:            callerID,
			referenceType: transcribe.ReferenceTypeCall,
			setup: func(ctx context.Context, mockReq *requesthandler.MockRequestHandler, _ *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, _ *streaminghandler.MockStreamingHandler) {
				mockReq.EXPECT().CallV1CallGet(ctx, referenceID).Return(&cmcall.Call{Status: cmcall.StatusProgressing}, nil)
				mockDB.EXPECT().TranscribeGet(ctx, callerID).Return(nil, boom)
			},
		},
		{
			name:          "the recording transcribe cannot be created",
			referenceType: transcribe.ReferenceTypeRecording,
			setup: func(ctx context.Context, mockReq *requesthandler.MockRequestHandler, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, _ *streaminghandler.MockStreamingHandler) {
				mockReq.EXPECT().CallV1RecordingGet(ctx, referenceID).Return(nil, nil)
				mockUtil.EXPECT().UUIDCreate().Return(generatedID)
				mockDB.EXPECT().TranscribeGetByReferenceIDAndLanguage(ctx, referenceID, "en-US").Return(nil, dbhandler.ErrNotFound)
				mockDB.EXPECT().TranscribeCreate(ctx, gomock.Any()).Return(boom)
			},
			expectErrContains: "could not start the recording transcribe. reference_id: " + referenceID.String(),
		},
		{
			name:          "the live streaming cannot be started",
			referenceType: transcribe.ReferenceTypeCall,
			setup: func(ctx context.Context, mockReq *requesthandler.MockRequestHandler, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockStreaming *streaminghandler.MockStreamingHandler) {
				mockReq.EXPECT().CallV1CallGet(ctx, referenceID).Return(&cmcall.Call{Status: cmcall.StatusProgressing}, nil)
				mockUtil.EXPECT().UUIDCreate().Return(generatedID)
				mockDB.EXPECT().TranscribeList(ctx, uint64(1), "", gomock.Any()).Return([]*transcribe.Transcribe{}, nil)
				mockStreaming.EXPECT().Start(ctx, customerID, generatedID, transcribe.ReferenceTypeCall, referenceID, "en-US", transcript.DirectionIn, transcribe.ProviderEmpty).Return(nil, boom)
			},
			expectErrContains: "could not start the live transcribe. reference_id: " + referenceID.String(),
		},
		{
			name:          "the live transcribe cannot be created and the started streaming is rolled back",
			referenceType: transcribe.ReferenceTypeCall,
			setup: func(ctx context.Context, mockReq *requesthandler.MockRequestHandler, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler, mockStreaming *streaminghandler.MockStreamingHandler) {
				mockReq.EXPECT().CallV1CallGet(ctx, referenceID).Return(&cmcall.Call{Status: cmcall.StatusProgressing}, nil)
				mockUtil.EXPECT().UUIDCreate().Return(generatedID)
				mockDB.EXPECT().TranscribeList(ctx, uint64(1), "", gomock.Any()).Return([]*transcribe.Transcribe{}, nil)
				mockStreaming.EXPECT().Start(ctx, customerID, generatedID, transcribe.ReferenceTypeCall, referenceID, "en-US", transcript.DirectionIn, transcribe.ProviderEmpty).Return(&streaming.Streaming{Identity: commonidentity.Identity{ID: streamingID}}, nil)
				mockDB.EXPECT().TranscribeCreate(ctx, gomock.Any()).Return(boom)
				mockStreaming.EXPECT().Stop(ctx, streamingID).Return(&streaming.Streaming{}, nil)
			},
			expectErrContains: "could not create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

			h := &transcribeHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				db:               mockDB,
				streamingHandler: mockStreaming,
			}
			ctx := context.Background()

			tt.setup(ctx, mockReq, mockUtil, mockDB, mockStreaming)

			res, err := h.Start(ctx, tt.id, customerID, uuid.Nil, uuid.Nil, tt.referenceType, referenceID, "en-US", transcribe.DirectionIn, transcribe.ProviderEmpty)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if !stderrors.Is(err, boom) {
				t.Errorf("Wrong match. expect: the original error to be kept, got: %v", err)
			}
			if tt.expectErrContains != "" && !strings.Contains(err.Error(), tt.expectErrContains) {
				t.Errorf("Wrong match. expect error containing: %q, got: %v", tt.expectErrContains, err)
			}
		})
	}
}
