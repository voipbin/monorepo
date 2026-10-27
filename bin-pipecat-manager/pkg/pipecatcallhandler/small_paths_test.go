package pipecatcallhandler

import (
	"context"
	stderrors "errors"
	"sync"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_shouldNotifyPipelineError_unknownCategory(t *testing.T) {
	// a category the classifier does not know is not shown to the customer, fatal or not handled elsewhere.
	if shouldNotifyPipelineError(message.ErrorCategory("not-a-category"), false, false) {
		t.Errorf("Wrong match. an unlisted category must not notify")
	}
	if !shouldNotifyPipelineError(message.ErrorCategory("not-a-category"), true, false) {
		t.Errorf("Wrong match. a fatal error must notify whatever the category")
	}
}

func Test_Get_notFoundMapsToTypedError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &pipecatcallHandler{db: mockDB}

	id := uuid.FromStringOrNil("a1b2c3d4-6501-4000-8000-000000000001")
	mockDB.EXPECT().PipecatcallGet(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Get(context.Background(), id)
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("Wrong error type. expect: *VoipbinError, got: %T (%v)", err, err)
	}
	if ve.Reason != "PIPECATCALL_NOT_FOUND" || ve.Status != cerrors.StatusNotFound {
		t.Errorf("Wrong error. expect: PIPECATCALL_NOT_FOUND not found, got: %s %v", ve.Reason, ve.Status)
	}
	if !stderrors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("Wrong error. the cause must stay reachable for the caller")
	}
}

func Test_SendMessage_recordsInReplyToMessageID(t *testing.T) {
	tests := []struct {
		name string

		messageID string

		expectPending uuid.UUID
	}{
		{
			name:          "a uuid message id is recorded",
			messageID:     "5b374a54-b48c-11f0-8c36-477d3f6baf0d",
			expectPending: uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),
		},
		{
			name:          "a message id that is not a uuid leaves the previous value",
			messageID:     "msg-123",
			expectPending: uuid.FromStringOrNil("11111111-1111-4111-8111-111111111111"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockPF := NewMockPipecatframeHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &pipecatcallHandler{
				utilHandler:           mockUtil,
				pipecatframeHandler:   mockPF,
				mapPipecatcallSession: map[uuid.UUID]*pipecatcall.Session{},
				muPipecatcallSession:  sync.Mutex{},
			}

			id := uuid.FromStringOrNil("a1b2c3d4-6502-4000-8000-000000000001")
			se := &pipecatcall.Session{Identity: commonidentity.Identity{ID: id}}
			se.SetPendingInReplyToMessageID(uuid.FromStringOrNil("11111111-1111-4111-8111-111111111111"))
			h.mapPipecatcallSession[id] = se

			mockPF.EXPECT().SendRTVIText(se, tt.messageID, "hello", false, false).Return(nil)
			mockUtil.EXPECT().UUIDCreate().Return(uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf01"))

			if _, err := h.SendMessage(context.Background(), id, tt.messageID, "hello", false, false); err != nil {
				t.Fatalf("Wrong error: %v", err)
			}
			if got := se.SnapshotPendingInReplyToMessageID(); got != tt.expectPending {
				t.Errorf("Wrong pending id. expect: %s, got: %s", tt.expectPending, got)
			}
		})
	}
}
