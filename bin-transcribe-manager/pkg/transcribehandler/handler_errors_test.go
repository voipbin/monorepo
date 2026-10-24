package transcribehandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cmrecording "monorepo/bin-call-manager/models/recording"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/dbhandler"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

func Test_isValidReference_rejects(t *testing.T) {
	refID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000101")
	deleted := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string

		referenceType transcribe.ReferenceType

		responseCall       *cmcall.Call
		responseConfbridge *cmconfbridge.Confbridge
		responseRecording  *cmrecording.Recording
		responseErr        error

		expectRes bool
	}{
		{
			name:          "call lookup failure",
			referenceType: transcribe.ReferenceTypeCall,
			responseErr:   fmt.Errorf("rpc failed"),
			expectRes:     false,
		},
		{
			name:          "call that already hung up",
			referenceType: transcribe.ReferenceTypeCall,
			responseCall:  &cmcall.Call{Status: cmcall.StatusHangup},
			expectRes:     false,
		},
		{
			name:          "call that is ringing is allowed",
			referenceType: transcribe.ReferenceTypeCall,
			responseCall:  &cmcall.Call{Status: cmcall.StatusRinging},
			expectRes:     true,
		},
		{
			name:          "call that is soft deleted",
			referenceType: transcribe.ReferenceTypeCall,
			responseCall:  &cmcall.Call{Status: cmcall.StatusProgressing, TMDelete: &deleted},
			expectRes:     false,
		},
		{
			name:          "confbridge lookup failure",
			referenceType: transcribe.ReferenceTypeConfbridge,
			responseErr:   fmt.Errorf("rpc failed"),
			expectRes:     false,
		},
		{
			name:               "confbridge that is soft deleted",
			referenceType:      transcribe.ReferenceTypeConfbridge,
			responseConfbridge: &cmconfbridge.Confbridge{TMDelete: &deleted},
			expectRes:          false,
		},
		{
			name:              "recording that exists",
			referenceType:     transcribe.ReferenceTypeRecording,
			responseRecording: &cmrecording.Recording{},
			expectRes:         true,
		},
		{
			name:          "recording lookup failure",
			referenceType: transcribe.ReferenceTypeRecording,
			responseErr:   fmt.Errorf("rpc failed"),
			expectRes:     false,
		},
		{
			name:          "unsupported type",
			referenceType: transcribe.ReferenceType("unknown"),
			expectRes:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &transcribeHandler{reqHandler: mockReq}
			ctx := context.Background()

			switch tt.referenceType {
			case transcribe.ReferenceTypeCall:
				mockReq.EXPECT().CallV1CallGet(ctx, refID).Return(tt.responseCall, tt.responseErr)
			case transcribe.ReferenceTypeConfbridge:
				mockReq.EXPECT().CallV1ConfbridgeGet(ctx, refID).Return(tt.responseConfbridge, tt.responseErr)
			case transcribe.ReferenceTypeRecording:
				mockReq.EXPECT().CallV1RecordingGet(ctx, refID).Return(tt.responseRecording, tt.responseErr)
			}

			res := h.isValidReference(ctx, tt.referenceType, refID)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Get_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000201")
	refID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000202")

	tests := []struct {
		name string

		dbErr error

		expectNotFound bool
	}{
		{"missing row is a typed not found", dbhandler.ErrNotFound, true},
		{"other failure is wrapped", fmt.Errorf("select failed"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name+" Get", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &transcribeHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().TranscribeGet(ctx, id).Return(nil, tt.dbErr)

			res, err := h.Get(ctx, id)
			assertLookupError(t, res == nil, err, tt.expectNotFound)
		})

		t.Run(tt.name+" GetByReferenceIDAndLanguage", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &transcribeHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().TranscribeGetByReferenceIDAndLanguage(ctx, refID, "en-US").Return(nil, tt.dbErr)

			res, err := h.GetByReferenceIDAndLanguage(ctx, refID, "en-US")
			assertLookupError(t, res == nil, err, tt.expectNotFound)
		})
	}
}

func assertLookupError(t *testing.T, resNil bool, err error, expectNotFound bool) {
	t.Helper()

	if err == nil {
		t.Fatalf("Wrong match. expect: err, got: ok")
	}
	if !resNil {
		t.Errorf("Wrong match. expect: nil result")
	}

	var ve *cerrors.VoipbinError
	gotTyped := stderrors.As(err, &ve) && ve.Status == cerrors.StatusNotFound
	if gotTyped != expectNotFound {
		t.Errorf("Wrong match. expect typed not found: %v, got: %v (err: %v)", expectNotFound, gotTyped, err)
	}
}

