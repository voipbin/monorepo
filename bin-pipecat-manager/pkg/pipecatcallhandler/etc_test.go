package pipecatcallhandler

import (
	"context"
	"errors"
	"sync"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func TestSendMessage(t *testing.T) {
	tests := []struct {
		name string

		id             uuid.UUID
		messageID      string
		messageText    string
		runImmediately bool
		audioResponse  bool

		session        *pipecatcall.Session
		sessionGetErr  error
		sendRTVIErr    error
		createdUUID    uuid.UUID

		expectErr bool
	}{
		{
			name: "success",

			id:             uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
			messageID:      "msg-123",
			messageText:    "Hello, world!",
			runImmediately: true,
			audioResponse:  true,

			session: &pipecatcall.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
					CustomerID: uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
				},
				PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"),
			},
			sessionGetErr: nil,
			sendRTVIErr:   nil,
			createdUUID:   uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),

			expectErr: false,
		},
		{
			name: "session not found",

			id:             uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
			messageID:      "msg-123",
			messageText:    "Hello, world!",
			runImmediately: true,
			audioResponse:  true,

			session:       nil,
			sessionGetErr: errors.New("session not found"),
			sendRTVIErr:   nil,
			createdUUID:   uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),

			expectErr: true,
		},
		{
			name: "send rtvi error",

			id:             uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
			messageID:      "msg-123",
			messageText:    "Hello, world!",
			runImmediately: true,
			audioResponse:  true,

			session: &pipecatcall.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
					CustomerID: uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
				},
				PipecatcallReferenceType: pipecatcall.ReferenceTypeAICall,
				PipecatcallReferenceID:   uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"),
			},
			sessionGetErr: nil,
			sendRTVIErr:   errors.New("send error"),
			createdUUID:   uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockPipecatframe := NewMockPipecatframeHandler(mc)

			h := &pipecatcallHandler{
				utilHandler:         mockUtil,
				pipecatframeHandler: mockPipecatframe,
				mapPipecatcallSession: map[uuid.UUID]*pipecatcall.Session{},
				muPipecatcallSession:  sync.Mutex{},
			}

			ctx := context.Background()

			if tt.session != nil {
				h.mapPipecatcallSession[tt.session.ID] = tt.session
			}

			if tt.sessionGetErr == nil {
				mockUtil.EXPECT().UUIDCreate().Return(tt.createdUUID)
				mockPipecatframe.EXPECT().SendRTVIText(
					tt.session,
					tt.messageID,
					tt.messageText,
					tt.runImmediately,
					tt.audioResponse,
				).Return(tt.sendRTVIErr)
			}

			result, err := h.SendMessage(ctx, tt.id, tt.messageID, tt.messageText, tt.runImmediately, tt.audioResponse)

			if tt.expectErr {
				if err == nil {
					t.Errorf("SendMessage() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("SendMessage() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("SendMessage() returned nil result")
				return
			}

			if result.ID != tt.createdUUID {
				t.Errorf("SendMessage() ID = %v, want %v", result.ID, tt.createdUUID)
			}

			if result.Text != tt.messageText {
				t.Errorf("SendMessage() Text = %v, want %v", result.Text, tt.messageText)
			}

			if result.PipecatcallID != tt.session.ID {
				t.Errorf("SendMessage() PipecatcallID = %v, want %v", result.PipecatcallID, tt.session.ID)
			}
		})
	}
}
