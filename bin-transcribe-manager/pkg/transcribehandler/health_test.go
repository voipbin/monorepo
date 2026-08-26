package transcribehandler

import (
	"context"
	stderrors "errors"
	"testing"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"

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
				TMDelete:      nil,
			},
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3f7775c7-af75-4fa7-85f2-3e6e9d27663f"),
				},
				Status:   cmcall.StatusProgressing,
				TMHangup: nil,
				TMDelete: nil,
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
				TMDelete:      nil,
			},
			responseConfbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fe812e35-b30e-4b38-9705-4cc22cbe3678"),
				},
				TMDelete: nil,
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
				TMDelete:      nil,
			},
			responseCall: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6f459316-678f-4c22-aa16-5f91cd8c4a2d"),
				},
				Status:   cmcall.StatusHangup,
				TMDelete: nil,
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
				TMDelete:      nil,
			},
			responseConfbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cdc3edd9-ee22-43ec-a598-4f27c896a4ca"),
				},
				TMDelete: func() *time.Time { t := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC); return &t }(),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &transcribeHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
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

// Test_HealthCheck_maxRetryExceeded_stopFails_reschedules verifies that when
// the retry count is exceeded and the resulting Stop() genuinely fails(a
// streaming session could not be stopped), HealthCheck does not just give up:
// it reschedules another health check at the same retryCount so the existing
// periodic health-check mechanism keeps retrying, instead of leaving the
// transcribe stuck in progressing forever with nothing left to ever retry it.
func Test_HealthCheck_maxRetryExceeded_stopFails_reschedules(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

	id := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000001")
	stID := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000002")
	retryCount := defaultHealthMaxRetryCount + 1

	tr := &transcribe.Transcribe{
		Identity: commonidentity.Identity{
			ID: id,
		},
		ReferenceType: transcribe.ReferenceTypeCall,
		Status:        transcribe.StatusProgressing,
		StreamingIDs:  []uuid.UUID{stID},
	}

	h := &transcribeHandler{
		reqHandler:       mockReq,
		db:               mockDB,
		notifyHandler:    mockNotify,
		streamingHandler: mockStreaming,
	}
	ctx := context.Background()

	// Get is called twice: once by HealthCheck's own unconditional
	// done/deleted guard (which now runs before the max-retry branch), and
	// once more inside the nested h.Stop() call triggered by
	// stopOrReschedule.
	mockDB.EXPECT().TranscribeGet(ctx, id).Return(tr, nil).Times(2)
	mockStreaming.EXPECT().Stop(ctx, stID).Return(nil, stderrors.New("could not stop the external media"))
	// stopLive wraps this into a reasonStreamingStopFailed
	// FailedPrecondition error, which is the one retryable case - so
	// stopOrReschedule reschedules at retryCount+1.
	mockReq.EXPECT().TranscribeV1TranscribeHealthCheck(ctx, id, defaultHealthDelay, retryCount+1).Return(nil)

	h.HealthCheck(ctx, id, retryCount)

	time.Sleep(time.Millisecond * 100)
}