func Test_List_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &transcribeHandler{db: mockDB}
	ctx := context.Background()

	filters := map[transcribe.Field]any{transcribe.FieldCustomerID: uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000211")}
	mockDB.EXPECT().TranscribeList(ctx, uint64(5), "tok", filters).Return(nil, fmt.Errorf("list failed"))

	res, err := h.List(ctx, 5, "tok", filters)
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: nil and err, got: %v and %v", res, err)
	}
}

func Test_Create_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000221")
	customerID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000222")
	activeflowID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000223")

	tests := []struct {
		name string

		createErr error
		getErr    error
		varErr    error

		expectGet      bool
		expectVariable bool
		expectErr      bool
	}{
		{
			name:      "insert failure",
			createErr: fmt.Errorf("insert failed"),
			expectErr: true,
		},
		{
			name:      "lookup failure after insert",
			getErr:    fmt.Errorf("select failed"),
			expectGet: true,
			expectErr: true,
		},
		{
			name:           "variable failure does not fail the create",
			varErr:         fmt.Errorf("variable set failed"),
			expectGet:      true,
			expectVariable: true,
			expectErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &transcribeHandler{reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify}
			ctx := context.Background()

			created := &transcribe.Transcribe{
				Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
				Language: "en-US",
			}

			mockDB.EXPECT().TranscribeCreate(ctx, gomock.Any()).Return(tt.createErr)
			if tt.expectGet {
				if tt.getErr != nil {
					mockDB.EXPECT().TranscribeGet(ctx, id).Return(nil, tt.getErr)
				} else {
					mockDB.EXPECT().TranscribeGet(ctx, id).Return(created, nil)
				}
			}
			if tt.expectVariable {
				mockReq.EXPECT().FlowV1VariableSetVariable(ctx, activeflowID, gomock.Any()).Return(tt.varErr)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, transcribe.EventTypeTranscribeCreated, created)
			}

			res, err := h.Create(ctx, id, customerID, activeflowID, uuid.Nil, transcribe.ReferenceTypeCall, uuid.Nil, "en-US", transcribe.DirectionBoth, "", nil)
			if tt.expectErr {
				if err == nil || res != nil {
					t.Errorf("Wrong match. expect: nil and err, got: %v and %v", res, err)
				}
				return
			}
			if err != nil || res != created {
				t.Errorf("Wrong match. expect: created and ok, got: %v and %v", res, err)
			}
		})
	}
}

func Test_Delete_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000231")
	deleted := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string

		getErr     error
		listErr    error
		deleteErr  error
		already    bool
		expectList bool
		expectDel  bool
		expectErr  bool
	}{
		{
			name:      "lookup failure",
			getErr:    fmt.Errorf("select failed"),
			expectErr: true,
		},
		{
			name:    "an already deleted transcribe is returned as is",
			already: true,
		},
		{
			name:       "transcript list failure aborts the delete",
			listErr:    fmt.Errorf("list failed"),
			expectList: true,
			expectErr:  true,
		},
		{
			name:       "row delete failure",
			deleteErr:  fmt.Errorf("delete failed"),
			expectList: true,
			expectDel:  true,
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
			h := &transcribeHandler{db: mockDB, transcriptHandler: mockTranscript}
			ctx := context.Background()

			// done status: the stop-before-delete branch is skipped
			tr := &transcribe.Transcribe{Identity: commonidentity.Identity{ID: id}, Status: transcribe.StatusDone}
			if tt.already {
				tr.TMDelete = &deleted
			}

			if tt.getErr != nil {
				mockDB.EXPECT().TranscribeGet(ctx, id).Return(nil, tt.getErr)
			} else {
				mockDB.EXPECT().TranscribeGet(ctx, id).Return(tr, nil)
			}
			if tt.expectList {
				filters := map[transcript.Field]any{
					transcript.FieldTranscribeID: id,
					transcript.FieldDeleted:      false,
				}
				mockTranscript.EXPECT().List(ctx, uint64(1000), "", filters).Return(nil, tt.listErr)
			}
			if tt.expectDel {
				mockDB.EXPECT().TranscribeDelete(ctx, id).Return(tt.deleteErr)
			}

			res, err := h.Delete(ctx, id)
			if tt.expectErr {
				if err == nil || res != nil {
					t.Errorf("Wrong match. expect: nil and err, got: %v and %v", res, err)
				}
				return
			}
			if err != nil || res != tr {
				t.Errorf("Wrong match. expect the stored transcribe and ok, got: %v and %v", res, err)
			}
		})
	}
}

