package transcribehandler

import (
	"context"
	stderrors "errors"
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	pkgerrors "github.com/pkg/errors"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

func Test_TranscribingStop_call(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseTranscribe *transcribe.Transcribe

		expectRes *transcribe.Transcribe
	}{
		{
			"normal",

			uuid.FromStringOrNil("e28b21dc-8218-11ed-b54f-d394b81cda3b"),

			&transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e28b21dc-8218-11ed-b54f-d394b81cda3b"),
				},
				ReferenceType: transcribe.ReferenceTypeCall,
				Status:        transcribe.StatusProgressing,
			},

			&transcribe.Transcribe{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e28b21dc-8218-11ed-b54f-d394b81cda3b"),
				},
				ReferenceType: transcribe.ReferenceTypeCall,
				Status:        transcribe.StatusProgressing,
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
			}

			ctx := context.Background()

			mockDB.EXPECT().TranscribeGet(ctx, tt.id).Return(tt.responseTranscribe, nil)

			// streamingTranscribeStop
			mockDB.EXPECT().TranscribeUpdate(ctx, gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().TranscribeGet(gomock.Any(), gomock.Any()).Return(tt.responseTranscribe, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

			res, err := h.Stop(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// newStopLiveTestTranscribe returns a fresh transcribe for each subtest so
// subtests never share a pointer(a shared pointer risks one subtest's
// mutations leaking into another when run in parallel or in sequence).
func newStopLiveTestTranscribe() *transcribe.Transcribe {
	return &transcribe.Transcribe{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("58ad260c-8789-11ec-87ad-63d573434c69"),
		},
		StreamingIDs: []uuid.UUID{
			uuid.FromStringOrNil("d5824a14-8788-11ec-9e71-a7cedf6ca3e1"),
			uuid.FromStringOrNil("df402f8a-8788-11ec-a14b-af9efb78ed6a"),
		},
	}
}

func Test_stopLive(t *testing.T) {

	tests := []struct {
		name string

		// streamingErrs holds the error h.streamingHandler.Stop should return for the
		// streaming id at the same index of transcribe.StreamingIDs. nil means success.
		streamingErrs []error

		expectDone bool
	}{
		{
			"all streamings stop successfully, transcribe becomes done",

			[]error{nil, nil},

			true,
		},
		{
			"a streaming is already gone(typed not found), still becomes done",

			[]error{
				cerrors.NotFound(commonoutline.ServiceNameTranscribeManager, "STREAMING_NOT_FOUND", "The streaming was not found."),
				nil,
			},

			true,
		},
		{
			"a streaming is already gone via the legacy call-manager sentinel, still becomes done",

			[]error{
				nil,
				pkgerrors.Wrap(requesthandler.ErrNotFound, "could not stop the external media"),
			},

			true,
		},
		{
			"a streaming reports STT not configured(Unavailable), still becomes done",

			[]error{
				cerrors.Unavailable(commonoutline.ServiceNameTranscribeManager, streaminghandler.ErrSTTNotConfiguredReason, "No STT provider is configured."),
				nil,
			},

			true,
		},
		{
			"a streaming genuinely fails to stop, must not become done(zombie session invariant)",

			[]error{
				nil,
				stderrors.New("could not stop the external media"),
			},

			false,
		},
		{
			"all streamings genuinely fail to stop, must not become done",

			[]error{
				stderrors.New("could not stop the external media 1"),
				stderrors.New("could not stop the external media 2"),
			},

			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			tr := newStopLiveTestTranscribe()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

			h := &transcribeHandler{
				reqHandler:        mockReq,
				db:                mockDB,
				notifyHandler:     mockNotify,
				transcriptHandler: mockTranscript,
				streamingHandler:  mockStreaming,
			}

			ctx := context.Background()

			for i, stID := range tr.StreamingIDs {
				if tt.streamingErrs[i] != nil {
					mockStreaming.EXPECT().Stop(ctx, stID).Return(nil, tt.streamingErrs[i])
				} else {
					mockStreaming.EXPECT().Stop(ctx, stID).Return(&streaming.Streaming{}, nil)
				}
			}

			if tt.expectDone {
				mockDB.EXPECT().TranscribeUpdate(ctx, tr.ID, map[transcribe.Field]any{
					transcribe.FieldStatus: transcribe.StatusDone,
				}).Return(nil)
				mockDB.EXPECT().TranscribeGet(gomock.Any(), tr.ID).Return(tr, nil)
				mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tr.CustomerID, transcribe.EventTypeTranscribeDone, tr)
			}

			res, err := h.stopLive(ctx, tr)

			if !tt.expectDone {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok, res: %v", res)
				}
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}

				// verify error identity: callers(pkg/transcribehandler/stop.go's
				// CLAUDE.md structured-error convention) rely on a typed
				// *cerrors.VoipbinError, not an opaque fmt.Errorf.
				var ve *cerrors.VoipbinError
				if !stderrors.As(err, &ve) {
					t.Fatalf("Wrong match. expect: a *cerrors.VoipbinError, got: %T (%v)", err, err)
				}
				if ve.Status != cerrors.StatusFailedPrecondition {
					t.Errorf("Wrong match. expect status: %s, got: %s", cerrors.StatusFailedPrecondition, ve.Status)
				}
				if ve.Reason != "STREAMING_STOP_FAILED" {
					t.Errorf("Wrong match. expect reason: STREAMING_STOP_FAILED, got: %s", ve.Reason)
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tr, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tr, res)
			}
		})
	}
}

func Test_isSafeToConsiderStopped(t *testing.T) {

	tests := []struct {
		name string
		err  error

		expectRes bool
	}{
		{
			"typed not found",
			cerrors.NotFound(commonoutline.ServiceNameTranscribeManager, "STREAMING_NOT_FOUND", "The streaming was not found."),
			true,
		},
		{
			"typed unavailable(STT not configured)",
			cerrors.Unavailable(commonoutline.ServiceNameTranscribeManager, streaminghandler.ErrSTTNotConfiguredReason, "No STT provider is configured."),
			true,
		},
		{
			// HIGH-1 regression test: StatusUnavailable alone must not be
			// treated as "safe to consider stopped". Only the specific
			// STT_NOT_CONFIGURED reason is safe; any other Unavailable
			// reason (e.g. a future typed-error migration of call-manager's
			// CallV1ExternalMediaStop surfacing a transient failure as
			// Unavailable) must still be treated as a genuine failure, or a
			// live streaming session could be misclassified as already
			// stopped.
			"typed unavailable with a different reason is a genuine failure",
			cerrors.Unavailable(commonoutline.ServiceNameTranscribeManager, "SOME_OTHER_TRANSIENT_FAILURE", "a transient failure unrelated to STT configuration"),
			false,
		},
		{
			"legacy sentinel wrapped",
			pkgerrors.Wrap(requesthandler.ErrNotFound, "could not stop the external media"),
			true,
		},
		{
			"legacy sentinel bare",
			requesthandler.ErrNotFound,
			true,
		},
		{
			"typed failed precondition is not safe to consider stopped",
			cerrors.FailedPrecondition(commonoutline.ServiceNameTranscribeManager, "SOME_OTHER_REASON", "some other failure"),
			false,
		},
		{
			"generic error is not safe to consider stopped",
			stderrors.New("could not stop the external media"),
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := isSafeToConsiderStopped(tt.err)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