// Test_HealthCheck_maxRetryExceeded_stopSucceeds_noReschedule verifies that
// when Stop() succeeds, HealthCheck does not reschedule another health check
// (the transcribe is now done, so there is nothing left to check).
func Test_HealthCheck_maxRetryExceeded_stopSucceeds_noReschedule(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

	id := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000003")
	stID := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000004")
	retryCount := defaultHealthMaxRetryCount + 1

	tr := &transcribe.Transcribe{
		Identity: commonidentity.Identity{
			ID: id,
		},
		ReferenceType: transcribe.ReferenceTypeCall,
		Status:        transcribe.StatusProgressing,
		StreamingIDs:  []uuid.UUID{stID},
	}

	h := &transcribeHandler{
		reqHandler:       mockReq,
		db:               mockDB,
		notifyHandler:    mockNotify,
		streamingHandler: mockStreaming,
	}
	ctx := context.Background()

	// Get is called twice with the exact (ctx, id) matcher: once by
	// HealthCheck's own unconditional done/deleted guard, and once more
	// inside the nested h.Stop() call triggered by stopOrReschedule. A third,
	// separate Get (matched via gomock.Any() below) happens inside
	// UpdateStatus's post-update refetch.
	mockDB.EXPECT().TranscribeGet(ctx, id).Return(tr, nil).Times(2)
	mockStreaming.EXPECT().Stop(ctx, stID).Return(&streaming.Streaming{}, nil)
	mockDB.EXPECT().TranscribeUpdate(ctx, id, map[transcribe.Field]any{
		transcribe.FieldStatus: transcribe.StatusDone,
	}).Return(nil)
	mockDB.EXPECT().TranscribeGet(gomock.Any(), id).Return(tr, nil)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tr.CustomerID, transcribe.EventTypeTranscribeDone, tr)
	// deliberately no mockReq.EXPECT().TranscribeV1TranscribeHealthCheck(...): a
	// successful stop must not reschedule another health check.

	h.HealthCheck(ctx, id, retryCount)

	time.Sleep(time.Millisecond * 100)
}

// Test_HealthCheck_referenceGetError_stopFails_reschedules verifies the same
// reschedule-on-genuine-stop-failure behavior for the "could not get
// reference call info" branch, not just the max-retry-exceeded branch.
func Test_HealthCheck_referenceGetError_stopFails_reschedules(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

	id := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000005")
	stID := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000006")
	referenceID := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000007")
	retryCount := 0

	tr := &transcribe.Transcribe{
		Identity: commonidentity.Identity{
			ID: id,
		},
		ReferenceType: transcribe.ReferenceTypeCall,
		ReferenceID:   referenceID,
		Status:        transcribe.StatusProgressing,
		StreamingIDs:  []uuid.UUID{stID},
	}

	h := &transcribeHandler{
		reqHandler:       mockReq,
		db:               mockDB,
		notifyHandler:    mockNotify,
		streamingHandler: mockStreaming,
	}
	ctx := context.Background()

	// once for HealthCheck's own validation Get, once more inside the nested
	// h.Stop() call triggered by the reference-lookup failure.
	mockDB.EXPECT().TranscribeGet(ctx, id).Return(tr, nil).Times(2)
	mockReq.EXPECT().CallV1CallGet(ctx, referenceID).Return(nil, stderrors.New("call not found"))
	mockStreaming.EXPECT().Stop(ctx, stID).Return(nil, stderrors.New("could not stop the external media"))
	// stopLive wraps this into a reasonStreamingStopFailed FailedPrecondition
	// error, which is the one retryable case - so stopOrReschedule
	// reschedules at retryCount+1.
	mockReq.EXPECT().TranscribeV1TranscribeHealthCheck(ctx, id, defaultHealthDelay, retryCount+1).Return(nil)

	h.HealthCheck(ctx, id, retryCount)

	time.Sleep(time.Millisecond * 100)
}

// Test_HealthCheck_maxRetryExceeded_permanentStopFailure_noReschedule verifies
// that when Stop() fails with a non-retryable error (i.e. anything other than
// the specific reasonStreamingStopFailed FailedPrecondition), stopOrReschedule
// gives up instead of rescheduling - retrying the exact same permanent
// failure (e.g. an invalid reference type) on a later health check could
// never succeed, so it must not loop forever.
func Test_HealthCheck_maxRetryExceeded_permanentStopFailure_noReschedule(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	id := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000008")
	retryCount := defaultHealthMaxRetryCount + 1

	tr := &transcribe.Transcribe{
		Identity: commonidentity.Identity{
			ID: id,
		},
		// Not ReferenceTypeCall/ReferenceTypeConfbridge, so Stop()'s switch
		// default branch returns a permanent
		// cerrors.InvalidArgument("INVALID_REFERENCE_TYPE") error - not the
		// retryable reasonStreamingStopFailed - without needing a streaming
		// handler mock at all.
		ReferenceType: transcribe.ReferenceTypeRecording,
		Status:        transcribe.StatusProgressing,
	}

	h := &transcribeHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	mockDB.EXPECT().TranscribeGet(ctx, id).Return(tr, nil).Times(2)
	// deliberately no mockReq.EXPECT().TranscribeV1TranscribeHealthCheck(...):
	// a permanent failure must not be rescheduled.

	h.HealthCheck(ctx, id, retryCount)

	time.Sleep(time.Millisecond * 100)
}