func Test_deleteTranscripts_continuesOnItemFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
	h := &transcribeHandler{transcriptHandler: mockTranscript}
	ctx := context.Background()

	transcribeID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000241")
	first := &transcript.Transcript{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000242")}}
	second := &transcript.Transcript{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000243")}}

	filters := map[transcript.Field]any{
		transcript.FieldTranscribeID: transcribeID,
		transcript.FieldDeleted:      false,
	}
	mockTranscript.EXPECT().List(ctx, uint64(1000), "", filters).Return([]*transcript.Transcript{first, second}, nil)
	mockTranscript.EXPECT().Delete(ctx, first.ID).Return(nil, fmt.Errorf("delete failed"))
	mockTranscript.EXPECT().Delete(ctx, second.ID).Return(second, nil)

	if err := h.deleteTranscripts(ctx, transcribeID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_UpdateStatus_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000251")

	tests := []struct {
		name string

		updateErr error
		getErr    error
	}{
		{"update failure", fmt.Errorf("update failed"), nil},
		{"lookup failure after update", nil, fmt.Errorf("select failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &transcribeHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().TranscribeUpdate(ctx, id, map[transcribe.Field]any{transcribe.FieldStatus: transcribe.StatusProgressing}).Return(tt.updateErr)
			if tt.updateErr == nil {
				mockDB.EXPECT().TranscribeGet(ctx, id).Return(nil, tt.getErr)
			}

			res, err := h.UpdateStatus(ctx, id, transcribe.StatusProgressing)
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: nil and err, got: %v and %v", res, err)
			}
		})
	}
}

func Test_dbDelete_errors(t *testing.T) {
	id := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000261")

	tests := []struct {
		name string

		deleteErr error
		getErr    error
	}{
		{"delete failure", fmt.Errorf("delete failed"), nil},
		{"lookup failure after delete", nil, fmt.Errorf("select failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &transcribeHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().TranscribeDelete(ctx, id).Return(tt.deleteErr)
			if tt.deleteErr == nil {
				mockDB.EXPECT().TranscribeGet(ctx, id).Return(nil, tt.getErr)
			}

			res, err := h.dbDelete(ctx, id)
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: nil and err, got: %v and %v", res, err)
			}
		})
	}
}

func Test_variableSet_branches(t *testing.T) {
	activeflowID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000271")
	tr := &transcribe.Transcribe{
		Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000272")},
		Language:  "ko-KR",
		Direction: transcribe.DirectionIn,
	}

	t.Run("nil activeflow id sets nothing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		// no expectation on the request handler: any call fails the test
		h := &transcribeHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

		if err := h.variableSet(context.Background(), uuid.Nil, tr); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("rpc failure is wrapped", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &transcribeHandler{reqHandler: mockReq}
		ctx := context.Background()

		rpcErr := fmt.Errorf("rpc failed")
		expectVariables := map[string]string{
			variableTranscribeID:        tr.ID.String(),
			variableTranscribeLanguage:  "ko-KR",
			variableTranscribeDirection: "in",
		}
		mockReq.EXPECT().FlowV1VariableSetVariable(ctx, activeflowID, expectVariables).Return(rpcErr)

		err := h.variableSet(ctx, activeflowID, tr)
		if !stderrors.Is(err, rpcErr) {
			t.Errorf("Wrong match. expect: wrapped rpc error, got: %v", err)
		}
	})
}

func Test_Event_listFailures(t *testing.T) {
	listErr := fmt.Errorf("list failed")
	targetID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000281")

	tests := []struct {
		name string

		filters map[transcribe.Field]any
		call    func(h *transcribeHandler, ctx context.Context) error
	}{
		{
			name:    "customer deleted",
			filters: map[transcribe.Field]any{transcribe.FieldCustomerID: targetID, transcribe.FieldDeleted: false},
			call: func(h *transcribeHandler, ctx context.Context) error {
				return h.EventCUCustomerDeleted(ctx, newTestCustomer(targetID))
			},
		},
		{
			name:    "call hangup",
			filters: map[transcribe.Field]any{transcribe.FieldReferenceID: targetID, transcribe.FieldDeleted: false},
			call: func(h *transcribeHandler, ctx context.Context) error {
				return h.EventCMCallHangup(ctx, &cmcall.Call{Identity: commonidentity.Identity{ID: targetID}})
			},
		},
		{
			name:    "confbridge terminated",
			filters: map[transcribe.Field]any{transcribe.FieldReferenceID: targetID, transcribe.FieldDeleted: false},
			call: func(h *transcribeHandler, ctx context.Context) error {
				return h.EventCMConfbridgeTerminated(ctx, &cmconfbridge.Confbridge{Identity: commonidentity.Identity{ID: targetID}})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &transcribeHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().TranscribeList(ctx, uint64(1000), "", tt.filters).Return(nil, listErr)

			if err := tt.call(h, ctx); !stderrors.Is(err, listErr) {
				t.Errorf("Wrong match. expect: wrapped list error, got: %v", err)
			}
		})
	}
}