// Test_HealthCheck_maxRetryExceeded_softDeleted_noReschedule verifies that a
// soft-deleted transcribe (TMDelete != nil) never reaches stopOrReschedule,
// even when retryCount has already exceeded the max. The done/deleted guard
// at the top of HealthCheck must run unconditionally, before the max-retry
// branch: TranscribeGet does not filter out soft-deleted rows, so without
// this ordering a soft-deleted-but-still-"progressing" transcribe could be
// rescheduled by this health check forever.
func Test_HealthCheck_maxRetryExceeded_softDeleted_noReschedule(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	id := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-000000000009")
	retryCount := defaultHealthMaxRetryCount + 1
	deletedAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	tr := &transcribe.Transcribe{
		Identity: commonidentity.Identity{
			ID: id,
		},
		ReferenceType: transcribe.ReferenceTypeCall,
		Status:        transcribe.StatusProgressing,
		TMDelete:      &deletedAt,
	}

	h := &transcribeHandler{
		reqHandler:    mockReq,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	// Only the single top-level guard Get - Stop() must never be reached
	// (no streamingHandler is even wired up on this handler), so there must
	// be no second Get from inside it.
	mockDB.EXPECT().TranscribeGet(ctx, id).Return(tr, nil).Times(1)
	// deliberately no CallV1CallGet and no
	// TranscribeV1TranscribeHealthCheck expectations: a soft-deleted
	// transcribe must short-circuit before ever reaching stopOrReschedule.

	h.HealthCheck(ctx, id, retryCount)

	time.Sleep(time.Millisecond * 100)
}

// Test_HealthCheck_stopReschedule_capEnforced verifies that stopOrReschedule
// stops rescheduling once defaultStopRescheduleMaxRetryCount is reached, even
// though Stop() keeps failing with the retryable reasonStreamingStopFailed
// error - otherwise a persistent stop failure (e.g. call-manager unreachable
// indefinitely) would reschedule forever.
func Test_HealthCheck_stopReschedule_capEnforced(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

	id := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-00000000000a")
	stID := uuid.FromStringOrNil("a1b2c3d4-0000-4000-8000-00000000000b")
	retryCount := defaultStopRescheduleMaxRetryCount

	tr := &transcribe.Transcribe{
		Identity: commonidentity.Identity{
			ID: id,
		},
		ReferenceType: transcribe.ReferenceTypeCall,
		Status:        transcribe.StatusProgressing,
		StreamingIDs:  []uuid.UUID{stID},
	}

	h := &transcribeHandler{
		reqHandler:       mockReq,
		db:               mockDB,
		notifyHandler:    mockNotify,
		streamingHandler: mockStreaming,
	}
	ctx := context.Background()

	mockDB.EXPECT().TranscribeGet(ctx, id).Return(tr, nil).Times(2)
	mockStreaming.EXPECT().Stop(ctx, stID).Return(nil, stderrors.New("could not stop the external media"))
	// deliberately no mockReq.EXPECT().TranscribeV1TranscribeHealthCheck(...):
	// retryCount has already reached the cap, so no further reschedule
	// should be requested.

	h.HealthCheck(ctx, id, retryCount)

	time.Sleep(time.Millisecond * 100)
}