func Test_Event_itemFailuresAreSkipped(t *testing.T) {
	targetID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000291")
	firstID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000292")
	secondID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000293")

	// the transcribes are of an unsupported reference type, so Stop fails for every item
	newList := func() []*transcribe.Transcribe {
		return []*transcribe.Transcribe{
			{Identity: commonidentity.Identity{ID: firstID}, ReferenceType: transcribe.ReferenceTypeRecording, Status: transcribe.StatusProgressing},
			{Identity: commonidentity.Identity{ID: secondID}, ReferenceType: transcribe.ReferenceTypeRecording, Status: transcribe.StatusProgressing},
		}
	}

	tests := []struct {
		name string

		filters map[transcribe.Field]any
		call    func(h *transcribeHandler, ctx context.Context) error
	}{
		{
			name:    "call hangup",
			filters: map[transcribe.Field]any{transcribe.FieldReferenceID: targetID, transcribe.FieldDeleted: false},
			call: func(h *transcribeHandler, ctx context.Context) error {
				return h.EventCMCallHangup(ctx, &cmcall.Call{Identity: commonidentity.Identity{ID: targetID}})
			},
		},
		{
			name:    "confbridge terminated",
			filters: map[transcribe.Field]any{transcribe.FieldReferenceID: targetID, transcribe.FieldDeleted: false},
			call: func(h *transcribeHandler, ctx context.Context) error {
				return h.EventCMConfbridgeTerminated(ctx, &cmconfbridge.Confbridge{Identity: commonidentity.Identity{ID: targetID}})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &transcribeHandler{db: mockDB}
			ctx := context.Background()

			list := newList()
			mockDB.EXPECT().TranscribeList(ctx, uint64(1000), "", tt.filters).Return(list, nil)
			// every Stop fails at the status lookup stage with an invalid reference type; the loop must visit both items
			mockDB.EXPECT().TranscribeGet(ctx, firstID).Return(list[0], nil)
			mockDB.EXPECT().TranscribeGet(ctx, secondID).Return(list[1], nil)

			if err := tt.call(h, ctx); err != nil {
				t.Errorf("Wrong match. expect: ok (item failures are skipped), got: %v", err)
			}
		})
	}
}

func Test_HealthCheck_lookupFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// the request handler has no expectation: a lookup failure must neither stop nor reschedule
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &transcribeHandler{reqHandler: mockReq, db: mockDB}
	ctx := context.Background()
	id := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000301")

	mockDB.EXPECT().TranscribeGet(ctx, id).Return(nil, fmt.Errorf("select failed"))

	h.HealthCheck(ctx, id, 0)
}

func Test_startLive_failures(t *testing.T) {
	id := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000401")
	customerID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000402")
	referenceID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000403")
	inStreamID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000404")

	dupFilters := map[transcribe.Field]any{
		transcribe.FieldCustomerID:  customerID,
		transcribe.FieldReferenceID: referenceID,
		transcribe.FieldLanguage:    "en-US",
		transcribe.FieldStatus:      transcribe.StatusProgressing,
		transcribe.FieldDeleted:     false,
	}

	t.Run("duplicate check failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &transcribeHandler{db: mockDB, streamingHandler: streaminghandler.NewMockStreamingHandler(mc)}
		ctx := context.Background()

		mockDB.EXPECT().TranscribeList(ctx, uint64(1), "", dupFilters).Return(nil, fmt.Errorf("list failed"))

		res, err := h.startLive(ctx, id, customerID, uuid.Nil, uuid.Nil, transcribe.ReferenceTypeCall, referenceID, "en-US", transcribe.DirectionBoth, "")
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: nil and err, got: %v and %v", res, err)
		}
	})

	t.Run("second leg failure rolls back the first leg and tolerates a rollback failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		mockStreaming := streaminghandler.NewMockStreamingHandler(mc)
		h := &transcribeHandler{db: mockDB, streamingHandler: mockStreaming}
		ctx := context.Background()

		startErr := fmt.Errorf("second leg failed")
		mockDB.EXPECT().TranscribeList(ctx, uint64(1), "", dupFilters).Return(nil, nil)
		gomock.InOrder(
			mockStreaming.EXPECT().Start(ctx, customerID, id, transcribe.ReferenceTypeCall, referenceID, "en-US", transcript.DirectionIn, transcribe.Provider("")).Return(newTestStreaming(inStreamID), nil),
			mockStreaming.EXPECT().Start(ctx, customerID, id, transcribe.ReferenceTypeCall, referenceID, "en-US", transcript.DirectionOut, transcribe.Provider("")).Return(nil, startErr),
			mockStreaming.EXPECT().Stop(ctx, inStreamID).Return(nil, fmt.Errorf("rollback failed")),
		)

		res, err := h.startLive(ctx, id, customerID, uuid.Nil, uuid.Nil, transcribe.ReferenceTypeCall, referenceID, "en-US", transcribe.DirectionBoth, "")
		if !stderrors.Is(err, startErr) || res != nil {
			t.Errorf("Wrong match. expect: nil and the leg error, got: %v and %v", res, err)
		}
	})

	t.Run("record creation failure rolls back the started leg", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		mockStreaming := streaminghandler.NewMockStreamingHandler(mc)
		h := &transcribeHandler{db: mockDB, streamingHandler: mockStreaming}
		ctx := context.Background()

		createErr := fmt.Errorf("insert failed")
		mockDB.EXPECT().TranscribeList(ctx, uint64(1), "", dupFilters).Return(nil, nil)
		gomock.InOrder(
			mockStreaming.EXPECT().Start(ctx, customerID, id, transcribe.ReferenceTypeCall, referenceID, "en-US", transcript.DirectionIn, transcribe.Provider("")).Return(newTestStreaming(inStreamID), nil),
			mockDB.EXPECT().TranscribeCreate(ctx, gomock.Any()).Return(createErr),
			mockStreaming.EXPECT().Stop(ctx, inStreamID).Return(newTestStreaming(inStreamID), nil),
		)

		res, err := h.startLive(ctx, id, customerID, uuid.Nil, uuid.Nil, transcribe.ReferenceTypeCall, referenceID, "en-US", transcribe.DirectionIn, "")
		if !stderrors.Is(err, createErr) || res != nil {
			t.Errorf("Wrong match. expect: nil and the create error, got: %v and %v", res, err)
		}
	})
}

func Test_startRecording_failures(t *testing.T) {
	id := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000501")
	customerID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000502")
	recordingID := uuid.FromStringOrNil("6c1d0a10-0000-11ee-8d11-000000000503")

	tests := []struct {
		name string

		createErr error
		recordErr error
		updateErr error

		expectRecording bool
		expectUpdate    bool
	}{
		{
			name:      "transcribe creation failure",
			createErr: fmt.Errorf("insert failed"),
		},
		{
			name:            "recording transcription failure",
			recordErr:       fmt.Errorf("stt failed"),
			expectRecording: true,
		},
		{
			name:            "status update failure",
			updateErr:       fmt.Errorf("update failed"),
			expectRecording: true,
			expectUpdate:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)
			h := &transcribeHandler{db: mockDB, notifyHandler: mockNotify, transcriptHandler: mockTranscript}
			ctx := context.Background()

			created := &transcribe.Transcribe{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}}

			mockDB.EXPECT().TranscribeGetByReferenceIDAndLanguage(ctx, recordingID, "en-US").Return(nil, dbhandler.ErrNotFound)
			mockDB.EXPECT().TranscribeCreate(ctx, gomock.Any()).Return(tt.createErr)
			if tt.createErr == nil {
				mockDB.EXPECT().TranscribeGet(ctx, id).Return(created, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, transcribe.EventTypeTranscribeCreated, created)
			}
			if tt.expectRecording {
				mockTranscript.EXPECT().Recording(ctx, customerID, id, recordingID, "en-US").Return(nil, tt.recordErr)
			}
			if tt.expectUpdate {
				mockDB.EXPECT().TranscribeUpdate(ctx, id, map[transcribe.Field]any{transcribe.FieldStatus: transcribe.StatusDone}).Return(tt.updateErr)
			}

			res, err := h.startRecording(ctx, id, false, customerID, uuid.Nil, uuid.Nil, recordingID, "en-US", "")
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: nil and err, got: %v and %v", res, err)
			}
		})
	}
}
